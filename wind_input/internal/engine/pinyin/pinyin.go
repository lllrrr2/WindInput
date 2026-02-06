package pinyin

import (
	"log"
	"sort"
	"strings"

	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
)

// Config 拼音引擎配置
type Config struct {
	ShowWubiHint    bool   // 显示五笔编码提示
	FilterMode      string // 候选过滤模式
	UseSmartCompose bool   // 启用智能组句（Viterbi）
	CandidateOrder  string // 候选排序模式：char_first(单字优先)/phrase_first(词组优先)/smart(智能混排)
}

// Engine 拼音引擎
type Engine struct {
	dict         dict.Dict
	syllableTrie *SyllableTrie       // 音节 Trie
	unigram      *UnigramModel       // Unigram 语言模型
	bigram       *BigramModel        // Bigram 语言模型（可选）
	wubiTable    *dict.CodeTable     // 五笔码表（用于反查）
	wubiReverse  map[string][]string // 汉字 -> 五笔编码（反向索引）
	config       *Config
}

// NewEngine 创建拼音引擎
func NewEngine(d dict.Dict) *Engine {
	return &Engine{
		dict:         d,
		syllableTrie: NewSyllableTrie(),
		config:       &Config{ShowWubiHint: false, FilterMode: "smart"},
	}
}

// NewEngineWithConfig 创建带配置的拼音引擎
func NewEngineWithConfig(d dict.Dict, config *Config) *Engine {
	if config == nil {
		config = &Config{ShowWubiHint: false, FilterMode: "smart"}
	}
	return &Engine{
		dict:         d,
		syllableTrie: NewSyllableTrie(),
		config:       config,
	}
}

// SetConfig 设置配置
func (e *Engine) SetConfig(config *Config) {
	e.config = config
}

// GetConfig 获取配置
func (e *Engine) GetConfig() *Config {
	return e.config
}

// LoadUnigram 加载 Unigram 语言模型
func (e *Engine) LoadUnigram(path string) error {
	m := NewUnigramModel()
	if err := m.Load(path); err != nil {
		return err
	}
	e.unigram = m
	return nil
}

// LoadBigram 加载 Bigram 语言模型
func (e *Engine) LoadBigram(path string) error {
	if e.unigram == nil {
		return nil // Bigram 需要 Unigram 作为回退
	}
	m := NewBigramModel(e.unigram)
	if err := m.Load(path); err != nil {
		return err
	}
	e.bigram = m
	return nil
}

// SetUnigram 直接设置 Unigram 模型
func (e *Engine) SetUnigram(m *UnigramModel) {
	e.unigram = m
}

// LoadWubiTable 加载五笔码表（用于反查）
func (e *Engine) LoadWubiTable(path string) error {
	ct, err := dict.LoadCodeTable(path)
	if err != nil {
		return err
	}
	e.wubiTable = ct
	e.wubiReverse = ct.BuildReverseIndex()
	return nil
}

// lookupWubiCode 查找汉字的五笔编码
func (e *Engine) lookupWubiCode(text string) string {
	if e.wubiReverse == nil {
		return ""
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	// 单字：直接返回编码
	if len(runes) == 1 {
		codes := e.wubiReverse[text]
		if len(codes) > 0 {
			return codes[0]
		}
		return ""
	}

	// 词组：只有五笔码表中真实存在该词组时才返回编码
	codes := e.wubiReverse[text]
	if len(codes) > 0 {
		return codes[0]
	}
	return ""
}

// Convert 转换拼音为候选词
func (e *Engine) Convert(input string, maxCandidates int) ([]candidate.Candidate, error) {
	candidates, err := e.convertInternal(input, maxCandidates)
	if err != nil {
		return nil, err
	}

	// 应用过滤
	filterMode := "smart"
	if e.config != nil && e.config.FilterMode != "" {
		filterMode = e.config.FilterMode
	}
	beforeFilter := len(candidates)
	candidates = candidate.FilterCandidates(candidates, filterMode)
	log.Printf("[PinyinEngine] Filter: mode=%s before=%d after=%d", filterMode, beforeFilter, len(candidates))

	// 限制返回数量
	if maxCandidates > 0 && len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	// 添加五笔编码提示
	e.addWubiHints(candidates)

	return candidates, nil
}

// ConvertRaw 转换拼音为候选词（不应用过滤，用于测试）
func (e *Engine) ConvertRaw(input string, maxCandidates int) ([]candidate.Candidate, error) {
	candidates, err := e.convertInternal(input, maxCandidates)
	if err != nil {
		return nil, err
	}

	// 限制返回数量
	if maxCandidates > 0 && len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	// 添加五笔编码提示
	e.addWubiHints(candidates)

	return candidates, nil
}

// smartComposeThreshold 智能组句的输入长度阈值
const smartComposeThreshold = 4

// convertInternal 内部转换逻辑
func (e *Engine) convertInternal(input string, maxCandidates int) ([]candidate.Candidate, error) {
	if len(input) == 0 {
		return nil, nil
	}

	input = strings.ToLower(input)
	candidatesMap := make(map[string]*candidate.Candidate)

	// 1. 使用 DAG 进行音节切分
	dag := BuildDAG(input, e.syllableTrie)

	// 2. 智能组句：长输入且启用时使用 Viterbi
	useViterbi := e.config != nil && e.config.UseSmartCompose &&
		e.unigram != nil && len(input) >= smartComposeThreshold

	log.Printf("[PinyinEngine] input=%q len=%d useViterbi=%v (smartCompose=%v unigram=%v threshold=%d)",
		input, len(input), useViterbi,
		e.config != nil && e.config.UseSmartCompose,
		e.unigram != nil, smartComposeThreshold)

	if useViterbi {
		lattice := BuildLattice(input, e.syllableTrie, e.dict, e.unigram)
		log.Printf("[PinyinEngine] Lattice: empty=%v size=%d", lattice.IsEmpty(), lattice.Size())
		if !lattice.IsEmpty() {
			result := ViterbiDecode(lattice, e.bigram)
			if result != nil && len(result.Words) > 0 {
				// Viterbi 最优路径作为第一候选（最高权重）
				sentence := result.String()
				log.Printf("[PinyinEngine] Viterbi result: %q words=%v logprob=%.4f", sentence, result.Words, result.LogProb)
				c := candidate.Candidate{
					Text:           sentence,
					Code:           input,
					Weight:         weightViterbi,
					IsCommon:       true,
					ConsumedLength: len(input),
				}
				candidatesMap[sentence] = &c
			} else {
				log.Printf("[PinyinEngine] Viterbi returned nil or empty")
			}
		}
	}

	// 3. 获取最大匹配路径
	mainPath := dag.MaximumMatch()
	log.Printf("[PinyinEngine] DAG mainPath=%v joined=%q fullMatch=%v",
		mainPath, strings.Join(mainPath, ""), strings.Join(mainPath, "") == input)

	// 4. 如果最大匹配覆盖完整输入，查找词组
	if len(mainPath) > 0 {
		joined := strings.Join(mainPath, "")
		if joined == input {
			// 查找完整词组（给予最高加成，确保词组优先于单字）
			// 完整匹配的词组应该排在最前面
			fullPhraseBonus := 200000
			e.lookupAndAdd(input, candidatesMap, fullPhraseBonus)

			// 按不同长度查找子词组（从长到短）
			if len(mainPath) > 1 {
				e.lookupSubPhrases(mainPath, candidatesMap)
			}

			// 查找每个音节的单字（降低权重，使词组优先）
			singleCharPenalty := -len(mainPath) * 50000
			for _, syllable := range mainPath {
				e.lookupAndAdd(syllable, candidatesMap, singleCharPenalty)
			}
		}
	}

	// 5. 获取其他切分路径
	allPaths := dag.AllPaths(5)
	for _, path := range allPaths {
		joined := strings.Join(path, "")
		if joined != input {
			continue
		}
		e.lookupAndAdd(joined, candidatesMap, 0)

		if len(path) > 1 {
			e.lookupSubPhrases(path, candidatesMap)
		}
	}

	// 6. 前缀匹配补充候选（边打边出）
	log.Printf("[PinyinEngine] before prefix: candidatesMap size=%d", len(candidatesMap))
	if ps, ok := e.dict.(dict.PrefixSearchable); ok {
		log.Printf("[PinyinEngine] dict implements PrefixSearchable")
		prefixLimit := 20
		if maxCandidates > 0 {
			prefixLimit = maxCandidates
		}
		prefixResults := ps.LookupPrefix(input, prefixLimit)
		for _, cand := range prefixResults {
			if _, exists := candidatesMap[cand.Text]; !exists {
				c := cand
				c.Weight = c.Weight - 5000
				candidatesMap[c.Text] = &c
			}
		}
	}

	// 7. 如果 DAG 匹配失败，回退到旧的音节解析
	if len(candidatesMap) == 0 {
		e.fallbackParseSyllables(input, candidatesMap)
	}

	// 8. 转换为列表并排序
	candidates := make(candidate.CandidateList, 0, len(candidatesMap))
	for _, cand := range candidatesMap {
		candidates = append(candidates, *cand)
	}
	sort.Sort(candidates)

	// 调试：输出前5个候选
	topN := 5
	if len(candidates) < topN {
		topN = len(candidates)
	}
	for i := 0; i < topN; i++ {
		log.Printf("[PinyinEngine] candidate[%d]: %q code=%q weight=%d", i, candidates[i].Text, candidates[i].Code, candidates[i].Weight)
	}
	log.Printf("[PinyinEngine] total candidates=%d", len(candidates))

	return candidates, nil
}

// lookupAndAdd 查找编码并添加到候选映射
func (e *Engine) lookupAndAdd(code string, m map[string]*candidate.Candidate, weightAdjust int) {
	results := e.dict.Lookup(code)
	log.Printf("[PinyinEngine] lookupAndAdd: code=%q results=%d", code, len(results))
	for _, cand := range results {
		cand.Weight += weightAdjust
		if existing, ok := m[cand.Text]; ok {
			if cand.Weight > existing.Weight {
				*existing = cand
			}
		} else {
			c := cand
			m[c.Text] = &c
		}
	}
}

// lookupSubPhrases 查找子词组
func (e *Engine) lookupSubPhrases(syllables []string, m map[string]*candidate.Candidate) {
	n := len(syllables)
	log.Printf("[PinyinEngine] lookupSubPhrases: syllables=%v n=%d", syllables, n)

	// 从长到短尝试各种子序列
	for length := n; length >= 2; length-- {
		for start := 0; start+length <= n; start++ {
			sub := syllables[start : start+length]
			key := strings.Join(sub, "")
			results := e.dict.Lookup(key)
			log.Printf("[PinyinEngine] lookupSubPhrases: key=%q results=%d", key, len(results))
			// 子词组的权重根据长度提升
			bonus := length * 10000
			for _, cand := range results {
				cand.Weight += bonus
				if existing, ok := m[cand.Text]; ok {
					if cand.Weight > existing.Weight {
						*existing = cand
					}
				} else {
					c := cand
					m[c.Text] = &c
				}
			}
		}
	}
}

// fallbackParseSyllables 回退到旧的音节解析方式
func (e *Engine) fallbackParseSyllables(input string, m map[string]*candidate.Candidate) {
	syllablesList := ParseSyllables(input)
	for _, syllables := range syllablesList {
		// 尝试查找完整短语
		phraseCandidates := e.dict.LookupPhrase(syllables)
		for _, cand := range phraseCandidates {
			if existing, ok := m[cand.Text]; ok {
				if cand.Weight > existing.Weight {
					*existing = cand
				}
			} else {
				c := cand
				m[cand.Text] = &c
			}
		}

		// 查找单个音节
		for _, syllable := range syllables {
			singleCandidates := e.dict.Lookup(syllable)
			for _, cand := range singleCandidates {
				if len(syllables) > 1 {
					cand.Weight = cand.Weight / 2
				}
				if existing, ok := m[cand.Text]; ok {
					if cand.Weight > existing.Weight {
						*existing = cand
					}
				} else {
					c := cand
					m[cand.Text] = &c
				}
			}
		}
	}
}

// addWubiHints 添加五笔编码提示
func (e *Engine) addWubiHints(candidates []candidate.Candidate) {
	if e.config == nil || !e.config.ShowWubiHint || e.wubiReverse == nil {
		return
	}
	for i := range candidates {
		wubiCode := e.lookupWubiCode(candidates[i].Text)
		if wubiCode != "" {
			candidates[i].Hint = wubiCode
		}
	}
}

// Reset 重置引擎状态
func (e *Engine) Reset() {
	// 拼音引擎目前无状态，无需重置
}

// Type 返回引擎类型
func (e *Engine) Type() string {
	return "pinyin"
}
