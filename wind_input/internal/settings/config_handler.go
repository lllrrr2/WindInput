package settings

import (
	"encoding/json"
	"net/http"

	"github.com/huanfeng/wind_input/internal/config"
)

// ConfigHandler 配置处理器
type ConfigHandler struct {
	services *Services
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(services *Services) *ConfigHandler {
	return &ConfigHandler{
		services: services,
	}
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	General    config.GeneralConfig    `json:"general"`
	Dictionary config.DictionaryConfig `json:"dictionary"`
	Engine     config.EngineConfig     `json:"engine"`
	Hotkeys    config.HotkeyConfig     `json:"hotkeys"`
	UI         config.UIConfig         `json:"ui"`
	Toolbar    config.ToolbarConfig    `json:"toolbar"`
	Input      config.InputConfig      `json:"input"`
}

// GetConfig 获取完整配置
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if h.services.Config == nil {
		WriteError(w, http.StatusInternalServerError, "配置未初始化")
		return
	}

	WriteSuccess(w, ConfigResponse{
		General:    h.services.Config.General,
		Dictionary: h.services.Config.Dictionary,
		Engine:     h.services.Config.Engine,
		Hotkeys:    h.services.Config.Hotkeys,
		UI:         h.services.Config.UI,
		Toolbar:    h.services.Config.Toolbar,
		Input:      h.services.Config.Input,
	})
}

// ConfigUpdateRequest 配置更新请求
type ConfigUpdateRequest struct {
	General    *config.GeneralConfig    `json:"general,omitempty"`
	Dictionary *config.DictionaryConfig `json:"dictionary,omitempty"`
	Engine     *config.EngineConfig     `json:"engine,omitempty"`
	Hotkeys    *config.HotkeyConfig     `json:"hotkeys,omitempty"`
	UI         *config.UIConfig         `json:"ui,omitempty"`
	Toolbar    *config.ToolbarConfig    `json:"toolbar,omitempty"`
	Input      *config.InputConfig      `json:"input,omitempty"`
}

// ConfigUpdateResponse 配置更新响应
type ConfigUpdateResponse struct {
	Applied     []string `json:"applied"`     // 已立即生效的字段
	NeedReload  []string `json:"needReload"`  // 需要重载的模块
	NeedRestart bool     `json:"needRestart"` // 是否需要重启
}

// UpdateConfig 更新配置（PATCH - 部分更新）
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.services.Config == nil {
		WriteError(w, http.StatusInternalServerError, "配置未初始化")
		return
	}

	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	response := ConfigUpdateResponse{
		Applied:    make([]string, 0),
		NeedReload: make([]string, 0),
	}

	cfg := h.services.Config

	// 更新 General 配置
	if req.General != nil {
		cfg.General = *req.General
		response.Applied = append(response.Applied, "general")
	}

	// 更新 Dictionary 配置
	if req.Dictionary != nil {
		cfg.Dictionary = *req.Dictionary
		response.NeedReload = append(response.NeedReload, "dictionary")
	}

	// 更新 Engine 配置
	if req.Engine != nil {
		oldType := cfg.Engine.Type
		oldFilterMode := cfg.Engine.FilterMode
		cfg.Engine = *req.Engine

		// 如果引擎类型改变，需要重载
		if cfg.Engine.Type != oldType {
			response.NeedReload = append(response.NeedReload, "engine")
		} else {
			response.Applied = append(response.Applied, "engine")
		}

		// 如果过滤模式改变，立即更新引擎
		if cfg.Engine.FilterMode != oldFilterMode && h.services.EngineMgr != nil {
			h.services.EngineMgr.UpdateFilterMode(cfg.Engine.FilterMode)
			h.services.Logger.Info("Filter mode updated", "mode", cfg.Engine.FilterMode)
		}
	}

	// 更新 Hotkeys 配置
	if req.Hotkeys != nil {
		cfg.Hotkeys = *req.Hotkeys
		response.Applied = append(response.Applied, "hotkeys")
	}

	// 更新 UI 配置（可热更新）
	if req.UI != nil {
		cfg.UI = *req.UI
		response.Applied = append(response.Applied, "ui")

		// 通知 Coordinator 更新 UI 配置
		if h.services.Coordinator != nil {
			h.services.Coordinator.UpdateUIConfig(&cfg.UI)
		}
	}

	// 更新 Toolbar 配置（可热更新）
	if req.Toolbar != nil {
		cfg.Toolbar = *req.Toolbar
		response.Applied = append(response.Applied, "toolbar")

		// 通知 Coordinator 更新工具栏配置
		if h.services.Coordinator != nil {
			h.services.Coordinator.UpdateToolbarConfig(&cfg.Toolbar)
		}
	}

	// 更新 Input 配置（可热更新）
	if req.Input != nil {
		cfg.Input = *req.Input
		response.Applied = append(response.Applied, "input")

		// 通知 Coordinator 更新输入配置
		if h.services.Coordinator != nil {
			h.services.Coordinator.UpdateInputConfig(&cfg.Input)
		}
	}

	// 保存配置到文件
	if h.services.OnConfigSave != nil {
		if err := h.services.OnConfigSave(cfg); err != nil {
			WriteError(w, http.StatusInternalServerError, "保存配置失败: "+err.Error())
			return
		}
	}

	WriteSuccess(w, response)
}

// FieldMeta 字段元数据
type FieldMeta struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	UpdateMode  string `json:"updateMode"` // hot, reload, restart
	Default     any    `json:"default,omitempty"`
}

// ConfigMetaResponse 配置元数据响应
type ConfigMetaResponse struct {
	Fields map[string][]FieldMeta `json:"fields"`
}

// GetConfigMeta 获取配置元数据
func (h *ConfigHandler) GetConfigMeta(w http.ResponseWriter, r *http.Request) {
	meta := ConfigMetaResponse{
		Fields: map[string][]FieldMeta{
			"general": {
				{Name: "start_in_chinese_mode", Type: "bool", Description: "启动时默认中文模式", UpdateMode: "hot", Default: true},
				{Name: "log_level", Type: "string", Description: "日志级别 (debug/info/warn/error)", UpdateMode: "restart", Default: "info"},
			},
			"dictionary": {
				{Name: "system_dict", Type: "string", Description: "系统词库路径", UpdateMode: "reload"},
				{Name: "user_dict", Type: "string", Description: "用户词库路径", UpdateMode: "reload"},
				{Name: "pinyin_dict", Type: "string", Description: "拼音词库路径（用于反查）", UpdateMode: "reload"},
			},
			"engine": {
				{Name: "type", Type: "string", Description: "引擎类型 (pinyin/wubi)", UpdateMode: "reload", Default: "pinyin"},
				{Name: "filter_mode", Type: "string", Description: "字符过滤模式 (general/gb18030/smart)", UpdateMode: "reload", Default: "smart"},
			},
			"engine.pinyin": {
				{Name: "show_wubi_hint", Type: "bool", Description: "显示五笔编码提示", UpdateMode: "hot", Default: true},
			},
			"engine.wubi": {
				{Name: "auto_commit", Type: "string", Description: "自动上屏模式", UpdateMode: "reload", Default: "unique_at_4"},
				{Name: "empty_code", Type: "string", Description: "空码处理模式", UpdateMode: "reload", Default: "clear_at_4"},
				{Name: "top_code_commit", Type: "bool", Description: "五码顶字上屏", UpdateMode: "reload", Default: true},
				{Name: "punct_commit", Type: "bool", Description: "标点顶字上屏", UpdateMode: "reload", Default: true},
			},
			"hotkeys": {
				{Name: "toggle_mode", Type: "string", Description: "切换中英文模式热键", UpdateMode: "hot", Default: "shift"},
				{Name: "switch_engine", Type: "string", Description: "切换引擎热键", UpdateMode: "hot", Default: "ctrl+`"},
			},
			"ui": {
				{Name: "font_size", Type: "float64", Description: "字体大小", UpdateMode: "hot", Default: 18.0},
				{Name: "candidates_per_page", Type: "int", Description: "每页候选数", UpdateMode: "hot", Default: 9},
				{Name: "font_path", Type: "string", Description: "自定义字体路径", UpdateMode: "hot"},
			},
		},
	}

	WriteSuccess(w, meta)
}

// ValidateConfigRequest 配置验证请求
type ValidateConfigRequest struct {
	Config ConfigUpdateRequest `json:"config"`
}

// ValidateConfigResponse 配置验证响应
type ValidateConfigResponse struct {
	Valid    bool              `json:"valid"`
	Errors   map[string]string `json:"errors,omitempty"`
	Warnings map[string]string `json:"warnings,omitempty"`
}

// ValidateConfig 验证配置
func (h *ConfigHandler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	var req ValidateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	response := ValidateConfigResponse{
		Valid:    true,
		Errors:   make(map[string]string),
		Warnings: make(map[string]string),
	}

	// 验证引擎类型
	if req.Config.Engine != nil {
		switch req.Config.Engine.Type {
		case "pinyin", "wubi":
			// 有效
		default:
			response.Valid = false
			response.Errors["engine.type"] = "无效的引擎类型，必须是 pinyin 或 wubi"
		}

		switch req.Config.Engine.FilterMode {
		case "", "general", "gb18030", "smart":
			// 有效
		default:
			response.Valid = false
			response.Errors["engine.filter_mode"] = "无效的过滤模式，必须是 general / gb18030 / smart"
		}
	}

	// 验证 UI 配置
	if req.Config.UI != nil {
		if req.Config.UI.FontSize < 10 || req.Config.UI.FontSize > 72 {
			response.Warnings["ui.font_size"] = "字体大小建议在 10-72 之间"
		}
		if req.Config.UI.CandidatesPerPage < 1 || req.Config.UI.CandidatesPerPage > 10 {
			response.Valid = false
			response.Errors["ui.candidates_per_page"] = "每页候选数必须在 1-10 之间"
		}
	}

	// 验证五笔配置
	if req.Config.Engine != nil {
		switch req.Config.Engine.Wubi.AutoCommit {
		case "", "none", "unique", "unique_at_4", "unique_full_match":
			// 有效
		default:
			response.Valid = false
			response.Errors["engine.wubi.auto_commit"] = "无效的自动上屏模式"
		}

		switch req.Config.Engine.Wubi.EmptyCode {
		case "", "none", "clear", "clear_at_4", "to_english":
			// 有效
		default:
			response.Valid = false
			response.Errors["engine.wubi.empty_code"] = "无效的空码处理模式"
		}
	}

	WriteSuccess(w, response)
}

// ReloadConfigResponse 重载配置响应
type ReloadConfigResponse struct {
	Reloaded []string `json:"reloaded"`
	Errors   []string `json:"errors,omitempty"`
}

// ReloadConfig 重载配置（词库、引擎等）
func (h *ConfigHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	response := ReloadConfigResponse{
		Reloaded: make([]string, 0),
		Errors:   make([]string, 0),
	}

	// 重新加载配置文件
	newCfg, err := config.Load()
	if err != nil {
		response.Errors = append(response.Errors, "加载配置文件失败: "+err.Error())
	} else {
		*h.services.Config = *newCfg
		response.Reloaded = append(response.Reloaded, "config")
	}

	// TODO: 重载词库
	// TODO: 重载引擎配置

	if len(response.Errors) > 0 {
		WriteJSON(w, http.StatusOK, APIResponse{
			Success: false,
			Data:    response,
			Error:   "部分重载失败",
		})
		return
	}

	WriteSuccess(w, response)
}
