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

	// Load 加载词库
	Load(path string) error
}

// Entry 词库条目
type Entry struct {
	Pinyin string
	Text   string
	Weight int
}
