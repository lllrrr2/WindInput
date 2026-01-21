package pinyin

import (
	"strings"
)

// 声母
var shengmu = []string{
	"b", "p", "m", "f",
	"d", "t", "n", "l",
	"g", "k", "h",
	"j", "q", "x",
	"zh", "ch", "sh", "r",
	"z", "c", "s",
	"y", "w",
}

// 韵母
var yunmu = []string{
	"a", "o", "e", "i", "u", "v",
	"ai", "ei", "ui", "ao", "ou", "iu", "ie", "ve", "er",
	"an", "en", "in", "un", "vn",
	"ang", "eng", "ing", "ong",
}

// 整体认读音节
var wholeSyllables = []string{
	"zhi", "chi", "shi", "ri",
	"zi", "ci", "si",
	"yi", "wu", "yu",
	"ye", "yue", "yuan", "yin", "yun", "ying",
}

// ParseSyllables 解析拼音为音节列表
// 例如: "nihao" -> ["ni", "hao"]
func ParseSyllables(pinyin string) [][]string {
	pinyin = strings.ToLower(pinyin)

	if len(pinyin) == 0 {
		return nil
	}

	// 使用动态规划找到所有可能的分割方式
	results := [][]string{}
	parseSyllablesRecursive(pinyin, 0, []string{}, &results)

	return results
}

// parseSyllablesRecursive 递归解析音节
func parseSyllablesRecursive(pinyin string, start int, current []string, results *[][]string) {
	if start >= len(pinyin) {
		if len(current) > 0 {
			// 复制当前结果
			result := make([]string, len(current))
			copy(result, current)
			*results = append(*results, result)
		}
		return
	}

	// 尝试匹配整体认读音节（优先）
	for _, syllable := range wholeSyllables {
		if start+len(syllable) <= len(pinyin) &&
			pinyin[start:start+len(syllable)] == syllable {
			parseSyllablesRecursive(pinyin, start+len(syllable),
				append(current, syllable), results)
		}
	}

	// 尝试匹配声母+韵母
	for _, sm := range shengmu {
		if start+len(sm) > len(pinyin) {
			continue
		}
		if pinyin[start:start+len(sm)] != sm {
			continue
		}

		smEnd := start + len(sm)
		for _, ym := range yunmu {
			if smEnd+len(ym) > len(pinyin) {
				continue
			}
			if pinyin[smEnd:smEnd+len(ym)] != ym {
				continue
			}

			syllable := sm + ym
			parseSyllablesRecursive(pinyin, smEnd+len(ym),
				append(current, syllable), results)
		}
	}

	// 尝试匹配单个韵母（零声母）
	for _, ym := range yunmu {
		if start+len(ym) <= len(pinyin) &&
			pinyin[start:start+len(ym)] == ym {
			parseSyllablesRecursive(pinyin, start+len(ym),
				append(current, ym), results)
		}
	}
}

// IsValidSyllable 检查是否是有效的拼音音节
func IsValidSyllable(syllable string) bool {
	syllable = strings.ToLower(syllable)

	// 检查整体认读音节
	for _, ws := range wholeSyllables {
		if syllable == ws {
			return true
		}
	}

	// 检查声母+韵母组合
	for _, sm := range shengmu {
		if strings.HasPrefix(syllable, sm) {
			rest := syllable[len(sm):]
			for _, ym := range yunmu {
				if rest == ym {
					return true
				}
			}
		}
	}

	// 检查单个韵母
	for _, ym := range yunmu {
		if syllable == ym {
			return true
		}
	}

	return false
}
