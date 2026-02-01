package dict

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/huanfeng/wind_input/internal/candidate"
)

// DictManager 词库管理器
// 统一管理所有词库层的加载、保存和生命周期
type DictManager struct {
	mu sync.RWMutex

	// 数据目录（%APPDATA%\WindInput）
	dataDir string

	// 各层词库
	phraseLayer *PhraseLayer // Lv1: 特殊短语
	shadowLayer *ShadowLayer // Lv2: 用户修正
	userDict    *UserDict    // Lv3: 用户造词

	// 聚合词库
	compositeDict *CompositeDict

	// 系统词库适配器（由引擎加载后注册）
	systemLayers map[string]DictLayer
}

// NewDictManager 创建词库管理器
func NewDictManager(dataDir string) *DictManager {
	dm := &DictManager{
		dataDir:       dataDir,
		systemLayers:  make(map[string]DictLayer),
		compositeDict: NewCompositeDict(),
	}

	return dm
}

// Initialize 初始化所有词库层
func (dm *DictManager) Initialize() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 1. 初始化特殊短语层 (Lv1)
	phrasePath := filepath.Join(dm.dataDir, "phrases.yaml")
	dm.phraseLayer = NewPhraseLayer("phrases", phrasePath)
	if err := dm.phraseLayer.Load(); err != nil {
		log.Printf("[DictManager] 加载短语配置失败: %v", err)
		// 不返回错误，短语是可选的
	} else {
		log.Printf("[DictManager] 短语层加载成功: %d 短语, %d 命令",
			dm.phraseLayer.GetPhraseCount(), dm.phraseLayer.GetCommandCount())
	}
	dm.compositeDict.AddLayer(dm.phraseLayer)

	// 2. 初始化 Shadow 层 (Lv2)
	shadowPath := filepath.Join(dm.dataDir, "shadow.yaml")
	dm.shadowLayer = NewShadowLayer("shadow", shadowPath)
	if err := dm.shadowLayer.Load(); err != nil {
		log.Printf("[DictManager] 加载 Shadow 规则失败: %v", err)
	} else {
		log.Printf("[DictManager] Shadow 层加载成功: %d 规则", dm.shadowLayer.GetRuleCount())
	}
	// Shadow 层作为规则提供者，不作为搜索层
	dm.compositeDict.SetShadowProvider(dm.shadowLayer)

	// 3. 初始化用户词库层 (Lv3)
	userDictPath := filepath.Join(dm.dataDir, "user_words.txt")
	dm.userDict = NewUserDict("user", userDictPath)
	if err := dm.userDict.Load(); err != nil {
		log.Printf("[DictManager] 加载用户词库失败: %v", err)
	} else {
		log.Printf("[DictManager] 用户词库加载成功: %d 词条", dm.userDict.EntryCount())
	}
	dm.compositeDict.AddLayer(dm.userDict)

	return nil
}

// RegisterSystemLayer 注册系统词库层
// 由引擎加载码表后调用
func (dm *DictManager) RegisterSystemLayer(name string, layer DictLayer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.systemLayers[name] = layer
	dm.compositeDict.AddLayer(layer)
	log.Printf("[DictManager] 注册系统词库: %s", name)
}

// UnregisterSystemLayer 取消注册系统词库层
func (dm *DictManager) UnregisterSystemLayer(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, ok := dm.systemLayers[name]; ok {
		delete(dm.systemLayers, name)
		dm.compositeDict.RemoveLayer(name)
		log.Printf("[DictManager] 取消注册系统词库: %s", name)
	}
}

// GetCompositeDict 获取聚合词库
func (dm *DictManager) GetCompositeDict() *CompositeDict {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.compositeDict
}

// GetUserDict 获取用户词库（用于添加用户词）
func (dm *DictManager) GetUserDict() *UserDict {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.userDict
}

// GetShadowLayer 获取 Shadow 层（用于置顶/删除操作）
func (dm *DictManager) GetShadowLayer() *ShadowLayer {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.shadowLayer
}

// GetPhraseLayer 获取短语层
func (dm *DictManager) GetPhraseLayer() *PhraseLayer {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.phraseLayer
}

// AddUserWord 添加用户词
func (dm *DictManager) AddUserWord(code, text string, weight int) error {
	if dm.userDict == nil {
		return fmt.Errorf("用户词库未初始化")
	}
	return dm.userDict.Add(code, text, weight)
}

// TopWord 置顶词条
func (dm *DictManager) TopWord(code, word string) {
	if dm.shadowLayer != nil {
		dm.shadowLayer.Top(code, word)
	}
}

// DeleteWord 删除（隐藏）词条
func (dm *DictManager) DeleteWord(code, word string) {
	if dm.shadowLayer != nil {
		dm.shadowLayer.Delete(code, word)
	}
}

// Save 保存所有可写层
func (dm *DictManager) Save() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var errs []error

	if dm.userDict != nil {
		if err := dm.userDict.Save(); err != nil {
			errs = append(errs, fmt.Errorf("保存用户词库失败: %w", err))
		}
	}

	if dm.shadowLayer != nil && dm.shadowLayer.IsDirty() {
		if err := dm.shadowLayer.Save(); err != nil {
			errs = append(errs, fmt.Errorf("保存 Shadow 规则失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("保存词库时发生错误: %v", errs)
	}

	return nil
}

// Close 关闭词库管理器
func (dm *DictManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 保存所有数据
	if dm.userDict != nil {
		dm.userDict.Close()
	}

	if dm.shadowLayer != nil && dm.shadowLayer.IsDirty() {
		dm.shadowLayer.Save()
	}

	return nil
}

// Search 搜索候选词（便捷方法）
func (dm *DictManager) Search(code string, limit int) []candidate.Candidate {
	return dm.compositeDict.Search(code, limit)
}

// SearchPrefix 前缀搜索（便捷方法）
func (dm *DictManager) SearchPrefix(prefix string, limit int) []candidate.Candidate {
	return dm.compositeDict.SearchPrefix(prefix, limit)
}

// ReloadPhrases 重新加载短语配置
func (dm *DictManager) ReloadPhrases() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.phraseLayer != nil {
		return dm.phraseLayer.Load()
	}
	return nil
}

// ReloadShadow 重新加载 Shadow 规则
func (dm *DictManager) ReloadShadow() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.shadowLayer != nil {
		return dm.shadowLayer.Load()
	}
	return nil
}

// GetStats 获取统计信息
func (dm *DictManager) GetStats() map[string]int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := make(map[string]int)

	if dm.phraseLayer != nil {
		stats["phrases"] = dm.phraseLayer.GetPhraseCount()
		stats["commands"] = dm.phraseLayer.GetCommandCount()
	}

	if dm.shadowLayer != nil {
		stats["shadow_rules"] = dm.shadowLayer.GetRuleCount()
	}

	if dm.userDict != nil {
		stats["user_words"] = dm.userDict.EntryCount()
	}

	stats["total_layers"] = len(dm.compositeDict.GetLayers())

	return stats
}
