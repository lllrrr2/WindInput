package pinyin

import (
	"sort"

	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
)

// Engine 拼音引擎
type Engine struct {
	dict dict.Dict
}

// NewEngine 创建拼音引擎
func NewEngine(d dict.Dict) *Engine {
	return &Engine{
		dict: d,
	}
}

// Convert 转换拼音为候选词
func (e *Engine) Convert(input string, maxCandidates int) ([]candidate.Candidate, error) {
	if len(input) == 0 {
		return nil, nil
	}

	// 解析音节
	syllablesList := ParseSyllables(input)
	if len(syllablesList) == 0 {
		return nil, nil
	}

	// 收集所有候选词
	candidatesMap := make(map[string]*candidate.Candidate)

	// 对每一种音节分割方式
	for _, syllables := range syllablesList {
		// 尝试查找完整短语
		phraseCandidates := e.dict.LookupPhrase(syllables)
		for _, cand := range phraseCandidates {
			if existing, ok := candidatesMap[cand.Text]; ok {
				// 如果已存在，保留权重较高的
				if cand.Weight > existing.Weight {
					*existing = cand
				}
			} else {
				c := cand
				candidatesMap[cand.Text] = &c
			}
		}

		// 如果是单个音节，直接查找
		if len(syllables) == 1 {
			singleCandidates := e.dict.Lookup(syllables[0])
			for _, cand := range singleCandidates {
				if existing, ok := candidatesMap[cand.Text]; ok {
					if cand.Weight > existing.Weight {
						*existing = cand
					}
				} else {
					c := cand
					candidatesMap[cand.Text] = &c
				}
			}
		} else {
			// 多音节：尝试组合查找
			// 这里简化处理，实际可以实现更复杂的组合逻辑
			for _, syllable := range syllables {
				singleCandidates := e.dict.Lookup(syllable)
				for _, cand := range singleCandidates {
					// 降低组合词的权重
					cand.Weight = cand.Weight / 2
					if existing, ok := candidatesMap[cand.Text]; ok {
						if cand.Weight > existing.Weight {
							*existing = cand
						}
					} else {
						c := cand
						candidatesMap[cand.Text] = &c
					}
				}
			}
		}
	}

	// 转换为列表并排序
	candidates := make(candidate.CandidateList, 0, len(candidatesMap))
	for _, cand := range candidatesMap {
		candidates = append(candidates, *cand)
	}

	sort.Sort(candidates)

	// 限制返回数量
	if maxCandidates > 0 && len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	return candidates, nil
}

// Reset 重置引擎状态
func (e *Engine) Reset() {
	// 拼音引擎目前无状态，无需重置
}
