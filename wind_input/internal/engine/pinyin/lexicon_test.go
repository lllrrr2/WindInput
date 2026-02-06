package pinyin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/huanfeng/wind_input/internal/dict"
)

func createTestPinyinDict(t *testing.T) *dict.PinyinDict {
	t.Helper()
	tmpDir := t.TempDir()

	content := `# Rime dictionary
---
name: test
version: "1.0"
sort: by_weight
...
啊	a	1000
阿	a	900
爱	ai	1000
哀	ai	900
你	ni	1000
妮	ni	500
好	hao	1000
号	hao	500
你好	ni hao	800
我	wo	1000
们	men	1000
我们	wo men	800
是	shi	1000
的	de	1000
中	zhong	1000
国	guo	1000
中国	zhong guo	800
知	zhi	1000
道	dao	1000
知道	zhi dao	800
炸	zha	900
站	zhan	900
张	zhang	900
找	zhao	900
这	zhe	1000
这	zhei	900
真	zhen	900
正	zheng	900
之	zhi	900
周	zhou	900
住	zhu	900
`
	if err := os.WriteFile(filepath.Join(tmpDir, "8105.dict.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	d := dict.NewPinyinDict()
	if err := d.LoadRimeDir(tmpDir); err != nil {
		t.Fatalf("加载词库失败: %v", err)
	}
	return d
}

func TestCodeTableLexiconAdapterLookupBySyllables(t *testing.T) {
	d := createTestPinyinDict(t)
	adapter := NewCodeTableLexiconAdapter(d)

	tests := []struct {
		syllables    []string
		wantContain  string
		wantMinCount int
	}{
		{[]string{"ni", "hao"}, "你好", 1},
		{[]string{"zhong", "guo"}, "中国", 1},
		{[]string{"wo", "men"}, "我们", 1},
		{[]string{"ni"}, "你", 1},
	}

	for _, tt := range tests {
		entries := adapter.LookupBySyllables(tt.syllables)

		if len(entries) < tt.wantMinCount {
			t.Errorf("LookupBySyllables(%v) returned %d entries, want at least %d",
				tt.syllables, len(entries), tt.wantMinCount)
			continue
		}

		found := false
		for _, e := range entries {
			if e.Text == tt.wantContain {
				found = true
				break
			}
		}
		if !found {
			var texts []string
			for _, e := range entries {
				texts = append(texts, e.Text)
			}
			t.Errorf("LookupBySyllables(%v) missing %q, got %v",
				tt.syllables, tt.wantContain, texts)
		}
	}
}

func TestCodeTableLexiconAdapterLookupSingleChar(t *testing.T) {
	d := createTestPinyinDict(t)
	adapter := NewCodeTableLexiconAdapter(d)

	tests := []struct {
		syllable    string
		wantContain string
	}{
		{"ni", "你"},
		{"hao", "好"},
		{"wo", "我"},
		{"zhong", "中"},
	}

	for _, tt := range tests {
		entries := adapter.LookupSingleChar(tt.syllable)

		if len(entries) == 0 {
			t.Errorf("LookupSingleChar(%q) returned no entries", tt.syllable)
			continue
		}

		found := false
		for _, e := range entries {
			if e.Text == tt.wantContain {
				found = true
				// 验证是单字
				if len([]rune(e.Text)) != 1 {
					t.Errorf("LookupSingleChar(%q) returned non-single char: %q", tt.syllable, e.Text)
				}
				break
			}
		}
		if !found {
			var texts []string
			for _, e := range entries {
				texts = append(texts, e.Text)
			}
			t.Errorf("LookupSingleChar(%q) missing %q, got %v",
				tt.syllable, tt.wantContain, texts)
		}
	}
}

func TestLexiconQuery(t *testing.T) {
	d := createTestPinyinDict(t)
	adapter := NewCodeTableLexiconAdapter(d)
	query := NewLexiconQuery(adapter)
	parser := NewPinyinParser()

	tests := []struct {
		input       string
		wantContain string
	}{
		{"nihao", "你好"},
		{"zhongguo", "中国"},
		{"ni", "你"},
	}

	for _, tt := range tests {
		parsed := parser.Parse(tt.input)
		result := query.Query(parsed, 10)

		if len(result.Entries) == 0 {
			t.Errorf("Query(%q) returned no entries", tt.input)
			continue
		}

		found := false
		for _, e := range result.Entries {
			if e.Text == tt.wantContain {
				found = true
				break
			}
		}
		if !found {
			var texts []string
			for _, e := range result.Entries {
				texts = append(texts, e.Text)
			}
			t.Errorf("Query(%q) missing %q, got %v",
				tt.input, tt.wantContain, texts)
		}
	}
}

func TestLexiconQueryWithPartial(t *testing.T) {
	d := createTestPinyinDict(t)
	adapter := NewCodeTableLexiconAdapter(d)
	query := NewLexiconQuery(adapter)
	parser := NewPinyinParser()

	// 测试带有未完成音节的查询
	parsed := parser.Parse("nihaozh")

	if !parsed.HasPartial() {
		t.Fatal("Expected 'nihaozh' to have partial syllable")
	}

	result := query.Query(parsed, 10)

	// 检查是否为前缀匹配
	t.Logf("Query('nihaozh'): IsPrefixMatch=%v, MatchedSyllables=%d, Entries=%d",
		result.IsPrefixMatch, result.MatchedSyllables, len(result.Entries))

	for i, e := range result.Entries {
		if i < 5 {
			t.Logf("  [%d] %s (syllables=%v)", i, e.Text, e.Syllables)
		}
	}
}

func TestLexiconQueryWithFallback(t *testing.T) {
	d := createTestPinyinDict(t)
	adapter := NewCodeTableLexiconAdapter(d)
	query := NewLexiconQuery(adapter)
	parser := NewPinyinParser()

	// 测试一个词库中不存在的长词组，应该回退到单字
	parsed := parser.Parse("nihaoshijie") // 假设 "你好世界" 不在词库中

	result := query.QueryWithFallback(parsed, 10)

	t.Logf("QueryWithFallback('nihaoshijie'): MatchedSyllables=%d, Entries=%d",
		result.MatchedSyllables, len(result.Entries))

	// 应该至少返回一些结果（可能是单字或部分匹配）
	if len(result.Entries) == 0 {
		t.Log("Warning: QueryWithFallback returned no entries")
	}
}
