// quick_input_number.go — 快捷输入：数字/金额转换模块
package coordinator

import "strings"

var (
	chineseLowerDigits = [10]string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	chineseUpperDigits = [10]string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
	chineseLowerUnits  = []string{"", "十", "百", "千"}
	chineseUpperUnits  = []string{"", "拾", "佰", "仟"}
	chineseGroupUnits  = []string{"", "万", "亿", "万亿"}
)

func chineseDigitLower(r rune) string {
	if r >= '0' && r <= '9' {
		return chineseLowerDigits[r-'0']
	}
	return ""
}

func chineseDigitUpper(r rune) string {
	if r >= '0' && r <= '9' {
		return chineseUpperDigits[r-'0']
	}
	return ""
}

// numberToChineseLower 将数字字符串转换为中文小写（如 "12345" → "一万二千三百四十五"）
func numberToChineseLower(num string) string {
	return numberToChinese(num, chineseLowerDigits[:], chineseLowerUnits)
}

// numberToChineseUpper 将数字字符串转换为中文大写（如 "12345" → "壹万贰仟叁佰肆拾伍"）
func numberToChineseUpper(num string) string {
	return numberToChinese(num, chineseUpperDigits[:], chineseUpperUnits)
}

// numberToChinese 通用中文数字转换
// 按每 4 位一组（个/万/亿/万亿）分段处理，组内按千/百/十/个单位转换。
// 处理连续零的合并（多个零只读一个"零"）和尾部零的省略。
func numberToChinese(num string, digits []string, units []string) string {
	num = strings.TrimLeft(num, "0")
	if num == "" {
		return digits[0]
	}

	groups := make([]string, 0, 4)
	for len(num) > 0 {
		start := len(num) - 4
		if start < 0 {
			start = 0
		}
		groups = append(groups, num[start:])
		num = num[:start]
	}

	var result strings.Builder
	totalGroups := len(groups)
	for i := totalGroups - 1; i >= 0; i-- {
		groupStr := groups[i]
		groupText := groupToChinese(groupStr, digits, units)
		if groupText == "" {
			continue
		}

		if result.Len() > 0 && needsLeadingZero(groupStr) {
			result.WriteString(digits[0])
		}

		result.WriteString(groupText)
		if i < len(chineseGroupUnits) {
			result.WriteString(chineseGroupUnits[i])
		}
	}

	if result.Len() == 0 {
		return digits[0]
	}
	return result.String()
}

func needsLeadingZero(group string) bool {
	return len(group) < 4 || group[0] == '0'
}

func groupToChinese(group string, digits []string, units []string) string {
	var result strings.Builder
	allZero := true
	prevZero := false
	length := len(group)

	for i, r := range group {
		d := int(r - '0')
		unitIdx := length - 1 - i

		if d == 0 {
			prevZero = true
			continue
		}

		allZero = false
		if prevZero && result.Len() > 0 {
			result.WriteString(digits[0])
		}
		prevZero = false

		result.WriteString(digits[d])
		if unitIdx < len(units) {
			result.WriteString(units[unitIdx])
		}
	}

	if allZero {
		return ""
	}
	return result.String()
}

func numberToAmount(num string, upper bool) string {
	var text string
	if upper {
		text = numberToChineseUpper(num)
	} else {
		text = numberToChineseLower(num)
	}
	return text + "元整"
}

// isDecimalNumber 检查是否为数字（整数或小数，允许尾部点号）
// 不允许多个点号，不允许以点号开头。
func isDecimalNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] < '0' || s[0] > '9' {
		return false
	}
	dotCount := 0
	for _, r := range s {
		if r == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
		} else if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// splitDecimal 分割整数和小数部分
// "123.45" → ("123", "45")，"123." → ("123", "")，"123" → ("123", "")
func splitDecimal(s string) (intPart, decPart string) {
	idx := strings.IndexByte(s, '.')
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}

// decimalToAmount 带角分的金额转换
func decimalToAmount(intPart, decPart string, upper bool) string {
	var intText string
	if upper {
		intText = numberToChineseUpper(intPart)
	} else {
		intText = numberToChineseLower(intPart)
	}

	digits := chineseLowerDigits
	if upper {
		digits = chineseUpperDigits
	}

	if decPart == "" {
		return intText + "元整"
	}
	if len(decPart) > 2 {
		return ""
	}

	jiao := int(decPart[0] - '0')
	fen := 0
	if len(decPart) == 2 {
		fen = int(decPart[1] - '0')
	}

	if jiao == 0 && fen == 0 {
		return intText + "元整"
	}

	var b strings.Builder
	b.WriteString(intText)
	b.WriteString("元")

	if jiao == 0 {
		// 零角有分：X元零Z分
		b.WriteString("零")
		b.WriteString(digits[fen])
		b.WriteString("分")
	} else if fen == 0 {
		// 有角无分
		b.WriteString(digits[jiao])
		b.WriteString("角整")
	} else {
		// 有角有分
		b.WriteString(digits[jiao])
		b.WriteString("角")
		b.WriteString(digits[fen])
		b.WriteString("分")
	}

	return b.String()
}

// decimalToChineseText 中文小数读法
// "123", "456", false → "一百二十三点四五六"
func decimalToChineseText(intPart, decPart string, upper bool) string {
	var intText string
	if upper {
		intText = numberToChineseUpper(intPart)
	} else {
		intText = numberToChineseLower(intPart)
	}

	if decPart == "" {
		return intText
	}

	digits := chineseLowerDigits
	if upper {
		digits = chineseUpperDigits
	}

	var b strings.Builder
	b.WriteString(intText)
	b.WriteString("点")
	for _, r := range decPart {
		if r >= '0' && r <= '9' {
			b.WriteString(digits[r-'0'])
		}
	}
	return b.String()
}

func digitsToChineseChars(num string, upper bool) string {
	var b strings.Builder
	digits := chineseLowerDigits
	if upper {
		digits = chineseUpperDigits
	}
	for _, r := range num {
		if r >= '0' && r <= '9' {
			b.WriteString(digits[r-'0'])
		} else if r == '.' {
			b.WriteString("点")
		}
	}
	if b.Len() == 0 {
		return digits[0]
	}
	return b.String()
}

func formatThousands(num string) string {
	if len(num) <= 3 {
		return num
	}
	var b strings.Builder
	remainder := len(num) % 3
	if remainder > 0 {
		b.WriteString(num[:remainder])
	}
	for i := remainder; i < len(num); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(num[i : i+3])
	}
	return b.String()
}

// generateNumberCandidates 根据数字字符串生成完整候选列表
// 支持整数和小数。整数保持原有 7 个候选，小数生成金额（≤2位小数）、中文读法、单字逐位。
func generateNumberCandidates(s string) []string {
	intPart, decPart := splitDecimal(s)

	if decPart == "" {
		// 整数（含 "123." 情况，splitDecimal 返回 decPart=""）
		candidates := make([]string, 0, 7)
		candidates = append(candidates, numberToAmount(intPart, true))
		candidates = append(candidates, numberToAmount(intPart, false))
		candidates = append(candidates, numberToChineseLower(intPart))
		candidates = append(candidates, numberToChineseUpper(intPart))
		candidates = append(candidates, digitsToChineseChars(intPart, false))
		candidates = append(candidates, digitsToChineseChars(intPart, true))
		candidates = append(candidates, formatThousands(intPart))
		return candidates
	}

	// 小数
	candidates := make([]string, 0, 6)

	// 大写金额（≤2位小数）
	if amt := decimalToAmount(intPart, decPart, true); amt != "" {
		candidates = append(candidates, amt)
	}
	// 小写金额（≤2位小数）
	if amt := decimalToAmount(intPart, decPart, false); amt != "" {
		candidates = append(candidates, amt)
	}
	// 小写中文读法
	candidates = append(candidates, decimalToChineseText(intPart, decPart, false))
	// 大写中文读法
	candidates = append(candidates, decimalToChineseText(intPart, decPart, true))
	// 单字逐位（包含小数点）
	candidates = append(candidates, digitsToChineseChars(s, false))
	candidates = append(candidates, digitsToChineseChars(s, true))

	return candidates
}
