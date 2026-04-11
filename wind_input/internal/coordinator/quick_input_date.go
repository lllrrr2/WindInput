// quick_input_date.go — 快捷输入：日期格式化模块
package coordinator

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// isDateExpression 检查字符串是否为日期表达式（数字和点号组成，至少3段或2段合理数字）
func isDateExpression(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	if len(parts) == 2 {
		m, _ := strconv.Atoi(parts[0])
		d, _ := strconv.Atoi(parts[1])
		return m >= 1 && m <= 12 && d >= 1 && d <= 31
	}
	return len(parts) == 3
}

// parseDateParts 解析日期字符串为年月日，省略年份时 year=0
func parseDateParts(s string) (year, month, day int, ok bool) {
	parts := strings.Split(s, ".")
	if len(parts) == 2 {
		m, err1 := strconv.Atoi(parts[0])
		d, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || m < 1 || m > 12 || d < 1 || d > 31 {
			return 0, 0, 0, false
		}
		return 0, m, d, true
	}
	if len(parts) == 3 {
		y, err1 := strconv.Atoi(parts[0])
		m, err2 := strconv.Atoi(parts[1])
		d, err3 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		if m < 1 || m > 12 || d < 1 || d > 31 {
			return 0, 0, 0, false
		}
		return y, m, d, true
	}
	return 0, 0, 0, false
}

// isYearMonthExpression 检查是否为年月表达式（两段数字，首段>31，第二段1-12）
func isYearMonthExpression(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return y > 31 && m >= 1 && m <= 12
}

// generateYearMonthCandidates 根据年月字符串生成候选列表
func generateYearMonthCandidates(input string) []string {
	parts := strings.Split(input, ".")
	if len(parts) != 2 {
		return nil
	}
	y, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || y <= 31 || m < 1 || m > 12 {
		return nil
	}

	candidates := make([]string, 0, 4)
	candidates = append(candidates, fmt.Sprintf("%d年%d月", y, m))
	candidates = append(candidates, fmt.Sprintf("%d年%02d月", y, m))
	candidates = append(candidates, fmt.Sprintf("%04d-%02d", y, m))
	candidates = append(candidates, fmt.Sprintf("%04d/%02d", y, m))
	return candidates
}

// generateDateCandidates 根据日期字符串生成候选列表
func generateDateCandidates(input string) []string {
	year, month, day, ok := parseDateParts(input)
	if !ok {
		return nil
	}

	if year == 0 {
		year = time.Now().Year()
	}

	candidates := make([]string, 0, 5)
	candidates = append(candidates, fmt.Sprintf("%04d%02d%02d", year, month, day))
	candidates = append(candidates, fmt.Sprintf("%d年%d月%d日", year, month, day))
	candidates = append(candidates, fmt.Sprintf("%d年%02d月%02d日", year, month, day))
	candidates = append(candidates, fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	candidates = append(candidates, fmt.Sprintf("%04d/%02d/%02d", year, month, day))

	return candidates
}
