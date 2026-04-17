package dictio

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// RimeDictImporter 解析 Rime 用户词库格式（*.dict.yaml）。
type RimeDictImporter struct{}

func (p *RimeDictImporter) Name() string         { return "Rime词库" }
func (p *RimeDictImporter) Extensions() []string { return []string{".dict.yaml"} }

// rimeHeader Rime 词库文件的 YAML 头部。
type rimeHeader struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Columns []string `yaml:"columns"`
	Sort    string   `yaml:"sort"`
}

// Import 从 reader 中解析 Rime dict.yaml 格式。
func (p *RimeDictImporter) Import(r io.Reader, opts ImportOptions) (*ImportResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}

	content := string(data)

	// Rime 格式使用 "..." 分隔头部和数据
	headerYAML, dataSection, found := strings.Cut(content, "\n...")
	if !found {
		return nil, fmt.Errorf("无效的 Rime 词库格式: 缺少 '...' 数据分隔符")
	}
	// 去除可能的 --- 前缀
	headerYAML = strings.TrimPrefix(headerYAML, "---\n")
	headerYAML = strings.TrimPrefix(headerYAML, "---\r\n")

	var header rimeHeader
	if err := yaml.Unmarshal([]byte(headerYAML), &header); err != nil {
		return nil, fmt.Errorf("解析 Rime 头部失败: %w", err)
	}

	// 确定列定义
	columns := header.Columns
	if len(columns) == 0 {
		// Rime 默认列顺序：text, code, weight
		columns = []string{"text", "code", "weight"}
	}

	// 映射 Rime 列名到我们的列名
	mappedColumns := make([]string, len(columns))
	for i, col := range columns {
		switch col {
		case "text":
			mappedColumns[i] = "text"
		case "code":
			mappedColumns[i] = "code"
		case "weight":
			mappedColumns[i] = "weight"
		case "stem":
			mappedColumns[i] = "stem" // 忽略
		default:
			mappedColumns[i] = col
		}
	}

	cols := NewColumnDef(mappedColumns)

	// 解析数据部分（dataSection 已由 strings.Cut 获得）
	result := &ImportResult{}
	lineNum := 0

	scanner := bufio.NewScanner(strings.NewReader(dataSection))
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		// Rime 词库中可能含全角空格（\u3000）等，逐字段 trim
		for i := range fields {
			fields[i] = trimUnicodeSpaces(fields[i])
		}
		text := cols.GetUnescaped(fields, "text")
		code := cols.GetUnescaped(fields, "code")

		if text == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("第 %d 行: 缺少文本，已跳过", lineNum))
			result.Stats.SkippedCount++
			continue
		}

		weight := 0
		if ws := cols.Get(fields, "weight"); ws != "" {
			if w, err := strconv.Atoi(ws); err == nil {
				weight = w
			}
		}

		entry := UserWordEntry{
			Code:   code,
			Text:   text,
			Weight: weight,
		}
		result.UserWords = append(result.UserWords, entry)
	}

	result.UpdateStats()
	return result, nil
}
