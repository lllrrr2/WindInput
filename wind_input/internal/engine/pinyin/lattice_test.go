package pinyin

import (
	"testing"
)

func TestBuildLattice(t *testing.T) {
	d := createTestDictForViterbi(t)
	unigram := createTestUnigram(t)
	st := NewSyllableTrie()

	lattice := BuildLattice("nihao", st, d, nil)
	if lattice.IsEmpty() {
		// nihao 不在测试词库中，所以可能为空
		t.Log("BuildLattice(nihao) 返回空网格（词库中无此词）")
	}

	// 测试有词的情况
	lattice = BuildLattice("jintian", st, d, unigram)
	if lattice.IsEmpty() {
		t.Error("BuildLattice(jintian) 返回空网格")
	}

	if lattice.Size() < 1 {
		t.Errorf("网格节点数 = %d, 期望至少 1", lattice.Size())
	}

	// 检查结束位置节点
	endNodes := lattice.GetNodesEndingAt(len("jintian"))
	if len(endNodes) == 0 {
		t.Log("无结束于最终位置的节点")
	}
}

func TestBuildLatticeWithUnigram(t *testing.T) {
	d := createTestDictForViterbi(t)
	unigram := createTestUnigram(t)
	st := NewSyllableTrie()

	lattice := BuildLattice("henhao", st, d, unigram)
	if lattice.IsEmpty() {
		t.Fatal("BuildLattice(henhao) 返回空网格")
	}

	// 应该有"很好"这个词组节点
	endPos := len("henhao")
	nodes := lattice.GetNodesEndingAt(endPos)

	foundHenhao := false
	for _, node := range nodes {
		if node.Word == "很好" {
			foundHenhao = true
			t.Logf("找到节点: %s (logprob=%.4f)", node.Word, node.LogProb)
		}
	}

	if !foundHenhao {
		t.Error("网格中未找到 '很好' 节点")
	}
}

func TestLatticeKey(t *testing.T) {
	key1 := latticeKey(0, 3, "你")
	key2 := latticeKey(0, 3, "你")
	key3 := latticeKey(0, 3, "泥")

	if key1 != key2 {
		t.Error("相同参数应生成相同 key")
	}
	if key1 == key3 {
		t.Error("不同词应生成不同 key")
	}
}
