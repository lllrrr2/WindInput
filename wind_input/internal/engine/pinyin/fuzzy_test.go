package pinyin

import (
	"sort"
	"testing"
)

func TestFuzzyConfig_Enabled(t *testing.T) {
	tests := []struct {
		name   string
		config *FuzzyConfig
		want   bool
	}{
		{"nil config", nil, false},
		{"empty config", &FuzzyConfig{}, false},
		{"ZhZ enabled", &FuzzyConfig{ZhZ: true}, true},
		{"NL enabled", &FuzzyConfig{NL: true}, true},
		{"AnAng enabled", &FuzzyConfig{AnAng: true}, true},
		{"FH enabled", &FuzzyConfig{FH: true}, true},
		{"RL enabled", &FuzzyConfig{RL: true}, true},
		{"IanIang enabled", &FuzzyConfig{IanIang: true}, true},
		{"UanUang enabled", &FuzzyConfig{UanUang: true}, true},
		{"multiple enabled", &FuzzyConfig{ZhZ: true, NL: true, AnAng: true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.Enabled(); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitInitialFinal(t *testing.T) {
	tests := []struct {
		syllable    string
		wantInitial string
		wantFinal   string
	}{
		{"zhi", "zh", "i"},
		{"chi", "ch", "i"},
		{"shi", "sh", "i"},
		{"zi", "z", "i"},
		{"ci", "c", "i"},
		{"si", "s", "i"},
		{"ni", "n", "i"},
		{"li", "l", "i"},
		{"an", "", "an"},
		{"ang", "", "ang"},
		{"en", "", "en"},
		{"zhong", "zh", "ong"},
		{"chang", "ch", "ang"},
		{"shang", "sh", "ang"},
		{"ban", "b", "an"},
		{"bang", "b", "ang"},
		{"ren", "r", "en"},
		{"reng", "r", "eng"},
		{"lin", "l", "in"},
		{"ling", "l", "ing"},
	}

	for _, tt := range tests {
		t.Run(tt.syllable, func(t *testing.T) {
			gotInitial, gotFinal := splitInitialFinal(tt.syllable)
			if gotInitial != tt.wantInitial || gotFinal != tt.wantFinal {
				t.Errorf("splitInitialFinal(%q) = (%q, %q), want (%q, %q)",
					tt.syllable, gotInitial, gotFinal, tt.wantInitial, tt.wantFinal)
			}
		})
	}
}

func TestFuzzyConfig_Variants(t *testing.T) {
	tests := []struct {
		name     string
		config   *FuzzyConfig
		syllable string
		want     []string
	}{
		{
			"zh↔z: zhi → zhi variants",
			&FuzzyConfig{ZhZ: true},
			"zhi",
			[]string{"zi"},
		},
		{
			"zh↔z: zi → zhi",
			&FuzzyConfig{ZhZ: true},
			"zi",
			[]string{"zhi"},
		},
		{
			"ch↔c: chi → ci",
			&FuzzyConfig{ChC: true},
			"chi",
			[]string{"ci"},
		},
		{
			"sh↔s: shi → si",
			&FuzzyConfig{ShS: true},
			"shi",
			[]string{"si"},
		},
		{
			"n↔l: ni → li",
			&FuzzyConfig{NL: true},
			"ni",
			[]string{"li"},
		},
		{
			"n↔l: li → ni",
			&FuzzyConfig{NL: true},
			"li",
			[]string{"ni"},
		},
		{
			"an↔ang: ban → bang",
			&FuzzyConfig{AnAng: true},
			"ban",
			[]string{"bang"},
		},
		{
			"an↔ang: bang → ban",
			&FuzzyConfig{AnAng: true},
			"bang",
			[]string{"ban"},
		},
		{
			"en↔eng: ren → reng",
			&FuzzyConfig{EnEng: true},
			"ren",
			[]string{"reng"},
		},
		{
			"en↔eng: reng → ren",
			&FuzzyConfig{EnEng: true},
			"reng",
			[]string{"ren"},
		},
		{
			"in↔ing: lin → ling",
			&FuzzyConfig{InIng: true},
			"lin",
			[]string{"ling"},
		},
		{
			"in↔ing: ling → lin",
			&FuzzyConfig{InIng: true},
			"ling",
			[]string{"lin"},
		},
		{
			"f↔h: fan → han",
			&FuzzyConfig{FH: true},
			"fan",
			[]string{"han"},
		},
		{
			"f↔h: hua → fua (invalid pinyin, no variant)",
			&FuzzyConfig{FH: true},
			"hua",
			nil,
		},
		{
			"f↔h: hu → fu",
			&FuzzyConfig{FH: true},
			"hu",
			[]string{"fu"},
		},
		{
			"f↔h: fei → hei",
			&FuzzyConfig{FH: true},
			"fei",
			[]string{"hei"},
		},
		{
			"r↔l: ri → li",
			&FuzzyConfig{RL: true},
			"ri",
			[]string{"li"},
		},
		{
			"r↔l: rou → lou",
			&FuzzyConfig{RL: true},
			"rou",
			[]string{"lou"},
		},
		{
			"r↔l: lao → rao",
			&FuzzyConfig{RL: true},
			"lao",
			[]string{"rao"},
		},
		{
			"ian↔iang: lian → liang",
			&FuzzyConfig{IanIang: true},
			"lian",
			[]string{"liang"},
		},
		{
			"ian↔iang: liang → lian",
			&FuzzyConfig{IanIang: true},
			"liang",
			[]string{"lian"},
		},
		{
			"uan↔uang: guan → guang",
			&FuzzyConfig{UanUang: true},
			"guan",
			[]string{"guang"},
		},
		{
			"uan↔uang: guang → guan",
			&FuzzyConfig{UanUang: true},
			"guang",
			[]string{"guan"},
		},
		{
			"an↔ang should not affect ian",
			&FuzzyConfig{AnAng: true},
			"lian",
			nil,
		},
		{
			"an↔ang should not affect uan",
			&FuzzyConfig{AnAng: true},
			"guan",
			nil,
		},
		{
			"an↔ang should not affect iang",
			&FuzzyConfig{AnAng: true},
			"liang",
			nil,
		},
		{
			"an↔ang should not affect uang",
			&FuzzyConfig{AnAng: true},
			"guang",
			nil,
		},
		{
			"combined: zhan → zhan variants (zh↔z + an↔ang)",
			&FuzzyConfig{ZhZ: true, AnAng: true},
			"zhan",
			[]string{"zan", "zhang", "zang"},
		},
		{
			"no match: ba with ZhZ",
			&FuzzyConfig{ZhZ: true},
			"ba",
			nil,
		},
		{
			"nil config",
			nil,
			"zhi",
			nil,
		},
		{
			"disabled config",
			&FuzzyConfig{},
			"zhi",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Variants(tt.syllable)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Errorf("Variants(%q) = %v (len=%d), want %v (len=%d)",
					tt.syllable, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Variants(%q) = %v, want %v", tt.syllable, got, tt.want)
					break
				}
			}
		})
	}
}

func TestFuzzyConfig_ExpandCode(t *testing.T) {
	tests := []struct {
		name      string
		config    *FuzzyConfig
		syllables []string
		wantLen   int // 只检查长度，因为笛卡尔积结果较多
	}{
		{
			"single syllable zh↔z",
			&FuzzyConfig{ZhZ: true},
			[]string{"zhi"},
			1, // zi
		},
		{
			"two syllables zh↔z",
			&FuzzyConfig{ZhZ: true},
			[]string{"zhi", "zhao"},
			3, // zi+zhao, zhi+zao, zi+zao
		},
		{
			"no fuzzy match",
			&FuzzyConfig{ZhZ: true},
			[]string{"ba", "ma"},
			0, // ba 和 ma 没有 zh↔z 变体
		},
		{
			"nil config",
			nil,
			[]string{"zhi"},
			0,
		},
		{
			"empty syllables",
			&FuzzyConfig{ZhZ: true},
			nil,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ExpandCode(tt.syllables)
			if len(got) != tt.wantLen {
				t.Errorf("ExpandCode(%v) returned %d results %v, want %d",
					tt.syllables, len(got), got, tt.wantLen)
			}
		})
	}
}

func TestIsValidPinyin(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"a", true},
		{"zhi", true},
		{"chi", true},
		{"shi", true},
		{"ni", true},
		{"hao", true},
		{"zhx", false},
		{"xyz", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isValidPinyin(tt.input); got != tt.want {
				t.Errorf("isValidPinyin(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
