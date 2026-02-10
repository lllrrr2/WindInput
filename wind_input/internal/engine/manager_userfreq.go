package engine

import (
	"log"

	"github.com/huanfeng/wind_input/internal/engine/pinyin"
)

// OnCandidateSelected 选词回调，通知引擎用户选择了某个候选词
// code 为输入的编码（如拼音字符串），text 为选中的文字
func (m *Manager) OnCandidateSelected(code, text string) {
	engine := m.GetCurrentEngine()
	if engine == nil {
		return
	}

	// 拼音引擎：记录选词到用户词典
	if pinyinEngine, ok := engine.(*pinyin.Engine); ok {
		pinyinEngine.OnCandidateSelected(code, text)
	}
}

// SaveUserFreqs 保存拼音引擎的用户词频到文件
func (m *Manager) SaveUserFreqs() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, eng := range m.engines {
		if pinyinEngine, ok := eng.(*pinyin.Engine); ok {
			userFreqPath := "dict/pinyin/user_freq.txt"
			if m.exeDir != "" {
				userFreqPath = m.exeDir + "/" + userFreqPath
			}
			savePinyinUserFreqs(pinyinEngine, userFreqPath)
		}
	}
}

// loadPinyinUserFreqs 加载拼音引擎的用户词频（兼容内存模式和二进制模式）
func loadPinyinUserFreqs(engine *pinyin.Engine, path string) {
	if engine.GetUnigram() == nil {
		return
	}
	if m := engine.GetUnigramModel(); m != nil {
		if err := m.LoadUserFreqs(path); err != nil {
			log.Printf("[EngineManager] 加载用户词频失败: %v", err)
		} else {
			log.Printf("[EngineManager] 用户词频加载成功")
		}
		return
	}
	if bm := engine.GetBinaryUnigramModel(); bm != nil {
		if err := bm.LoadUserFreqs(path); err != nil {
			log.Printf("[EngineManager] 加载用户词频失败: %v", err)
		} else {
			log.Printf("[EngineManager] 用户词频加载成功")
		}
	}
}

// savePinyinUserFreqs 保存拼音引擎的用户词频
func savePinyinUserFreqs(engine *pinyin.Engine, path string) {
	if engine.GetUnigram() == nil {
		return
	}
	if m := engine.GetUnigramModel(); m != nil {
		if err := m.SaveUserFreqs(path); err != nil {
			log.Printf("[EngineManager] 保存用户词频失败: %v", err)
		}
		return
	}
	if bm := engine.GetBinaryUnigramModel(); bm != nil {
		if err := bm.SaveUserFreqs(path); err != nil {
			log.Printf("[EngineManager] 保存用户词频失败: %v", err)
		}
	}
}

// EnsurePinyinLoaded 确保拼音引擎已加载（不切换当前引擎）
// 用于临时拼音模式：在五笔模式下按需加载拼音引擎
func (m *Manager) EnsurePinyinLoaded() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.engines[EngineTypePinyin]; ok {
		return nil // 已加载
	}

	log.Printf("[EngineManager] 临时拼音：加载拼音引擎...")
	return m.loadPinyinEngineLocked()
}

// ConvertWithPinyin 使用拼音引擎转换输入（不切换当前引擎）
// 强制添加五笔编码提示，用于临时拼音模式
func (m *Manager) ConvertWithPinyin(input string, maxCandidates int) *ConvertResult {
	m.mu.RLock()
	pinyinEngine, ok := m.engines[EngineTypePinyin]
	m.mu.RUnlock()

	if !ok {
		return &ConvertResult{IsEmpty: true}
	}

	pe, ok := pinyinEngine.(*pinyin.Engine)
	if !ok {
		return &ConvertResult{IsEmpty: true}
	}

	pinyinResult := pe.ConvertEx(input, maxCandidates)

	// 强制添加五笔编码提示
	pe.AddWubiHintsForced(pinyinResult.Candidates)

	result := &ConvertResult{
		Candidates:     pinyinResult.Candidates,
		IsEmpty:        pinyinResult.IsEmpty,
		PreeditDisplay: pinyinResult.PreeditDisplay,
	}
	if pinyinResult.Composition != nil {
		result.CompletedSyllables = pinyinResult.Composition.CompletedSyllables
		result.PartialSyllable = pinyinResult.Composition.PartialSyllable
		result.HasPartial = pinyinResult.Composition.HasPartial()
	}
	return result
}
