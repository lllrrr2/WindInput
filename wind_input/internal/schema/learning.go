package schema

import (
	"github.com/huanfeng/wind_input/internal/candidate"
	"github.com/huanfeng/wind_input/internal/dict"
)

// LearningStrategy 学习策略接口
// 由方案配置中的 learning.mode 决定使用哪种策略
type LearningStrategy interface {
	// OnCandidateCommitted 用户提交候选词时的回调
	OnCandidateCommitted(code, text string, cand candidate.Candidate)

	// Reset 重置学习状态
	Reset()
}

// ManualLearning 手动学习策略（codetable 类型默认）
// 不自动造词，用户通过右键菜单操作 Shadow
type ManualLearning struct{}

func (m *ManualLearning) OnCandidateCommitted(code, text string, cand candidate.Candidate) {
	// 手动模式不自动学词
}

func (m *ManualLearning) Reset() {}

// AutoLearning 自动学习策略（pinyin 类型默认）
// 选词即学，记录到临时词库（如有），否则记录到用户词库
type AutoLearning struct {
	userDict *dict.UserDict
	tempDict *dict.TempDict
}

func NewAutoLearning(userDict *dict.UserDict) *AutoLearning {
	return &AutoLearning{userDict: userDict}
}

// SetTempDict 设置临时词库（自动学习优先写入临时词库）
func (a *AutoLearning) SetTempDict(td *dict.TempDict) {
	a.tempDict = td
}

func (a *AutoLearning) OnCandidateCommitted(code, text string, cand candidate.Candidate) {
	if cand.IsCommand {
		return
	}
	// 仅学习多字词
	if len([]rune(text)) < 2 {
		return
	}

	// 优先写入临时词库
	if a.tempDict != nil {
		promoted := a.tempDict.LearnWord(code, text, 10)
		if promoted {
			// 达到晋升条件，自动迁移到用户词库
			a.tempDict.PromoteWord(code, text)
		}
		return
	}

	// 没有临时词库时，直接写入用户词库（兼容旧行为）
	if a.userDict != nil {
		a.userDict.IncreaseWeight(code, text, 10)
	}
}

func (a *AutoLearning) Reset() {}

// FrequencyLearning 仅调频策略
// 不造新词，仅增加已有词条的选择频次
type FrequencyLearning struct {
	userDict *dict.UserDict
}

func NewFrequencyLearning(userDict *dict.UserDict) *FrequencyLearning {
	return &FrequencyLearning{userDict: userDict}
}

func (f *FrequencyLearning) OnCandidateCommitted(code, text string, cand candidate.Candidate) {
	if f.userDict == nil || cand.IsCommand {
		return
	}
	f.userDict.IncreaseWeight(code, text, 1)
}

func (f *FrequencyLearning) Reset() {}

// NewLearningStrategy 根据方案配置创建学习策略
func NewLearningStrategy(mode LearningMode, userDict *dict.UserDict) LearningStrategy {
	switch mode {
	case LearningAuto:
		return NewAutoLearning(userDict)
	case LearningFrequency:
		return NewFrequencyLearning(userDict)
	default:
		return &ManualLearning{}
	}
}
