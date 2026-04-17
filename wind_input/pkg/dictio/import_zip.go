package dictio

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
)

// ZipImportResult ZIP 导入的解析结果。
type ZipImportResult struct {
	Manifest *ZipManifest
	Schemas  map[string]*ImportResult // schemaID → 导入结果
	Phrases  *ImportResult            // 短语导入结果（可能为 nil）
}

// ImportZip 解析 ZIP 备份包。
func ImportZip(r io.ReaderAt, size int64, opts ImportOptions) (*ZipImportResult, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}

	// 读取所有文件内容到内存
	files := make(map[string][]byte)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("打开 ZIP entry %q: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("读取 ZIP entry %q: %w", f.Name, err)
		}
		files[f.Name] = data
	}

	// 解析 manifest
	manifestData, ok := files["manifest.yaml"]
	if !ok {
		return nil, fmt.Errorf("ZIP 中缺少 manifest.yaml")
	}
	manifest, err := ParseZipManifest(manifestData)
	if err != nil {
		return nil, err
	}

	result := &ZipImportResult{
		Manifest: manifest,
		Schemas:  make(map[string]*ImportResult),
	}

	// 解析各方案文件
	importer := &WindDictImporter{}
	for _, schema := range manifest.Schemas {
		data, ok := files[schema.File]
		if !ok {
			return nil, fmt.Errorf("ZIP 中缺少方案文件 %q", schema.File)
		}

		importResult, err := importer.Import(bytes.NewReader(data), opts)
		if err != nil {
			return nil, fmt.Errorf("解析方案 %q (%s): %w", schema.ID, schema.File, err)
		}
		result.Schemas[schema.ID] = importResult
	}

	// 解析短语文件
	if manifest.Phrases != nil {
		data, ok := files[manifest.Phrases.File]
		if !ok {
			return nil, fmt.Errorf("ZIP 中缺少短语文件 %q", manifest.Phrases.File)
		}

		phraseOpts := ImportOptions{Sections: []string{SectionPhrases}}
		importResult, err := importer.Import(bytes.NewReader(data), phraseOpts)
		if err != nil {
			return nil, fmt.Errorf("解析短语文件 (%s): %w", manifest.Phrases.File, err)
		}
		result.Phrases = importResult
	}

	return result, nil
}

// PreviewZip 预览 ZIP 备份包内容（不完整解析数据）。
func PreviewZip(r io.ReaderAt, size int64) (*ZipManifest, map[string]map[string]int, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, nil, fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}

	files := make(map[string][]byte)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		files[f.Name] = data
	}

	manifestData, ok := files["manifest.yaml"]
	if !ok {
		return nil, nil, fmt.Errorf("ZIP 中缺少 manifest.yaml")
	}
	manifest, err := ParseZipManifest(manifestData)
	if err != nil {
		return nil, nil, err
	}

	// 统计各方案的 section 行数
	schemaCounts := make(map[string]map[string]int)

	for _, schema := range manifest.Schemas {
		data, ok := files[schema.File]
		if !ok {
			continue
		}
		_, counts, err := PreviewWindDict(data)
		if err != nil {
			continue
		}
		schemaCounts[schema.ID] = counts
	}

	// 统计短语
	if manifest.Phrases != nil {
		if data, ok := files[manifest.Phrases.File]; ok {
			_, counts, err := PreviewWindDict(data)
			if err == nil {
				schemaCounts["_phrases"] = counts
			}
		}
	}

	return manifest, schemaCounts, nil
}
