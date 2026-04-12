//go:build windows

package systemfont

import "testing"

func TestExtractFamilies(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "regular face",
			in:   "Segoe UI (TrueType)",
			want: []string{"Segoe UI"},
		},
		{
			name: "styled face",
			in:   "Segoe UI Bold (TrueType)",
			want: []string{"Segoe UI"},
		},
		{
			name: "compound family",
			in:   "Microsoft YaHei & Microsoft YaHei UI (TrueType)",
			want: []string{"Microsoft YaHei", "Microsoft YaHei UI"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractFamilies(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("extractFamilies(%q) len=%d want=%d (%v)", tc.in, len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("extractFamilies(%q)[%d]=%q want %q (all=%v)", tc.in, i, got[i], tc.want[i], got)
				}
			}
		})
	}
}
