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
	entries map[string][]candidate.Candidate // pinyin -> candidates
	phrases map[string][]candidate.Candidate // joined syllables -> candidates
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
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析行
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pinyin := parts[0]
		text := parts[1]
		weight := 50 // 默认权重

		if len(parts) >= 3 {
			if w, err := strconv.Atoi(parts[2]); err == nil {
				weight = w
			}
		}

		// 添加到词库
		cand := candidate.Candidate{
			Text:   text,
			Pinyin: pinyin,
			Weight: weight,
		}

		d.entries[pinyin] = append(d.entries[pinyin], cand)
	}

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
func (d *SimpleDict) LookupPhrase(syllables []string) []candidate.Candidate {
	key := strings.Join(syllables, "")
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
