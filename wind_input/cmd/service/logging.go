// logging.go — 日志轮转与多路处理器
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// rotatingWriter 实现日志文件轮转的 io.Writer
type rotatingWriter struct {
	mu       sync.Mutex
	file     *os.File
	filePath string
	maxSize  int64 // 单文件最大字节数
	maxFiles int   // 最大备份文件数
	curSize  int64 // 当前文件大小
}

func newRotatingWriter(filePath string, maxSize int64, maxFiles int) (*rotatingWriter, error) {
	w := &rotatingWriter{
		filePath: filePath,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}

	// 检查已有文件大小，若超限则先轮转
	if info, err := os.Stat(filePath); err == nil {
		if info.Size() >= maxSize {
			w.rotateFiles()
		}
	}

	// 打开或创建文件
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	// 获取当前文件大小
	if info, err := f.Stat(); err == nil {
		w.curSize = info.Size()
	}

	w.file = f
	return w, nil
}

func (w *rotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err = w.file.Write(p)
	if err != nil {
		return n, err
	}
	w.curSize += int64(n)

	if w.curSize >= w.maxSize {
		w.rotate()
	}
	return n, nil
}

func (w *rotatingWriter) rotate() {
	w.file.Close()
	w.rotateFiles()

	f, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 无法创建新文件，尝试回退
		return
	}
	w.file = f
	w.curSize = 0
}

// rotateFiles 移动备份链：3→删除, 2→3, 1→2, current→1
func (w *rotatingWriter) rotateFiles() {
	ext := filepath.Ext(w.filePath)
	base := w.filePath[:len(w.filePath)-len(ext)]

	// 删除最旧的备份
	oldest := fmt.Sprintf("%s.%d%s", base, w.maxFiles, ext)
	os.Remove(oldest)

	// 依次重命名：N-1→N, ..., 1→2
	for i := w.maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d%s", base, i, ext)
		dst := fmt.Sprintf("%s.%d%s", base, i+1, ext)
		os.Rename(src, dst)
	}

	// 当前文件→.1
	first := fmt.Sprintf("%s.%d%s", base, 1, ext)
	os.Rename(w.filePath, first)
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// multiHandler wraps multiple slog handlers
type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			handler.Handle(ctx, r)
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return newMultiHandler(newHandlers...)
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(newHandlers...)
}
