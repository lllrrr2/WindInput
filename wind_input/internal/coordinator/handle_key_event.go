// handle_key_event.go — 键事件主路由与各按键处理器
package coordinator

import (
	"strings"
	"time"

	"github.com/huanfeng/wind_input/internal/bridge"
	"github.com/huanfeng/wind_input/internal/engine"
	"github.com/huanfeng/wind_input/internal/transform"
)

// HandleKeyEvent handles key events from C++ Bridge
// Returns a result indicating what action to take
func (c *Coordinator) HandleKeyEvent(data bridge.KeyEventData) *bridge.KeyEventResult {
	startTime := time.Now()

	c.mu.Lock()
	lockTime := time.Since(startTime)
	defer c.mu.Unlock()

	// Use Debug for high-frequency key events to reduce log noise
	c.logger.Debug("HandleKeyEvent", "key", data.Key, "keycode", data.KeyCode, "modifiers", data.Modifiers, "chineseMode", c.chineseMode, "lockWait", lockTime.String())

	// Check for Ctrl or Alt modifiers
	hasCtrl := data.Modifiers&ModCtrl != 0
	hasAlt := data.Modifiers&ModAlt != 0
	hasShift := data.Modifiers&ModShift != 0

	// Handle switch engine hotkey
	if c.config != nil && c.matchHotkey(c.config.Hotkeys.SwitchEngine, hasCtrl, hasShift, hasAlt, data.KeyCode) {
		return c.handleEngineSwitchKey()
	}

	// Handle full-width toggle hotkey
	if c.config != nil && c.matchHotkey(c.config.Hotkeys.ToggleFullWidth, hasCtrl, hasShift, hasAlt, data.KeyCode) {
		return c.handleToggleFullWidth()
	}

	// Handle punctuation toggle hotkey
	if c.config != nil && c.matchHotkey(c.config.Hotkeys.TogglePunct, hasCtrl, hasShift, hasAlt, data.KeyCode) {
		return c.handleTogglePunct()
	}

	// Handle mode toggle keys (lshift, rshift, lctrl, rctrl, capslock)
	// IMPORTANT: This must be checked BEFORE the Ctrl/Alt pass-through check,
	// because lctrl/rctrl are toggle mode keys but also set hasCtrl=true
	if toggleKey := c.getToggleModeKey(data.KeyCode); toggleKey != "" {
		c.logger.Debug("Toggle mode key detected", "key", toggleKey, "keyCode", data.KeyCode,
			"isConfigured", c.config != nil && c.config.IsToggleModeKey(toggleKey),
			"configuredKeys", c.config.Hotkeys.ToggleModeKeys)
		if c.config != nil && c.config.IsToggleModeKey(toggleKey) {
			// 检查是否需要在切换前上屏已有内容
			// CommitOnSwitch: 上屏编码（而非候选词），因为用户切换到英文意味着想输入英文
			var commitText string
			if c.config.Hotkeys.CommitOnSwitch && len(c.inputBuffer) > 0 {
				// 只在从中文切换到英文时上屏
				if c.chineseMode {
					commitText = c.inputBuffer
					if c.fullWidth {
						commitText = transform.ToFullWidth(commitText)
					}
				}
			}

			c.chineseMode = !c.chineseMode
			c.logger.Debug("Mode toggled", "key", toggleKey, "chineseMode", c.chineseMode)

			// Clear any pending input when switching modes
			if len(c.inputBuffer) > 0 {
				c.clearState()
				c.hideUI()
			}

			// Sync punctuation with mode if enabled
			if c.punctFollowMode {
				c.chinesePunctuation = c.chineseMode
			}

			// Reset punctuation converter state when switching modes
			c.punctConverter.Reset()

			// Save runtime state if remember_last_state is enabled
			c.saveRuntimeState()

			// Show mode indicator
			c.showModeIndicator()

			// Broadcast state to toolbar and all TSF clients
			c.broadcastState()

			// Return mode_changed with optional commit text
			if commitText != "" {
				return &bridge.KeyEventResult{
					Type:        bridge.ResponseTypeInsertText,
					Text:        commitText,
					ModeChanged: true,
					ChineseMode: c.chineseMode,
				}
			}

			return &bridge.KeyEventResult{
				Type:        bridge.ResponseTypeModeChanged,
				ChineseMode: c.chineseMode,
			}
		} else if toggleKey == "capslock" {
			// CapsLock is not configured as mode toggle key, but we still need to show indicator
			// C++ side sets 0x8000 bit in modifiers to indicate "state notification only"
			// Use the CapsLock state from C++ side (data.Toggles) as it's more accurate
			capsLockOn := data.IsCapsLockOn()
			c.logger.Debug("CapsLock state notification", "on", capsLockOn)

			// Update CapsLock state and broadcast if changed
			if c.capsLockOn != capsLockOn {
				c.capsLockOn = capsLockOn
				c.broadcastState()
			}

			// Show CapsLock indicator (A/a) - use NoLock version since we already hold the lock
			c.handleCapsLockStateNoLock(capsLockOn)

			// Return Consumed to indicate we handled it (no mode change)
			return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
		} else {
			// Toggle key recognized (lshift/rshift/lctrl/rctrl) but not configured
			// Consume the key to avoid passing Shift/Ctrl through to the application
			// This ensures consistent behavior: modifier key releases are always eaten by IME
			c.logger.Debug("Toggle key not configured, consuming", "key", toggleKey)
			return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
		}
	}

	// Other Ctrl/Alt combinations should be passed to the system
	// (after checking toggle mode keys, since lctrl/rctrl are valid toggle keys)
	if hasCtrl || hasAlt {
		c.logger.Debug("Key has Ctrl/Alt modifier, passing to system")
		return nil
	}

	// Preserve original key for English mode (uppercase letters should stay uppercase)
	key := data.Key

	// English mode: pass through all keys directly to system
	if !c.chineseMode {
		// 纯英文模式：所有按键直接透传给系统，不经过 Go 处理
		return nil
	}

	// Chinese mode with CapsLock: output letters directly (no full-width)
	// CapsLock ON: letters are uppercase, Shift+letter are lowercase
	// This allows users to quickly type English while in Chinese mode
	// Use the CapsLock state from C++ side (data.Toggles) as it's more accurate
	if data.IsCapsLockOn() {
		if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')) {
			// If there's pending input, commit it first then output the letter
			if len(c.inputBuffer) > 0 && len(c.candidates) > 0 {
				// Commit first candidate
				candidate := c.candidates[0]
				text := candidate.Text
				if c.fullWidth {
					text = transform.ToFullWidth(text)
				}
				c.clearState()
				c.hideUI()

				// Shift+letter = lowercase, letter = uppercase (CapsLock behavior)
				// Note: no full-width conversion for CapsLock English output
				var outputKey string
				if hasShift {
					outputKey = strings.ToLower(key)
				} else {
					outputKey = strings.ToUpper(key)
				}

				return &bridge.KeyEventResult{
					Type: bridge.ResponseTypeInsertText,
					Text: text + outputKey,
				}
			}

			// No pending input, just output letter
			c.clearState()
			c.hideUI()

			// Shift+letter = lowercase, letter = uppercase (CapsLock behavior)
			// Note: no full-width conversion for CapsLock English output
			var outputKey string
			if hasShift {
				outputKey = strings.ToLower(key)
			} else {
				outputKey = strings.ToUpper(key)
			}

			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: outputKey,
			}
		}
	}

	// 检查是否处于临时英文模式
	if c.tempEnglishMode {
		return c.handleTempEnglishKey(key, &data)
	}

	// 检查是否处于临时拼音模式
	if c.tempPinyinMode {
		return c.handleTempPinyinKey(key, &data)
	}

	// 检查是否应触发临时拼音模式
	if c.isTempPinyinTrigger(key, data.KeyCode) {
		return c.enterTempPinyinMode()
	}

	// 中文模式下，Shift+字母进入临时英文模式（CapsLock OFF 时）
	if c.chineseMode && !data.IsCapsLockOn() && hasShift {
		if c.config != nil && c.config.Input.ShiftTempEnglish.Enabled {
			if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')) {
				return c.enterTempEnglishMode(key)
			}
		}
	}

	// Chinese mode handling
	switch {
	case data.KeyCode == 37: // Left arrow
		return c.handleCursorLeft()

	case data.KeyCode == 39: // Right arrow
		return c.handleCursorRight()

	case data.KeyCode == 36: // Home
		return c.handleCursorHome()

	case data.KeyCode == 35: // End
		return c.handleCursorEnd()

	case data.KeyCode == 8: // Backspace
		return c.handleBackspace()

	case data.KeyCode == 13: // Enter
		return c.handleEnter()

	case data.KeyCode == 27: // Escape
		return c.handleEscape()

	case data.KeyCode == 32: // Space
		return c.handleSpace()

	case c.isPageUpKey(key, data.KeyCode, uint32(data.Modifiers)):
		return c.handlePageUp()

	case c.isPageDownKey(key, data.KeyCode, uint32(data.Modifiers)):
		return c.handlePageDown()

	case len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')):
		// Chinese mode: convert to lowercase for pinyin
		return c.handleAlphaKey(strings.ToLower(key))

	case len(key) == 1 && key[0] >= '1' && key[0] <= '9':
		return c.handleNumberKey(int(key[0] - '0'))

	case c.isSelectKey2(key, data.KeyCode):
		// Handle 2nd candidate selection key (e.g., semicolon)
		if len(c.candidates) >= 2 && len(c.inputBuffer) > 0 {
			return c.selectCandidate(1) // Select 2nd candidate (index 1)
		}
		// If no candidates, treat as punctuation
		if len(key) == 1 && c.isPunctuation(rune(key[0])) {
			return c.handlePunctuation(rune(key[0]))
		}
		return nil

	case c.isSelectKey3(key, data.KeyCode):
		// Handle 3rd candidate selection key (e.g., quote)
		if len(c.candidates) >= 3 && len(c.inputBuffer) > 0 {
			return c.selectCandidate(2) // Select 3rd candidate (index 2)
		}
		// If no candidates, treat as punctuation
		if len(key) == 1 && c.isPunctuation(rune(key[0])) {
			return c.handlePunctuation(rune(key[0]))
		}
		return nil

	case len(key) == 1 && c.isPunctuation(rune(key[0])):
		return c.handlePunctuation(rune(key[0]))

	default:
		c.logger.Debug("Unhandled key", "key", key, "keycode", data.KeyCode)
		return nil
	}
}

func (c *Coordinator) handleAlphaKey(key string) *bridge.KeyEventResult {
	startTime := time.Now()
	// 在光标位置插入字符
	c.inputBuffer = c.inputBuffer[:c.inputCursorPos] + key + c.inputBuffer[c.inputCursorPos:]
	c.inputCursorPos += len(key)
	c.logger.Debug("Input buffer updated", "buffer", c.inputBuffer, "cursor", c.inputCursorPos)

	// 处理顶码（如五笔的五码顶字）
	if c.engineMgr != nil {
		commitText, newInput, shouldCommit := c.engineMgr.HandleTopCode(c.inputBuffer)
		if shouldCommit {
			c.inputBuffer = newInput
			c.inputCursorPos = len(newInput)
			c.logger.Debug("Top code commit", "newInputLen", len(newInput))

			// Apply full-width conversion if enabled
			if c.fullWidth {
				commitText = transform.ToFullWidth(commitText)
			}

			// 如果还有剩余输入，继续处理并更新候选
			if len(c.inputBuffer) > 0 {
				c.updateCandidates()
				c.showUI()
				// 返回带有新组合的响应，让 C++ 端同时插入文字并开始新组合
				return &bridge.KeyEventResult{
					Type:           bridge.ResponseTypeInsertText,
					Text:           commitText,
					NewComposition: c.inputBuffer,
				}
			} else {
				c.hideUI()
				return &bridge.KeyEventResult{
					Type: bridge.ResponseTypeInsertText,
					Text: commitText,
				}
			}
		}
	}

	// 更新候选词
	updateStart := time.Now()
	result := c.updateCandidatesEx()
	updateElapsed := time.Since(updateStart)

	// 检查自动上屏
	if result != nil && result.ShouldCommit {
		text := result.CommitText
		// Apply full-width conversion if enabled
		if c.fullWidth {
			text = transform.ToFullWidth(text)
		}
		c.clearState()
		c.hideUI()
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
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
			// Apply full-width conversion if enabled
			if c.fullWidth {
				text = transform.ToFullWidth(text)
			}
			c.clearState()
			c.hideUI()
			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: text,
			}
		}
	}

	showStart := time.Now()
	c.showUI()
	showElapsed := time.Since(showStart)

	totalElapsed := time.Since(startTime)
	c.logger.Debug("handleAlphaKey timing", "total", totalElapsed.String(), "updateCandidates", updateElapsed.String(), "showUI", showElapsed.String())

	// Handle Inline Preedit
	if c.config != nil && c.config.UI.InlinePreedit {
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     c.compositionText(),
			CaretPos: c.displayCursorPos(),
		}
	}

	// When InlinePreedit is disabled, we still need to tell TSF that we have an active
	// composition so that subsequent keys (ESC, Backspace, Enter) are intercepted.
	// Return UpdateComposition with empty text - TSF will set _isComposing=TRUE but
	// won't display anything in the application.
	return &bridge.KeyEventResult{
		Type:     bridge.ResponseTypeUpdateComposition,
		Text:     "",
		CaretPos: 0,
	}
}

func (c *Coordinator) handleBackspace() *bridge.KeyEventResult {
	if len(c.inputBuffer) > 0 && c.inputCursorPos > 0 {
		// 在光标位置删除前一个字符
		c.inputBuffer = c.inputBuffer[:c.inputCursorPos-1] + c.inputBuffer[c.inputCursorPos:]
		c.inputCursorPos--
		c.logger.Debug("Input buffer after backspace", "buffer", c.inputBuffer, "cursor", c.inputCursorPos)

		if len(c.inputBuffer) == 0 {
			c.clearState()
			c.hideUI()
			return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
		}

		c.updateCandidates()
		c.showUI()

		// Handle Inline Preedit
		if c.config != nil && c.config.UI.InlinePreedit {
			return &bridge.KeyEventResult{
				Type:     bridge.ResponseTypeUpdateComposition,
				Text:     c.compositionText(),
				CaretPos: c.displayCursorPos(),
			}
		}

		// When InlinePreedit is disabled, still use UpdateComposition with empty text
		// so that TSF knows there's an active composition
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     "",
			CaretPos: 0,
		}
	}

	if len(c.inputBuffer) == 0 {
		// Buffer is already empty - pass through to system
		c.logger.Debug("Backspace with empty buffer, passing through to system")
		return nil
	}

	// Cursor at beginning but buffer not empty - consume the key (don't pass to system)
	return &bridge.KeyEventResult{
		Type: bridge.ResponseTypeConsumed,
	}
}

func (c *Coordinator) handleCursorLeft() *bridge.KeyEventResult {
	if len(c.inputBuffer) == 0 {
		return nil // 无输入时透传
	}
	if c.inputCursorPos > 0 {
		c.inputCursorPos--
		c.logger.Debug("Cursor left", "cursor", c.inputCursorPos)
		if c.config != nil && c.config.UI.InlinePreedit {
			return &bridge.KeyEventResult{
				Type:     bridge.ResponseTypeUpdateComposition,
				Text:     c.compositionText(),
				CaretPos: c.displayCursorPos(),
			}
		}
		// InlinePreedit 关闭时，刷新候选窗口中的光标位置
		c.showUI()
	}
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) handleCursorRight() *bridge.KeyEventResult {
	if len(c.inputBuffer) == 0 {
		return nil // 无输入时透传
	}
	if c.inputCursorPos < len(c.inputBuffer) {
		c.inputCursorPos++
		c.logger.Debug("Cursor right", "cursor", c.inputCursorPos)
		if c.config != nil && c.config.UI.InlinePreedit {
			return &bridge.KeyEventResult{
				Type:     bridge.ResponseTypeUpdateComposition,
				Text:     c.compositionText(),
				CaretPos: c.displayCursorPos(),
			}
		}
		// InlinePreedit 关闭时，刷新候选窗口中的光标位置
		c.showUI()
	}
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) handleCursorHome() *bridge.KeyEventResult {
	if len(c.inputBuffer) == 0 {
		return nil
	}
	c.inputCursorPos = 0
	c.logger.Debug("Cursor home", "cursor", c.inputCursorPos)
	if c.config != nil && c.config.UI.InlinePreedit {
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     c.compositionText(),
			CaretPos: c.displayCursorPos(),
		}
	}
	// InlinePreedit 关闭时，刷新候选窗口中的光标位置
	c.showUI()
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) handleCursorEnd() *bridge.KeyEventResult {
	if len(c.inputBuffer) == 0 {
		return nil
	}
	c.inputCursorPos = len(c.inputBuffer)
	c.logger.Debug("Cursor end", "cursor", c.inputCursorPos)
	if c.config != nil && c.config.UI.InlinePreedit {
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     c.compositionText(),
			CaretPos: c.displayCursorPos(),
		}
	}
	// InlinePreedit 关闭时，刷新候选窗口中的光标位置
	c.showUI()
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) handleEnter() *bridge.KeyEventResult {
	// Commit raw input as text
	if len(c.inputBuffer) > 0 {
		text := c.inputBuffer

		// Apply full-width conversion if enabled
		if c.fullWidth {
			text = transform.ToFullWidth(text)
		}

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
	// If toolbar context menu is open, close it and consume ESC
	if c.uiManager != nil && c.uiManager.IsToolbarMenuOpen() {
		c.logger.Debug("ESC closes toolbar context menu")
		c.uiManager.HideToolbarMenu()
		// Return Consumed to eat the ESC key (don't pass to app)
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
	}

	// If candidate context menu is open, close it and consume ESC
	if c.uiManager != nil && c.uiManager.IsCandidateMenuOpen() {
		c.logger.Debug("ESC closes candidate context menu")
		c.uiManager.HideCandidateMenu()
		// Return Consumed to eat the ESC key (don't pass to app)
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
	}

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

		// Apply full-width conversion if enabled
		if c.fullWidth {
			text = transform.ToFullWidth(text)
		}

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
	// Pass through only if no candidates
	if len(c.candidates) == 0 {
		return nil
	}

	// Have candidates - always consume the key, even if at first page
	if c.currentPage > 1 {
		c.currentPage--
		c.logger.Debug("Page up", "currentPage", c.currentPage, "totalPages", c.totalPages)
		c.showUI()
	}
	// Return Consumed to indicate key was handled (don't pass to application)
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) handlePageDown() *bridge.KeyEventResult {
	// Pass through only if no candidates
	if len(c.candidates) == 0 {
		return nil
	}

	// Have candidates - always consume the key, even if at last page
	if c.currentPage < c.totalPages {
		c.currentPage++
		c.logger.Debug("Page down", "currentPage", c.currentPage, "totalPages", c.totalPages)
		c.showUI()
	}
	// Return Consumed to indicate key was handled (don't pass to application)
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
}

func (c *Coordinator) selectCandidate(index int) *bridge.KeyEventResult {
	if index < 0 || index >= len(c.candidates) {
		return nil
	}

	cand := c.candidates[index]
	originalText := cand.Text
	text := originalText

	// Apply full-width conversion if enabled
	if c.fullWidth {
		text = transform.ToFullWidth(text)
	}

	// 拼音引擎部分上屏：候选消耗的输入长度小于缓冲区长度时，保留剩余部分
	isPinyin := c.engineMgr != nil && c.engineMgr.GetCurrentType() == engine.EngineTypePinyin
	if isPinyin && cand.ConsumedLength > 0 && cand.ConsumedLength < len(c.inputBuffer) {
		// 用户词频学习：命令候选不进入学习，避免污染用户词典（如 uuid）。
		if !cand.IsCommand {
			consumedCode := c.inputBuffer[:cand.ConsumedLength]
			c.engineMgr.OnCandidateSelected(consumedCode, originalText)
		}

		remaining := c.inputBuffer[cand.ConsumedLength:]
		c.logger.Debug("Partial commit (pinyin)", "index", index, "text", text,
			"consumed", cand.ConsumedLength, "remaining", remaining)

		// 更新缓冲区为剩余部分，光标重置到末尾，重新触发候选更新
		c.inputBuffer = remaining
		c.inputCursorPos = len(remaining)
		c.currentPage = 1
		c.updateCandidates()
		c.showUI()

		return &bridge.KeyEventResult{
			Type:           bridge.ResponseTypeInsertText,
			Text:           text,
			NewComposition: remaining,
		}
	}

	// 用户词频学习：命令候选不进入学习，避免污染用户词典（如 uuid）。
	if isPinyin && c.engineMgr != nil && !cand.IsCommand {
		c.engineMgr.OnCandidateSelected(c.inputBuffer, originalText)
	}

	c.logger.Debug("Candidate selected", "index", index, "original", originalText, "output", text, "fullWidth", c.fullWidth)

	c.clearState()
	c.hideUI()

	return &bridge.KeyEventResult{
		Type: bridge.ResponseTypeInsertText,
		Text: text,
	}
}
