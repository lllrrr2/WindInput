package binformat

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadMeta(t *testing.T) {
	// 构建包含 meta 的 wdb
	w := NewDictWriter()
	w.AddCode("abc", []DictEntry{
		{Text: "测试", Weight: 100},
		{Text: "测试2", Weight: 50},
	})
	w.AddCode("def", []DictEntry{
		{Text: "词条", Weight: 200},
	})

	metaJSON := []byte(`{"name":"TestDict","code_length":4}`)
	w.SetMeta(metaJSON)

	// 写入到临时文件
	tmpDir := t.TempDir()
	wdbPath := filepath.Join(tmpDir, "test.wdb")
	f, err := os.Create(wdbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// 读取并验证
	r, err := OpenDict(wdbPath)
	if err != nil {
		t.Fatalf("OpenDict 失败: %v", err)
	}
	defer r.Close()

	// 验证基本查询
	results := r.Lookup("abc")
	if len(results) != 2 {
		t.Fatalf("期望 2 个候选, 实际=%d", len(results))
	}
	if results[0].Text != "测试" {
		t.Errorf("期望 Text=测试, 实际=%s", results[0].Text)
	}

	// 验证 meta
	if !r.HasMeta() {
		t.Fatal("期望 HasMeta=true")
	}
	meta := r.ReadMeta()
	if meta == nil {
		t.Fatal("ReadMeta 不应返回 nil")
	}
	if !bytes.Equal(meta, metaJSON) {
		t.Errorf("Meta 不匹配: 期望=%s, 实际=%s", metaJSON, meta)
	}
}

func TestWriteAndReadWithoutMeta(t *testing.T) {
	w := NewDictWriter()
	w.AddCode("xyz", []DictEntry{
		{Text: "无meta", Weight: 10},
	})
	// 不调用 SetMeta

	tmpDir := t.TempDir()
	wdbPath := filepath.Join(tmpDir, "nometa.wdb")
	f, err := os.Create(wdbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	r, err := OpenDict(wdbPath)
	if err != nil {
		t.Fatalf("OpenDict 失败: %v", err)
	}
	defer r.Close()

	if r.HasMeta() {
		t.Error("不应有 meta")
	}
	if r.ReadMeta() != nil {
		t.Error("ReadMeta 应返回 nil")
	}

	// 基本查询仍正常
	results := r.Lookup("xyz")
	if len(results) != 1 || results[0].Text != "无meta" {
		t.Error("基本查询失败")
	}
}

func TestWriteAndReadWithAbbrevAndMeta(t *testing.T) {
	w := NewDictWriter()
	w.AddCode("nihao", []DictEntry{
		{Text: "你好", Weight: 1000},
	})
	w.AddAbbrev("nh", []DictEntry{
		{Text: "你好", Weight: 1000},
	})
	w.SetMeta([]byte(`{"type":"pinyin"}`))

	tmpDir := t.TempDir()
	wdbPath := filepath.Join(tmpDir, "abbrev_meta.wdb")
	f, err := os.Create(wdbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write(f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	r, err := OpenDict(wdbPath)
	if err != nil {
		t.Fatalf("OpenDict 失败: %v", err)
	}
	defer r.Close()

	// 验证主查询
	results := r.Lookup("nihao")
	if len(results) != 1 || results[0].Text != "你好" {
		t.Error("主查询失败")
	}

	// 验证简拼查询
	abbrevResults := r.LookupAbbrev("nh", 10)
	if len(abbrevResults) != 1 || abbrevResults[0].Text != "你好" {
		t.Error("简拼查询失败")
	}

	// 验证 meta
	if !r.HasMeta() {
		t.Fatal("期望 HasMeta=true")
	}
	meta := r.ReadMeta()
	if string(meta) != `{"type":"pinyin"}` {
		t.Errorf("Meta 不匹配: %s", meta)
	}
}
