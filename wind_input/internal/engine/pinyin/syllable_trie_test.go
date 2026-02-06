package pinyin

import "testing"

func TestSyllableTrieContains(t *testing.T) {
	st := NewSyllableTrie()

	validSyllables := []string{
		"a", "ai", "an", "ang", "ao",
		"ba", "bai", "ban", "bang", "bao", "bei", "ben", "beng", "bi", "bian", "biao", "bie", "bin", "bing", "bo", "bu",
		"ni", "hao", "shi", "wo", "de",
		"zhi", "chi", "shi", "ri",
		"yi", "wu", "yu",
		"zhong", "guo", "ren", "min",
	}

	for _, s := range validSyllables {
		if !st.Contains(s) {
			t.Errorf("Contains(%q) = false, want true", s)
		}
	}

	invalidSyllables := []string{
		"bv", "fiu", "gv",
	}
	for _, s := range invalidSyllables {
		if st.Contains(s) {
			t.Errorf("Contains(%q) = true, want false", s)
		}
	}
}

func TestSyllableTrieMatchAt(t *testing.T) {
	st := NewSyllableTrie()

	tests := []struct {
		input   string
		pos     int
		wantLen int    // 期望匹配数量
		first   string // 期望第一个匹配（最长的）
	}{
		{"nihao", 0, 1, "ni"},
		{"nihao", 2, 1, "hao"},
		{"zhongguo", 0, 2, "zhong"}, // zhong, zho（如果存在）
		{"xian", 0, 2, "xian"},      // xian, xi
		{"a", 0, 1, "a"},
	}

	for _, tt := range tests {
		matches := st.MatchAt(tt.input, tt.pos)
		if len(matches) < 1 {
			t.Errorf("MatchAt(%q, %d) 无匹配", tt.input, tt.pos)
			continue
		}
		if matches[0] != tt.first {
			t.Errorf("MatchAt(%q, %d)[0] = %q, want %q", tt.input, tt.pos, matches[0], tt.first)
		}
	}
}

func TestSyllableTrieHasPrefix(t *testing.T) {
	st := NewSyllableTrie()

	if !st.HasPrefix("zh") {
		t.Error("HasPrefix(zh) = false, want true")
	}
	if !st.HasPrefix("n") {
		t.Error("HasPrefix(n) = false, want true")
	}
	// 单个字母基本都是前缀
	for c := byte('a'); c <= byte('z'); c++ {
		prefix := string(c)
		// 只有少数字母不是声母或韵母的开头
		_ = st.HasPrefix(prefix)
	}
}

func TestSyllableTrieMatchPrefixAt(t *testing.T) {
	st := NewSyllableTrie()

	tests := []struct {
		input        string
		pos          int
		wantPrefix   string
		wantComplete bool
		wantPossible bool // 是否期望有可能的续写
	}{
		// 完整音节
		{"zhang", 0, "zhang", true, false},
		{"ni", 0, "ni", true, true},    // "ni" 完整，但还有 "nian", "niang" 等
		{"hao", 0, "hao", true, false}, // "hao" 完整，无续写
		// 不完整音节（前缀）
		{"zh", 0, "zh", false, true},   // "zh" 不完整，可续写 "a", "ai", "an" 等
		{"zho", 0, "zho", false, true}, // "zho" 不完整，可续写 "ng", "u"
		{"shang", 0, "shang", true, false},
		// 从非零位置开始
		{"nihao", 2, "hao", true, false},
		// 部分有效输入
		{"xyz", 0, "x", false, true},    // "x" 是有效前缀（xi, xu, xia 等）
		{"qwert", 1, "we", false, true}, // "we" 是有效前缀（wei, wen, weng）
		// 无效输入
		{"vvv", 0, "", false, false}, // "v" 不是有效前缀
		{"", 0, "", false, false},
	}

	for _, tt := range tests {
		prefix, isComplete, possible := st.MatchPrefixAt(tt.input, tt.pos)

		if prefix != tt.wantPrefix {
			t.Errorf("MatchPrefixAt(%q, %d) prefix = %q, want %q", tt.input, tt.pos, prefix, tt.wantPrefix)
		}
		if isComplete != tt.wantComplete {
			t.Errorf("MatchPrefixAt(%q, %d) isComplete = %v, want %v", tt.input, tt.pos, isComplete, tt.wantComplete)
		}
		hasPossible := len(possible) > 0
		if hasPossible != tt.wantPossible {
			t.Errorf("MatchPrefixAt(%q, %d) hasPossible = %v, want %v (got %v)", tt.input, tt.pos, hasPossible, tt.wantPossible, possible)
		}
	}
}

func TestSyllableTrieGetPossibleSyllables(t *testing.T) {
	st := NewSyllableTrie()

	tests := []struct {
		prefix      string
		wantContain []string
	}{
		{"zh", []string{"zha", "zhai", "zhan", "zhang", "zhao", "zhe", "zhei", "zhen", "zheng", "zhi", "zhong", "zhou", "zhu"}},
		{"ni", []string{"ni", "nian", "niang", "niao", "nie", "nin", "ning", "niu"}},
		{"wo", []string{"wo"}},
	}

	for _, tt := range tests {
		results := st.GetPossibleSyllables(tt.prefix)
		resultSet := make(map[string]bool)
		for _, r := range results {
			resultSet[r] = true
		}

		for _, want := range tt.wantContain {
			if !resultSet[want] {
				t.Errorf("GetPossibleSyllables(%q) missing %q, got %v", tt.prefix, want, results)
			}
		}
	}
}
