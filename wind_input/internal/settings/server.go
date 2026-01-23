// Package settings 提供设置管理 HTTP 服务
package settings

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	// DefaultAddr 默认监听地址（仅本地访问）
	DefaultAddr = "127.0.0.1:18923"
)

// Server 设置服务器
type Server struct {
	addr       string
	httpServer *http.Server
	router     *Router
	logger     *slog.Logger
}

// NewServer 创建设置服务器
func NewServer(logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		addr:   DefaultAddr,
		logger: logger,
	}

	// 创建路由
	s.router = NewRouter(logger)

	return s
}

// SetAddr 设置监听地址
func (s *Server) SetAddr(addr string) {
	s.addr = addr
}

// RegisterServices 注册服务处理器
func (s *Server) RegisterServices(services *Services) {
	s.router.RegisterServices(services)
}

// GetLogHandler 获取日志处理器（供外部创建 slog handler）
func (s *Server) GetLogHandler() *LogHandler {
	return s.router.GetLogHandler()
}

// Start 启动服务器（阻塞）
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 注册路由
	s.router.SetupRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      s.corsMiddleware(s.loggingMiddleware(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Settings server starting", "addr", s.addr)
	return s.httpServer.ListenAndServe()
}

// StartAsync 异步启动服务器
func (s *Server) StartAsync() {
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Settings server error", "error", err)
		}
	}()
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.logger.Info("Settings server stopping...")
	return s.httpServer.Shutdown(ctx)
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		s.logger.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start),
		)
	})
}

// corsMiddleware CORS 中间件（允许本地 Wails 应用访问）
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许本地访问
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter 包装 ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// APIResponse 统一 API 响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccess 写入成功响应
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}
