package dictio

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"time"
)

// SchemaExportData 单个方案的导出数据（用于 ZIP 打包）。
type SchemaExportData struct {
	SchemaID   string
	SchemaName string
	Data       *ExportData
}

// ZipExportOptions ZIP 导出选项。
type ZipExportOptions struct {
	Generator string
	Sections  []string // 每个方案要导出的 section（nil = 全部）
}

// ExportZip 将多个方案的数据和短语打包为 ZIP 文件。
func ExportZip(w io.Writer, schemas []SchemaExportData, phrases []PhraseEntry, opts ZipExportOptions) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	generator := opts.Generator
	if generator == "" {
		generator = "WindInput"
	}
	exportedAt := time.Now().Format(time.RFC3339)

	// 1. 构建 manifest
	manifest := ZipManifest{
		Version:    FormatVersion,
		Generator:  generator,
		ExportedAt: exportedAt,
	}

	exporter := &WindDictExporter{}

	// 2. 写入各方案文件
	for _, s := range schemas {
		filename := s.SchemaID + ".wdict.yaml"
		manifest.Schemas = append(manifest.Schemas, ZipSchemaEntry{
			ID:   s.SchemaID,
			Name: s.SchemaName,
			File: filename,
		})

		fw, err := zw.Create(filename)
		if err != nil {
			return fmt.Errorf("创建 zip entry %q: %w", filename, err)
		}

		exportOpts := ExportOptions{
			SchemaID:   s.SchemaID,
			SchemaName: s.SchemaName,
			Sections:   opts.Sections,
			Generator:  generator,
		}
		if err := exporter.Export(fw, s.Data, exportOpts); err != nil {
			return fmt.Errorf("导出方案 %q: %w", s.SchemaID, err)
		}
	}

	// 3. 写入短语文件
	if len(phrases) > 0 {
		phrasesFile := "phrases.wdict.yaml"
		manifest.Phrases = &ZipPhrasesEntry{File: phrasesFile}

		fw, err := zw.Create(phrasesFile)
		if err != nil {
			return fmt.Errorf("创建 zip entry %q: %w", phrasesFile, err)
		}

		phraseData := &ExportData{Phrases: phrases}
		phraseOpts := ExportOptions{
			Sections:  []string{SectionPhrases},
			Generator: generator,
		}
		if err := exporter.Export(fw, phraseData, phraseOpts); err != nil {
			return fmt.Errorf("导出短语: %w", err)
		}
	}

	// 4. 写入 manifest.yaml
	fw, err := zw.Create("manifest.yaml")
	if err != nil {
		return fmt.Errorf("创建 manifest: %w", err)
	}

	var buf bytes.Buffer
	if err := WriteZipManifest(&buf, &manifest); err != nil {
		return err
	}
	if _, err := fw.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("写入 manifest: %w", err)
	}

	return nil
}
