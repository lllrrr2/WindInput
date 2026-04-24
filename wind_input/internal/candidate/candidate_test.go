package candidate

import (
	"sort"
	"testing"
)

func TestBetter_SameWeight_DifferentCode_GlobalOrder(t *testing.T) {
	// 模拟前缀匹配场景：三个不同编码的候选，同权重，按全局顺序排序
	// 码表文件顺序: uukg(三, order=0) → uuka(一, order=1) → uukc(二, order=2)
	candidates := CandidateList{
		{Text: "一", Code: "uuka", Weight: 1, NaturalOrder: 1},
		{Text: "二", Code: "uukc", Weight: 1, NaturalOrder: 2},
		{Text: "三", Code: "uukg", Weight: 1, NaturalOrder: 0},
	}

	sort.Sort(candidates)

	expected := []string{"三", "一", "二"}
	for i, exp := range expected {
		if candidates[i].Text != exp {
			t.Errorf("Better 排序 [%d]: 期望 %q, 实际 %q", i, exp, candidates[i].Text)
		}
	}
}

func TestBetter_DifferentWeight(t *testing.T) {
	// 不同权重时，权重高的排前面，不受 NaturalOrder 影响
	candidates := CandidateList{
		{Text: "低", Code: "aa", Weight: 10, NaturalOrder: 0},
		{Text: "高", Code: "bb", Weight: 100, NaturalOrder: 5},
	}

	sort.Sort(candidates)

	if candidates[0].Text != "高" {
		t.Errorf("期望 '高' 排第一, 实际 %q", candidates[0].Text)
	}
}

func TestBetter_SameCode_SameWeight_GlobalOrder(t *testing.T) {
	// 同编码同权重时，NaturalOrder 小的排前面
	candidates := CandidateList{
		{Text: "B", Code: "aa", Weight: 10, NaturalOrder: 5},
		{Text: "A", Code: "aa", Weight: 10, NaturalOrder: 2},
	}

	sort.Sort(candidates)

	if candidates[0].Text != "A" {
		t.Errorf("期望 'A'(order=2) 排第一, 实际 %q", candidates[0].Text)
	}
}

func TestBetterNatural_CrossCode(t *testing.T) {
	// BetterNatural 模式下，跨编码也按全局顺序排
	candidates := []Candidate{
		{Text: "一", Code: "uuka", Weight: 100, NaturalOrder: 5},
		{Text: "三", Code: "uukg", Weight: 100, NaturalOrder: 1},
		{Text: "二", Code: "uukc", Weight: 100, NaturalOrder: 3},
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return BetterNatural(candidates[i], candidates[j])
	})

	expected := []string{"三", "二", "一"}
	for i, exp := range expected {
		if candidates[i].Text != exp {
			t.Errorf("BetterNatural 排序 [%d]: 期望 %q, 实际 %q", i, exp, candidates[i].Text)
		}
	}
}

func TestBetter_PrefixPenalty_PreservesFileOrder(t *testing.T) {
	// 模拟实际前缀匹配场景：原始 weight=1，降权后都变成 -999999
	// 验证同权重时按 NaturalOrder（文件顺序）排序
	penalty := 1000000
	candidates := CandidateList{
		{Text: "一", Code: "uuka", Weight: 1 - penalty, NaturalOrder: 1},
		{Text: "二", Code: "uukc", Weight: 1 - penalty, NaturalOrder: 2},
		{Text: "三", Code: "uukg", Weight: 1 - penalty, NaturalOrder: 0},
	}

	sort.Sort(candidates)

	expected := []string{"三", "一", "二"}
	for i, exp := range expected {
		if candidates[i].Text != exp {
			t.Errorf("降权后排序 [%d]: 期望 %q, 实际 %q", i, exp, candidates[i].Text)
		}
	}
}
