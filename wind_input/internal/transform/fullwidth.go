// Package transform provides text transformation utilities
package transform

// ASCII 半角字符范围: 0x0021 - 0x007E
// 全角字符范围: 0xFF01 - 0xFF5E
// 转换公式: 全角 = 半角 + 0xFEE0
// 空格特殊处理: 半角空格 0x0020, 全角空格 0x3000

const (
	halfWidthStart = 0x0021 // !
	halfWidthEnd   = 0x007E // ~
	fullWidthStart = 0xFF01 // ！
	fullWidthDiff  = 0xFEE0 // 全角与半角的差值

	halfWidthSpace = 0x0020 // 半角空格
	fullWidthSpace = 0x3000 // 全角空格
)

// ToFullWidth converts half-width characters to full-width
// 半角转全角
func ToFullWidth(s string) string {
	runes := []rune(s)
	result := make([]rune, len(runes))

	for i, r := range runes {
		if r == halfWidthSpace {
			// 空格特殊处理
			result[i] = fullWidthSpace
		} else if r >= halfWidthStart && r <= halfWidthEnd {
			// ASCII 可打印字符转全角
			result[i] = r + fullWidthDiff
		} else {
			// 其他字符保持不变
			result[i] = r
		}
	}

	return string(result)
}

// ToHalfWidth converts full-width characters to half-width
// 全角转半角
func ToHalfWidth(s string) string {
	runes := []rune(s)
	result := make([]rune, len(runes))

	for i, r := range runes {
		if r == fullWidthSpace {
			// 全角空格转半角空格
			result[i] = halfWidthSpace
		} else if r >= fullWidthStart && r <= fullWidthStart+(halfWidthEnd-halfWidthStart) {
			// 全角字符转半角
			result[i] = r - fullWidthDiff
		} else {
			// 其他字符保持不变
			result[i] = r
		}
	}

	return string(result)
}
