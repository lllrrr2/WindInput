// Package coordinator orchestrates communication between C++ Bridge, Engine, and UI
package coordinator

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/huanfeng/wind_input/internal/bridge"
	"github.com/huanfeng/wind_input/internal/config"
	"github.com/huanfeng/wind_input/internal/engine"
	"github.com/huanfeng/wind_input/internal/ui"
)

// Modifier key flags (must match C++ side)
const (
	ModShift = 0x01
	ModCtrl  = 0x02
	ModAlt   = 0x04
)

// Coordinator orchestrates between C++ Bridge, Engine, and native UI
type Coordinator struct {
	engineMgr *engine.Manager
	uiManager *ui.Manager
	logger    *slog.Logger
	config    *config.Config

	mu sync.Mutex

	// Input mode state
	chineseMode bool // true = Chinese, false = English

	// Input state
	inputBuffer       string
	candidates        []ui.Candidate
	currentPage       int
	totalPages        int
	candidatesPerPage int

	// Caret position (from C++)
	caretX      int
	caretY      int
	caretHeight int

	// Last known valid window position (for fallback)
	lastValidX int
	lastValidY int
}

// NewCoordinator creates a new Coordinator
func NewCoordinator(engineMgr *engine.Manager, uiManager *ui.Manager, cfg *config.Config, logger *slog.Logger) *Coordinator {
	candidatesPerPage := 9
	if cfg != nil && cfg.UI.CandidatesPerPage > 0 {
		candidatesPerPage = cfg.UI.CandidatesPerPage
	}

	startInChineseMode := true
	if cfg != nil {
		startInChineseMode = cfg.General.StartInChineseMode
	}

	return &Coordinator{
		engineMgr:         engineMgr,
		uiManager:         uiManager,
		logger:            logger,
		config:            cfg,
		chineseMode:       startInChineseMode,
		inputBuffer:       "",
		candidates:        nil,
		currentPage:       1,
		totalPages:        1,
		candidatesPerPage: candidatesPerPage,
		caretX:            100,
		caretY:            100,
		caretHeight:       20,
	}
}

// HandleKeyEvent handles key events from C++ Bridge
// Returns a result indicating what action to take
func (c *Coordinator) HandleKeyEvent(data bridge.KeyEventData) *bridge.KeyEventResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use Debug for high-frequency key events to reduce log noise
	c.logger.Debug("HandleKeyEvent", "key", data.Key, "keycode", data.KeyCode, "modifiers", data.Modifiers, "chineseMode", c.chineseMode)

	// Check for Ctrl or Alt modifiers
	hasCtrl := data.Modifiers&ModCtrl != 0
	hasAlt := data.Modifiers&ModAlt != 0

	// Handle Ctrl+` for engine switch (VK_OEM_3 = 192)
	if hasCtrl && data.KeyCode == 192 {
		return c.handleEngineSwitchKey()
	}

	// Other Ctrl/Alt combinations should be passed to the system
	if hasCtrl || hasAlt {
		c.logger.Debug("Key has Ctrl/Alt modifier, passing to system")
		return nil
	}

	// Preserve original key for English mode (uppercase letters should stay uppercase)
	key := data.Key

	// Handle Shift key for mode toggle
	if data.KeyCode == 16 { // VK_SHIFT
		c.chineseMode = !c.chineseMode
		c.logger.Debug("Mode toggled by Shift", "chineseMode", c.chineseMode)

		// Clear any pending input when switching modes
		if len(c.inputBuffer) > 0 {
			c.clearState()
			c.hideUI()
		}

		// Show mode indicator
		c.showModeIndicator()

		// Return mode_changed so C++ can update language bar icon
		return &bridge.KeyEventResult{
			Type:        bridge.ResponseTypeModeChanged,
			ChineseMode: c.chineseMode,
		}
	}

	// English mode: pass through all keys
	if !c.chineseMode {
		// In English mode, letters should be passed through directly (both upper and lower case)
		if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')) {
			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: key,
			}
		}
		return nil // Let other keys pass through
	}

	// Chinese mode handling
	switch {
	case data.KeyCode == 8: // Backspace
		return c.handleBackspace()

	case data.KeyCode == 13: // Enter
		return c.handleEnter()

	case data.KeyCode == 27: // Escape
		return c.handleEscape()

	case data.KeyCode == 32: // Space
		return c.handleSpace()

	case key == "page_up" || data.KeyCode == 189: // - key (VK_OEM_MINUS = 0xBD = 189)
		return c.handlePageUp()

	case key == "page_down" || data.KeyCode == 187: // = key (VK_OEM_PLUS = 0xBB = 187)
		return c.handlePageDown()

	case len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')):
		// Chinese mode: convert to lowercase for pinyin
		return c.handleAlphaKey(strings.ToLower(key))

	case len(key) == 1 && key[0] >= '1' && key[0] <= '9':
		return c.handleNumberKey(int(key[0] - '0'))

	default:
		c.logger.Debug("Unhandled key", "key", key, "keycode", data.KeyCode)
		return nil
	}
}

func (c *Coordinator) handleAlphaKey(key string) *bridge.KeyEventResult {
	c.inputBuffer += key
	c.logger.Debug("Input buffer updated", "buffer", c.inputBuffer)

	// 处理顶码（如五笔的五码顶字）
	if c.engineMgr != nil {
		commitText, newInput, shouldCommit := c.engineMgr.HandleTopCode(c.inputBuffer)
		if shouldCommit {
			c.inputBuffer = newInput
			c.logger.Debug("Top code commit", "text", commitText, "newInput", newInput)

			// 如果还有剩余输入，继续处理
			if len(c.inputBuffer) > 0 {
				c.updateCandidates()
				c.showUI()
			} else {
				c.hideUI()
			}

			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: commitText,
			}
		}
	}

	// 更新候选词
	result := c.updateCandidatesEx()

	// 检查自动上屏
	if result != nil && result.ShouldCommit {
		c.clearState()
		c.hideUI()
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: result.CommitText,
		}
	}

	// 检查空码处理
	if result != nil && result.IsEmpty {
		if result.ShouldClear {
			c.clearState()
			c.hideUI()
			return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
		}
		if result.ToEnglish {
			text := c.inputBuffer
			c.clearState()
			c.hideUI()
			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: text,
			}
		}
	}

	c.showUI()
	return nil // Just show candidates, don't insert anything yet
}

func (c *Coordinator) handleBackspace() *bridge.KeyEventResult {
	if len(c.inputBuffer) > 0 {
		c.inputBuffer = c.inputBuffer[:len(c.inputBuffer)-1]
		c.logger.Debug("Input buffer after backspace", "buffer", c.inputBuffer)

		if len(c.inputBuffer) == 0 {
			c.clearState()
			c.hideUI()
			return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
		}

		c.updateCandidates()
		c.showUI()
	} else {
		// Buffer is already empty - this shouldn't happen normally
		// Return ClearComposition to reset C++ side's _isComposing state
		c.logger.Debug("Backspace with empty buffer, clearing composition state")
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
	}
	return nil
}

func (c *Coordinator) handleEnter() *bridge.KeyEventResult {
	// Commit raw pinyin as text
	if len(c.inputBuffer) > 0 {
		text := c.inputBuffer
		c.clearState()
		c.hideUI()

		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
		}
	}
	return nil
}

func (c *Coordinator) handleEscape() *bridge.KeyEventResult {
	if len(c.inputBuffer) > 0 {
		c.clearState()
		c.hideUI()
	}
	// Always return ClearComposition to ensure C++ side's _isComposing is reset
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
}

func (c *Coordinator) handleSpace() *bridge.KeyEventResult {
	// Select first candidate of current page
	if len(c.candidates) > 0 {
		// Calculate index of first candidate on current page
		index := (c.currentPage - 1) * c.candidatesPerPage
		if index < len(c.candidates) {
			return c.selectCandidate(index)
		}
	} else if len(c.inputBuffer) > 0 {
		// No candidates, commit raw input
		text := c.inputBuffer
		c.clearState()
		c.hideUI()
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
		}
	}
	return nil
}

func (c *Coordinator) handleNumberKey(num int) *bridge.KeyEventResult {
	// num is 1-9, convert to 0-based index within current page
	index := (c.currentPage-1)*c.candidatesPerPage + (num - 1)
	if index < len(c.candidates) {
		return c.selectCandidate(index)
	}
	return nil
}

func (c *Coordinator) handlePageUp() *bridge.KeyEventResult {
	// Only handle if we have candidates and are not on the first page
	if len(c.candidates) == 0 || c.currentPage <= 1 {
		return nil
	}

	c.currentPage--
	c.logger.Debug("Page up", "currentPage", c.currentPage, "totalPages", c.totalPages)
	c.showUI()
	return nil
}

func (c *Coordinator) handlePageDown() *bridge.KeyEventResult {
	// Only handle if we have candidates and are not on the last page
	if len(c.candidates) == 0 || c.currentPage >= c.totalPages {
		return nil
	}

	c.currentPage++
	c.logger.Debug("Page down", "currentPage", c.currentPage, "totalPages", c.totalPages)
	c.showUI()
	return nil
}

func (c *Coordinator) selectCandidate(index int) *bridge.KeyEventResult {
	if index < 0 || index >= len(c.candidates) {
		return nil
	}

	candidate := c.candidates[index]
	c.logger.Debug("Candidate selected", "index", index, "text", candidate.Text)

	text := candidate.Text
	c.clearState()
	c.hideUI()

	return &bridge.KeyEventResult{
		Type: bridge.ResponseTypeInsertText,
		Text: text,
	}
}

func (c *Coordinator) updateCandidates() {
	c.updateCandidatesEx()
}

func (c *Coordinator) updateCandidatesEx() *engine.ConvertResult {
	if len(c.inputBuffer) == 0 {
		c.candidates = nil
		return nil
	}

	if c.engineMgr == nil {
		return nil
	}

	// 使用扩展转换获取更多信息
	result := c.engineMgr.ConvertEx(c.inputBuffer, 50)

	// Convert to UI candidates
	c.candidates = make([]ui.Candidate, len(result.Candidates))
	for i, ec := range result.Candidates {
		cand := ui.Candidate{
			Text:   ec.Text,
			Index:  i + 1,
			Weight: ec.Weight,
		}
		// 如果有提示信息（如反查编码），添加到注释
		if ec.Hint != "" {
			cand.Comment = ec.Hint
		}
		c.candidates[i] = cand
	}

	c.logger.Debug("Got candidates", "count", len(c.candidates), "empty", result.IsEmpty)

	// Calculate pagination
	c.totalPages = (len(c.candidates) + c.candidatesPerPage - 1) / c.candidatesPerPage
	if c.totalPages == 0 {
		c.totalPages = 1
	}
	c.currentPage = 1

	return result
}

func (c *Coordinator) showUI() {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		c.logger.Warn("UI manager not ready")
		return
	}

	// Get current page candidates
	startIdx := (c.currentPage - 1) * c.candidatesPerPage
	endIdx := startIdx + c.candidatesPerPage
	if endIdx > len(c.candidates) {
		endIdx = len(c.candidates)
	}

	var pageCandidates []ui.Candidate
	if startIdx < len(c.candidates) {
		pageCandidates = c.candidates[startIdx:endIdx]
	}

	// Re-index for display (1-9)
	displayCandidates := make([]ui.Candidate, len(pageCandidates))
	for i, cand := range pageCandidates {
		displayCandidates[i] = ui.Candidate{
			Text:    cand.Text,
			Index:   i + 1,
			Comment: cand.Comment,
			Weight:  cand.Weight,
		}
	}

	// Calculate window position (below caret)
	windowX := c.caretX
	windowY := c.caretY + c.caretHeight + 5

	// Multi-monitor support: coordinates can be negative (monitors to the left/above primary)
	// Only use fallback if we haven't received valid caret info yet (both X and Y are 0)
	// or if coordinates are extremely large (likely garbage values)
	const maxCoord = 32000 // Windows virtual screen limit is typically around 32767
	if (c.caretX == 0 && c.caretY == 0) || windowX > maxCoord || windowX < -maxCoord || windowY > maxCoord || windowY < -maxCoord {
		// Use last known good position or a reasonable default
		if c.lastValidX != 0 || c.lastValidY != 0 {
			windowX = c.lastValidX
			windowY = c.lastValidY
		} else {
			// Fallback to a safe position on primary monitor
			windowX = 400
			windowY = 300
		}
		c.logger.Debug("Using fallback position", "x", windowX, "y", windowY, "caretX", c.caretX, "caretY", c.caretY)
	} else {
		// Save valid position for future fallback
		c.lastValidX = windowX
		c.lastValidY = windowY
	}

	c.uiManager.ShowCandidates(
		displayCandidates,
		c.inputBuffer,
		windowX,
		windowY,
		c.currentPage,
		c.totalPages,
	)
}

func (c *Coordinator) showModeIndicator() {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	// Show a brief indicator of the current mode
	modeText := "中"
	if !c.chineseMode {
		modeText = "En"
	}

	// Use valid position or fallback (multi-monitor: coordinates can be negative)
	x := c.caretX
	y := c.caretY + c.caretHeight + 5
	const maxCoord = 32000
	if (c.caretX == 0 && c.caretY == 0) || x > maxCoord || x < -maxCoord || y > maxCoord || y < -maxCoord {
		if c.lastValidX != 0 || c.lastValidY != 0 {
			x = c.lastValidX
			y = c.lastValidY
		} else {
			x = 400
			y = 300
		}
	}

	c.uiManager.ShowModeIndicator(modeText, x, y)
}

func (c *Coordinator) hideUI() {
	if c.uiManager != nil {
		c.uiManager.Hide()
	}
}

func (c *Coordinator) clearState() {
	c.inputBuffer = ""
	c.candidates = nil
	c.currentPage = 1
	c.totalPages = 1
}

// HandleCaretUpdate handles caret position updates from C++ Bridge
func (c *Coordinator) HandleCaretUpdate(data bridge.CaretData) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.caretX = data.X
	c.caretY = data.Y
	c.caretHeight = data.Height

	c.logger.Debug("Caret position updated", "x", c.caretX, "y", c.caretY, "height", c.caretHeight)
	return nil
}

// HandleFocusLost handles focus lost events
func (c *Coordinator) HandleFocusLost() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Focus lost, clearing state")
	c.clearState()
	c.hideUI()
}

// HandleToggleMode toggles the input mode and returns the new state
func (c *Coordinator) HandleToggleMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chineseMode = !c.chineseMode
	c.logger.Debug("Mode toggled via IPC", "chineseMode", c.chineseMode)

	// Clear any pending input when switching modes
	if len(c.inputBuffer) > 0 {
		c.clearState()
		c.hideUI()
	}

	// Show mode indicator
	c.showModeIndicator()

	return c.chineseMode
}

// HandleCapsLockState shows Caps Lock indicator (A/a)
func (c *Coordinator) HandleCapsLockState(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	// Show A for Caps Lock ON, a for OFF
	indicator := "a"
	if on {
		indicator = "A"
	}

	// Use valid position or fallback (multi-monitor: coordinates can be negative)
	x := c.caretX
	y := c.caretY + c.caretHeight + 5
	const maxCoord = 32000
	if (c.caretX == 0 && c.caretY == 0) || x > maxCoord || x < -maxCoord || y > maxCoord || y < -maxCoord {
		if c.lastValidX != 0 || c.lastValidY != 0 {
			x = c.lastValidX
			y = c.lastValidY
		} else {
			x = 400
			y = 300
		}
	}

	c.uiManager.ShowModeIndicator(indicator, x, y)
}

// handleEngineSwitchKey 处理引擎切换快捷键 (Ctrl+`)
func (c *Coordinator) handleEngineSwitchKey() *bridge.KeyEventResult {
	if c.engineMgr == nil {
		return nil
	}

	// 检查是否有输入需要清除
	hadInput := len(c.inputBuffer) > 0

	// 清除当前输入状态
	c.clearState()
	c.hideUI()

	// 切换引擎
	newType, err := c.engineMgr.ToggleEngine()
	if err != nil {
		c.logger.Error("Failed to switch engine", "error", err)
		return nil
	}

	c.logger.Info("Engine switched", "newType", newType)

	// 保存到用户配置
	go func() {
		if err := config.UpdateEngineType(string(newType)); err != nil {
			c.logger.Error("Failed to save engine type to config", "error", err)
		} else {
			c.logger.Debug("Engine type saved to config", "type", newType)
		}
	}()

	// 显示引擎指示器
	c.showEngineIndicator()

	// 返回 ClearComposition 让 C++ 端清除 _isComposing 状态
	if hadInput {
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
	}
	return nil
}

// showEngineIndicator 显示引擎切换指示器
func (c *Coordinator) showEngineIndicator() {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	// 获取引擎显示名称
	engineName := c.engineMgr.GetEngineDisplayName()

	// Use valid position or fallback
	x := c.caretX
	y := c.caretY + c.caretHeight + 5
	const maxCoord = 32000
	if (c.caretX == 0 && c.caretY == 0) || x > maxCoord || x < -maxCoord || y > maxCoord || y < -maxCoord {
		if c.lastValidX != 0 || c.lastValidY != 0 {
			x = c.lastValidX
			y = c.lastValidY
		} else {
			x = 400
			y = 300
		}
	}

	c.uiManager.ShowModeIndicator(engineName, x, y)
}

// GetCurrentEngineName 获取当前引擎名称
func (c *Coordinator) GetCurrentEngineName() string {
	if c.engineMgr == nil {
		return "unknown"
	}
	return string(c.engineMgr.GetCurrentType())
}

// UpdateUIConfig 更新 UI 配置（热更新）
func (c *Coordinator) UpdateUIConfig(uiConfig *config.UIConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if uiConfig == nil {
		return
	}

	// 更新每页候选数
	if uiConfig.CandidatesPerPage > 0 {
		c.candidatesPerPage = uiConfig.CandidatesPerPage
		// 重新计算总页数
		if len(c.candidates) > 0 {
			c.totalPages = (len(c.candidates) + c.candidatesPerPage - 1) / c.candidatesPerPage
			if c.currentPage > c.totalPages {
				c.currentPage = c.totalPages
			}
		}
	}

	// 更新配置引用
	if c.config != nil {
		c.config.UI = *uiConfig
	}

	// 通知 UI Manager 更新字体等设置
	if c.uiManager != nil {
		c.uiManager.UpdateConfig(uiConfig.FontSize, uiConfig.FontPath)
	}

	c.logger.Debug("UI config updated", "candidatesPerPage", c.candidatesPerPage)
}

// ClearInputState 清空输入状态（供外部调用）
func (c *Coordinator) ClearInputState() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.clearState()
	c.hideUI()
	c.logger.Debug("Input state cleared")
}
