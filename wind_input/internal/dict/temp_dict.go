package dict

import (
	"log"
	"sort"
)

// TempDict 临时词库
// 包装 UserDict，用于存储自动学习的词条。
// 与 UserDict 使用相同的文件格式，但在 CompositeDict 中优先级较低。
// 支持晋升（将词条迁移到用户词库）和清空操作。
type TempDict struct {
	*UserDict
	maxEntries   int       // 最大条目数（0=不限制）
	promoteCount int       // 晋升所需选择次数（0=不自动晋升）
	targetDict   *UserDict // 晋升目标词库
}

// NewTempDict 创建临时词库
func NewTempDict(name string, filePath string, maxEntries, promoteCount int) *TempDict {
	return &TempDict{
		UserDict:     NewUserDict(name, filePath),
		maxEntries:   maxEntries,
		promoteCount: promoteCount,
	}
}

// Type 返回层类型（覆盖 UserDict 的 Type）
func (td *TempDict) Type() LayerType {
	return LayerTypeTemp
}

// SetTargetDict 设置晋升目标词库
func (td *TempDict) SetTargetDict(target *UserDict) {
	td.targetDict = target
}

// LearnWord 学习新词（自动学习入口）
// 如果词已存在则增加权重和计数，不存在则新增
// 返回 true 表示该词达到晋升条件
func (td *TempDict) LearnWord(code, text string, weightDelta int) bool {
	td.mu.Lock()
	defer td.mu.Unlock()

	code = toLowerCode(code)
	words, ok := td.entries[code]
	if ok {
		for i, w := range words {
			if w.Text == text {
				newWeight := w.Weight + weightDelta
				if newWeight > MaxDynamicWeight {
					newWeight = MaxDynamicWeight
				}
				td.entries[code][i].Weight = newWeight
				td.entries[code][i].Count++
				td.markDirty()

				// 检查是否达到晋升条件
				if td.promoteCount > 0 && td.entries[code][i].Count >= td.promoteCount {
					return true
				}
				return false
			}
		}
	}

	// 新增词条
	td.entries[code] = append(td.entries[code], UserWord{
		Text:   text,
		Weight: weightDelta,
		Count:  1,
	})
	td.markDirty()

	// 检查是否超过最大条目数，执行淘汰
	if td.maxEntries > 0 {
		td.evictIfNeeded()
	}

	return false
}

// PromoteWord 将指定词条晋升到用户词库
func (td *TempDict) PromoteWord(code, text string) bool {
	if td.targetDict == nil {
		return false
	}

	td.mu.Lock()
	defer td.mu.Unlock()

	code = toLowerCode(code)
	words, ok := td.entries[code]
	if !ok {
		return false
	}

	for i, w := range words {
		if w.Text == text {
			// 添加到目标词库
			td.targetDict.Add(code, text, w.Weight)

			// 从临时词库移除
			td.entries[code] = append(words[:i], words[i+1:]...)
			if len(td.entries[code]) == 0 {
				delete(td.entries, code)
			}
			td.markDirty()
			return true
		}
	}
	return false
}

// PromoteAll 将所有临时词条晋升到用户词库
func (td *TempDict) PromoteAll() int {
	if td.targetDict == nil {
		return 0
	}

	td.mu.Lock()
	defer td.mu.Unlock()

	count := 0
	for code, words := range td.entries {
		for _, w := range words {
			td.targetDict.Add(code, w.Text, w.Weight)
			count++
		}
	}

	td.entries = make(map[string][]UserWord)
	td.markDirty()
	return count
}

// Clear 清空临时词库
func (td *TempDict) Clear() int {
	td.mu.Lock()
	defer td.mu.Unlock()

	count := td.entryCountLocked()
	td.entries = make(map[string][]UserWord)
	td.markDirty()
	return count
}

// GetWordCount 获取词条总数
func (td *TempDict) GetWordCount() int {
	td.mu.RLock()
	defer td.mu.RUnlock()
	return td.entryCountLocked()
}

// entryCountLocked 获取词条数（需持有锁）
func (td *TempDict) entryCountLocked() int {
	count := 0
	for _, words := range td.entries {
		count += len(words)
	}
	return count
}

// evictIfNeeded 如果超过最大条目数，淘汰最旧的词条（需持有写锁）
func (td *TempDict) evictIfNeeded() {
	total := td.entryCountLocked()
	if total <= td.maxEntries {
		return
	}

	// 收集所有词条并按权重排序（权重低的先淘汰）
	type entry struct {
		code string
		word UserWord
		idx  int
	}
	var all []entry
	for code, words := range td.entries {
		for i, w := range words {
			all = append(all, entry{code: code, word: w, idx: i})
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].word.Weight < all[j].word.Weight
	})

	// 淘汰多余的
	removeCount := total - td.maxEntries
	removed := 0
	removeSet := make(map[string]map[int]bool)

	for _, e := range all {
		if removed >= removeCount {
			break
		}
		if removeSet[e.code] == nil {
			removeSet[e.code] = make(map[int]bool)
		}
		removeSet[e.code][e.idx] = true
		removed++
	}

	// 执行移除
	for code, indices := range removeSet {
		var kept []UserWord
		for i, w := range td.entries[code] {
			if !indices[i] {
				kept = append(kept, w)
			}
		}
		if len(kept) == 0 {
			delete(td.entries, code)
		} else {
			td.entries[code] = kept
		}
	}

	if removed > 0 {
		log.Printf("[TempDict] 淘汰 %d 条低权重词条", removed)
	}
}

// toLowerCode 转小写（避免引入 strings 包）
func toLowerCode(code string) string {
	// 简单实现，UserDict 内部也用 strings.ToLower
	result := make([]byte, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
