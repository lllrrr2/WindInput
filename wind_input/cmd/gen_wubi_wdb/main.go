package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/huanfeng/wind_input/internal/dict/dictcache"
)

func main() {
	srcPath := flag.String("src", "dict/wubi/wubi86.txt", "五笔码表源文件路径")
	outDir := flag.String("out", "dict/wubi", "输出目录")
	flag.Parse()

	wdbPath := filepath.Join(*outDir, "wubi.wdb")

	if err := dictcache.ConvertCodeTableToWdb(*srcPath, wdbPath); err != nil {
		log.Fatalf("生成 wubi.wdb 失败: %v", err)
	}

	log.Printf("wubi.wdb 生成完毕 → %s", wdbPath)
}
