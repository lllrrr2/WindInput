package dictio

import (
	"bytes"
	"strings"
	"testing"
)

// ---- escape_test ----

func TestEscapeField(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"no special", "no special"},
		{"line1\nline2", `line1\nline2`},
		{"col1\tcol2", `col1\tcol2`},
		{`back\slash`, `back\\slash`},
		{"mix\nand\ttab", `mix\nand\ttab`},
		{`a\nb`, `a\\nb`}, // 反斜杠后跟 n，不是换行
		{"", ""},
		{"multi\n\nblank", `multi\n\nblank`},
	}
	for _, tt := range tests {
		got := EscapeField(tt.input)
		if got != tt.want {
			t.Errorf("EscapeField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUnescapeField(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{`line1\nline2`, "line1\nline2"},
		{`col1\tcol2`, "col1\tcol2"},
		{`back\\slash`, `back\slash`},
		{`mix\nand\ttab`, "mix\nand\ttab"},
		{`a\\nb`, "a\\nb"}, // \\n → 反斜杠 + n
		{"", ""},
		{`trailing\`, `trailing\`}, // 尾部孤立反斜杠保留
	}
	for _, tt := range tests {
		got := UnescapeField(tt.input)
		if got != tt.want {
			t.Errorf("UnescapeField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeUnescapeRoundtrip(t *testing.T) {
	inputs := []string{
		"普通文本",
		"第一行\n第二行\n第三行",
		"含\t制表符",
		`含\反斜杠`,
		"复合\n换行\t制表\\斜杠",
		"",
		"a",
	}
	for _, input := range inputs {
		escaped := EscapeField(input)
		got := UnescapeField(escaped)
		if got != input {
			t.Errorf("roundtrip failed: input=%q escaped=%q got=%q", input, escaped, got)
		}
	}
}

// ---- columns_test ----

func TestColumnDef(t *testing.T) {
	cols := NewColumnDef([]string{"code", "text", "weight", "count"})

	fields := []string{"abc", "工", "100", "5"}

	if got := cols.Get(fields, "code"); got != "abc" {
		t.Errorf("Get code = %q, want abc", got)
	}
	if got := cols.Get(fields, "text"); got != "工" {
		t.Errorf("Get text = %q, want 工", got)
	}
	if got := cols.GetInt(fields, "weight", 0); got != 100 {
		t.Errorf("GetInt weight = %d, want 100", got)
	}
	if got := cols.GetInt(fields, "count", 0); got != 5 {
		t.Errorf("GetInt count = %d, want 5", got)
	}
}

func TestColumnDefMissingFields(t *testing.T) {
	cols := NewColumnDef([]string{"code", "text", "weight", "count", "created_at"})

	// 只有 3 列数据，后面的列应该返回默认值
	fields := []string{"abc", "工", "100"}

	if got := cols.GetInt(fields, "count", 0); got != 0 {
		t.Errorf("missing field count = %d, want 0", got)
	}
	if got := cols.GetInt64(fields, "created_at", 0); got != 0 {
		t.Errorf("missing field created_at = %d, want 0", got)
	}
}

func TestColumnDefUnknownColumn(t *testing.T) {
	cols := NewColumnDef([]string{"code", "text"})
	fields := []string{"abc", "工"}

	if got := cols.Get(fields, "unknown"); got != "" {
		t.Errorf("unknown column = %q, want empty", got)
	}
	if got := cols.GetInt(fields, "unknown", 42); got != 42 {
		t.Errorf("unknown column int = %d, want 42", got)
	}
}

func TestColumnDefDifferentOrder(t *testing.T) {
	// 列顺序不同于默认
	cols := NewColumnDef([]string{"text", "code", "weight"})
	fields := []string{"工", "abc", "100"}

	if got := cols.Get(fields, "code"); got != "abc" {
		t.Errorf("reordered Get code = %q, want abc", got)
	}
	if got := cols.Get(fields, "text"); got != "工" {
		t.Errorf("reordered Get text = %q, want 工", got)
	}
}

func TestColumnDefBool(t *testing.T) {
	cols := NewColumnDef([]string{"enabled"})

	if got := cols.GetBool([]string{"1"}, "enabled", false); !got {
		t.Error("GetBool 1 should be true")
	}
	if got := cols.GetBool([]string{"0"}, "enabled", true); got {
		t.Error("GetBool 0 should be false")
	}
	if got := cols.GetBool([]string{""}, "enabled", true); !got {
		t.Error("GetBool empty should return default true")
	}
}

// ---- header_test ----

func TestParseHeader(t *testing.T) {
	input := `# WindInput 用户数据文件
wind_dict:
  version: 1
  generator: "WindInput v1.0"
  exported_at: "2026-04-16T10:30:00+08:00"
  schema_id: "wubi86"
  schema_name: "五笔86"
  sections:
    user_words:
      columns: [code, text, weight, count, created_at]
    freq:
      columns: [code, text, count, last_used, streak]

--- !user_words
a	工	100
`
	header, err := ParseHeader([]byte(input))
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if header.Version != 1 {
		t.Errorf("version = %d, want 1", header.Version)
	}
	if header.SchemaID != "wubi86" {
		t.Errorf("schema_id = %q, want wubi86", header.SchemaID)
	}
	if header.Generator != "WindInput v1.0" {
		t.Errorf("generator = %q, want WindInput v1.0", header.Generator)
	}

	uwMeta, ok := header.Sections[SectionUserWords]
	if !ok {
		t.Fatal("missing user_words section")
	}
	if len(uwMeta.Columns) != 5 {
		t.Errorf("user_words columns count = %d, want 5", len(uwMeta.Columns))
	}

	freqMeta, ok := header.Sections[SectionFreq]
	if !ok {
		t.Fatal("missing freq section")
	}
	if len(freqMeta.Columns) != 5 {
		t.Errorf("freq columns count = %d, want 5", len(freqMeta.Columns))
	}
}

func TestParseHeaderNoVersion(t *testing.T) {
	input := `wind_dict:
  generator: "test"
`
	_, err := ParseHeader([]byte(input))
	if err == nil {
		t.Error("expected error for missing version")
	}
}

func TestWriteHeader(t *testing.T) {
	header := &WindDictHeader{
		Version:    1,
		Generator:  "test",
		ExportedAt: "2026-04-16T10:30:00+08:00",
		SchemaID:   "wubi86",
		Sections: map[string]SectionMeta{
			SectionUserWords: {Columns: []string{"code", "text", "weight"}},
		},
	}

	var buf bytes.Buffer
	if err := WriteHeader(&buf, header); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "wind_dict:") {
		t.Error("output should contain wind_dict:")
	}
	if !strings.Contains(output, "version: 1") {
		t.Error("output should contain version: 1")
	}

	// 验证写入后可以解析回来
	parsed, err := ParseHeader([]byte(output))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if parsed.SchemaID != "wubi86" {
		t.Errorf("re-parsed schema_id = %q, want wubi86", parsed.SchemaID)
	}
}

func TestSplitSections(t *testing.T) {
	input := `wind_dict:
  version: 1
  sections:
    user_words:
      columns: [code, text, weight]

--- !user_words
a	工	100
aa	式	200

--- !freq
a	工	42	1713234000	3
`
	header, sections := SplitSections([]byte(input))

	if !bytes.Contains(header, []byte("wind_dict:")) {
		t.Error("header should contain wind_dict:")
	}

	if len(sections) != 2 {
		t.Fatalf("sections count = %d, want 2", len(sections))
	}

	if sections[0].Tag != "user_words" {
		t.Errorf("section[0].Tag = %q, want user_words", sections[0].Tag)
	}
	if !bytes.Contains(sections[0].Body, []byte("a\t工\t100")) {
		t.Error("section[0].Body should contain data line")
	}

	if sections[1].Tag != "freq" {
		t.Errorf("section[1].Tag = %q, want freq", sections[1].Tag)
	}
}

func TestIsWindDictFile(t *testing.T) {
	if !IsWindDictFile([]byte("wind_dict:\n  version: 1")) {
		t.Error("should detect wind_dict file")
	}
	if IsWindDictFile([]byte("phrases:\n  - code: sj")) {
		t.Error("should not detect plain yaml as wind_dict")
	}
}

// ---- format_test ----

func TestShouldImport(t *testing.T) {
	// nil sections = 全部
	opts := ImportOptions{}
	if !opts.ShouldImport(SectionUserWords) {
		t.Error("nil sections should import all")
	}

	// 指定 sections
	opts = ImportOptions{Sections: []string{SectionUserWords, SectionFreq}}
	if !opts.ShouldImport(SectionUserWords) {
		t.Error("should import user_words")
	}
	if opts.ShouldImport(SectionShadow) {
		t.Error("should not import shadow")
	}
}

func TestImportResultUpdateStats(t *testing.T) {
	r := &ImportResult{
		UserWords:  make([]UserWordEntry, 10),
		FreqData:   make([]FreqEntry, 5),
		ShadowPins: make([]ShadowPinEntry, 3),
	}
	r.UpdateStats()

	if r.Stats.UserWordsCount != 10 {
		t.Errorf("UserWordsCount = %d, want 10", r.Stats.UserWordsCount)
	}
	if r.Stats.FreqCount != 5 {
		t.Errorf("FreqCount = %d, want 5", r.Stats.FreqCount)
	}
}
