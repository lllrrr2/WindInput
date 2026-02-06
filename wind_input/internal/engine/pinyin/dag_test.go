package pinyin

import (
	"strings"
	"testing"
)

func TestBuildDAG(t *testing.T) {
	st := NewSyllableTrie()

	dag := BuildDAG("nihao", st)
	if dag == nil {
		t.Fatal("BuildDAG 返回 nil")
	}

	// 位置 0 应该有 "ni" 匹配
	if len(dag.nodes[0]) == 0 {
		t.Error("DAG 位置 0 无节点")
	}

	found := false
	for _, node := range dag.nodes[0] {
		if node.Syllables[0] == "ni" {
			found = true
			break
		}
	}
	if !found {
		t.Error("DAG 位置 0 未找到 'ni'")
	}
}

func TestDAGMaximumMatch(t *testing.T) {
	st := NewSyllableTrie()

	tests := []struct {
		input string
		want  string // 用空格分隔的音节
	}{
		{"nihao", "ni hao"},
		{"wo", "wo"},
		{"zhongguo", "zhong guo"},
		{"xian", "xian"},
		{"women", "wo men"},
	}

	for _, tt := range tests {
		dag := BuildDAG(tt.input, st)
		result := dag.MaximumMatch()
		got := strings.Join(result, " ")
		if got != tt.want {
			t.Errorf("MaximumMatch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDAGAllPaths(t *testing.T) {
	st := NewSyllableTrie()

	// "xian" 可以切分为 "xian" 或 "xi an"
	dag := BuildDAG("xian", st)
	paths := dag.AllPaths(10)

	if len(paths) < 2 {
		t.Errorf("AllPaths(xian) 返回 %d 条路径, 期望至少 2", len(paths))
	}

	// 验证包含 ["xian"] 和 ["xi", "an"]
	foundFull := false
	foundSplit := false
	for _, path := range paths {
		joined := strings.Join(path, " ")
		if joined == "xian" {
			foundFull = true
		}
		if joined == "xi an" {
			foundSplit = true
		}
	}
	if !foundFull {
		t.Error("AllPaths(xian) 未包含 [xian]")
	}
	if !foundSplit {
		t.Error("AllPaths(xian) 未包含 [xi an]")
	}
}

func TestDAGAllPathsLimit(t *testing.T) {
	st := NewSyllableTrie()

	dag := BuildDAG("xian", st)
	paths := dag.AllPaths(1)

	if len(paths) > 1 {
		t.Errorf("AllPaths(limit=1) 返回 %d 条, 期望最多 1", len(paths))
	}
}

func TestDAGIsFullMatch(t *testing.T) {
	st := NewSyllableTrie()

	// 合法拼音
	dag := BuildDAG("nihao", st)
	if !dag.IsFullMatch() {
		t.Error("IsFullMatch(nihao) = false, want true")
	}

	// 不合法
	dag = BuildDAG("xyz", st)
	if dag.IsFullMatch() {
		t.Error("IsFullMatch(xyz) = true, want false")
	}
}
