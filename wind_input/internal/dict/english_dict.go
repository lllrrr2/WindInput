package dict

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/huanfeng/wind_input/internal/candidate"
)

// EnglishDict 英文词库（基于 Trie 索引）
// 支持从 Rime dict.yaml 格式加载英文单词表
type EnglishDict struct {
	logger     *slog.Logger
	trie       *Trie
	seen       map[string]bool // 加载时去重（小写 → 是否已加载）
	entryCount int
}

// NewEnglishDict 创建英文词库
func NewEnglishDict(logger *slog.Logger) *EnglishDict {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnglishDict{logger: logger}
}

// LoadRimeDir 从目录加载 Rime dict.yaml 格式英文词库
// 自动查找并加载 en.dict.yaml 和 en_ext.dict.yaml
func (d *EnglishDict) LoadRimeDir(dirPath string) error {
	d.trie = NewTrie()
	d.seen = make(map[string]bool)
	d.entryCount = 0

	files := []string{
		"en.dict.yaml",     // 主词库
		"en_ext.dict.yaml", // 扩展词库
	}

	loaded := 0
	for _, name := range files {
		path := filepath.Join(dirPath, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			d.logger.Info("跳过不存在的英文词库文件", "name", name)
			continue
		}
		count, err := d.loadRimeFile(path)
		if err != nil {
			d.logger.Warn("加载英文词库文件失败", "name", name, "error", err)
			continue
		}
		d.logger.Info("加载英文词库文件", "name", name, "count", count)
		loaded++
	}

	if loaded == 0 {
		return fmt.Errorf("未找到任何英文词库文件（目录: %s）", dirPath)
	}

	d.entryCount = d.trie.EntryCount()
	return nil
}

// LoadRimeFile 加载单个 Rime dict.yaml 格式英文词库文件
// 如果 trie 未初始化，会自动创建
func (d *EnglishDict) LoadRimeFile(path string) error {
	if d.trie == nil {
		d.trie = NewTrie()
		d.seen = make(map[string]bool)
		d.entryCount = 0
	}
	if d.seen == nil {
		d.seen = make(map[string]bool)
	}
	count, err := d.loadRimeFile(path)
	if err != nil {
		return err
	}
	d.entryCount = d.trie.EntryCount()
	d.logger.Info("加载英文词库完成", "path", path, "count", count)
	return nil
}

// loadRimeFile 解析单个 Rime dict.yaml 文件
// 支持三种格式：
//   - 单列：word（默认权重 1）
//   - 两列：word\tweight 或 word\tcode
//   - 三列：word\tcode\tweight（标准 rime 格式）
func (d *EnglishDict) loadRimeFile(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	inHeader := true
	count := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过 YAML 头部（--- 到 ... 之间）
		if inHeader {
			if strings.TrimSpace(line) == "..." {
				inHeader = false
			}
			continue
		}

		// 跳过空行和注释
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")

		var word string
		weight := 1

		switch len(parts) {
		case 1:
			// 单列：word
			word = strings.TrimSpace(parts[0])
		case 2:
			// 两列：word\tweight
			word = strings.TrimSpace(parts[0])
			if w, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && w > 0 {
				weight = w
			}
		default:
			// 三列及以上：word\tcode\tweight（标准 rime 格式）
			word = strings.TrimSpace(parts[0])
			if w, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1])); err == nil && w > 0 {
				weight = w
			}
		}

		if word == "" {
			continue
		}

		code := strings.ToLower(word)

		// 去重：同一小写形式只保留第一次出现的
		if d.seen[code] {
			continue
		}
		d.seen[code] = true

		cand := candidate.Candidate{
			Text:   word,
			Code:   code,
			Weight: weight,
		}
		d.trie.Insert(code, cand)
		count++
	}

	if err := scanner.Err(); err != nil {
		return count, err
	}

	return count, nil
}

// EntryCount 返回词条数量
func (d *EnglishDict) EntryCount() int {
	return d.entryCount
}

// Lookup 精确查询（大小写不敏感）
func (d *EnglishDict) Lookup(word string) []candidate.Candidate {
	if d.trie == nil {
		return nil
	}
	return d.trie.Search(strings.ToLower(word))
}

// LookupPrefix 前缀查询（大小写不敏感）
// 排序：精确匹配 > 词库自然顺序（文件中的出现顺序）
func (d *EnglishDict) LookupPrefix(prefix string, limit int) []candidate.Candidate {
	if d.trie == nil {
		return nil
	}
	prefixLower := strings.ToLower(prefix)
	results := d.trie.SearchPrefix(prefixLower, 0)

	// 排序：精确匹配 > 短词优先 > 自然顺序
	sort.SliceStable(results, func(i, j int) bool {
		ci, cj := results[i], results[j]
		// 精确匹配优先
		exactI := ci.Code == prefixLower
		exactJ := cj.Code == prefixLower
		if exactI != exactJ {
			return exactI
		}
		// 短词优先（更常用/更相关）
		if len(ci.Code) != len(cj.Code) {
			return len(ci.Code) < len(cj.Code)
		}
		// 同长度按自然顺序（词库中的位置）
		return ci.NaturalOrder < cj.NaturalOrder
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Close 释放 trie
func (d *EnglishDict) Close() error {
	d.trie = nil
	return nil
}

// EnglishDictLayer 将 EnglishDict 适配为 DictLayer
type EnglishDictLayer struct {
	name string
	dict *EnglishDict
}

// NewEnglishDictLayer 创建 EnglishDict 适配器
func NewEnglishDictLayer(name string, dict *EnglishDict) *EnglishDictLayer {
	return &EnglishDictLayer{
		name: name,
		dict: dict,
	}
}

// Name 返回层名称
func (l *EnglishDictLayer) Name() string {
	return l.name
}

// Type 返回层类型
func (l *EnglishDictLayer) Type() LayerType {
	return LayerTypeSystem
}

// Search 精确查询
func (l *EnglishDictLayer) Search(code string, limit int) []candidate.Candidate {
	results := l.dict.Lookup(code)
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SearchPrefix 前缀查询
func (l *EnglishDictLayer) SearchPrefix(prefix string, limit int) []candidate.Candidate {
	return l.dict.LookupPrefix(prefix, limit)
}
