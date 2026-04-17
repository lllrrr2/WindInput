package dictio

import (
	"bufio"
	"io"
	"strings"
)

// TextListImporter 解析纯词语列表（一行一个词语，无编码）。
// 导入后 UserWords 条目的 Code 为空，需要调用方通过 ReverseEncoder 填充编码。
type TextListImporter struct{}

func (p *TextListImporter) Name() string         { return "纯词语列表" }
func (p *TextListImporter) Extensions() []string { return []string{".txt"} }

// Import 从 reader 中读取纯词语列表。
func (p *TextListImporter) Import(r io.Reader, opts ImportOptions) (*ImportResult, error) {
	result := &ImportResult{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 纯词语列表：Code 留空，由上层处理编码
		result.UserWords = append(result.UserWords, UserWordEntry{
			Text:   line,
			Weight: 0, // 默认权重，由上层决定
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result.UpdateStats()
	return result, nil
}
