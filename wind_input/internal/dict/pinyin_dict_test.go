package dict

import (
	"os"
	"path/filepath"
	"testing"
)

// createTestRimeDir 创建临时 Rime 格式词库目录
func createTestRimeDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	content := `# Rime dictionary
---
name: test
version: "1.0"
sort: by_weight
...
啊	a	1000
阿	a	500
爱	ai	900
哀	ai	400
你	ni	800
泥	ni	300
好	hao	700
号	hao	350
你好	ni hao	600
我	wo	850
们	men	200
是	shi	950
的	de	990
`
	if err := os.WriteFile(filepath.Join(tmpDir, "8105.dict.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	return tmpDir
}

func TestPinyinDictLoad(t *testing.T) {
	tmpDir := createTestRimeDir(t)

	d := NewPinyinDict()
	if err := d.LoadRimeDir(tmpDir); err != nil {
		t.Fatalf("加载词库失败: %v", err)
	}

	if d.EntryCount() != 13 {
		t.Errorf("词条数量不正确: got %d, want 13", d.EntryCount())
	}

	// 测试 Lookup
	results := d.Lookup("ni")
	if len(results) != 2 {
		t.Errorf("Lookup(ni) 返回 %d 条, 期望 2", len(results))
	}

	results = d.Lookup("nihao")
	if len(results) != 1 {
		t.Errorf("Lookup(nihao) 返回 %d 条, 期望 1", len(results))
	}
	if len(results) > 0 && results[0].Text != "你好" {
		t.Errorf("Lookup(nihao) 首个结果 = %q, 期望 '你好'", results[0].Text)
	}

	// 测试 LookupPhrase
	results = d.LookupPhrase([]string{"ni", "hao"})
	if len(results) != 1 {
		t.Errorf("LookupPhrase([ni,hao]) 返回 %d 条, 期望 1", len(results))
	}
	if len(results) > 0 && results[0].Text != "你好" {
		t.Errorf("LookupPhrase([ni,hao]) 首个结果 = %q, 期望 '你好'", results[0].Text)
	}

	// 测试空查询
	results = d.Lookup("xyz")
	if len(results) != 0 {
		t.Errorf("Lookup(xyz) 返回 %d 条, 期望 0", len(results))
	}

	results = d.LookupPhrase(nil)
	if len(results) != 0 {
		t.Errorf("LookupPhrase(nil) 返回 %d 条, 期望 0", len(results))
	}
}

func TestPinyinDictAsDict(t *testing.T) {
	// 验证 PinyinDict 实现了 Dict 接口
	var _ Dict = (*PinyinDict)(nil)
}

func TestPinyinDictRealDict(t *testing.T) {
	paths := []string{
		"D:/Develop/workspace/go_dev/WindInput/build/dict/pinyin",
		"../../build/dict/pinyin",
		"../../../build/dict/pinyin",
	}

	var dictPath string
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "8105.dict.yaml")); err == nil {
			dictPath = p
			break
		}
	}

	if dictPath == "" {
		t.Skip("跳过测试：无法找到 Rime 词库目录")
	}

	d := NewPinyinDict()
	if err := d.LoadRimeDir(dictPath); err != nil {
		t.Skipf("跳过测试：无法加载词库 %s: %v", dictPath, err)
	}

	t.Logf("词库加载成功，词条数: %d", d.EntryCount())

	// 测试 "women" -> "我们"
	results := d.Lookup("women")
	t.Logf("Lookup('women'): %d 个结果", len(results))
	found := false
	for _, cand := range results {
		if cand.Text == "我们" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Lookup('women') 应该返回 '我们'，但没有找到")
	}

	// 测试 "zhongguo" -> "中国"
	results2 := d.Lookup("zhongguo")
	t.Logf("Lookup('zhongguo'): %d 个结果", len(results2))
	found = false
	for _, cand := range results2 {
		if cand.Text == "中国" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Lookup('zhongguo') 应该返回 '中国'，但没有找到")
	}
}

func TestCompositeDictWithPinyinLayer(t *testing.T) {
	paths := []string{
		"D:/Develop/workspace/go_dev/WindInput/build/dict/pinyin",
		"../../build/dict/pinyin",
	}

	var dictPath string
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "8105.dict.yaml")); err == nil {
			dictPath = p
			break
		}
	}

	if dictPath == "" {
		t.Skip("跳过测试：无法找到 Rime 词库目录")
	}

	pinyinDict := NewPinyinDict()
	if err := pinyinDict.LoadRimeDir(dictPath); err != nil {
		t.Fatalf("加载词库失败: %v", err)
	}
	t.Logf("PinyinDict 加载成功，词条数: %d", pinyinDict.EntryCount())

	layer := NewPinyinDictLayer("pinyin-system", LayerTypeSystem, pinyinDict)

	results := layer.Search("women", 0)
	t.Logf("PinyinDictLayer.Search('women'): %d 个结果", len(results))

	composite := NewCompositeDict()
	composite.AddLayer(layer)

	results2 := composite.Lookup("women")
	t.Logf("CompositeDict.Lookup('women'): %d 个结果", len(results2))

	found := false
	for _, cand := range results2 {
		if cand.Text == "我们" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CompositeDict.Lookup('women') 应该返回 '我们'，但没有找到")
	}
}

func TestPinyinDictLayer(t *testing.T) {
	tmpDir := createTestRimeDir(t)

	d := NewPinyinDict()
	if err := d.LoadRimeDir(tmpDir); err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	layer := NewPinyinDictLayer("test", LayerTypeSystem, d)

	if layer.Name() != "test" {
		t.Errorf("Name() = %q, want 'test'", layer.Name())
	}
	if layer.Type() != LayerTypeSystem {
		t.Errorf("Type() = %v, want LayerTypeSystem", layer.Type())
	}

	// 精确查询
	results := layer.Search("ni", 0)
	if len(results) != 2 {
		t.Errorf("Search(ni) 返回 %d 条, 期望 2", len(results))
	}

	// 前缀查询
	results = layer.SearchPrefix("ni", 0)
	if len(results) < 2 {
		t.Errorf("SearchPrefix(ni) 返回 %d 条, 期望至少 2", len(results))
	}
}
