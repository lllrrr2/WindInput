package engine

import "github.com/huanfeng/wind_input/internal/candidate"

// Engine 输入引擎接口
type Engine interface {
	// Convert 转换输入为候选词
	Convert(input string, maxCandidates int) ([]candidate.Candidate, error)

	// Reset 重置引擎状态
	Reset()
}
