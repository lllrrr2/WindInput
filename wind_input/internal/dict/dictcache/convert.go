package dictcache

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/dict/binformat"
)

// CodeTableMeta 存储 CodeTable 的 Header 信息（sidecar 文件）
type CodeTableMeta struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Author        string `json:"author"`
	CodeScheme    string `json:"code_scheme"`
	CodeLength    int    `json:"code_length"`
	BWCodeLength  int    `json:"bw_code_length"`
	SpecialPrefix string `json:"special_prefix"`
	PhraseRule    int    `json:"phrase_rule"`
	EntryCount    int    `json:"entry_count"`
}

// MetaPath 返回 wdb 文件对应的 meta.json 路径
func MetaPath(wdbPath string) string {
	return wdbPath + ".meta.json"
}

// ConvertCodeTableToWdb 将文本码表转换为 wdb 二进制格式
func ConvertCodeTableToWdb(srcPath, wdbPath string) error {
	log.Printf("[dictcache] 转换码表: %s → %s", srcPath, wdbPath)

	ct, err := dict.LoadCodeTable(srcPath)
	if err != nil {
		return fmt.Errorf("加载码表失败: %w", err)
	}

	// 构建 DictWriter
	writer := binformat.NewDictWriter()
	entries := ct.GetEntries()

	for code, candidates := range entries {
		binEntries := make([]binformat.DictEntry, len(candidates))
		for i, c := range candidates {
			binEntries[i] = binformat.DictEntry{
				Text:   c.Text,
				Weight: int32(c.Weight),
			}
		}
		writer.AddCode(code, binEntries)
	}

	// 确保输出目录存在
	os.MkdirAll(filepath.Dir(wdbPath), 0755)

	// 写入 wdb 文件
	f, err := os.Create(wdbPath)
	if err != nil {
		return fmt.Errorf("创建 wdb 文件失败: %w", err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	if err := writer.Write(bw); err != nil {
		return fmt.Errorf("写入 wdb 失败: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush 失败: %w", err)
	}

	// 写入 meta.json sidecar
	meta := CodeTableMeta{
		Name:          ct.Header.Name,
		Version:       ct.Header.Version,
		Author:        ct.Header.Author,
		CodeScheme:    ct.Header.CodeScheme,
		CodeLength:    ct.Header.CodeLength,
		BWCodeLength:  ct.Header.BWCodeLength,
		SpecialPrefix: ct.Header.SpecialPrefix,
		PhraseRule:    ct.Header.PhraseRule,
		EntryCount:    ct.EntryCount(),
	}
	if err := writeMetaJSON(MetaPath(wdbPath), &meta); err != nil {
		return fmt.Errorf("写入 meta.json 失败: %w", err)
	}

	log.Printf("[dictcache] 码表转换完成: %d 编码", len(entries))
	return nil
}

// LoadCodeTableMeta 加载 meta.json
func LoadCodeTableMeta(wdbPath string) (*CodeTableMeta, error) {
	data, err := os.ReadFile(MetaPath(wdbPath))
	if err != nil {
		return nil, err
	}
	var meta CodeTableMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func writeMetaJSON(path string, meta *CodeTableMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ConvertPinyinToWdb 将拼音 YAML 词库转换为 wdb 二进制格式
// 复用 gen_bindict 的核心逻辑
func ConvertPinyinToWdb(dictDir, wdbPath string) error {
	log.Printf("[dictcache] 转换拼音词库: %s → %s", dictDir, wdbPath)

	codeEntries := make(map[string][]dictEntry)
	abbrevEntries := make(map[string][]dictEntry)

	files := []string{"8105.dict.yaml", "base.dict.yaml"}
	totalCount := 0

	for _, name := range files {
		path := filepath.Join(dictDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("[dictcache] 跳过不存在的文件: %s", path)
			continue
		}

		count, err := loadRimeFile(path, codeEntries, abbrevEntries)
		if err != nil {
			return fmt.Errorf("加载 %s 失败: %w", name, err)
		}
		log.Printf("[dictcache] 加载 %s: %d 条", name, count)
		totalCount += count
	}

	if totalCount == 0 {
		return fmt.Errorf("未加载到任何拼音词条")
	}

	writer := binformat.NewDictWriter()

	for code, entries := range codeEntries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].weight > entries[j].weight
		})
		binEntries := make([]binformat.DictEntry, len(entries))
		for i, e := range entries {
			binEntries[i] = binformat.DictEntry{
				Text:   e.text,
				Weight: int32(e.weight),
			}
		}
		writer.AddCode(code, binEntries)
	}

	for abbrev, entries := range abbrevEntries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].weight > entries[j].weight
		})
		binEntries := make([]binformat.DictEntry, len(entries))
		for i, e := range entries {
			binEntries[i] = binformat.DictEntry{
				Text:   e.text,
				Weight: int32(e.weight),
			}
		}
		writer.AddAbbrev(abbrev, binEntries)
	}

	os.MkdirAll(filepath.Dir(wdbPath), 0755)

	f, err := os.Create(wdbPath)
	if err != nil {
		return fmt.Errorf("创建 wdb 文件失败: %w", err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	if err := writer.Write(bw); err != nil {
		return fmt.Errorf("写入 wdb 失败: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush 失败: %w", err)
	}

	log.Printf("[dictcache] 拼音词库转换完成: %d codes, %d abbrevs", len(codeEntries), len(abbrevEntries))
	return nil
}

// ConvertUnigramToWdb 将 unigram.txt 转换为 unigram.wdb
func ConvertUnigramToWdb(txtPath, wdbPath string) error {
	log.Printf("[dictcache] 转换 Unigram: %s → %s", txtPath, wdbPath)

	file, err := os.Open(txtPath)
	if err != nil {
		return fmt.Errorf("打开 unigram 文件失败: %w", err)
	}
	defer file.Close()

	freqs := make(map[string]float64)
	var total float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		word := parts[0]
		freq, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}
		freqs[word] = freq
		total += freq
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 unigram 文件失败: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("unigram 文件为空")
	}

	writer := binformat.NewUnigramWriter()
	for word, freq := range freqs {
		logProb := math.Log(freq / total)
		writer.Add(word, logProb)
	}

	os.MkdirAll(filepath.Dir(wdbPath), 0755)

	f, err := os.Create(wdbPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	if err := writer.Write(bw); err != nil {
		return fmt.Errorf("写入失败: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush 失败: %w", err)
	}

	log.Printf("[dictcache] Unigram 转换完成: %d 词条", len(freqs))
	return nil
}

// ---- 内部辅助 ----

type dictEntry struct {
	text   string
	weight int
}

func loadRimeFile(path string, codeEntries map[string][]dictEntry, abbrevEntries map[string][]dictEntry) (int, error) {
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

		if inHeader {
			if strings.TrimSpace(line) == "..." {
				inHeader = false
			}
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		text := parts[0]
		pinyin := parts[1]
		weight, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil || weight <= 0 {
			continue
		}

		code := strings.ReplaceAll(pinyin, " ", "")
		codeEntries[code] = append(codeEntries[code], dictEntry{
			text:   text,
			weight: weight,
		})

		// 构建简拼索引（2 字及以上）
		syllables := strings.Fields(pinyin)
		if len(syllables) >= 2 {
			var abbrevBuilder strings.Builder
			for _, s := range syllables {
				if len(s) == 0 {
					break
				}
				abbrevBuilder.WriteByte(s[0])
			}
			abbrev := abbrevBuilder.String()
			if abbrev != "" {
				abbrevEntries[abbrev] = append(abbrevEntries[abbrev], dictEntry{
					text:   text,
					weight: weight,
				})
			}
		}

		count++
	}

	return count, scanner.Err()
}
