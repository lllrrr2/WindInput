package dict

import (
	"github.com/huanfeng/wind_input/internal/candidate"
)

// Dict 词库接口
type Dict interface {
	// Lookup 查找拼音对应的候选词
	Lookup(pinyin string) []candidate.Candidate

	// LookupPhrase 查找短语
	LookupPhrase(syllables []string) []candidate.Candidate
}

// PrefixSearchable 支持前缀搜索的词库接口（可选扩展）
// 拼音引擎通过类型断言使用：if ps, ok := d.(PrefixSearchable); ok { ... }
type PrefixSearchable interface {
	LookupPrefix(prefix string, limit int) []candidate.Candidate
}
