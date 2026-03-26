package dict

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPhraseLayerSearchCommandMarksIsCommand(t *testing.T) {
	// 创建临时系统短语文件，包含一个动态短语（含 $uuid 变量）
	tmpDir := t.TempDir()
	systemFile := filepath.Join(tmpDir, "system.phrases.yaml")
	content := `phrases:
  - code: "uuid"
    text: "$uuid"
    position: 1
`
	if err := os.WriteFile(systemFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pl := NewPhraseLayer("phrases", systemFile, "")
	if err := pl.Load(); err != nil {
		t.Fatal(err)
	}

	results := pl.SearchCommand("uuid", 10)
	if len(results) == 0 {
		t.Fatal("SearchCommand(uuid) should return candidates")
	}

	for i, c := range results {
		if !c.IsCommand {
			t.Fatalf("candidate[%d] should be marked IsCommand=true", i)
		}
	}
}

func TestPhraseLayerStaticPhrase(t *testing.T) {
	tmpDir := t.TempDir()
	userFile := filepath.Join(tmpDir, "user.phrases.yaml")
	content := `phrases:
  - code: "dz"
    text: "我的地址"
    position: 1
`
	if err := os.WriteFile(userFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pl := NewPhraseLayer("phrases", "", userFile)
	if err := pl.Load(); err != nil {
		t.Fatal(err)
	}

	results := pl.Search("dz", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Text != "我的地址" {
		t.Fatalf("expected '我的地址', got %q", results[0].Text)
	}
}

func TestPhraseLayerDynamicExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	systemFile := filepath.Join(tmpDir, "system.phrases.yaml")
	content := `phrases:
  - code: "rq"
    text: "$Y-$MM-$DD"
    position: 1
`
	if err := os.WriteFile(systemFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pl := NewPhraseLayer("phrases", systemFile, "")
	if err := pl.Load(); err != nil {
		t.Fatal(err)
	}

	// 动态短语不应出现在 Search 中
	results := pl.Search("rq", 10)
	if len(results) != 0 {
		t.Fatalf("dynamic phrase should not appear in Search, got %d", len(results))
	}

	// 应出现在 SearchCommand 中，且已展开
	cmdResults := pl.SearchCommand("rq", 10)
	if len(cmdResults) == 0 {
		t.Fatal("dynamic phrase should appear in SearchCommand")
	}
	// 展开后不应包含 $
	if cmdResults[0].Text == "$Y-$MM-$DD" {
		t.Fatal("dynamic phrase text should be expanded, not raw template")
	}
}
