package settings

import (
	"encoding/json"
	"net/http"

	"github.com/huanfeng/wind_input/internal/engine"
)

// EngineHandler 引擎处理器
type EngineHandler struct {
	services *Services
}

// NewEngineHandler 创建引擎处理器
func NewEngineHandler(services *Services) *EngineHandler {
	return &EngineHandler{
		services: services,
	}
}

// EngineInfo 引擎信息
type EngineInfo struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Info        string `json:"info"`
	IsActive    bool   `json:"isActive"`
}

// GetEngineResponse 获取引擎响应
type GetEngineResponse struct {
	Current EngineInfo `json:"current"`
}

// GetEngine 获取当前引擎信息
func (h *EngineHandler) GetEngine(w http.ResponseWriter, r *http.Request) {
	if h.services.EngineMgr == nil {
		WriteError(w, http.StatusInternalServerError, "引擎管理器未初始化")
		return
	}

	engineType := h.services.EngineMgr.GetCurrentType()

	response := GetEngineResponse{
		Current: EngineInfo{
			Type:        string(engineType),
			DisplayName: h.services.EngineMgr.GetEngineDisplayName(),
			Description: getEngineDescription(engineType),
			Info:        h.services.EngineMgr.GetEngineInfo(),
			IsActive:    true,
		},
	}

	WriteSuccess(w, response)
}

// ListEnginesResponse 引擎列表响应
type ListEnginesResponse struct {
	Engines []EngineInfo `json:"engines"`
	Current string       `json:"current"`
}

// ListEngines 列出所有可用引擎
func (h *EngineHandler) ListEngines(w http.ResponseWriter, r *http.Request) {
	currentType := ""
	if h.services.EngineMgr != nil {
		currentType = string(h.services.EngineMgr.GetCurrentType())
	}

	engines := []EngineInfo{
		{
			Type:        "pinyin",
			DisplayName: "拼",
			Description: "拼音输入法 - 支持全拼输入，可显示五笔编码提示",
			IsActive:    currentType == "pinyin",
		},
		{
			Type:        "wubi",
			DisplayName: "五",
			Description: "五笔输入法 - 86版五笔，支持自动上屏、顶码等功能",
			IsActive:    currentType == "wubi",
		},
	}

	response := ListEnginesResponse{
		Engines: engines,
		Current: currentType,
	}

	WriteSuccess(w, response)
}

// SwitchEngineRequest 切换引擎请求
type SwitchEngineRequest struct {
	Type string `json:"type"` // pinyin 或 wubi
}

// SwitchEngineResponse 切换引擎响应
type SwitchEngineResponse struct {
	Previous    string `json:"previous"`
	Current     string `json:"current"`
	DisplayName string `json:"displayName"`
}

// SwitchEngine 切换引擎
func (h *EngineHandler) SwitchEngine(w http.ResponseWriter, r *http.Request) {
	if h.services.EngineMgr == nil {
		WriteError(w, http.StatusInternalServerError, "引擎管理器未初始化")
		return
	}

	var req SwitchEngineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	// 验证引擎类型
	var targetType engine.EngineType
	switch req.Type {
	case "pinyin":
		targetType = engine.EngineTypePinyin
	case "wubi":
		targetType = engine.EngineTypeWubi
	case "":
		// 如果没有指定类型，则切换（toggle）
		newType, err := h.services.EngineMgr.ToggleEngine()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "切换引擎失败: "+err.Error())
			return
		}

		// 更新配置文件
		if h.services.Config != nil {
			h.services.Config.Engine.Type = string(newType)
			if h.services.OnConfigSave != nil {
				h.services.OnConfigSave(h.services.Config)
			}
		}

		// 清空输入状态
		if h.services.Coordinator != nil {
			h.services.Coordinator.ClearInputState()
		}

		WriteSuccess(w, SwitchEngineResponse{
			Previous:    "", // toggle 模式不返回 previous
			Current:     string(newType),
			DisplayName: h.services.EngineMgr.GetEngineDisplayName(),
		})
		return
	default:
		WriteError(w, http.StatusBadRequest, "无效的引擎类型，必须是 pinyin 或 wubi")
		return
	}

	// 获取当前引擎类型
	previousType := h.services.EngineMgr.GetCurrentType()

	// 如果已经是目标引擎，直接返回
	if previousType == targetType {
		WriteSuccess(w, SwitchEngineResponse{
			Previous:    string(previousType),
			Current:     string(targetType),
			DisplayName: h.services.EngineMgr.GetEngineDisplayName(),
		})
		return
	}

	// 切换引擎
	if err := h.services.EngineMgr.SwitchEngine(targetType); err != nil {
		WriteError(w, http.StatusInternalServerError, "切换引擎失败: "+err.Error())
		return
	}

	// 更新配置文件
	if h.services.Config != nil {
		h.services.Config.Engine.Type = string(targetType)
		if h.services.OnConfigSave != nil {
			h.services.OnConfigSave(h.services.Config)
		}
	}

	// 清空输入状态
	if h.services.Coordinator != nil {
		h.services.Coordinator.ClearInputState()
	}

	WriteSuccess(w, SwitchEngineResponse{
		Previous:    string(previousType),
		Current:     string(targetType),
		DisplayName: h.services.EngineMgr.GetEngineDisplayName(),
	})
}

// getEngineDescription 获取引擎描述
func getEngineDescription(engineType engine.EngineType) string {
	switch engineType {
	case engine.EngineTypePinyin:
		return "拼音输入法 - 支持全拼输入，可显示五笔编码提示"
	case engine.EngineTypeWubi:
		return "五笔输入法 - 86版五笔，支持自动上屏、顶码等功能"
	default:
		return "未知引擎"
	}
}
