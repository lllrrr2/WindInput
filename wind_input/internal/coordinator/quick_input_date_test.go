package coordinator

import "testing"

func TestIsDateExpression(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2026.4.2", true},
		{"2026.04.02", true},
		{"2026.12.31", true},
		{"12.31", true},
		{"2026.4", false}, // 两段且首段>31，这是年月而非日期
		{"12345", false},
		{"abc.def", false},
	}
	for _, tt := range tests {
		got := isDateExpression(tt.input)
		if got != tt.want {
			t.Errorf("isDateExpression(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseDateParts(t *testing.T) {
	tests := []struct {
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
		wantOk    bool
	}{
		{"2026.4.2", 2026, 4, 2, true},
		{"2026.04.02", 2026, 4, 2, true},
		{"2026.12.31", 2026, 12, 31, true},
		{"12.31", 0, 12, 31, true},
		{"2026.13.1", 0, 0, 0, false},
		{"2026.1.32", 0, 0, 0, false},
	}
	for _, tt := range tests {
		y, m, d, ok := parseDateParts(tt.input)
		if ok != tt.wantOk {
			t.Errorf("parseDateParts(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			continue
		}
		if !ok {
			continue
		}
		if y != tt.wantYear || m != tt.wantMonth || d != tt.wantDay {
			t.Errorf("parseDateParts(%q) = (%d,%d,%d), want (%d,%d,%d)",
				tt.input, y, m, d, tt.wantYear, tt.wantMonth, tt.wantDay)
		}
	}
}

func TestGenerateDateCandidates(t *testing.T) {
	candidates := generateDateCandidates("2026.4.2")
	if len(candidates) < 3 {
		t.Fatalf("expected at least 3 candidates, got %d", len(candidates))
	}
	expected := []string{
		"20260402",
		"2026年4月2日",
		"2026年04月02日",
		"2026-04-02",
		"2026/04/02",
	}
	for i, want := range expected {
		if i >= len(candidates) {
			break
		}
		if candidates[i] != want {
			t.Errorf("candidate[%d] = %q, want %q", i, candidates[i], want)
		}
	}
}

func TestIsYearMonthExpression(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2026.03", true},
		{"2025.3", true},
		{"2026.12", true},
		{"2026.0", false},  // 月份无效
		{"2026.13", false}, // 月份无效
		{"12.3", false},    // 首段<=31，不是年月
		{"31.3", false},    // 首段==31，不是年月
		{"32.3", true},     // 首段>31
		{"abc.3", false},
		{"2026", false},
		{"2026.3.1", false}, // 三段不是年月
	}
	for _, tt := range tests {
		got := isYearMonthExpression(tt.input)
		if got != tt.want {
			t.Errorf("isYearMonthExpression(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGenerateYearMonthCandidates(t *testing.T) {
	candidates := generateYearMonthCandidates("2026.3")
	if len(candidates) != 4 {
		t.Fatalf("expected 4 candidates, got %d", len(candidates))
	}
	expected := []string{
		"2026年3月",
		"2026年03月",
		"2026-03",
		"2026/03",
	}
	for i, want := range expected {
		if candidates[i] != want {
			t.Errorf("candidate[%d] = %q, want %q", i, candidates[i], want)
		}
	}
}
