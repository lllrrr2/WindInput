package settings

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// LogHandler 日志处理器
type LogHandler struct {
	services *Services
	logs     []LogEntry
	mu       sync.RWMutex
	maxLogs  int
}

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// NewLogHandler 创建日志处理器
func NewLogHandler(services *Services) *LogHandler {
	return &LogHandler{
		services: services,
		logs:     make([]LogEntry, 0),
		maxLogs:  500,
	}
}

// SlogHandler 实现 slog.Handler 接口，用于捕获日志到 LogHandler
type SlogHandler struct {
	logHandler *LogHandler
	next       slog.Handler // 下一个 handler（用于链式处理）
	level      slog.Level
	attrs      []slog.Attr
	groups     []string
}

// NewSlogHandler 创建一个新的 slog Handler
func NewSlogHandler(logHandler *LogHandler, next slog.Handler, level slog.Level) *SlogHandler {
	return &SlogHandler{
		logHandler: logHandler,
		next:       next,
		level:      level,
	}
}

// Enabled 实现 slog.Handler
func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle 实现 slog.Handler
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// 构建消息
	msg := r.Message
	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	// 添加到 LogHandler
	if h.logHandler != nil {
		h.logHandler.AddLog(r.Level.String(), msg)
	}

	// 传递给下一个 handler
	if h.next != nil {
		return h.next.Handle(ctx, r)
	}
	return nil
}

// WithAttrs 实现 slog.Handler
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &SlogHandler{
		logHandler: h.logHandler,
		next:       h.next,
		level:      h.level,
		attrs:      append(h.attrs, attrs...),
		groups:     h.groups,
	}
	if h.next != nil {
		newHandler.next = h.next.WithAttrs(attrs)
	}
	return newHandler
}

// WithGroup 实现 slog.Handler
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	newHandler := &SlogHandler{
		logHandler: h.logHandler,
		next:       h.next,
		level:      h.level,
		attrs:      h.attrs,
		groups:     append(h.groups, name),
	}
	if h.next != nil {
		newHandler.next = h.next.WithGroup(name)
	}
	return newHandler
}

// AddLog 添加日志（供外部调用）
func (h *LogHandler) AddLog(level, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Message: message,
	}

	h.logs = append(h.logs, entry)

	// 限制日志数量
	if len(h.logs) > h.maxLogs {
		h.logs = h.logs[len(h.logs)-h.maxLogs:]
	}
}

// GetLogs 获取日志
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 获取查询参数
	level := r.URL.Query().Get("level")
	filter := r.URL.Query().Get("filter")

	// 过滤日志
	var filteredLogs []LogEntry
	for _, log := range h.logs {
		// 级别过滤
		if level != "" && level != "all" && log.Level != level {
			continue
		}
		// 文本过滤
		if filter != "" {
			if !containsIgnoreCase(log.Message, filter) {
				continue
			}
		}
		filteredLogs = append(filteredLogs, log)
	}

	WriteSuccess(w, map[string]interface{}{
		"logs":  filteredLogs,
		"total": len(h.logs),
	})
}

// ClearLogs 清空日志
func (h *LogHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.logs = make([]LogEntry, 0)
	h.mu.Unlock()

	WriteSuccess(w, map[string]string{"status": "cleared"})
}

// containsIgnoreCase 忽略大小写的包含检查
func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
