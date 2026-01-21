package candidate

// Candidate 候选词
type Candidate struct {
	Text   string // 候选文字
	Pinyin string // 拼音
	Weight int    // 权重（用于排序）
}

// CandidateList 候选词列表
type CandidateList []Candidate

// Len 返回候选词数量
func (c CandidateList) Len() int {
	return len(c)
}

// Less 比较候选词（按权重降序）
func (c CandidateList) Less(i, j int) bool {
	return c[i].Weight > c[j].Weight
}

// Swap 交换候选词
func (c CandidateList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
