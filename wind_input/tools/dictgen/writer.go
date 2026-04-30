package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func writeRimeYAML(path string, entries []Entry, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	bw := bufio.NewWriter(f)
	version := time.Now().Format("2006-01-02")
	boost := fmt.Sprintf("%.1f", cfg.CharBoostFactor)

	fmt.Fprintf(bw, "# Rime dictionary\n")
	fmt.Fprintf(bw, "# encoding: utf-8\n")
	fmt.Fprintf(bw, "#\n")
	fmt.Fprintf(bw, "# WindInput 五笔86词库\n")
	fmt.Fprintf(bw, "# 来源: rime-wubi86-jidian (https://github.com/KyleBing/rime-wubi86-jidian)\n")
	fmt.Fprintf(bw, "# 处理: 按 unigram 真实词频重新排序，单字提权（×%s），生僻字保底权重\n", boost)
	fmt.Fprintf(bw, "# 生成: WindInput/tools/dictgen  版本: %s\n", version)
	fmt.Fprintf(bw, "---\n")
	fmt.Fprintf(bw, "name: %s\n", cfg.OutputName)
	fmt.Fprintf(bw, "version: \"%s\"\n", version)
	fmt.Fprintf(bw, "sort: by_weight\n")
	if len(cfg.ImportTables) > 0 {
		fmt.Fprintf(bw, "import_tables:\n")
		for _, t := range cfg.ImportTables {
			fmt.Fprintf(bw, "  - %s\n", t)
		}
	}
	fmt.Fprintf(bw, "columns:\n")
	fmt.Fprintf(bw, "  - code\n")
	fmt.Fprintf(bw, "  - text\n")
	fmt.Fprintf(bw, "  - weight\n")
	fmt.Fprintf(bw, "...\n")

	for _, e := range entries {
		fmt.Fprintf(bw, "%s\t%s\t%d\n", e.Code, e.Text, e.OrigWeight)
	}

	if err := bw.Flush(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// 原子替换（Windows 需先删除目标）
	os.Remove(path)
	return os.Rename(tmpPath, path)
}

func writeDropped(path string, dropped []droppedEntry) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	fmt.Fprintf(bw, "reason\tcode\ttext\torig_weight\n")

	// 按原因+编码排序
	sorted := make([]droppedEntry, len(dropped))
	copy(sorted, dropped)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].reason != sorted[j].reason {
			return sorted[i].reason < sorted[j].reason
		}
		return sorted[i].e.Code < sorted[j].e.Code
	})

	for _, d := range sorted {
		text := strings.ReplaceAll(d.e.Text, "\t", " ")
		fmt.Fprintf(bw, "%s\t%s\t%s\t%d\n", d.reason, d.e.Code, text, d.e.OrigWeight)
	}
	bw.Flush()
}
