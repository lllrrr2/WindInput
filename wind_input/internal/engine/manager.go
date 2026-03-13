package engine

import (
	"fmt"
	"log"
	"sync"

	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/engine/pinyin"
	"github.com/huanfeng/wind_input/internal/engine/wubi"
)

// Manager 引擎管理器
type Manager struct {
	mu            sync.RWMutex
	engines       map[EngineType]Engine
	currentType   EngineType
	currentEngine Engine

	// 词库路径（用于动态切换时加载）
	pinyinDictPath string
	wubiDictPath   string

	// 引擎配置
	pinyinConfig *pinyin.Config
	wubiConfig   *wubi.Config

	// 可执行文件目录（用于相对路径）
	exeDir string

	// 词库管理器
	dictManager *dict.DictManager
}

// NewManager 创建引擎管理器
func NewManager() *Manager {
	return &Manager{
		engines: make(map[EngineType]Engine),
	}
}

// RegisterEngine 注册引擎
func (m *Manager) RegisterEngine(engineType EngineType, engine Engine) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.engines[engineType] = engine
	log.Printf("[EngineManager] 注册引擎: %s", engineType)
}

// SetCurrentEngine 设置当前引擎
func (m *Manager) SetCurrentEngine(engineType EngineType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, ok := m.engines[engineType]
	if !ok {
		return fmt.Errorf("引擎未注册: %s", engineType)
	}

	m.currentType = engineType
	m.currentEngine = engine
	log.Printf("[EngineManager] 切换引擎: %s", engineType)
	return nil
}

// GetCurrentEngine 获取当前引擎
func (m *Manager) GetCurrentEngine() Engine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentEngine
}

// GetCurrentType 获取当前引擎类型
func (m *Manager) GetCurrentType() EngineType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentType
}

// Convert 使用当前引擎转换输入
func (m *Manager) Convert(input string, maxCandidates int) ([]candidate.Candidate, error) {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return nil, fmt.Errorf("未设置当前引擎")
	}
	return engine.Convert(input, maxCandidates)
}

// ConvertRaw 使用当前引擎转换输入（不应用过滤，用于测试）
func (m *Manager) ConvertRaw(input string, maxCandidates int) ([]candidate.Candidate, error) {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return nil, fmt.Errorf("未设置当前引擎")
	}

	// 检查引擎是否支持 ConvertRaw
	if pinyinEngine, ok := engine.(*pinyin.Engine); ok {
		return pinyinEngine.ConvertRaw(input, maxCandidates)
	}
	if wubiEngine, ok := engine.(*wubi.Engine); ok {
		return wubiEngine.ConvertRaw(input, maxCandidates)
	}

	// 回退到普通 Convert
	return engine.Convert(input, maxCandidates)
}

// ConvertEx 扩展转换，返回更多信息
func (m *Manager) ConvertEx(input string, maxCandidates int) *ConvertResult {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return &ConvertResult{}
	}

	// 五笔引擎：使用五笔扩展功能
	if wubiEngine, ok := engine.(*wubi.Engine); ok {
		wubiResult := wubiEngine.ConvertEx(input, maxCandidates)
		return &ConvertResult{
			Candidates:   wubiResult.Candidates,
			ShouldCommit: wubiResult.ShouldCommit,
			CommitText:   wubiResult.CommitText,
			IsEmpty:      wubiResult.IsEmpty,
			ShouldClear:  wubiResult.ShouldClear,
			ToEnglish:    wubiResult.ToEnglish,
		}
	}

	// 拼音引擎：使用新的 ConvertEx 方法
	if pinyinEngine, ok := engine.(*pinyin.Engine); ok {
		pinyinResult := pinyinEngine.ConvertEx(input, maxCandidates)
		result := &ConvertResult{
			Candidates:     pinyinResult.Candidates,
			IsEmpty:        pinyinResult.IsEmpty,
			PreeditDisplay: pinyinResult.PreeditDisplay,
		}
		// 填充组合态信息
		if pinyinResult.Composition != nil {
			result.CompletedSyllables = pinyinResult.Composition.CompletedSyllables
			result.PartialSyllable = pinyinResult.Composition.PartialSyllable
			result.HasPartial = pinyinResult.Composition.HasPartial()
		}
		return result
	}

	// 普通引擎
	candidates, err := engine.Convert(input, maxCandidates)
	if err != nil {
		log.Printf("[EngineManager] 转换错误: %v", err)
	}

	return &ConvertResult{
		Candidates: candidates,
		IsEmpty:    len(candidates) == 0,
	}
}

// Reset 重置当前引擎
func (m *Manager) Reset() {
	engine := m.GetCurrentEngine()
	if engine != nil {
		engine.Reset()
	}
}

// GetMaxCodeLength 获取当前引擎的最大码长
func (m *Manager) GetMaxCodeLength() int {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return 0
	}

	// 五笔引擎
	if wubiEngine, ok := engine.(*wubi.Engine); ok {
		return wubiEngine.GetConfig().MaxCodeLength
	}

	// 拼音默认不限制
	return 100
}

// HandleTopCode 处理顶码
func (m *Manager) HandleTopCode(input string) (commitText string, newInput string, shouldCommit bool) {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return "", input, false
	}

	// 五笔引擎支持顶码
	if wubiEngine, ok := engine.(*wubi.Engine); ok {
		return wubiEngine.HandleTopCode(input)
	}

	return "", input, false
}

// InvalidateCommandCache 清除命令结果缓存（uuid/date/time 等）
// 在输入提交或状态清除后调用，确保下次查询生成新的结果
func (m *Manager) InvalidateCommandCache() {
	if m.dictManager == nil {
		return
	}
	if phraseLayer := m.dictManager.GetPhraseLayer(); phraseLayer != nil {
		phraseLayer.InvalidateCache()
	}
}

// GetEngineInfo 获取当前引擎信息
func (m *Manager) GetEngineInfo() string {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return "未加载引擎"
	}

	engineType := m.GetCurrentType()

	// 五笔引擎
	if wubiEngine, ok := engine.(*wubi.Engine); ok {
		info := wubiEngine.GetCodeTableInfo()
		if info != nil {
			return fmt.Sprintf("%s: %s (%d词条)", engineType, info.Name, wubiEngine.GetEntryCount())
		}
	}

	return string(engineType)
}

// GetEngineDisplayName 获取引擎显示名称
func (m *Manager) GetEngineDisplayName() string {
	switch m.GetCurrentType() {
	case EngineTypePinyin:
		return "拼"
	case EngineTypeWubi:
		return "五"
	default:
		return "?"
	}
}

// SwitchEngine 切换到指定引擎
func (m *Manager) SwitchEngine(targetType EngineType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已经是目标引擎，不做任何操作
	if m.currentType == targetType {
		return nil
	}

	// 检查是否已注册
	if engine, ok := m.engines[targetType]; ok {
		m.currentType = targetType
		m.currentEngine = engine
		log.Printf("[EngineManager] 切换到已注册引擎: %s", targetType)
		return nil
	}

	// 需要动态加载引擎
	var err error
	switch targetType {
	case EngineTypePinyin:
		err = m.loadPinyinEngineLocked()
	case EngineTypeWubi:
		err = m.loadWubiEngineLocked()
	default:
		return fmt.Errorf("未知引擎类型: %s", targetType)
	}

	if err != nil {
		return err
	}

	m.currentType = targetType
	m.currentEngine = m.engines[targetType]
	log.Printf("[EngineManager] 动态加载并切换引擎: %s", targetType)
	return nil
}

// ToggleEngine 在拼音和五笔之间切换
func (m *Manager) ToggleEngine() (EngineType, error) {
	currentType := m.GetCurrentType()
	var targetType EngineType

	switch currentType {
	case EngineTypePinyin:
		targetType = EngineTypeWubi
	case EngineTypeWubi:
		targetType = EngineTypePinyin
	default:
		targetType = EngineTypePinyin
	}

	if err := m.SwitchEngine(targetType); err != nil {
		return currentType, err
	}

	return targetType, nil
}

// isAbsPath 判断是否为绝对路径
func isAbsPath(path string) bool {
	if len(path) == 0 {
		return false
	}
	// Windows 绝对路径: C:\ 或 \\
	if len(path) >= 2 && path[1] == ':' {
		return true
	}
	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		return true
	}
	// Unix 绝对路径
	return path[0] == '/'
}
