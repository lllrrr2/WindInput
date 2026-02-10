// handle_config.go — 配置热更新、菜单命令、持久化、getter
package coordinator

import (
	"github.com/huanfeng/wind_input/internal/bridge"
	"github.com/huanfeng/wind_input/internal/engine"
	"github.com/huanfeng/wind_input/internal/transform"
	"github.com/huanfeng/wind_input/internal/ui"
	"github.com/huanfeng/wind_input/pkg/config"
)

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
		c.uiManager.UpdateConfig(uiConfig.FontSize, uiConfig.FontPath, uiConfig.HideCandidateWindow)
		// Update candidate layout
		if uiConfig.CandidateLayout != "" {
			c.uiManager.SetCandidateLayout(uiConfig.CandidateLayout)
		}
		// Update hide preedit setting
		c.uiManager.SetHidePreedit(uiConfig.InlinePreedit)
		// Update status indicator config
		c.uiManager.UpdateStatusIndicatorConfig(
			uiConfig.StatusIndicatorDuration,
			uiConfig.StatusIndicatorOffsetX,
			uiConfig.StatusIndicatorOffsetY,
		)
		// 设置编码提示延迟
		c.uiManager.SetTooltipDelay(uiConfig.TooltipDelay)
		// 更新主题
		if uiConfig.Theme != "" {
			c.uiManager.LoadTheme(uiConfig.Theme)
		}
	}

	c.logger.Debug("UI config updated", "candidatesPerPage", c.candidatesPerPage)
}

// UpdateToolbarConfig 更新工具栏配置（热更新）
func (c *Coordinator) UpdateToolbarConfig(toolbarConfig *config.ToolbarConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if toolbarConfig == nil {
		return
	}

	c.toolbarVisible = toolbarConfig.Visible

	// 更新配置引用
	if c.config != nil {
		c.config.Toolbar = *toolbarConfig
	}

	// 通知 UI Manager 更新工具栏状态
	if c.uiManager != nil {
		if c.toolbarVisible && c.imeActivated {
			c.uiManager.ShowToolbarWithState(toolbarConfig.PositionX, toolbarConfig.PositionY, ui.ToolbarState{
				ChineseMode:   c.chineseMode,
				FullWidth:     c.fullWidth,
				ChinesePunct:  c.chinesePunctuation,
				CapsLock:      c.capsLockOn,
				EffectiveMode: int(c.getEffectiveModeNoLock()),
			})
		} else {
			c.uiManager.SetToolbarVisible(false)
		}
	}

	c.logger.Debug("Toolbar config updated", "visible", c.toolbarVisible)
}

// UpdateInputConfig 更新输入配置（热更新）
// 注意：fullWidth 和 chinesePunctuation 是运行时状态，不从配置更新
func (c *Coordinator) UpdateInputConfig(inputConfig *config.InputConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if inputConfig == nil {
		return
	}

	// 只更新配置项，不更新运行时状态（fullWidth, chinesePunctuation）
	c.punctFollowMode = inputConfig.PunctFollowMode

	// 更新配置引用
	if c.config != nil {
		c.config.Input = *inputConfig
	}

	c.hotkeysDirty = true // SelectKeyGroups/PageKeys 变化也影响热键
	c.logger.Debug("Input config updated", "punctFollowMode", c.punctFollowMode)
}

// UpdateEngineConfig 更新引擎配置
func (c *Coordinator) UpdateEngineConfig(engineConfig *config.EngineConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if engineConfig == nil || c.engineMgr == nil {
		return
	}

	// 检查引擎类型是否改变
	currentType := c.engineMgr.GetCurrentType()
	newType := engine.EngineType(engineConfig.Type)

	if currentType != newType {
		// 清除当前输入状态
		c.clearState()
		c.hideUI()

		// 切换引擎
		if err := c.engineMgr.SwitchEngine(newType); err != nil {
			c.logger.Error("Failed to switch engine", "error", err, "targetType", newType)
		} else {
			c.logger.Info("Engine switched via config reload", "from", currentType, "to", newType)
			// 同步词库管理器的活跃引擎
			if dm := c.engineMgr.GetDictManager(); dm != nil {
				dm.SetActiveEngine(string(newType))
			}
		}
	}

	// 更新引擎选项
	c.engineMgr.UpdateFilterMode(engineConfig.FilterMode)
	c.engineMgr.UpdateWubiOptions(
		engineConfig.Wubi.AutoCommitAt4,
		engineConfig.Wubi.ClearOnEmptyAt4,
		engineConfig.Wubi.TopCodeCommit,
		engineConfig.Wubi.PunctCommit,
		engineConfig.Wubi.ShowCodeHint,
		engineConfig.Wubi.SingleCodeInput,
	)
	c.engineMgr.UpdatePinyinOptions(&engineConfig.Pinyin)

	// 更新配置引用
	if c.config != nil {
		c.config.Engine = *engineConfig
	}

	c.logger.Debug("Engine config updated", "type", engineConfig.Type, "filterMode", engineConfig.FilterMode)
}

// UpdateHotkeyConfig 更新快捷键配置
func (c *Coordinator) UpdateHotkeyConfig(hotkeyConfig *config.HotkeyConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if hotkeyConfig == nil {
		return
	}

	// 更新配置引用
	if c.config != nil {
		c.config.Hotkeys = *hotkeyConfig
	}

	// 重新编译快捷键（如果有编译器的话）
	if c.hotkeyCompiler != nil {
		c.hotkeyCompiler.UpdateConfig(c.config)
	}
	c.hotkeysDirty = true // 标记缓存失效，下次获取时重新编译

	c.logger.Debug("Hotkey config updated",
		"toggleModeKeys", hotkeyConfig.ToggleModeKeys,
		"switchEngine", hotkeyConfig.SwitchEngine)
}

// UpdateStartupConfig 更新启动配置
func (c *Coordinator) UpdateStartupConfig(startupConfig *config.StartupConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if startupConfig == nil {
		return
	}

	// 更新配置引用
	if c.config != nil {
		c.config.Startup = *startupConfig
	}

	c.logger.Debug("Startup config updated", "rememberLastState", startupConfig.RememberLastState)
}

// ClearInputState 清空输入状态（供外部调用）
func (c *Coordinator) ClearInputState() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.clearState()
	c.hideUI()
	c.logger.Debug("Input state cleared")
}

// SetIMEActivated sets the IME activation state
// When activated, show toolbar if enabled; when deactivated, hide toolbar
func (c *Coordinator) SetIMEActivated(activated bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	wasActivated := c.imeActivated
	c.imeActivated = activated
	c.logger.Debug("IME activation", "activated", activated, "wasActivated", wasActivated)

	if c.uiManager == nil {
		return
	}

	if activated {
		// IME activated - show toolbar if enabled
		if c.toolbarVisible {
			// Always recalculate toolbar position based on current caret/focus position
			// This ensures toolbar follows the active screen when switching between apps
			toolbarWidth, toolbarHeight := 140, 30 // Base size, will be scaled by DPI
			var posX, posY int

			// Use caret position to determine which monitor to show toolbar on
			// Note: coordinates can be negative in multi-monitor setups, use caretValid flag
			if c.caretValid {
				posX, posY = ui.GetToolbarPositionForCaret(
					c.caretX, c.caretY,
					ui.ScaleIntForDPI(toolbarWidth),
					ui.ScaleIntForDPI(toolbarHeight),
				)
			} else {
				// No valid caret position yet, use mouse position as fallback
				posX, posY = ui.GetDefaultToolbarPosition(
					ui.ScaleIntForDPI(toolbarWidth),
					ui.ScaleIntForDPI(toolbarHeight),
				)
			}
			c.logger.Debug("Toolbar position calculated", "x", posX, "y", posY, "caretX", c.caretX, "caretY", c.caretY)

			// Show toolbar with position and state in one atomic operation
			c.uiManager.ShowToolbarWithState(posX, posY, ui.ToolbarState{
				ChineseMode:   c.chineseMode,
				FullWidth:     c.fullWidth,
				ChinesePunct:  c.chinesePunctuation,
				CapsLock:      c.capsLockOn,
				EffectiveMode: int(c.getEffectiveModeNoLock()),
			})
		}
	} else {
		// IME deactivated - always hide toolbar
		c.uiManager.SetToolbarVisible(false)
		// Also hide candidate window
		c.hideUI()
	}
}

// HandleMenuCommand handles menu commands from C++ (toggle_mode, toggle_width, toggle_punct, open_settings, toggle_toolbar)
func (c *Coordinator) HandleMenuCommand(command string) *bridge.StatusUpdateData {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("HandleMenuCommand", "command", command)

	needBroadcast := false

	switch command {
	case "toggle_mode":
		c.chineseMode = !c.chineseMode
		c.logger.Debug("Mode toggled via menu", "chineseMode", c.chineseMode)

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

		// Save runtime state
		c.saveRuntimeState()

		// Show mode indicator
		c.showModeIndicator()

		needBroadcast = true

	case "toggle_width":
		c.fullWidth = !c.fullWidth
		c.logger.Debug("Full-width toggled via menu", "fullWidth", c.fullWidth)

		// Show indicator
		indicator := "半"
		if c.fullWidth {
			indicator = "全"
		}
		c.showIndicator(indicator)

		// Save runtime state
		c.saveRuntimeState()

		needBroadcast = true

	case "toggle_punct":
		c.chinesePunctuation = !c.chinesePunctuation
		c.logger.Debug("Chinese punctuation toggled via menu", "chinesePunctuation", c.chinesePunctuation)

		// Reset punctuation converter state
		c.punctConverter.Reset()

		// Show indicator
		indicator := "英."
		if c.chinesePunctuation {
			indicator = "中，"
		}
		c.showIndicator(indicator)

		// Save runtime state
		c.saveRuntimeState()

		needBroadcast = true

	case "toggle_toolbar":
		c.toolbarVisible = !c.toolbarVisible
		c.logger.Debug("Toolbar visibility toggled via menu", "toolbarVisible", c.toolbarVisible)

		// Update UI
		if c.uiManager != nil {
			if c.toolbarVisible && c.imeActivated {
				// Calculate position based on current caret
				// Note: coordinates can be negative in multi-monitor setups, use caretValid flag
				toolbarWidth, toolbarHeight := 140, 30
				var posX, posY int
				if c.caretValid {
					posX, posY = ui.GetToolbarPositionForCaret(
						c.caretX, c.caretY,
						ui.ScaleIntForDPI(toolbarWidth),
						ui.ScaleIntForDPI(toolbarHeight),
					)
				} else {
					posX, posY = ui.GetDefaultToolbarPosition(
						ui.ScaleIntForDPI(toolbarWidth),
						ui.ScaleIntForDPI(toolbarHeight),
					)
				}
				c.uiManager.ShowToolbarWithState(posX, posY, ui.ToolbarState{
					ChineseMode:   c.chineseMode,
					FullWidth:     c.fullWidth,
					ChinesePunct:  c.chinesePunctuation,
					CapsLock:      c.capsLockOn,
					EffectiveMode: int(c.getEffectiveModeNoLock()),
				})
			} else {
				c.uiManager.SetToolbarVisible(false)
			}
		}

		// Save to config
		c.saveToolbarConfig()

		needBroadcast = true

	case "open_settings":
		c.logger.Info("Opening settings requested")
		// Open settings window (will be implemented in UI)
		if c.uiManager != nil {
			c.uiManager.OpenSettings()
		}

	case "open_dictionary":
		c.logger.Info("Opening dictionary manager requested")
		if c.uiManager != nil {
			c.uiManager.OpenSettingsWithPage("dictionary")
		}

	case "show_about":
		c.logger.Info("Showing about dialog requested")
		if c.uiManager != nil {
			c.uiManager.OpenSettingsWithPage("about")
		}

	case "exit":
		c.logger.Info("Exit requested from menu")
		// TODO: Signal application exit
		// For now, just log the request
	}

	// Broadcast state to all clients if needed
	if needBroadcast {
		c.broadcastState()
	}

	// Return current status
	return c.buildStatusUpdate()
}

// getStatusUpdate returns the current status (caller must hold lock)
func (c *Coordinator) getStatusUpdate() *bridge.StatusUpdateData {
	return &bridge.StatusUpdateData{
		ChineseMode:        c.chineseMode,
		FullWidth:          c.fullWidth,
		ChinesePunctuation: c.chinesePunctuation,
		ToolbarVisible:     c.toolbarVisible,
		CapsLock:           c.capsLockOn,
	}
}

// showIndicator shows a brief indicator text
func (c *Coordinator) showIndicator(text string) {
	if c.uiManager == nil || !c.uiManager.IsReady() {
		return
	}

	x, y := c.getIndicatorPosition()
	c.uiManager.ShowModeIndicator(text, x, y)
}

// saveToolbarConfig saves the toolbar configuration to file
func (c *Coordinator) saveToolbarConfig() {
	go func() {
		cfg, err := config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}

		cfg.Toolbar.Visible = c.toolbarVisible

		if err := config.Save(cfg); err != nil {
			c.logger.Error("Failed to save toolbar config", "error", err)
		} else {
			c.logger.Debug("Toolbar config saved")
		}
	}()
}

// saveThemeConfig saves the theme name to config
func (c *Coordinator) saveThemeConfig(themeName string) {
	go func() {
		cfg, err := config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}

		cfg.UI.Theme = themeName

		if err := config.Save(cfg); err != nil {
			c.logger.Error("Failed to save theme config", "error", err)
		} else {
			c.logger.Debug("Theme config saved", "theme", themeName)
		}
	}()
}

// GetFullWidth returns the current full-width mode state
func (c *Coordinator) GetFullWidth() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fullWidth
}

// GetChinesePunctuation returns the current Chinese punctuation mode state
func (c *Coordinator) GetChinesePunctuation() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chinesePunctuation
}

// GetToolbarVisible returns the current toolbar visibility state
func (c *Coordinator) GetToolbarVisible() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.toolbarVisible
}

// GetChineseMode returns the current Chinese mode state
func (c *Coordinator) GetChineseMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chineseMode
}

// TransformOutput applies full-width and punctuation transformations to output text
func (c *Coordinator) TransformOutput(text string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := text

	// Apply full-width conversion if enabled
	if c.fullWidth {
		result = transform.ToFullWidth(result)
	}

	return result
}

// TransformPunctuation transforms a punctuation character based on current settings
func (c *Coordinator) TransformPunctuation(r rune) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.chinesePunctuation {
		return string(r), false
	}

	// Use punctuation converter which handles paired punctuation (quotes)
	return c.punctConverter.ToChinesePunctStr(r)
}

// saveRuntimeState saves the current state if remember_last_state is enabled
func (c *Coordinator) saveRuntimeState() {
	if c.config == nil || !c.config.Startup.RememberLastState {
		return
	}

	go func() {
		state := &config.RuntimeState{
			ChineseMode:  c.chineseMode,
			FullWidth:    c.fullWidth,
			ChinesePunct: c.chinesePunctuation,
			EngineType:   c.GetCurrentEngineName(),
		}
		if err := config.SaveRuntimeState(state); err != nil {
			c.logger.Error("Failed to save runtime state", "error", err)
		} else {
			c.logger.Debug("Runtime state saved", "chineseMode", state.ChineseMode)
		}
	}()
}

// saveRuntimeStateNoLock saves runtime state without acquiring lock (caller must hold lock)
func (c *Coordinator) saveRuntimeStateNoLock() {
	if c.config == nil || !c.config.Startup.RememberLastState {
		return
	}

	// Capture values while we hold the lock
	state := &config.RuntimeState{
		ChineseMode:  c.chineseMode,
		FullWidth:    c.fullWidth,
		ChinesePunct: c.chinesePunctuation,
		EngineType:   c.getCurrentEngineNameNoLock(),
	}

	go func() {
		if err := config.SaveRuntimeState(state); err != nil {
			c.logger.Error("Failed to save runtime state", "error", err)
		} else {
			c.logger.Debug("Runtime state saved", "chineseMode", state.ChineseMode)
		}
	}()
}
