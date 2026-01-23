package settings

import (
	"encoding/json"
	"net/http"

	"github.com/huanfeng/wind_input/internal/candidate"
)

// TestHandler 测试处理器
type TestHandler struct {
	services *Services
}

// NewTestHandler 创建测试处理器
func NewTestHandler(services *Services) *TestHandler {
	return &TestHandler{
		services: services,
	}
}

// TestConvertRequest 测试转换请求
type TestConvertRequest struct {
	Input      string `json:"input"`
	Engine     string `json:"engine"`     // current / pinyin / wubi
	FilterMode string `json:"filterMode"` // current / smart / general / gb18030
}

// TestCandidate 测试候选词
type TestCandidate struct {
	Text     string `json:"text"`
	Code     string `json:"code,omitempty"`
	IsCommon bool   `json:"isCommon"`
	Weight   int    `json:"weight"`
}

// TestConvertResponse 测试转换响应
type TestConvertResponse struct {
	Candidates []TestCandidate `json:"candidates"`
	Engine     string          `json:"engine"`
	FilterMode string          `json:"filterMode"`
}

// TestConvert 测试转换
func (h *TestHandler) TestConvert(w http.ResponseWriter, r *http.Request) {
	var req TestConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	if req.Input == "" {
		WriteSuccess(w, TestConvertResponse{
			Candidates: []TestCandidate{},
			Engine:     req.Engine,
			FilterMode: req.FilterMode,
		})
		return
	}

	if h.services.EngineMgr == nil {
		WriteError(w, http.StatusInternalServerError, "引擎管理器未初始化")
		return
	}

	// 获取候选词（使用 ConvertRaw 获取未过滤的原始候选词）
	candidates, err := h.services.EngineMgr.ConvertRaw(req.Input, 100)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "转换失败: "+err.Error())
		return
	}

	// 应用请求指定的过滤模式
	filterMode := req.FilterMode
	if filterMode == "current" && h.services.Config != nil {
		filterMode = h.services.Config.Engine.FilterMode
	}
	// 应用过滤（gb18030 模式不过滤）
	if filterMode != "" && filterMode != "gb18030" {
		candidates = candidate.FilterCandidates(candidates, filterMode)
	}

	// 限制返回数量
	if len(candidates) > 50 {
		candidates = candidates[:50]
	}

	// 转换为响应格式
	testCandidates := make([]TestCandidate, len(candidates))
	for i, c := range candidates {
		testCandidates[i] = TestCandidate{
			Text:     c.Text,
			Code:     c.Code,
			IsCommon: c.IsCommon,
			Weight:   c.Weight,
		}
	}

	// 获取当前引擎类型
	engineType := "unknown"
	if h.services.EngineMgr != nil {
		engineType = string(h.services.EngineMgr.GetCurrentType())
	}

	WriteSuccess(w, TestConvertResponse{
		Candidates: testCandidates,
		Engine:     engineType,
		FilterMode: filterMode,
	})
}
