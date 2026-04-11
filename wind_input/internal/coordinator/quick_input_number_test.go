package coordinator

import "testing"

func TestChineseDigitLower(t *testing.T) {
	tests := []struct {
		digit rune
		want  string
	}{
		{'0', "零"}, {'1', "一"}, {'2', "二"}, {'3', "三"}, {'4', "四"},
		{'5', "五"}, {'6', "六"}, {'7', "七"}, {'8', "八"}, {'9', "九"},
	}
	for _, tt := range tests {
		got := chineseDigitLower(tt.digit)
		if got != tt.want {
			t.Errorf("chineseDigitLower(%c) = %q, want %q", tt.digit, got, tt.want)
		}
	}
}

func TestChineseDigitUpper(t *testing.T) {
	tests := []struct {
		digit rune
		want  string
	}{
		{'0', "零"}, {'1', "壹"}, {'2', "贰"}, {'3', "叁"}, {'4', "肆"},
		{'5', "伍"}, {'6', "陆"}, {'7', "柒"}, {'8', "捌"}, {'9', "玖"},
	}
	for _, tt := range tests {
		got := chineseDigitUpper(tt.digit)
		if got != tt.want {
			t.Errorf("chineseDigitUpper(%c) = %q, want %q", tt.digit, got, tt.want)
		}
	}
}

func TestNumberToChinese(t *testing.T) {
	tests := []struct {
		num  string
		want string
	}{
		{"0", "零"},
		{"1", "一"},
		{"10", "一十"},
		{"100", "一百"},
		{"1000", "一千"},
		{"10000", "一万"},
		{"12345", "一万二千三百四十五"},
		{"10001", "一万零一"},
		{"10010", "一万零一十"},
		{"10100", "一万零一百"},
		{"100000000", "一亿"},
		{"100000001", "一亿零一"},
		{"123456789", "一亿二千三百四十五万六千七百八十九"},
		{"1000000000000", "一万亿"},
	}
	for _, tt := range tests {
		got := numberToChineseLower(tt.num)
		if got != tt.want {
			t.Errorf("numberToChineseLower(%q) = %q, want %q", tt.num, got, tt.want)
		}
	}
}

func TestNumberToChineseUpper(t *testing.T) {
	tests := []struct {
		num  string
		want string
	}{
		{"12345", "壹万贰仟叁佰肆拾伍"},
		{"100", "壹佰"},
	}
	for _, tt := range tests {
		got := numberToChineseUpper(tt.num)
		if got != tt.want {
			t.Errorf("numberToChineseUpper(%q) = %q, want %q", tt.num, got, tt.want)
		}
	}
}

func TestNumberToAmount(t *testing.T) {
	tests := []struct {
		num   string
		upper bool
		want  string
	}{
		{"12345", true, "壹万贰仟叁佰肆拾伍元整"},
		{"12345", false, "一万二千三百四十五元整"},
		{"0", true, "零元整"},
	}
	for _, tt := range tests {
		got := numberToAmount(tt.num, tt.upper)
		if got != tt.want {
			t.Errorf("numberToAmount(%q, %v) = %q, want %q", tt.num, tt.upper, got, tt.want)
		}
	}
}

func TestDigitsToChineseChars(t *testing.T) {
	tests := []struct {
		num   string
		upper bool
		want  string
	}{
		{"12345", false, "一二三四五"},
		{"12345", true, "壹贰叁肆伍"},
		{"0", false, "零"},
	}
	for _, tt := range tests {
		got := digitsToChineseChars(tt.num, tt.upper)
		if got != tt.want {
			t.Errorf("digitsToChineseChars(%q, %v) = %q, want %q", tt.num, tt.upper, got, tt.want)
		}
	}
}

func TestFormatThousands(t *testing.T) {
	tests := []struct {
		num  string
		want string
	}{
		{"0", "0"},
		{"123", "123"},
		{"1234", "1,234"},
		{"12345", "12,345"},
		{"123456789", "123,456,789"},
	}
	for _, tt := range tests {
		got := formatThousands(tt.num)
		if got != tt.want {
			t.Errorf("formatThousands(%q) = %q, want %q", tt.num, got, tt.want)
		}
	}
}

func TestGenerateNumberCandidates(t *testing.T) {
	candidates := generateNumberCandidates("12345")
	if len(candidates) != 7 {
		t.Fatalf("expected 7 candidates, got %d", len(candidates))
	}
	expected := []string{
		"壹万贰仟叁佰肆拾伍元整",
		"一万二千三百四十五元整",
		"一万二千三百四十五",
		"壹万贰仟叁佰肆拾伍",
		"一二三四五",
		"壹贰叁肆伍",
		"12,345",
	}
	for i, want := range expected {
		if candidates[i] != want {
			t.Errorf("candidate[%d] = %q, want %q", i, candidates[i], want)
		}
	}
}

func TestIsDecimalNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"123.45", true},
		{"123.", true},
		{"0.5", true},
		{".5", false},       // 不允许以点号开头
		{"123.45.6", false}, // 多个点号
		{"abc", false},
		{"", false},
		{"12+3", false},
		{"0", true},
	}
	for _, tt := range tests {
		got := isDecimalNumber(tt.input)
		if got != tt.want {
			t.Errorf("isDecimalNumber(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSplitDecimal(t *testing.T) {
	tests := []struct {
		input   string
		wantInt string
		wantDec string
	}{
		{"123.45", "123", "45"},
		{"123.", "123", ""},
		{"123", "123", ""},
		{"0.5", "0", "5"},
	}
	for _, tt := range tests {
		gotInt, gotDec := splitDecimal(tt.input)
		if gotInt != tt.wantInt || gotDec != tt.wantDec {
			t.Errorf("splitDecimal(%q) = (%q, %q), want (%q, %q)",
				tt.input, gotInt, gotDec, tt.wantInt, tt.wantDec)
		}
	}
}

func TestDecimalToAmount(t *testing.T) {
	tests := []struct {
		intPart string
		decPart string
		upper   bool
		want    string
	}{
		{"123", "34", true, "壹佰贰拾叁元叁角肆分"},
		{"123", "40", true, "壹佰贰拾叁元肆角整"},
		{"123", "04", true, "壹佰贰拾叁元零肆分"},
		{"123", "", true, "壹佰贰拾叁元整"},
		{"0", "50", false, "零元五角整"},
		{"0", "05", false, "零元零五分"},
		{"100", "00", true, "壹佰元整"},
		{"123", "456", true, ""}, // 超过2位小数返回空
	}
	for _, tt := range tests {
		got := decimalToAmount(tt.intPart, tt.decPart, tt.upper)
		if got != tt.want {
			t.Errorf("decimalToAmount(%q, %q, %v) = %q, want %q",
				tt.intPart, tt.decPart, tt.upper, got, tt.want)
		}
	}
}

func TestDecimalToChineseText(t *testing.T) {
	tests := []struct {
		intPart string
		decPart string
		upper   bool
		want    string
	}{
		{"123", "456", false, "一百二十三点四五六"},
		{"123", "", false, "一百二十三"},
		{"0", "5", false, "零点五"},
		{"123", "45", true, "壹佰贰拾叁点肆伍"},
	}
	for _, tt := range tests {
		got := decimalToChineseText(tt.intPart, tt.decPart, tt.upper)
		if got != tt.want {
			t.Errorf("decimalToChineseText(%q, %q, %v) = %q, want %q",
				tt.intPart, tt.decPart, tt.upper, got, tt.want)
		}
	}
}

func TestGenerateNumberCandidatesDecimal(t *testing.T) {
	// 小数有2位小数 - 应包含金额
	candidates := generateNumberCandidates("123.34")
	if len(candidates) < 4 {
		t.Fatalf("expected at least 4 candidates for 123.34, got %d", len(candidates))
	}
	// 第一个应是大写金额
	if candidates[0] != "壹佰贰拾叁元叁角肆分" {
		t.Errorf("candidate[0] = %q, want %q", candidates[0], "壹佰贰拾叁元叁角肆分")
	}

	// 小数超过2位 - 不应包含金额，但应有中文读法
	candidates3 := generateNumberCandidates("123.456")
	for _, c := range candidates3 {
		if c == "" {
			t.Error("got empty candidate")
		}
	}
	// 应包含中文读法
	found := false
	for _, c := range candidates3 {
		if c == "一百二十三点四五六" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected '一百二十三点四五六' in candidates, got %v", candidates3)
	}

	// "123." 应视为整数 123
	candidatesDot := generateNumberCandidates("123.")
	if len(candidatesDot) != 7 {
		t.Fatalf("expected 7 candidates for '123.', got %d", len(candidatesDot))
	}
}
