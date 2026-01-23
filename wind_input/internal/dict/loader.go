package dict

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/huanfeng/wind_input/internal/candidate"
)

// SimpleDict 简单的词库实现（使用 map）
type SimpleDict struct {
	entries    map[string][]candidate.Candidate // pinyin -> candidates
	phrases    map[string][]candidate.Candidate // joined syllables -> candidates
	entryCount int                              // 总词条数
}

// NewSimpleDict 创建简单词库
func NewSimpleDict() *SimpleDict {
	return &SimpleDict{
		entries: make(map[string][]candidate.Candidate),
		phrases: make(map[string][]candidate.Candidate),
	}
}

// Load 加载词库文件
// 文件格式: 拼音 汉字 权重
// 例如: ni 你 100
func (d *SimpleDict) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open dict file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entryOrder := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 跳过码表头部标记
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		// 解析行（支持 tab 或空格分隔）
		var parts []string
		if strings.Contains(line, "\t") {
			parts = strings.Split(line, "\t")
		} else {
			parts = strings.Fields(line)
		}
		if len(parts) < 2 {
			continue
		}

		pinyin := strings.TrimSpace(parts[0])
		text := strings.TrimSpace(parts[1])

		// 解析权重：如果有显式权重则使用，否则按文件顺序递减
		weight := 0
		hasExplicitWeight := false
		if len(parts) >= 3 {
			if w, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
				weight = w
				hasExplicitWeight = true
			}
		}

		// 如果没有显式权重，使用递减的顺序权重
		if !hasExplicitWeight {
			weight = 1000000 - entryOrder
			if weight < 0 {
				weight = 0
			}
		}

		// 添加到词库
		cand := candidate.Candidate{
			Text:   text,
			Pinyin: pinyin,
			Weight: weight,
		}

		d.entries[pinyin] = append(d.entries[pinyin], cand)
		entryOrder++
	}

	d.entryCount = entryOrder

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read dict file: %w", err)
	}

	return nil
}

// Lookup 查找单个拼音
func (d *SimpleDict) Lookup(pinyin string) []candidate.Candidate {
	pinyin = strings.ToLower(pinyin)
	return d.entries[pinyin]
}

// LookupPhrase 查找短语
// syllables 是音节列表，如 ["ni", "hao"]
// 会拼接成 "nihao" 在 entries 中查找
func (d *SimpleDict) LookupPhrase(syllables []string) []candidate.Candidate {
	key := strings.ToLower(strings.Join(syllables, ""))
	// 优先从 entries 查找（词库中的词组是以完整拼音为 key 存储的）
	if results := d.entries[key]; len(results) > 0 {
		return results
	}
	// 回退到 phrases（用于手动添加的短语）
	return d.phrases[key]
}

// AddEntry 添加词条
func (d *SimpleDict) AddEntry(pinyin, text string, weight int) {
	cand := candidate.Candidate{
		Text:   text,
		Pinyin: pinyin,
		Weight: weight,
	}
	d.entries[pinyin] = append(d.entries[pinyin], cand)
}

// AddPhrase 添加短语
func (d *SimpleDict) AddPhrase(syllables []string, text string, weight int) {
	key := strings.Join(syllables, "")
	cand := candidate.Candidate{
		Text:   text,
		Pinyin: key,
		Weight: weight,
	}
	d.phrases[key] = append(d.phrases[key], cand)
}

// EntryCount 返回词条数量
func (d *SimpleDict) EntryCount() int {
	return d.entryCount
}
