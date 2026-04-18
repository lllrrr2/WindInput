package dict

import (
	"os"
	"testing"
)

func TestEnglishDict_LoadAndSearch(t *testing.T) {
	// 创建临时 rime 格式文件
	content := `---
name: english
version: "1"
...
# 注释行
hello	100
world	200
help	50
helper	30
he	10

`
	tmpFile, err := os.CreateTemp("", "english_test_*.dict.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	dict := NewEnglishDict(nil)
	if err := dict.LoadRimeFile(tmpFile.Name()); err != nil {
		t.Fatalf("LoadRimeFile failed: %v", err)
	}

	// 检查词条数量
	if dict.EntryCount() != 5 {
		t.Errorf("EntryCount = %d, want 5", dict.EntryCount())
	}

	// 精确查询
	results := dict.Lookup("hello")
	if len(results) == 0 {
		t.Error("Lookup('hello') returned empty")
	} else if results[0].Text != "hello" {
		t.Errorf("Lookup('hello') text = %q, want 'hello'", results[0].Text)
	} else if results[0].Weight != 100 {
		t.Errorf("Lookup('hello') weight = %d, want 100", results[0].Weight)
	}

	// 大小写不敏感
	resultsUpper := dict.Lookup("HELLO")
	if len(resultsUpper) == 0 {
		t.Error("Lookup('HELLO') returned empty (case-insensitive failed)")
	}

	// 不存在的词
	notFound := dict.Lookup("nonexistent")
	if len(notFound) != 0 {
		t.Errorf("Lookup('nonexistent') = %v, want empty", notFound)
	}

	// 前缀查询
	prefixResults := dict.LookupPrefix("hel", 10)
	if len(prefixResults) == 0 {
		t.Error("LookupPrefix('hel') returned empty")
	}
	// 应包含 hello、help、helper
	found := make(map[string]bool)
	for _, c := range prefixResults {
		found[c.Text] = true
	}
	for _, word := range []string{"hello", "help", "helper"} {
		if !found[word] {
			t.Errorf("LookupPrefix('hel') missing %q", word)
		}
	}

	// limit 测试
	limitResults := dict.LookupPrefix("hel", 2)
	if len(limitResults) > 2 {
		t.Errorf("LookupPrefix('hel', 2) returned %d results, want <= 2", len(limitResults))
	}

	// 前缀大小写不敏感
	upperPrefix := dict.LookupPrefix("HEL", 10)
	if len(upperPrefix) == 0 {
		t.Error("LookupPrefix('HEL') returned empty (case-insensitive failed)")
	}
}

func TestEnglishDict_ThreeColumnFormat(t *testing.T) {
	content := `---
name: english
...
apple	apple	500
banana	banana	300
cherry	cherry	100
`
	tmpFile, err := os.CreateTemp("", "english_three_col_*.dict.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	dict := NewEnglishDict(nil)
	if err := dict.LoadRimeFile(tmpFile.Name()); err != nil {
		t.Fatalf("LoadRimeFile failed: %v", err)
	}

	if dict.EntryCount() != 3 {
		t.Errorf("EntryCount = %d, want 3", dict.EntryCount())
	}

	results := dict.Lookup("apple")
	if len(results) == 0 {
		t.Error("Lookup('apple') returned empty")
	} else {
		if results[0].Text != "apple" {
			t.Errorf("text = %q, want 'apple'", results[0].Text)
		}
		if results[0].Weight != 500 {
			t.Errorf("weight = %d, want 500", results[0].Weight)
		}
	}

	results2 := dict.Lookup("banana")
	if len(results2) == 0 {
		t.Error("Lookup('banana') returned empty")
	} else if results2[0].Weight != 300 {
		t.Errorf("banana weight = %d, want 300", results2[0].Weight)
	}
}

func TestEnglishDictLayer(t *testing.T) {
	content := `---
name: english
...
test	100
testing	80
tested	60
`
	tmpFile, err := os.CreateTemp("", "english_layer_*.dict.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	dict := NewEnglishDict(nil)
	if err := dict.LoadRimeFile(tmpFile.Name()); err != nil {
		t.Fatalf("LoadRimeFile failed: %v", err)
	}

	layer := NewEnglishDictLayer("english", dict)

	// Name
	if layer.Name() != "english" {
		t.Errorf("Name() = %q, want 'english'", layer.Name())
	}

	// Type
	if layer.Type() != LayerTypeSystem {
		t.Errorf("Type() = %v, want LayerTypeSystem", layer.Type())
	}

	// Search 精确查询
	results := layer.Search("test", 0)
	if len(results) == 0 {
		t.Error("Search('test') returned empty")
	} else if results[0].Text != "test" {
		t.Errorf("Search('test') text = %q, want 'test'", results[0].Text)
	}

	// Search 带 limit
	limited := layer.Search("test", 1)
	if len(limited) > 1 {
		t.Errorf("Search with limit=1 returned %d results", len(limited))
	}

	// SearchPrefix
	prefixResults := layer.SearchPrefix("test", 10)
	if len(prefixResults) < 3 {
		t.Errorf("SearchPrefix('test') returned %d results, want >= 3", len(prefixResults))
	}

	// SearchPrefix 带 limit
	limitedPrefix := layer.SearchPrefix("test", 2)
	if len(limitedPrefix) > 2 {
		t.Errorf("SearchPrefix with limit=2 returned %d results", len(limitedPrefix))
	}

	// 实现 DictLayer 接口检查（编译期保证，此处为运行时确认）
	var _ DictLayer = layer
}
