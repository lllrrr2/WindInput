package dictio

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// PhraseYAMLImporter 解析旧版纯 YAML 短语格式。
type PhraseYAMLImporter struct{}

func (p *PhraseYAMLImporter) Name() string         { return "短语YAML" }
func (p *PhraseYAMLImporter) Extensions() []string { return []string{".yaml", ".yml"} }

// phraseYAMLFile 旧版短语文件结构。
type phraseYAMLFile struct {
	Phrases []phraseYAMLEntry `yaml:"phrases"`
}

type phraseYAMLEntry struct {
	Code     string `yaml:"code"`
	Text     string `yaml:"text,omitempty"`
	Texts    string `yaml:"texts,omitempty"`
	Name     string `yaml:"name,omitempty"`
	Position int    `yaml:"position,omitempty"`
	Disabled bool   `yaml:"disabled,omitempty"`
}

// Import 从 reader 中解析旧版短语 YAML 格式。
func (p *PhraseYAMLImporter) Import(r io.Reader, opts ImportOptions) (*ImportResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}

	// 如果是 WindDict 格式，拒绝处理
	if IsWindDictFile(data) {
		return nil, fmt.Errorf("此文件是 WindDict 格式，请使用 WindDict 导入器")
	}

	var file phraseYAMLFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	result := &ImportResult{}

	for i, e := range file.Phrases {
		if e.Code == "" || (e.Text == "" && e.Texts == "") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("第 %d 条: 缺少编码或文本，已跳过", i+1))
			result.Stats.SkippedCount++
			continue
		}

		pType := "static"
		text := e.Text
		if e.Texts != "" {
			pType = "array"
			text = e.Texts
		} else if strings.Contains(e.Text, "$") {
			pType = "dynamic"
		}

		pos := e.Position
		if pos <= 0 {
			pos = 1
		}

		entry := PhraseEntry{
			Code:     e.Code,
			Type:     pType,
			Text:     text,
			Position: pos,
			Enabled:  !e.Disabled,
			Name:     e.Name,
		}
		result.Phrases = append(result.Phrases, entry)
	}

	result.UpdateStats()
	return result, nil
}
