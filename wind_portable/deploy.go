//go:build windows

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// validateZip 验证 ZIP 文件是否包含必要的文件
func validateZip(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("无法打开 ZIP 文件: %w", err)
	}
	defer r.Close()

	required := map[string]bool{
		strings.ToLower(serviceName): false,
		strings.ToLower(dllName):     false,
	}

	for _, f := range r.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if _, ok := required[base]; ok {
			required[base] = true
		}
	}

	var missing []string
	for name, found := range required {
		if !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ZIP 缺少必要文件: %s", strings.Join(missing, ", "))
	}
	return nil
}

// deployFromZip 从 ZIP 文件部署到目标目录，返回是否需要重启
func deployFromZip(zipPath, targetDir string) (needsRestart bool, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return false, fmt.Errorf("无法打开 ZIP 文件: %w", err)
	}
	defer r.Close()

	selfExe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("无法获取当前可执行文件路径: %w", err)
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		dstPath := filepath.Join(targetDir, filepath.FromSlash(f.Name))

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return needsRestart, fmt.Errorf("创建目录失败 %s: %w", filepath.Dir(dstPath), err)
		}

		// 检查是否是当前正在运行的可执行文件
		if samePath(dstPath, selfExe) {
			oldPath := fmt.Sprintf("%s.old_%d", selfExe, rand.Intn(100000))
			if err := os.Rename(selfExe, oldPath); err != nil {
				return needsRestart, fmt.Errorf("重命名当前可执行文件失败: %w", err)
			}
			needsRestart = true
		}

		if err := extractZipFile(f, dstPath); err != nil {
			return needsRestart, fmt.Errorf("解压文件失败 %s: %w", f.Name, err)
		}
	}

	return needsRestart, nil
}

// extractZipFile 从 ZIP 条目中解压单个文件到目标路径
func extractZipFile(f *zip.File, dstPath string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("打开 ZIP 条目失败: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		// 文件可能被锁定，尝试重命名后再创建
		oldPath := fmt.Sprintf("%s.old_%d", dstPath, rand.Intn(100000))
		if renameErr := os.Rename(dstPath, oldPath); renameErr != nil {
			return fmt.Errorf("创建文件失败且无法重命名: %w", err)
		}
		out, err = os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("重命名后仍无法创建文件: %w", err)
		}
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// deployFromDirectory 将源目录中的文件复制到目标目录
func deployFromDirectory(srcDir, targetDir string) error {
	srcDir = filepath.Clean(srcDir)
	targetDir = filepath.Clean(targetDir)

	if strings.EqualFold(srcDir, targetDir) {
		return fmt.Errorf("源目录与目标目录相同")
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// 跳过 userdata 目录
		if info.IsDir() && relPath == "userdata" {
			return filepath.SkipDir
		}

		dstPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return copyFile(path, dstPath)
	})
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// cleanOldFiles 清理目录中的旧文件（.old_* 和 .bak）
func cleanOldFiles(dir string) {
	patterns := []string{
		filepath.Join(dir, "*.old_*"),
		filepath.Join(dir, "*.bak"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}
