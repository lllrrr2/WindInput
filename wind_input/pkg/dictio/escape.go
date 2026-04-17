package dictio

import (
	"strings"
	"unicode"
)

// EscapeField 对 TSV 数据段中的字段值进行转义。
// 将换行符、制表符和反斜杠替换为字面量转义序列。
func EscapeField(s string) string {
	// 快速路径：大多数字段不含需要转义的字符
	if !strings.ContainsAny(s, "\\\n\t") {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// UnescapeField 对 TSV 数据段中的字段值进行反转义。
// 将字面量转义序列还原为原始字符。
func UnescapeField(s string) string {
	// 快速路径
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
			case 't':
				b.WriteByte('\t')
				i++
			case '\\':
				b.WriteByte('\\')
				i++
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// trimUnicodeSpaces 去除字符串两端的所有 Unicode 空白字符，
// 包括全角空格（\u3000）等 strings.TrimSpace 不处理的字符。
func trimUnicodeSpaces(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}
