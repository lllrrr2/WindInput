package dictio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// headerWrapper 用于解析 YAML 头部中的 wind_dict 顶层键。
type headerWrapper struct {
	WindDict WindDictHeader `yaml:"wind_dict"`
}

// zipManifestWrapper 用于解析 ZIP 清单中的 wind_backup 顶层键。
type zipManifestWrapper struct {
	WindBackup ZipManifest `yaml:"wind_backup"`
}

// ParseHeader 从文件内容中解析 WindDict YAML 头部。
// 只解析第一个 YAML 文档（--- 分隔符之前的部分）。
func ParseHeader(data []byte) (*WindDictHeader, error) {
	// 提取第一个 YAML 文档
	firstDoc := extractFirstDocument(data)

	var wrapper headerWrapper
	if err := yaml.Unmarshal(firstDoc, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 YAML 头部失败: %w", err)
	}

	h := &wrapper.WindDict
	if h.Version == 0 {
		return nil, fmt.Errorf("无效的 WindDict 文件: 缺少 version 字段")
	}

	return h, nil
}

// WriteHeader 将 WindDict YAML 头部写入 writer。
func WriteHeader(w io.Writer, header *WindDictHeader) error {
	wrapper := headerWrapper{WindDict: *header}
	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return fmt.Errorf("序列化 YAML 头部失败: %w", err)
	}

	// 写入注释
	if _, err := fmt.Fprintln(w, "# WindInput 用户数据文件"); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ParseZipManifest 解析 ZIP 备份包的清单文件。
func ParseZipManifest(data []byte) (*ZipManifest, error) {
	var wrapper zipManifestWrapper
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 ZIP 清单失败: %w", err)
	}
	m := &wrapper.WindBackup
	if m.Version == 0 {
		return nil, fmt.Errorf("无效的 ZIP 清单: 缺少 version 字段")
	}
	return m, nil
}

// WriteZipManifest 将 ZIP 清单写入 writer。
func WriteZipManifest(w io.Writer, manifest *ZipManifest) error {
	wrapper := zipManifestWrapper{WindBackup: *manifest}
	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return fmt.Errorf("序列化 ZIP 清单失败: %w", err)
	}
	_, err = w.Write(data)
	return err
}

// SplitSections 将 .wdict.yaml 文件内容按 section 分割。
// 返回 YAML 头部内容和各 section 的 (tag, body) 对。
func SplitSections(data []byte) (header []byte, sections []SectionBlock) {
	// 使用 "\n---" 作为分隔符（YAML 多文档分隔）
	parts := splitByDocSeparator(data)

	if len(parts) == 0 {
		return nil, nil
	}

	header = parts[0]

	for _, part := range parts[1:] {
		tag, body := parseSectionTag(part)
		if tag != "" {
			sections = append(sections, SectionBlock{Tag: tag, Body: body})
		}
	}

	return header, sections
}

// SectionBlock 表示一个数据段。
type SectionBlock struct {
	Tag  string // section 名称（如 "user_words"）
	Body []byte // 数据内容（不含 tag 行）
}

// splitByDocSeparator 按 YAML 文档分隔符分割内容。
// 识别 "\n---" 或文件开头的 "---"。
func splitByDocSeparator(data []byte) [][]byte {
	var parts [][]byte
	sep := []byte("\n---")

	for {
		idx := bytes.Index(data, sep)
		if idx == -1 {
			parts = append(parts, data)
			break
		}
		parts = append(parts, data[:idx])
		data = data[idx+len(sep):]
	}

	return parts
}

// parseSectionTag 从 section 块中解析 tag 名称和 body 内容。
// 输入格式: " !tag_name\n...body..."
func parseSectionTag(block []byte) (tag string, body []byte) {
	s := string(block)

	// 查找 !tag_name
	bangIdx := strings.Index(s, "!")
	if bangIdx == -1 {
		return "", block
	}

	// 找到 tag 的结束位置（换行符或空格）
	tagStart := bangIdx + 1
	tagEnd := len(s)
	for i := tagStart; i < len(s); i++ {
		if s[i] == '\n' || s[i] == ' ' || s[i] == '\r' {
			tagEnd = i
			break
		}
	}

	tag = s[tagStart:tagEnd]

	// body 从 tag 行之后开始
	bodyStart := tagEnd
	if bodyStart < len(s) && s[bodyStart] == '\r' {
		bodyStart++
	}
	if bodyStart < len(s) && s[bodyStart] == '\n' {
		bodyStart++
	}

	return tag, []byte(s[bodyStart:])
}

// extractFirstDocument 提取第一个 YAML 文档（第一个 \n--- 之前的内容）。
func extractFirstDocument(data []byte) []byte {
	before, _, found := bytes.Cut(data, []byte("\n---"))
	if !found {
		return data
	}
	return before
}

// IsWindDictFile 检查文件内容是否是 WindDict 格式（通过检测 wind_dict: 头部）。
func IsWindDictFile(header []byte) bool {
	return bytes.Contains(header, []byte("wind_dict:"))
}
