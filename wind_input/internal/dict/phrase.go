package dict

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/huanfeng/wind_input/internal/candidate"
	"gopkg.in/yaml.v3"
)

// PhraseLayer 短语层
// 加载系统短语和用户短语，支持变量模板展开（$Y, $MM, $DD 等）。
// 含变量的短语为"动态短语"，仅精确匹配（通过 SearchCommand），
// 不含变量的为"静态短语"，支持前缀搜索。
type PhraseLayer struct {
	mu             sync.RWMutex
	name           string
	systemFilePath string // 系统短语文件（随程序打包，只读）
	userFilePath   string // 用户短语文件（用户可编辑）

	// 静态短语（不含变量）: code -> []PhraseEntry，参与前缀搜索
	staticPhrases map[string][]PhraseEntry

	// 动态短语（含 $ 变量）: code -> []PhraseEntry，仅精确匹配
	dynamicPhrases map[string][]PhraseEntry

	// 模板引擎
	templateEngine *TemplateEngine

	// 命令结果缓存（动态短语）
	cmdCache    map[string][]candidate.Candidate
	cmdCacheKey string
}

// PhraseEntry 短语条目
type PhraseEntry struct {
	Text     string // 输出文本（可含 $变量模板）
	Position int    // 候选位置
	IsSystem bool   // 是否来自系统短语
	Disabled bool   // 是否被禁用
}

// PhrasesFileConfig 短语文件的 YAML 结构
type PhrasesFileConfig struct {
	Phrases []PhraseFileEntry `yaml:"phrases"`
}

// PhraseFileEntry 短语文件中的单条配置
type PhraseFileEntry struct {
	Code     string `yaml:"code"`
	Text     string `yaml:"text"`
	Position int    `yaml:"position"`
	Disabled bool   `yaml:"disabled,omitempty"`
}

// NewPhraseLayer 创建短语层
// systemPath: 系统短语文件路径（只读，可为空）
// userPath: 用户短语文件路径（可读写）
func NewPhraseLayer(name string, systemPath, userPath string) *PhraseLayer {
	return &PhraseLayer{
		name:           name,
		systemFilePath: systemPath,
		userFilePath:   userPath,
		staticPhrases:  make(map[string][]PhraseEntry),
		dynamicPhrases: make(map[string][]PhraseEntry),
		templateEngine: GetTemplateEngine(),
		cmdCache:       make(map[string][]candidate.Candidate),
	}
}

// Name 返回层名称
func (pl *PhraseLayer) Name() string {
	return pl.name
}

// Type 返回层类型
func (pl *PhraseLayer) Type() LayerType {
	return LayerTypeLogic
}

// Search 精确查询静态短语（不含变量的短语）
func (pl *PhraseLayer) Search(code string, limit int) []candidate.Candidate {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	code = strings.ToLower(code)
	entries, ok := pl.staticPhrases[code]
	if !ok {
		return nil
	}

	results := make([]candidate.Candidate, 0, len(entries))
	for _, e := range entries {
		results = append(results, candidate.Candidate{
			Text:     e.Text,
			Code:     code,
			Weight:   positionToWeight(e.Position),
			IsCommon: true, // 短语由用户/系统配置，不应被 smart 过滤
		})
	}

	sortByPosition(results)

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SearchCommand 查询动态短语（含变量的短语），展开模板后返回
func (pl *PhraseLayer) SearchCommand(code string, limit int) []candidate.Candidate {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	code = strings.ToLower(code)
	entries, ok := pl.dynamicPhrases[code]
	if !ok {
		return nil
	}

	// 使用缓存
	if cached, hit := pl.cmdCache[code]; hit {
		if limit > 0 && len(cached) > limit {
			return cached[:limit]
		}
		return cached
	}

	results := make([]candidate.Candidate, 0, len(entries))
	for _, e := range entries {
		expanded := pl.templateEngine.Expand(e.Text)
		results = append(results, candidate.Candidate{
			Text:      expanded,
			Code:      code,
			Weight:    positionToWeight(e.Position),
			IsCommand: true,
			IsCommon:  true, // 动态短语不应被 smart 过滤
		})
	}

	sortByPosition(results)
	pl.cmdCache[code] = results

	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

// SearchPrefix 前缀查询（仅静态短语）
func (pl *PhraseLayer) SearchPrefix(prefix string, limit int) []candidate.Candidate {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	prefix = strings.ToLower(prefix)
	var results []candidate.Candidate

	for code, entries := range pl.staticPhrases {
		if strings.HasPrefix(code, prefix) {
			for _, e := range entries {
				results = append(results, candidate.Candidate{
					Text:     e.Text,
					Code:     code,
					Weight:   positionToWeight(e.Position),
					IsCommon: true,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return candidate.Better(results[i], results[j])
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// InvalidateCache 清除动态短语缓存
func (pl *PhraseLayer) InvalidateCache() {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.cmdCache = make(map[string][]candidate.Candidate)
}

// InvalidateCacheForInput 根据输入变化清除缓存
func (pl *PhraseLayer) InvalidateCacheForInput(input string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	if pl.cmdCacheKey != input {
		pl.cmdCache = make(map[string][]candidate.Candidate)
		pl.cmdCacheKey = input
	}
}

// Load 加载系统短语和用户短语
func (pl *PhraseLayer) Load() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	pl.staticPhrases = make(map[string][]PhraseEntry)
	pl.dynamicPhrases = make(map[string][]PhraseEntry)
	pl.cmdCache = make(map[string][]candidate.Candidate)

	// 1. 加载系统短语
	if pl.systemFilePath != "" {
		if err := pl.loadFile(pl.systemFilePath, true); err != nil {
			// 系统文件不存在不是错误（开发环境下可能没有）
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load system phrases: %w", err)
			}
		}
	}

	// 2. 加载用户短语
	if pl.userFilePath != "" {
		if err := pl.loadFile(pl.userFilePath, false); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load user phrases: %w", err)
			}
		}
	}

	return nil
}

// loadFile 从文件加载短语（需持有写锁）
func (pl *PhraseLayer) loadFile(path string, isSystem bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config PhrasesFileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse phrases file %s: %w", path, err)
	}

	for _, p := range config.Phrases {
		code := strings.ToLower(p.Code)
		if code == "" || p.Text == "" {
			continue
		}

		position := p.Position
		if position <= 0 {
			position = 1
		}

		// 用户文件可以覆盖系统短语（同 code+text 的条目）
		if !isSystem {
			phraseID := code + "||" + p.Text
			// 检查并移除系统短语中的同名条目（用户覆盖）
			pl.removeEntryByID(phraseID)
		}

		entry := PhraseEntry{
			Text:     p.Text,
			Position: position,
			IsSystem: isSystem,
			Disabled: p.Disabled,
		}

		// 被禁用的条目不加入查询索引（但仍可被前端读取）
		if p.Disabled {
			continue
		}

		// $[...] 数组映射：展开为多个静态候选
		if HasArrayMapping(p.Text) {
			items := ExpandArrayMapping(p.Text)
			for idx, item := range items {
				arrEntry := PhraseEntry{
					Text:     item,
					Position: position + idx,
					IsSystem: isSystem,
				}
				pl.staticPhrases[code] = append(pl.staticPhrases[code], arrEntry)
			}
			continue
		}

		// 含 $变量 的为动态短语，否则为静态短语
		if HasVariable(p.Text) {
			pl.dynamicPhrases[code] = append(pl.dynamicPhrases[code], entry)
		} else {
			pl.staticPhrases[code] = append(pl.staticPhrases[code], entry)
		}
	}

	return nil
}

// removeEntryByID 根据 "code||text" 从所有索引中移除条目（用于用户覆盖系统短语）
func (pl *PhraseLayer) removeEntryByID(id string) {
	for code, entries := range pl.staticPhrases {
		for i, e := range entries {
			if code+"||"+e.Text == id {
				pl.staticPhrases[code] = append(entries[:i], entries[i+1:]...)
				if len(pl.staticPhrases[code]) == 0 {
					delete(pl.staticPhrases, code)
				}
				return
			}
		}
	}
	for code, entries := range pl.dynamicPhrases {
		for i, e := range entries {
			if code+"||"+e.Text == id {
				pl.dynamicPhrases[code] = append(entries[:i], entries[i+1:]...)
				if len(pl.dynamicPhrases[code]) == 0 {
					delete(pl.dynamicPhrases, code)
				}
				return
			}
		}
	}
}

// GetPhraseCount 获取静态短语数量
func (pl *PhraseLayer) GetPhraseCount() int {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	count := 0
	for _, entries := range pl.staticPhrases {
		count += len(entries)
	}
	return count
}

// GetCommandCount 获取动态短语数量
func (pl *PhraseLayer) GetCommandCount() int {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	count := 0
	for _, entries := range pl.dynamicPhrases {
		count += len(entries)
	}
	return count
}

// GetUserFilePath 获取用户短语文件路径
func (pl *PhraseLayer) GetUserFilePath() string {
	return pl.userFilePath
}

// GetSystemFilePath 获取系统短语文件路径
func (pl *PhraseLayer) GetSystemFilePath() string {
	return pl.systemFilePath
}

// ===== 辅助函数 =====

// positionToWeight 将位置转换为权重（position 1 → 最高权重）
func positionToWeight(position int) int {
	if position <= 0 {
		position = 1
	}
	return 10000 - position
}

// sortByPosition 按位置排序候选
func sortByPosition(candidates []candidate.Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Weight > candidates[j].Weight
	})
}
