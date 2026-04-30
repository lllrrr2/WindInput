package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
)

// buildCharCodeMap 从 jidian 单字条目建立 汉字→首选编码 的反查表。
// 每个字取 OrigWeight 最高的编码作为首选（极点词库中 weight=30 的为首选全码）。
func buildCharCodeMap(entries []Entry) map[rune]string {
	type best struct {
		code   string
		weight int
	}
	m := make(map[rune]best)
	for _, e := range entries {
		runes := []rune(e.Text)
		if len(runes) != 1 {
			continue
		}
		r := runes[0]
		if b, ok := m[r]; !ok || e.OrigWeight > b.weight {
			m[r] = best{code: e.Code, weight: e.OrigWeight}
		}
	}
	result := make(map[rune]string, len(m))
	for r, b := range m {
		result[r] = b.code
	}
	return result
}

// encodePhrase 按五笔86词组取码规则计算编码：
//
//	2字：字1前2码 + 字2前2码
//	3字：字1首码 + 字2首码 + 字3前2码
//	4字+：字1/2/3首码 + 末字首码
//
// 返回 (code, true)；若有字在码表中缺失则返回 ("", false)。
func encodePhrase(text string, charCodes map[rune]string) (string, bool) {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return "", false
	}

	get := func(r rune) (string, bool) {
		c, ok := charCodes[r]
		return c, ok
	}
	// 取编码的前 n 位，不足时补 'l'（五笔末笔识别码占位）
	prefix := func(code string, n int) string {
		for len(code) < n {
			code += "l"
		}
		return code[:n]
	}

	switch {
	case n == 1:
		return get(runes[0])

	case n == 2:
		c1, ok1 := get(runes[0])
		c2, ok2 := get(runes[1])
		if !ok1 || !ok2 {
			return "", false
		}
		return prefix(c1, 2) + prefix(c2, 2), true

	case n == 3:
		c1, ok1 := get(runes[0])
		c2, ok2 := get(runes[1])
		c3, ok3 := get(runes[2])
		if !ok1 || !ok2 || !ok3 {
			return "", false
		}
		return prefix(c1, 1) + prefix(c2, 1) + prefix(c3, 2), true

	default:
		c1, ok1 := get(runes[0])
		c2, ok2 := get(runes[1])
		c3, ok3 := get(runes[2])
		cL, okL := get(runes[n-1])
		if !ok1 || !ok2 || !ok3 || !okL {
			return "", false
		}
		return prefix(c1, 1) + prefix(c2, 1) + prefix(c3, 1) + prefix(cL, 1), true
	}
}

// loadCustomWords 从自定义词表文件加载词条，自动反查编码并从 unigram 获取词频。
//
// 格式：每行一词，可选 TAB 分隔频率；# 开头为注释。
// 示例：
//
//	人工智能
//	深度学习	5000
func loadCustomWords(path string, charCodes map[rune]string, unigram map[string]int64, logMedian float64, cfg *Config) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	skipped := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		word := strings.TrimSpace(parts[0])
		if word == "" {
			continue
		}

		code, ok := encodePhrase(word, charCodes)
		if !ok {
			fmt.Printf("        [跳过] 无法编码: %s\n", word)
			skipped++
			continue
		}

		// 权重：优先 unigram，否则用 target_median
		weight := cfg.TargetMedian
		if logMedian > 0 {
			if freq, ok2 := unigram[word]; ok2 {
				w := float64(cfg.TargetMedian) * math.Log10(float64(freq)+1) / logMedian
				weight = int(math.Round(w))
				if weight < cfg.WeightMin {
					weight = cfg.WeightMin
				}
				if weight > cfg.WeightMax {
					weight = cfg.WeightMax
				}
			}
		}
		// 手动指定频率覆盖
		if len(parts) == 2 {
			// ignore for now; freq field in custom words is informational
		}

		entries = append(entries, Entry{Text: word, Code: code, OrigWeight: weight})
	}
	if skipped > 0 {
		fmt.Printf("        跳过 %d 条（无法反查编码）\n", skipped)
	}
	return entries, scanner.Err()
}
