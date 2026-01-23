package settings

import (
	"log/slog"
	"net/http"

	"github.com/huanfeng/wind_input/internal/config"
	"github.com/huanfeng/wind_input/internal/coordinator"
	"github.com/huanfeng/wind_input/internal/engine"
)

// Services 包含所有需要的服务依赖
type Services struct {
	Config      *config.Config
	EngineMgr   *engine.Manager
	Coordinator *coordinator.Coordinator
	Logger      *slog.Logger

	// 配置保存回调
	OnConfigSave func(*config.Config) error
}

// Router 路由管理器
type Router struct {
	logger   *slog.Logger
	services *Services

	// 处理器
	configHandler *ConfigHandler
	statusHandler *StatusHandler
	engineHandler *EngineHandler
	testHandler   *TestHandler
	logHandler    *LogHandler
}

// NewRouter 创建路由管理器
func NewRouter(logger *slog.Logger) *Router {
	r := &Router{
		logger: logger,
	}
	// 提前创建 LogHandler，使其可以在 RegisterServices 之前使用
	r.logHandler = NewLogHandler(nil)
	return r
}

// RegisterServices 注册服务
func (r *Router) RegisterServices(services *Services) {
	r.services = services

	// 更新 logHandler 的 services
	r.logHandler.services = services

	// 创建其他处理器
	r.configHandler = NewConfigHandler(services)
	r.statusHandler = NewStatusHandler(services)
	r.engineHandler = NewEngineHandler(services)
	r.testHandler = NewTestHandler(services)
}

// GetLogHandler 获取日志处理器（供外部添加日志）
func (r *Router) GetLogHandler() *LogHandler {
	return r.logHandler
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes(mux *http.ServeMux) {
	// 健康检查
	mux.HandleFunc("GET /api/health", r.handleHealth)

	// 配置相关
	mux.HandleFunc("GET /api/config", r.configHandler.GetConfig)
	mux.HandleFunc("PATCH /api/config", r.configHandler.UpdateConfig)
	mux.HandleFunc("GET /api/config/meta", r.configHandler.GetConfigMeta)
	mux.HandleFunc("POST /api/config/validate", r.configHandler.ValidateConfig)
	mux.HandleFunc("POST /api/config/reload", r.configHandler.ReloadConfig)

	// 状态相关
	mux.HandleFunc("GET /api/status", r.statusHandler.GetStatus)

	// 引擎相关
	mux.HandleFunc("GET /api/engine", r.engineHandler.GetEngine)
	mux.HandleFunc("POST /api/engine/switch", r.engineHandler.SwitchEngine)
	mux.HandleFunc("GET /api/engine/list", r.engineHandler.ListEngines)

	// 测试相关
	mux.HandleFunc("POST /api/test/convert", r.testHandler.TestConvert)

	// 日志相关
	mux.HandleFunc("GET /api/logs", r.logHandler.GetLogs)
	mux.HandleFunc("DELETE /api/logs", r.logHandler.ClearLogs)
}

// handleHealth 健康检查
func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	WriteSuccess(w, map[string]string{
		"status":  "ok",
		"service": "wind_input_settings",
	})
}
