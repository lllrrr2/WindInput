package dict

import (
	"testing"

	"github.com/huanfeng/wind_input/internal/candidate"
)

func TestTrieInsertAndSearch(t *testing.T) {
	trie := NewTrie()

	trie.Insert("ni", candidate.Candidate{Text: "你", Weight: 100})
	trie.Insert("ni", candidate.Candidate{Text: "泥", Weight: 50})
	trie.Insert("nihao", candidate.Candidate{Text: "你好", Weight: 200})
	trie.Insert("nimen", candidate.Candidate{Text: "你们", Weight: 150})
	trie.Insert("hao", candidate.Candidate{Text: "好", Weight: 100})

	// 精确查找
	results := trie.Search("ni")
	if len(results) != 2 {
		t.Errorf("Search(ni) = %d条, want 2", len(results))
	}

	results = trie.Search("nihao")
	if len(results) != 1 || results[0].Text != "你好" {
		t.Errorf("Search(nihao) 结果不正确")
	}

	results = trie.Search("nih")
	if len(results) != 0 {
		t.Errorf("Search(nih) 应返回空, 得到 %d条", len(results))
	}
}

func TestTrieSearchPrefix(t *testing.T) {
	trie := NewTrie()

	trie.Insert("ni", candidate.Candidate{Text: "你", Weight: 100})
	trie.Insert("nihao", candidate.Candidate{Text: "你好", Weight: 200})
	trie.Insert("nimen", candidate.Candidate{Text: "你们", Weight: 150})
	trie.Insert("hao", candidate.Candidate{Text: "好", Weight: 100})

	// 前缀查找
	results := trie.SearchPrefix("ni", 0)
	if len(results) != 3 {
		t.Errorf("SearchPrefix(ni) = %d条, want 3", len(results))
	}

	// 带 limit
	results = trie.SearchPrefix("ni", 2)
	if len(results) != 2 {
		t.Errorf("SearchPrefix(ni, 2) = %d条, want 2", len(results))
	}
	// 应按权重降序
	if results[0].Weight < results[1].Weight {
		t.Errorf("SearchPrefix 结果未按权重排序")
	}

	// 完整匹配也是前缀
	results = trie.SearchPrefix("nihao", 0)
	if len(results) != 1 {
		t.Errorf("SearchPrefix(nihao) = %d条, want 1", len(results))
	}
}

func TestTrieHasPrefix(t *testing.T) {
	trie := NewTrie()

	trie.Insert("nihao", candidate.Candidate{Text: "你好", Weight: 200})

	if !trie.HasPrefix("n") {
		t.Error("HasPrefix(n) = false, want true")
	}
	if !trie.HasPrefix("ni") {
		t.Error("HasPrefix(ni) = false, want true")
	}
	if !trie.HasPrefix("nih") {
		t.Error("HasPrefix(nih) = false, want true")
	}
	if trie.HasPrefix("x") {
		t.Error("HasPrefix(x) = true, want false")
	}
}

func TestTrieEntryCount(t *testing.T) {
	trie := NewTrie()

	trie.Insert("a", candidate.Candidate{Text: "啊"})
	trie.Insert("a", candidate.Candidate{Text: "阿"})
	trie.Insert("ni", candidate.Candidate{Text: "你"})

	if trie.EntryCount() != 3 {
		t.Errorf("EntryCount() = %d, want 3", trie.EntryCount())
	}
}
