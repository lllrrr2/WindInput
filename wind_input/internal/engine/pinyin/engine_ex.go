package pinyin

import (
	"log"
	"sort"
	"strings"

	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
)

// ============================================================
// Engine 扩展方法
// 使用新的 Parser → Lexicon → Ranker 流水线
// ============================================================

// ============================================================
// 权重层级常量（统一权重体系）
// ============================================================
const (
	weightViterbi       = 3000000 // Viterbi 整句候选
	weightExactMatch    = 2000000 // 精确匹配（字数=音节数）
	weightPrefixClose   = 1800000 // 前缀匹配（字数接近）
	weightPrefixMatch   = 1700000 // 前缀匹配（一般）
	weightExactOther    = 1500000 // 精确匹配（字数≠音节数）
	weightSubPhrase     = 1000000 // 子词组
	weightPartialPrefix = 800000  // 未完成音节前缀词
	weightPartialChar   = 600000  // 未完成音节单字
	weightSingleChar    = 500000  // 首音节单字
)

// ConvertEx 扩展版转换方法
// 返回包含组合态的完整转换结果
func (e *Engine) ConvertEx(input string, maxCandidates int) *PinyinConvertResult {
	result := &PinyinConvertResult{
		Candidates: make([]candidate.Candidate, 0),
	}

	if len(input) == 0 {
		result.IsEmpty = true
		return result
	}

	input = strings.ToLower(input)

	// 1. 解析输入为音节
	parser := NewPinyinParser()
	parsed := parser.Parse(input)

	// 2. 构建组合态
	builder := NewCompositionBuilder()
	result.Composition = builder.Build(parsed)
	result.PreeditDisplay = result.Composition.PreeditText

	completedSyllables := parsed.CompletedSyllables()
	syllableCount := len(completedSyllables)
	partial := parsed.PartialSyllable()

	log.Printf("[PinyinEngine.ConvertEx] input=%q preedit=%q completed=%v partial=%q",
		input, result.PreeditDisplay, completedSyllables, partial)

	// 3. 收集候选词
	candidatesMap := make(map[string]*candidate.Candidate)

	// 获取候选排序模式
	candidateOrder := "char_first"
	if e.config != nil && e.config.CandidateOrder != "" {
		candidateOrder = e.config.CandidateOrder
	}

	// 3a. Viterbi 智能组句（多音节且无未完成音节时使用）
	useViterbi := e.config != nil && e.config.UseSmartCompose &&
		e.unigram != nil && syllableCount >= 2 && partial == "" &&
		len(input) >= smartComposeThreshold

	if useViterbi {
		lattice := BuildLattice(input, e.syllableTrie, e.dict, e.unigram)
		if !lattice.IsEmpty() {
			// 获取 Top-3 最优路径
			vResults := ViterbiTopK(lattice, e.bigram, 3)
			for rank, vResult := range vResults {
				if vResult == nil || len(vResult.Words) == 0 {
					continue
				}
				sentence := vResult.String()
				if _, exists := candidatesMap[sentence]; exists {
					continue
				}
				log.Printf("[PinyinEngine.ConvertEx] Viterbi[%d]: %q words=%v logprob=%.4f",
					rank, sentence, vResult.Words, vResult.LogProb)
				c := candidate.Candidate{
					Text:           sentence,
					Code:           input,
					Weight:         weightViterbi - rank,
					IsCommon:       true,
					ConsumedLength: len(input),
				}
				candidatesMap[sentence] = &c
			}
		}
	}

	// 3b. 精确匹配完整音节序列的词组
	if syllableCount > 0 && partial == "" {
		exactResults := e.dict.Lookup(input)
		// 使用 Unigram 单字频率对精确匹配进行二次排序
		type scoredExact struct {
			cand  candidate.Candidate
			score float64
		}
		scored := make([]scoredExact, 0, len(exactResults))
		for _, cand := range exactResults {
			charCount := len([]rune(cand.Text))
			lmScore := float64(0)
			if e.unigram != nil {
				lmScore = e.unigram.CharBasedScore(cand.Text)
			}
			se := scoredExact{cand: cand, score: lmScore}
			// 字数匹配音节数的给予额外加成
			if charCount == syllableCount {
				se.score += 100
			}
			scored = append(scored, se)
		}
		// 按 LM 分数降序排列
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})
		for i, se := range scored {
			if _, exists := candidatesMap[se.cand.Text]; exists {
				continue
			}
			c := se.cand
			charCount := len([]rune(c.Text))
			if charCount == syllableCount {
				c.Weight = weightExactMatch - i
			} else {
				c.Weight = weightExactOther - i
			}
			c.ConsumedLength = len(input)
			candidatesMap[c.Text] = &c
		}
		log.Printf("[PinyinEngine.ConvertEx] exact match for %q: %d results", input, len(exactResults))
	}

	// 3c. 前缀匹配（输入 "wome" 时找到 "women"→我们）
	if ps, ok := e.dict.(dict.PrefixSearchable); ok {
		prefixLimit := 50
		if maxCandidates > 0 {
			prefixLimit = maxCandidates * 2
		}
		prefixResults := ps.LookupPrefix(input, prefixLimit)
		for i, cand := range prefixResults {
			if _, exists := candidatesMap[cand.Text]; exists {
				continue
			}
			c := cand
			charCount := len([]rune(c.Text))
			if charCount == syllableCount+1 && partial == "" {
				c.Weight = weightPrefixClose - i
			} else if charCount == syllableCount {
				c.Weight = weightPrefixMatch - i
			} else {
				c.Weight = weightSubPhrase - i
			}
			c.ConsumedLength = len(input)
			candidatesMap[c.Text] = &c
		}
		log.Printf("[PinyinEngine.ConvertEx] prefix match for %q: %d results", input, len(prefixResults))
	}

	// 3d. 子词组查找（如 "nihao" → 查找 "ni"+"hao" 对应的词组）
	var mainPath []string
	if syllableCount > 1 {
		dag := BuildDAG(input, e.syllableTrie)
		mainPath = dag.MaximumMatch()
		if len(mainPath) > 1 {
			joined := strings.Join(mainPath, "")
			if joined == input {
				e.lookupSubPhrasesEx(mainPath, candidatesMap)
			}
		}
	}

	// 3e. 单字候选
	if syllableCount > 0 {
		// 使用 Unigram 对首音节单字排序
		firstSyllable := completedSyllables[0]
		charResults := e.dict.Lookup(firstSyllable)

		type scoredChar struct {
			cand  candidate.Candidate
			score float64
		}
		scoredChars := make([]scoredChar, 0, len(charResults))
		for _, cand := range charResults {
			lmScore := float64(0)
			if e.unigram != nil {
				lmScore = e.unigram.LogProb(cand.Text)
			}
			scoredChars = append(scoredChars, scoredChar{cand: cand, score: lmScore})
		}
		sort.Slice(scoredChars, func(i, j int) bool {
			return scoredChars[i].score > scoredChars[j].score
		})

		for j, sc := range scoredChars {
			if _, exists := candidatesMap[sc.cand.Text]; exists {
				continue
			}
			c := sc.cand
			c.Weight = weightSingleChar - j
			c.ConsumedLength = len(firstSyllable)
			candidatesMap[c.Text] = &c
		}

		// 非首音节的单字（更低权重）
		for i := 1; i < syllableCount; i++ {
			syllable := completedSyllables[i]
			others := e.dict.Lookup(syllable)
			for j, cand := range others {
				if _, exists := candidatesMap[cand.Text]; exists {
					continue
				}
				c := cand
				c.Weight = weightSingleChar - i*10000 - j
				// 非首音节单字的 ConsumedLength 需要计算到该音节的结束位置
				consumedLen := 0
				for k := 0; k <= i; k++ {
					consumedLen += len(completedSyllables[k])
				}
				c.ConsumedLength = consumedLen
				candidatesMap[c.Text] = &c
			}
		}
	}

	// 3f. 未完成音节的前缀查找
	if partial != "" {
		if ps, ok := e.dict.(dict.PrefixSearchable); ok {
			partialPrefix := input
			prefixResults := ps.LookupPrefix(partialPrefix, 30)
			for i, cand := range prefixResults {
				if _, exists := candidatesMap[cand.Text]; exists {
					continue
				}
				c := cand
				c.Weight = weightPartialPrefix - i
				c.ConsumedLength = len(input)
				candidatesMap[c.Text] = &c
			}
		}
		// 按完整音节前缀查找单字
		if len(completedSyllables) == 0 {
			st := e.syllableTrie
			possibles := st.GetPossibleSyllables(partial)
			for _, syllable := range possibles {
				charResults := e.dict.Lookup(syllable)
				for j, cand := range charResults {
					if _, exists := candidatesMap[cand.Text]; exists {
						continue
					}
					c := cand
					c.Weight = weightPartialChar - j
					c.ConsumedLength = len(input)
					candidatesMap[c.Text] = &c
				}
			}
		}
	}

	// 4. 转换为列表
	result.Candidates = make([]candidate.Candidate, 0, len(candidatesMap))
	for _, cand := range candidatesMap {
		result.Candidates = append(result.Candidates, *cand)
	}

	// 5. 排序（根据排序模式）
	e.sortCandidates(result.Candidates, candidateOrder, syllableCount)

	// 6. 应用过滤
	filterMode := "smart"
	if e.config != nil && e.config.FilterMode != "" {
		filterMode = e.config.FilterMode
	}
	beforeFilter := len(result.Candidates)
	result.Candidates = candidate.FilterCandidates(result.Candidates, filterMode)
	log.Printf("[PinyinEngine.ConvertEx] Filter: mode=%s before=%d after=%d", filterMode, beforeFilter, len(result.Candidates))

	// 7. 检查是否空码
	if len(result.Candidates) == 0 {
		result.IsEmpty = true
		result.NeedRefine = result.Composition.HasPartial()
	}

	// 8. 限制数量
	if maxCandidates > 0 && len(result.Candidates) > maxCandidates {
		result.Candidates = result.Candidates[:maxCandidates]
		result.HasMore = true
	}

	// 9. 添加五笔编码提示
	e.addWubiHints(result.Candidates)

	log.Printf("[PinyinEngine.ConvertEx] final candidates=%d isEmpty=%v",
		len(result.Candidates), result.IsEmpty)

	return result
}

// sortCandidates 根据排序模式对候选进行排序
func (e *Engine) sortCandidates(candidates []candidate.Candidate, order string, syllableCount int) {
	switch order {
	case "phrase_first":
		// 词组优先：词组排在单字前面（在同级别内按权重排序）
		sort.SliceStable(candidates, func(i, j int) bool {
			iLen := len([]rune(candidates[i].Text))
			jLen := len([]rune(candidates[j].Text))
			iIsPhrase := iLen > 1
			jIsPhrase := jLen > 1
			if iIsPhrase != jIsPhrase {
				return iIsPhrase // 词组排前面
			}
			return candidates[i].Weight > candidates[j].Weight
		})
	case "smart":
		// 智能混排：完全按权重排序，但对单字候选使用 Unigram 分数微调
		if e.unigram != nil {
			for i := range candidates {
				if len([]rune(candidates[i].Text)) == 1 && candidates[i].Weight >= weightSingleChar && candidates[i].Weight < weightPartialChar {
					// 用 Unigram 分数微调单字权重
					lmScore := e.unigram.LogProb(candidates[i].Text)
					// 将 logprob（通常 -20 到 -5）映射到 0-9999 的范围
					bonus := int((lmScore + 20) * 600)
					if bonus < 0 {
						bonus = 0
					}
					if bonus > 9999 {
						bonus = 9999
					}
					candidates[i].Weight = weightSingleChar + bonus
				}
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Weight > candidates[j].Weight
		})
	default: // "char_first" 或默认
		// 单字优先：默认按权重排序即可（权重体系已保证单字在同音节下排在词组前面的逻辑）
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Weight > candidates[j].Weight
		})
	}
}

// lookupSubPhrasesEx 查找子词组
func (e *Engine) lookupSubPhrasesEx(syllables []string, candidatesMap map[string]*candidate.Candidate) {
	n := len(syllables)
	// 查找所有连续子序列组成的词组
	for length := n; length >= 2; length-- {
		for start := 0; start <= n-length; start++ {
			subKey := strings.Join(syllables[start:start+length], "")
			results := e.dict.Lookup(subKey)
			for i, cand := range results {
				if _, exists := candidatesMap[cand.Text]; exists {
					continue
				}
				c := cand
				charCount := len([]rune(c.Text))
				// 子词组权重：词越长越优先，匹配字数越接近音节数越好
				bonus := length * 100000
				if charCount == length {
					bonus += 50000
				}
				c.Weight = weightSubPhrase + bonus - start*10000 - i
				// 计算该子词组消耗的输入长度
				consumedLen := 0
				for k := 0; k < start+length; k++ {
					consumedLen += len(syllables[k])
				}
				c.ConsumedLength = consumedLen
				candidatesMap[c.Text] = &c
			}
		}
	}
}

// ============================================================
// 便捷方法
// ============================================================

// ParseInput 仅解析输入，不查询词库
// 用于 UI 层获取组合态显示
func (e *Engine) ParseInput(input string) *CompositionState {
	if len(input) == 0 {
		return &CompositionState{}
	}

	input = strings.ToLower(input)
	parser := NewPinyinParser()
	parsed := parser.Parse(input)

	builder := NewCompositionBuilder()
	return builder.Build(parsed)
}

// GetPossibleSyllables 获取以 prefix 开头的所有可能音节
// 用于 UI 显示可能的续写提示
func (e *Engine) GetPossibleSyllables(prefix string) []string {
	return e.syllableTrie.GetPossibleSyllables(strings.ToLower(prefix))
}

// IsValidSyllable 检查是否是有效的完整音节
func (e *Engine) IsValidSyllable(syllable string) bool {
	return e.syllableTrie.Contains(strings.ToLower(syllable))
}

// IsValidSyllablePrefix 检查是否是有效的音节前缀
func (e *Engine) IsValidSyllablePrefix(prefix string) bool {
	return e.syllableTrie.HasPrefix(strings.ToLower(prefix))
}
