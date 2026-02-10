// handle_temp_mode.go — 临时英文模式 + 临时拼音模式
package coordinator

import (
	"strings"

	"github.com/huanfeng/wind_input/internal/bridge"
	"github.com/huanfeng/wind_input/internal/engine"
	"github.com/huanfeng/wind_input/internal/transform"
	"github.com/huanfeng/wind_input/internal/ui"
)

// enterTempEnglishMode 进入临时英文模式
func (c *Coordinator) enterTempEnglishMode(key string) *bridge.KeyEventResult {
	c.tempEnglishMode = true
	c.tempEnglishBuffer = strings.ToUpper(key) // Shift+字母输出大写

	c.logger.Debug("Entered temp English mode", "buffer", c.tempEnglishBuffer)

	// 显示临时英文模式 UI
	c.showTempEnglishUI()

	// 返回 UpdateComposition 让 C++ 端知道进入了 composing 状态
	// 这样后续的 Backspace/Enter 才会被发送到 Go 端处理
	return &bridge.KeyEventResult{
		Type:     bridge.ResponseTypeUpdateComposition,
		Text:     c.tempEnglishBuffer,
		CaretPos: len(c.tempEnglishBuffer),
	}
}

// handleTempEnglishKey 处理临时英文模式下的按键
func (c *Coordinator) handleTempEnglishKey(key string, data *bridge.KeyEventData) *bridge.KeyEventResult {
	hasShift := data.Modifiers&ModShift != 0

	switch {
	case data.KeyCode == 8: // Backspace
		if len(c.tempEnglishBuffer) > 0 {
			c.tempEnglishBuffer = c.tempEnglishBuffer[:len(c.tempEnglishBuffer)-1]
			if len(c.tempEnglishBuffer) == 0 {
				return c.exitTempEnglishMode(false, "")
			}
			c.showTempEnglishUI()
			// 返回 UpdateComposition 保持 composing 状态
			return &bridge.KeyEventResult{
				Type:     bridge.ResponseTypeUpdateComposition,
				Text:     c.tempEnglishBuffer,
				CaretPos: len(c.tempEnglishBuffer),
			}
		}
		// 缓冲区已空，返回 ClearComposition
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}

	case data.KeyCode == 27: // Escape
		return c.exitTempEnglishMode(false, "")

	case data.KeyCode == 32: // Space
		// 上屏缓冲内容
		text := c.tempEnglishBuffer
		return c.exitTempEnglishMode(true, text)

	case data.KeyCode == 13: // Enter
		// Enter 也上屏缓冲内容
		return c.exitTempEnglishMode(true, c.tempEnglishBuffer)

	case len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z')):
		// 追加字母：Shift+字母=大写，字母=小写
		var letter string
		if hasShift {
			letter = strings.ToUpper(key)
		} else {
			letter = strings.ToLower(key)
		}
		c.tempEnglishBuffer += letter
		c.logger.Debug("Temp English buffer updated", "buffer", c.tempEnglishBuffer)
		c.showTempEnglishUI()
		// 返回 UpdateComposition 保持 composing 状态
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     c.tempEnglishBuffer,
			CaretPos: len(c.tempEnglishBuffer),
		}

	case len(key) == 1 && key[0] >= '0' && key[0] <= '9':
		// 数字：当前没有英文候选，上屏缓冲内容并输出数字
		// （如果将来有英文词库/候选，数字应该用于选择候选）
		if len(c.tempEnglishBuffer) > 0 {
			text := c.tempEnglishBuffer
			if c.fullWidth {
				text = transform.ToFullWidth(text)
			}
			c.tempEnglishMode = false
			c.tempEnglishBuffer = ""
			c.hideUI()
			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: text + key,
			}
		}
		// 缓冲区为空，退出并透传数字
		c.exitTempEnglishMode(false, "")
		return nil
	}

	// 其他按键（如标点）：上屏缓冲内容，然后处理按键
	if len(c.tempEnglishBuffer) > 0 {
		text := c.tempEnglishBuffer
		if c.fullWidth {
			text = transform.ToFullWidth(text)
		}
		c.tempEnglishMode = false
		c.tempEnglishBuffer = ""
		c.hideUI()
		// 如果是标点，处理标点
		if len(key) == 1 && c.isPunctuation(rune(key[0])) {
			punctResult := c.handlePunctuation(rune(key[0]))
			if punctResult != nil {
				return &bridge.KeyEventResult{
					Type: bridge.ResponseTypeInsertText,
					Text: text + punctResult.Text,
				}
			}
		}
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
		}
	}

	// 缓冲区为空，退出临时英文模式并透传
	c.exitTempEnglishMode(false, "")
	return nil
}

// exitTempEnglishMode 退出临时英文模式
func (c *Coordinator) exitTempEnglishMode(commit bool, text string) *bridge.KeyEventResult {
	c.tempEnglishMode = false
	c.tempEnglishBuffer = ""
	c.candidates = nil
	c.currentPage = 0
	c.totalPages = 0
	c.hideUI()

	c.logger.Debug("Exited temp English mode", "commit", commit, "textLen", len(text))

	if commit && len(text) > 0 {
		// 应用全角转换（如果启用）
		if c.fullWidth {
			text = transform.ToFullWidth(text)
		}
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
		}
	}

	// 取消时返回 ClearComposition 让 C++ 端清除 composing 状态
	return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
}

// showTempEnglishUI 显示临时英文模式的 UI
func (c *Coordinator) showTempEnglishUI() {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	// 使用光标位置
	caretX := c.caretX
	caretY := c.caretY
	caretHeight := c.caretHeight

	const maxCoord = 32000
	if (c.caretX == 0 && c.caretY == 0) || caretX > maxCoord || caretX < -maxCoord || caretY > maxCoord || caretY < -maxCoord {
		if c.lastValidX != 0 || c.lastValidY != 0 {
			caretX = c.lastValidX
			caretY = c.lastValidY
			caretHeight = 20
		} else {
			caretX = 400
			caretY = 300
			caretHeight = 20
		}
	}

	// 显示临时英文缓冲区内容（无候选词）
	c.uiManager.ShowCandidates(
		nil, // 无候选词
		c.tempEnglishBuffer,
		len(c.tempEnglishBuffer), // 光标在末尾
		caretX,
		caretY,
		caretHeight,
		1, // currentPage
		1, // totalPages
	)
}

// ============================================================
// 临时拼音模式
// ============================================================

// isTempPinyinTrigger 检查按键是否应触发临时拼音模式
func (c *Coordinator) isTempPinyinTrigger(key string, keyCode int) bool {
	// 仅五笔引擎下生效
	if c.engineMgr == nil || c.engineMgr.GetCurrentType() != engine.EngineTypeWubi {
		return false
	}
	// 仅输入缓冲区为空时触发
	if len(c.inputBuffer) > 0 {
		return false
	}
	if c.config == nil {
		return false
	}

	for _, tk := range c.config.Input.TempPinyin.TriggerKeys {
		switch tk {
		case "backtick":
			if key == "`" || keyCode == 192 {
				return true
			}
		case "semicolon":
			// 仅在输入缓冲区为空且无候选时触发
			// 有候选时 semicolon 仍用于二三候选选择
			if (key == ";" || keyCode == 186) && len(c.candidates) == 0 {
				return true
			}
		}
	}
	return false
}

// enterTempPinyinMode 进入临时拼音模式
func (c *Coordinator) enterTempPinyinMode() *bridge.KeyEventResult {
	// 确保拼音引擎已加载
	if c.engineMgr != nil {
		if err := c.engineMgr.EnsurePinyinLoaded(); err != nil {
			c.logger.Warn("Failed to load pinyin engine for temp pinyin", "error", err)
			return nil
		}
	}

	c.tempPinyinMode = true
	c.tempPinyinBuffer = ""

	c.logger.Debug("Entered temp pinyin mode")

	// 显示初始 UI（仅前缀标识）
	c.showTempPinyinUI()

	return &bridge.KeyEventResult{
		Type:     bridge.ResponseTypeUpdateComposition,
		Text:     "`",
		CaretPos: 1,
	}
}

// handleTempPinyinKey 处理临时拼音模式下的按键
func (c *Coordinator) handleTempPinyinKey(key string, data *bridge.KeyEventData) *bridge.KeyEventResult {
	switch {
	case len(key) == 1 && key[0] >= 'a' && key[0] <= 'z':
		// 追加字母到拼音缓冲区
		c.tempPinyinBuffer += key
		c.updateTempPinyinCandidates()
		c.showTempPinyinUI()
		preedit := "`" + c.preeditDisplay
		if c.preeditDisplay == "" {
			preedit = "`" + c.tempPinyinBuffer
		}
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     preedit,
			CaretPos: len(preedit),
		}

	case len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z':
		// 大写字母也转小写处理
		c.tempPinyinBuffer += strings.ToLower(key)
		c.updateTempPinyinCandidates()
		c.showTempPinyinUI()
		preedit := "`" + c.preeditDisplay
		if c.preeditDisplay == "" {
			preedit = "`" + c.tempPinyinBuffer
		}
		return &bridge.KeyEventResult{
			Type:     bridge.ResponseTypeUpdateComposition,
			Text:     preedit,
			CaretPos: len(preedit),
		}

	case len(key) == 1 && key[0] >= '1' && key[0] <= '9':
		// 数字选择候选
		idx := int(key[0]-'0') - 1
		pageStart := (c.currentPage - 1) * c.candidatesPerPage
		globalIdx := pageStart + idx
		if globalIdx < len(c.candidates) {
			return c.selectTempPinyinCandidate(globalIdx)
		}
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}

	case data.KeyCode == 32: // Space
		// 选择第一个候选
		if len(c.candidates) > 0 {
			pageStart := (c.currentPage - 1) * c.candidatesPerPage
			return c.selectTempPinyinCandidate(pageStart)
		}
		// 无候选时退出
		return c.exitTempPinyinMode(false, "")

	case data.KeyCode == 13: // Enter
		// 上屏拼音原文字母
		if len(c.tempPinyinBuffer) > 0 {
			return c.exitTempPinyinMode(true, c.tempPinyinBuffer)
		}
		return c.exitTempPinyinMode(false, "")

	case data.KeyCode == 8: // Backspace
		if len(c.tempPinyinBuffer) > 0 {
			c.tempPinyinBuffer = c.tempPinyinBuffer[:len(c.tempPinyinBuffer)-1]
			if len(c.tempPinyinBuffer) == 0 {
				return c.exitTempPinyinMode(false, "")
			}
			c.currentPage = 1
			c.updateTempPinyinCandidates()
			c.showTempPinyinUI()
			preedit := "`" + c.preeditDisplay
			if c.preeditDisplay == "" {
				preedit = "`" + c.tempPinyinBuffer
			}
			return &bridge.KeyEventResult{
				Type:     bridge.ResponseTypeUpdateComposition,
				Text:     preedit,
				CaretPos: len(preedit),
			}
		}
		return c.exitTempPinyinMode(false, "")

	case data.KeyCode == 27: // Escape
		return c.exitTempPinyinMode(false, "")

	case c.isPageUpKey(key, data.KeyCode, uint32(data.Modifiers)):
		if c.currentPage > 1 {
			c.currentPage--
			c.showTempPinyinUI()
		}
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}

	case c.isPageDownKey(key, data.KeyCode, uint32(data.Modifiers)):
		if c.currentPage < c.totalPages {
			c.currentPage++
			c.showTempPinyinUI()
		}
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}

	default:
		// 其他按键（如标点）
		if len(c.candidates) > 0 {
			// 有候选时：选第一个候选后处理标点
			pageStart := (c.currentPage - 1) * c.candidatesPerPage
			cand := c.candidates[pageStart]
			text := cand.Text
			if c.fullWidth {
				text = transform.ToFullWidth(text)
			}

			c.tempPinyinMode = false
			c.tempPinyinBuffer = ""
			c.preeditDisplay = ""
			c.candidates = nil
			c.currentPage = 1
			c.totalPages = 1
			c.hideUI()

			// 处理标点
			if len(key) == 1 && c.isPunctuation(rune(key[0])) {
				punctResult := c.handlePunctuation(rune(key[0]))
				if punctResult != nil {
					return &bridge.KeyEventResult{
						Type: bridge.ResponseTypeInsertText,
						Text: text + punctResult.Text,
					}
				}
			}
			return &bridge.KeyEventResult{
				Type: bridge.ResponseTypeInsertText,
				Text: text,
			}
		}

		// 无候选时退出
		return c.exitTempPinyinMode(false, "")
	}
}

// updateTempPinyinCandidates 更新临时拼音候选
func (c *Coordinator) updateTempPinyinCandidates() {
	if c.engineMgr == nil || len(c.tempPinyinBuffer) == 0 {
		c.candidates = nil
		c.preeditDisplay = ""
		c.totalPages = 1
		return
	}

	maxCandidates := 100
	result := c.engineMgr.ConvertWithPinyin(c.tempPinyinBuffer, maxCandidates)

	// 转换为 ui.Candidate 列表
	uiCandidates := make([]ui.Candidate, len(result.Candidates))
	for i, cand := range result.Candidates {
		uiCandidates[i] = ui.Candidate{
			Text:           cand.Text,
			Index:          i + 1,
			Comment:        cand.Hint, // 五笔编码作为 Comment 显示
			Weight:         cand.Weight,
			ConsumedLength: cand.ConsumedLength,
		}
	}

	c.candidates = uiCandidates
	c.preeditDisplay = result.PreeditDisplay

	// 计算分页
	if c.candidatesPerPage <= 0 {
		c.candidatesPerPage = 7
	}
	total := len(c.candidates)
	c.totalPages = (total + c.candidatesPerPage - 1) / c.candidatesPerPage
	if c.totalPages < 1 {
		c.totalPages = 1
	}
	if c.currentPage > c.totalPages {
		c.currentPage = c.totalPages
	}
	if c.currentPage < 1 {
		c.currentPage = 1
	}
}

// selectTempPinyinCandidate 选择临时拼音候选
func (c *Coordinator) selectTempPinyinCandidate(index int) *bridge.KeyEventResult {
	if index < 0 || index >= len(c.candidates) {
		return &bridge.KeyEventResult{Type: bridge.ResponseTypeConsumed}
	}

	cand := c.candidates[index]
	text := cand.Text
	if c.fullWidth {
		text = transform.ToFullWidth(text)
	}

	// 支持拼音部分上屏
	if cand.ConsumedLength > 0 && cand.ConsumedLength < len(c.tempPinyinBuffer) {
		remaining := c.tempPinyinBuffer[cand.ConsumedLength:]
		c.tempPinyinBuffer = remaining
		c.currentPage = 1
		c.updateTempPinyinCandidates()
		c.showTempPinyinUI()

		preedit := "`" + c.preeditDisplay
		if c.preeditDisplay == "" {
			preedit = "`" + c.tempPinyinBuffer
		}
		return &bridge.KeyEventResult{
			Type:           bridge.ResponseTypeInsertText,
			Text:           text,
			NewComposition: preedit,
		}
	}

	// 全部上屏，退出临时拼音模式
	return c.exitTempPinyinMode(true, text)
}

// exitTempPinyinMode 退出临时拼音模式
func (c *Coordinator) exitTempPinyinMode(commit bool, text string) *bridge.KeyEventResult {
	c.tempPinyinMode = false
	c.tempPinyinBuffer = ""
	c.preeditDisplay = ""
	c.candidates = nil
	c.currentPage = 1
	c.totalPages = 1
	c.hideUI()

	c.logger.Debug("Exited temp pinyin mode", "commit", commit, "textLen", len(text))

	if commit && len(text) > 0 {
		return &bridge.KeyEventResult{
			Type: bridge.ResponseTypeInsertText,
			Text: text,
		}
	}

	return &bridge.KeyEventResult{Type: bridge.ResponseTypeClearComposition}
}

// showTempPinyinUI 显示临时拼音模式 UI
func (c *Coordinator) showTempPinyinUI() {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	// 使用光标位置
	caretX := c.caretX
	caretY := c.caretY
	caretHeight := c.caretHeight

	const maxCoord = 32000
	if (c.caretX == 0 && c.caretY == 0) || caretX > maxCoord || caretX < -maxCoord || caretY > maxCoord || caretY < -maxCoord {
		if c.lastValidX != 0 || c.lastValidY != 0 {
			caretX = c.lastValidX
			caretY = c.lastValidY
			caretHeight = 20
		} else {
			caretX = 400
			caretY = 300
			caretHeight = 20
		}
	}

	// 获取当前页候选
	startIdx := (c.currentPage - 1) * c.candidatesPerPage
	endIdx := startIdx + c.candidatesPerPage
	if endIdx > len(c.candidates) {
		endIdx = len(c.candidates)
	}

	var pageCandidates []ui.Candidate
	if startIdx < len(c.candidates) {
		pageCandidates = c.candidates[startIdx:endIdx]
	}

	// 重新编号显示（1-9）
	displayCandidates := make([]ui.Candidate, len(pageCandidates))
	for i, cand := range pageCandidates {
		displayCandidates[i] = ui.Candidate{
			Text:    cand.Text,
			Index:   i + 1,
			Comment: cand.Comment,
			Weight:  cand.Weight,
		}
	}

	// 构建预编辑文本
	preedit := "`" + c.preeditDisplay
	if c.preeditDisplay == "" && c.tempPinyinBuffer != "" {
		preedit = "`" + c.tempPinyinBuffer
	} else if c.tempPinyinBuffer == "" {
		preedit = "`"
	}

	c.uiManager.ShowCandidates(
		displayCandidates,
		preedit,
		len(preedit),
		caretX,
		caretY,
		caretHeight,
		c.currentPage,
		c.totalPages,
	)
}
