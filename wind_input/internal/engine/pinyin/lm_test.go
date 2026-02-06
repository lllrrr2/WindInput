package pinyin

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestUnigramModelLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "unigram.txt")

	content := `我	1000
的	800
是	600
你好	500
中国	400
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	m := NewUnigramModel()
	if err := m.Load(path); err != nil {
		t.Fatalf("加载模型失败: %v", err)
	}

	if m.Size() != 5 {
		t.Errorf("Size() = %d, want 5", m.Size())
	}

	// "我" 应该有最高概率
	probWo := m.LogProb("我")
	probDe := m.LogProb("的")
	if probWo <= probDe {
		t.Errorf("LogProb(我)=%f 应该 > LogProb(的)=%f", probWo, probDe)
	}

	// 未知词应返回 minProb
	probUnk := m.LogProb("未知词")
	if probUnk >= probDe {
		t.Errorf("LogProb(未知词)=%f 应该 < LogProb(的)=%f", probUnk, probDe)
	}

	if !m.Contains("我") {
		t.Error("Contains(我) = false, want true")
	}
	if m.Contains("未知词") {
		t.Error("Contains(未知词) = true, want false")
	}
}

func TestUnigramLoadFromFreqMap(t *testing.T) {
	m := NewUnigramModel()
	freqs := map[string]float64{
		"你好": 100,
		"世界": 50,
	}
	m.LoadFromFreqMap(freqs)

	if m.Size() != 2 {
		t.Errorf("Size() = %d, want 2", m.Size())
	}

	p1 := m.LogProb("你好")
	p2 := m.LogProb("世界")
	if p1 <= p2 {
		t.Errorf("LogProb(你好)=%f 应该 > LogProb(世界)=%f", p1, p2)
	}
}

func TestBigramModel(t *testing.T) {
	// 创建 Unigram
	uni := NewUnigramModel()
	uni.LoadFromFreqMap(map[string]float64{
		"今天": 100,
		"天气": 80,
		"很好": 60,
	})

	// 创建 Bigram
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bigram.txt")
	content := `今天	天气	50
天气	很好	30
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	bi := NewBigramModel(uni)
	if err := bi.Load(path); err != nil {
		t.Fatalf("加载 Bigram 失败: %v", err)
	}

	// P(天气|今天) 应该 > P(天气)（Bigram 信息增益）
	probBi := bi.LogProb("今天", "天气")
	probUni := uni.LogProb("天气")
	if probBi < probUni {
		t.Errorf("P(天气|今天)=%f 应该 >= P(天气)=%f", probBi, probUni)
	}
}

func TestLogSumExp(t *testing.T) {
	// log(exp(-1) + exp(-2)) ≈ -0.6867
	result := logSumExp(-1, -2)
	expected := math.Log(math.Exp(-1) + math.Exp(-2))
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("logSumExp(-1, -2) = %f, want %f", result, expected)
	}
}
