package pinyin

import (
	"strings"

	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
)

// ============================================================
// SyllableLexicon 音节词库接口
// 支持基于音节序列的查询，而非简单的编码字符串
// ============================================================

// SyllableLexicon 音节词库接口
type SyllableLexicon interface {
	// LookupBySyllables 根据完整音节序列查找候选词
	// 例如: LookupBySyllables(["ni", "hao"]) -> [你好, ...]
	LookupBySyllables(syllables []string) []LexiconEntry

	// LookupBySyllablesPrefix 根据音节序列+最后一个未完成音节前缀查找
	// 例如: LookupBySyllablesPrefix(["ni"], "ha") -> [你好, 你还, ...]
	LookupBySyllablesPrefix(completedSyllables []string, partialLast string) []LexiconEntry

	// LookupSingleChar 根据完整音节查找单字
	// 例如: LookupSingleChar("ni") -> [你, 妮, 泥, ...]
	LookupSingleChar(syllable string) []LexiconEntry

	// LookupSingleCharPrefix 根据音节前缀查找单字
	// 例如: LookupSingleCharPrefix("zh") -> [中, 之, 知, ...]
	LookupSingleCharPrefix(prefix string) []LexiconEntry
}

// ============================================================
// CodeTableLexiconAdapter 适配器
// 将现有的 CodeTable/Dict 适配为 SyllableLexicon 接口
// ============================================================

// CodeTableLexiconAdapter 将现有 dict.Dict 适配为 SyllableLexicon
type CodeTableLexiconAdapter struct {
	dict   dict.Dict
	parser *PinyinParser
}

// NewCodeTableLexiconAdapter 创建 CodeTable 适配器
func NewCodeTableLexiconAdapter(d dict.Dict) *CodeTableLexiconAdapter {
	return &CodeTableLexiconAdapter{
		dict:   d,
		parser: NewPinyinParser(),
	}
}

// LookupBySyllables 根据完整音节序列查找候选词
func (a *CodeTableLexiconAdapter) LookupBySyllables(syllables []string) []LexiconEntry {
	if len(syllables) == 0 {
		return nil
	}

	// 将音节拼接为编码字符串
	code := strings.Join(syllables, "")

	// 查找词库
	candidates := a.dict.Lookup(code)

	// 转换为 LexiconEntry
	return a.candidatesToEntries(candidates, syllables)
}

// LookupBySyllablesPrefix 根据音节序列+未完成音节前缀查找
func (a *CodeTableLexiconAdapter) LookupBySyllablesPrefix(completedSyllables []string, partialLast string) []LexiconEntry {
	// 构建前缀字符串
	var prefix string
	if len(completedSyllables) > 0 {
		prefix = strings.Join(completedSyllables, "") + partialLast
	} else {
		prefix = partialLast
	}

	if prefix == "" {
		return nil
	}

	// 使用前缀搜索（如果支持）
	if ps, ok := a.dict.(dict.PrefixSearchable); ok {
		candidates := ps.LookupPrefix(prefix, 100)

		// 过滤：只保留音节数匹配的候选
		// 例如，查找 "ni" + "ha" 时，应该返回 2 音节的词
		expectedSyllableCount := len(completedSyllables) + 1
		var filtered []LexiconEntry
		for _, cand := range candidates {
			// 解析候选词的编码确定音节数
			parsed := a.parser.Parse(cand.Code)
			syllableCount := len(parsed.Syllables)

			// 允许匹配或更长的词组
			if syllableCount >= expectedSyllableCount {
				entry := LexiconEntry{
					Text:      cand.Text,
					Syllables: parsed.SyllableTexts(),
					Freq:      cand.Weight,
					Source:    a.getSource(cand),
				}
				filtered = append(filtered, entry)
			}
		}
		return filtered
	}

	// 回退：使用精确查找
	return a.LookupBySyllables(append(completedSyllables, partialLast))
}

// LookupSingleChar 根据完整音节查找单字
func (a *CodeTableLexiconAdapter) LookupSingleChar(syllable string) []LexiconEntry {
	candidates := a.dict.Lookup(syllable)

	// 过滤只保留单字
	var entries []LexiconEntry
	for _, cand := range candidates {
		runes := []rune(cand.Text)
		if len(runes) == 1 {
			entry := LexiconEntry{
				Text:      cand.Text,
				Syllables: []string{syllable},
				Freq:      cand.Weight,
				Source:    a.getSource(cand),
			}
			entries = append(entries, entry)
		}
	}
	return entries
}

// LookupSingleCharPrefix 根据音节前缀查找单字
func (a *CodeTableLexiconAdapter) LookupSingleCharPrefix(prefix string) []LexiconEntry {
	if ps, ok := a.dict.(dict.PrefixSearchable); ok {
		candidates := ps.LookupPrefix(prefix, 50)

		// 过滤只保留单字
		var entries []LexiconEntry
		for _, cand := range candidates {
			runes := []rune(cand.Text)
			if len(runes) == 1 {
				entry := LexiconEntry{
					Text:      cand.Text,
					Syllables: []string{cand.Code}, // 使用完整编码作为音节
					Freq:      cand.Weight,
					Source:    a.getSource(cand),
				}
				entries = append(entries, entry)
			}
		}
		return entries
	}
	return nil
}

// candidatesToEntries 将 candidate.Candidate 转换为 LexiconEntry
func (a *CodeTableLexiconAdapter) candidatesToEntries(candidates []candidate.Candidate, syllables []string) []LexiconEntry {
	entries := make([]LexiconEntry, 0, len(candidates))
	for _, cand := range candidates {
		entry := LexiconEntry{
			Text:      cand.Text,
			Syllables: syllables,
			Freq:      cand.Weight,
			Source:    a.getSource(cand),
		}
		entries = append(entries, entry)
	}
	return entries
}

// getSource 从候选词推断来源
func (a *CodeTableLexiconAdapter) getSource(cand candidate.Candidate) EntrySource {
	// 根据候选词的 Category 或其他标记判断来源
	// 目前简化处理，都视为系统词库
	return SourceSystem
}

// ============================================================
// LexiconQuery 词库查询器
// 封装查询逻辑，支持多种查询策略
// ============================================================

// LexiconQuery 词库查询器
type LexiconQuery struct {
	lexicon SyllableLexicon
	parser  *PinyinParser
}

// NewLexiconQuery 创建词库查询器
func NewLexiconQuery(lexicon SyllableLexicon) *LexiconQuery {
	return &LexiconQuery{
		lexicon: lexicon,
		parser:  NewPinyinParser(),
	}
}

// QueryResult 查询结果
type QueryResult struct {
	Entries          []LexiconEntry // 候选词条
	IsPrefixMatch    bool           // 是否为前缀匹配
	MatchedSyllables int            // 匹配的音节数
}

// Query 根据解析结果查询候选词
func (q *LexiconQuery) Query(parsed *ParseResult, maxResults int) *QueryResult {
	if parsed == nil || len(parsed.Syllables) == 0 {
		return &QueryResult{}
	}

	result := &QueryResult{}

	// 获取完整音节和未完成音节
	completed := parsed.CompletedSyllables()
	partial := parsed.PartialSyllable()

	if partial != "" {
		// 有未完成音节：使用前缀查询
		entries := q.lexicon.LookupBySyllablesPrefix(completed, partial)
		result.Entries = entries
		result.IsPrefixMatch = true
		result.MatchedSyllables = len(completed) + 1
	} else if len(completed) > 0 {
		// 全部是完整音节：精确查询
		entries := q.lexicon.LookupBySyllables(completed)
		result.Entries = entries
		result.IsPrefixMatch = false
		result.MatchedSyllables = len(completed)
	}

	// 限制结果数量
	if maxResults > 0 && len(result.Entries) > maxResults {
		result.Entries = result.Entries[:maxResults]
	}

	return result
}

// QueryWithFallback 带回退的查询
// 如果主查询无结果，尝试查询更短的音节序列
func (q *LexiconQuery) QueryWithFallback(parsed *ParseResult, maxResults int) *QueryResult {
	mainResult := q.Query(parsed, maxResults)
	if len(mainResult.Entries) > 0 {
		return mainResult
	}

	// 主查询无结果，尝试单字查询
	completed := parsed.CompletedSyllables()
	if len(completed) > 0 {
		// 查找第一个音节的单字
		singleEntries := q.lexicon.LookupSingleChar(completed[0])
		if len(singleEntries) > 0 {
			if maxResults > 0 && len(singleEntries) > maxResults {
				singleEntries = singleEntries[:maxResults]
			}
			return &QueryResult{
				Entries:          singleEntries,
				IsPrefixMatch:    false,
				MatchedSyllables: 1,
			}
		}
	}

	// 如果有未完成音节，尝试前缀查询单字
	partial := parsed.PartialSyllable()
	if partial != "" {
		prefixEntries := q.lexicon.LookupSingleCharPrefix(partial)
		if len(prefixEntries) > 0 {
			if maxResults > 0 && len(prefixEntries) > maxResults {
				prefixEntries = prefixEntries[:maxResults]
			}
			return &QueryResult{
				Entries:          prefixEntries,
				IsPrefixMatch:    true,
				MatchedSyllables: 1,
			}
		}
	}

	return mainResult
}
