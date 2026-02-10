// handle_ui_callbacks.go — 工具栏与候选窗口 UI 回调
package coordinator

import (
	"github.com/huanfeng/wind_input/internal/transform"
	"github.com/huanfeng/wind_input/internal/ui"
	"github.com/huanfeng/wind_input/pkg/config"
)

// setupToolbarCallbacks sets up the callbacks for toolbar button clicks
// IMPORTANT: These callbacks are invoked from the UI thread (window procedure).
// We use goroutines to avoid blocking the UI thread with lock acquisition or I/O.
func (c *Coordinator) setupToolbarCallbacks() {
	if c.uiManager == nil {
		return
	}

	c.uiManager.SetToolbarCallbacks(&ui.ToolbarCallback{
		OnToggleMode: func() {
			// Run in goroutine to avoid blocking UI thread
			go c.handleToolbarToggleMode()
		},
		OnToggleWidth: func() {
			go c.handleToolbarToggleWidth()
		},
		OnTogglePunct: func() {
			go c.handleToolbarTogglePunct()
		},
		OnOpenSettings: func() {
			go c.handleToolbarOpenSettings()
		},
		OnPositionChanged: func(x, y int) {
			go c.handleToolbarPositionChanged(x, y)
		},
		OnContextMenu: func(action ui.ToolbarContextMenuAction) {
			go c.handleToolbarContextMenu(action)
		},
		OnShowMenu: func(screenX, screenY, flipRefY int) {
			go c.handleShowUnifiedMenu(screenX, screenY, flipRefY)
		},
	})
}

// setupCandidateCallbacks sets up the callbacks for candidate window mouse interactions
// IMPORTANT: These callbacks are invoked from the UI thread (window procedure).
// We use goroutines to avoid blocking the UI thread with lock acquisition or I/O.
func (c *Coordinator) setupCandidateCallbacks() {
	if c.uiManager == nil {
		return
	}

	c.uiManager.SetCandidateCallbacks(&ui.CandidateCallback{
		OnSelect: func(index int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateSelect(index)
		},
		OnHoverChange: func(index, tooltipX, tooltipY int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateHoverChange(index, tooltipX, tooltipY)
		},
		OnPageUp: func() {
			// Run in goroutine to avoid blocking UI thread
			go func() {
				c.mu.Lock()
				defer c.mu.Unlock()
				c.handlePageUp()
			}()
		},
		OnPageDown: func() {
			// Run in goroutine to avoid blocking UI thread
			go func() {
				c.mu.Lock()
				defer c.mu.Unlock()
				c.handlePageDown()
			}()
		},
		OnMoveUp: func(index int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateMoveUp(index)
		},
		OnMoveDown: func(index int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateMoveDown(index)
		},
		OnMoveTop: func(index int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateMoveTop(index)
		},
		OnDelete: func(index int) {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateDelete(index)
		},
		OnOpenSettings: func() {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateOpenSettings()
		},
		OnAbout: func() {
			// Run in goroutine to avoid blocking UI thread
			go c.handleCandidateAbout()
		},
		OnShowUnifiedMenu: func(screenX, screenY int) {
			go c.handleShowUnifiedMenu(screenX, screenY, 0)
		},
	})
}

// handleCandidateSelect handles candidate selection via mouse click
func (c *Coordinator) handleCandidateSelect(index int) {
	c.mu.Lock()

	// Convert page-local index to actual candidate index
	actualIndex := (c.currentPage-1)*c.candidatesPerPage + index

	c.logger.Debug("Candidate selected via mouse", "pageIndex", index, "actualIndex", actualIndex, "currentPage", c.currentPage)

	if actualIndex < 0 || actualIndex >= len(c.candidates) {
		c.logger.Warn("Invalid candidate index", "actualIndex", actualIndex, "candidateCount", len(c.candidates))
		c.mu.Unlock()
		return
	}

	candidate := c.candidates[actualIndex]
	text := candidate.Text

	// Apply full-width conversion if enabled
	if c.fullWidth {
		text = transform.ToFullWidth(text)
	}

	c.logger.Debug("Candidate selected via mouse click", "index", actualIndex)

	// Clear state and hide UI
	c.clearState()
	c.hideUI()

	// Get bridge server reference (release lock before network I/O)
	bridgeServer := c.bridgeServer
	c.mu.Unlock()

	// Send text to TSF via push pipe (only to active client for security)
	if bridgeServer != nil && text != "" {
		bridgeServer.PushCommitTextToActiveClient(text)
	}
}

// handleCandidateHoverChange handles hover state change
func (c *Coordinator) handleCandidateHoverChange(index, tooltipX, tooltipY int) {
	c.logger.Debug("Candidate hover changed", "index", index, "tooltipX", tooltipX, "tooltipY", tooltipY)

	// Refresh the candidate window to show/hide hover highlight
	if c.uiManager != nil {
		c.uiManager.RefreshCandidates()

		// Show/hide tooltip for encoding lookup
		if index >= 0 {
			c.uiManager.ShowTooltipForCandidate(index, tooltipX, tooltipY)
		} else {
			c.uiManager.HideTooltip()
		}
	}
}

// handleCandidateMoveUp handles move up action from context menu
func (c *Coordinator) handleCandidateMoveUp(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Candidate move up requested", "index", index)

	if index <= 0 || index >= len(c.candidates) {
		c.logger.Warn("Invalid candidate index for move up", "index", index)
		return
	}

	c.logger.Debug("Request to move candidate up", "index", index)

	// TODO: Implement candidate priority adjustment
	// This would require:
	// 1. Swap priority/weight with the previous candidate
	// 2. Save to user dictionary or priority file
	// 3. Refresh the candidate list
}

// handleCandidateMoveDown handles move down action from context menu
func (c *Coordinator) handleCandidateMoveDown(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Candidate move down requested", "index", index)

	if index < 0 || index >= len(c.candidates)-1 {
		c.logger.Warn("Invalid candidate index for move down", "index", index)
		return
	}

	c.logger.Debug("Request to move candidate down", "index", index)

	// TODO: Implement candidate priority adjustment
}

// handleCandidateMoveTop handles move to top action from context menu
func (c *Coordinator) handleCandidateMoveTop(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Candidate move to top requested", "index", index)

	if index <= 0 || index >= len(c.candidates) {
		c.logger.Warn("Invalid candidate index for move to top", "index", index)
		return
	}

	c.logger.Debug("Request to move candidate to top", "index", index)

	// TODO: Implement candidate priority adjustment
	// Set this candidate's priority to be highest
}

// handleCandidateDelete handles delete action from context menu
func (c *Coordinator) handleCandidateDelete(index int) {
	c.mu.Lock()

	c.logger.Debug("Candidate delete requested", "index", index)

	if index < 0 || index >= len(c.candidates) {
		c.logger.Warn("Invalid candidate index for delete", "index", index)
		c.mu.Unlock()
		return
	}

	c.mu.Unlock()

	c.logger.Debug("Request to delete user word", "index", index)

	// Show confirmation dialog
	// TODO: Implement confirmation dialog via UI manager
	// For now, just log the request
	// if confirmed {
	//     // Delete from user dictionary
	//     // Refresh candidate list
	// }
}

// handleCandidateOpenSettings handles open settings action from context menu
func (c *Coordinator) handleCandidateOpenSettings() {
	c.logger.Info("Opening settings from candidate context menu")
	if c.uiManager != nil {
		c.uiManager.OpenSettings()
	}
}

// handleCandidateAbout handles about action from context menu
func (c *Coordinator) handleCandidateAbout() {
	c.logger.Info("Opening about page from candidate context menu")
	if c.uiManager != nil {
		c.uiManager.OpenSettingsWithPage("about")
	}
}

// handleToolbarToggleMode handles mode toggle from toolbar click
func (c *Coordinator) handleToolbarToggleMode() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chineseMode = !c.chineseMode
	c.logger.Info("Mode toggled via toolbar", "chineseMode", c.chineseMode)

	// Clear any pending input when switching modes
	hadInput := len(c.inputBuffer) > 0
	if hadInput {
		c.clearState()
		c.hideUI()
	}

	// Notify TSF side to clear inline composition if there was active input
	if hadInput && c.bridgeServer != nil {
		go c.bridgeServer.PushClearCompositionToActiveClient()
	}

	// Sync punctuation with mode if enabled
	if c.punctFollowMode {
		c.chinesePunctuation = c.chineseMode
	}

	// Reset punctuation converter state
	c.punctConverter.Reset()

	// Save runtime state if remember_last_state is enabled
	c.saveRuntimeState()

	// Broadcast state to toolbar and all TSF clients
	c.broadcastState()
}

// handleToolbarToggleWidth handles width toggle from toolbar click
func (c *Coordinator) handleToolbarToggleWidth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.fullWidth = !c.fullWidth
	c.logger.Debug("Full-width toggled via toolbar", "fullWidth", c.fullWidth)

	// Save runtime state if remember_last_state is enabled
	c.saveRuntimeState()

	// Broadcast state to toolbar and all TSF clients
	c.broadcastState()
}

// handleToolbarTogglePunct handles punctuation toggle from toolbar click
func (c *Coordinator) handleToolbarTogglePunct() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chinesePunctuation = !c.chinesePunctuation
	c.logger.Debug("Chinese punctuation toggled via toolbar", "chinesePunctuation", c.chinesePunctuation)

	// Reset punctuation converter state
	c.punctConverter.Reset()

	// Save runtime state if remember_last_state is enabled
	c.saveRuntimeState()

	// Broadcast state to toolbar and all TSF clients
	c.broadcastState()
}

// handleToolbarOpenSettings handles settings button click from toolbar
func (c *Coordinator) handleToolbarOpenSettings() {
	c.logger.Info("Opening settings from toolbar")
	if c.uiManager != nil {
		c.uiManager.OpenSettings()
	}
}

// handleToolbarPositionChanged handles toolbar position change (after dragging)
func (c *Coordinator) handleToolbarPositionChanged(x, y int) {
	c.logger.Debug("Toolbar position changed", "x", x, "y", y)
	c.saveToolbarPosition(x, y)
}

// handleToolbarContextMenu handles toolbar right-click context menu action
func (c *Coordinator) handleToolbarContextMenu(action ui.ToolbarContextMenuAction) {
	c.logger.Debug("Toolbar context menu action", "action", action)

	switch action {
	case ui.ToolbarMenuSettings:
		c.logger.Info("Opening settings from toolbar context menu")
		c.handleToolbarOpenSettings()

	case ui.ToolbarMenuRestartService:
		c.logger.Info("Restart service requested from toolbar context menu")
		c.resetAndResync()

	case ui.ToolbarMenuAbout:
		c.logger.Info("Opening about page from toolbar context menu")
		// Open settings with "about" parameter
		if c.uiManager != nil {
			c.uiManager.OpenSettingsWithPage("about")
		}
	}
}

// handleShowUnifiedMenu shows the unified context menu at the given screen position
func (c *Coordinator) handleShowUnifiedMenu(screenX, screenY, flipRefY int) {
	if c.uiManager == nil {
		return
	}

	c.mu.Lock()
	state := ui.UnifiedMenuState{
		ChineseMode:    c.chineseMode,
		FullWidth:      c.fullWidth,
		ChinesePunct:   c.chinesePunctuation,
		ToolbarVisible: c.toolbarVisible,
		Themes:         c.uiManager.GetAvailableThemes(),
		CurrentTheme:   c.uiManager.GetCurrentThemeName(),
	}
	c.mu.Unlock()

	c.uiManager.ShowUnifiedMenu(screenX, screenY, flipRefY, state, func(id int) {
		go c.handleUnifiedMenuAction(id)
	})
}

// handleUnifiedMenuAction handles a menu item selection from the unified menu
func (c *Coordinator) handleUnifiedMenuAction(id int) {
	switch {
	case id == ui.UnifiedMenuToggleMode:
		c.handleToolbarToggleMode()
	case id == ui.UnifiedMenuToggleWidth:
		c.handleToolbarToggleWidth()
	case id == ui.UnifiedMenuTogglePunct:
		c.handleToolbarTogglePunct()
	case id == ui.UnifiedMenuToggleToolbar:
		c.HandleMenuCommand("toggle_toolbar")
	case id == ui.UnifiedMenuDictionary:
		if c.uiManager != nil {
			c.uiManager.OpenSettingsWithPage("dictionary")
		}
	case id == ui.UnifiedMenuSettings:
		c.handleToolbarOpenSettings()
	case id == ui.UnifiedMenuAbout:
		if c.uiManager != nil {
			c.uiManager.OpenSettingsWithPage("about")
		}
	case id >= ui.UnifiedMenuThemeBase && id < ui.UnifiedMenuThemeBase+100:
		// Theme selection
		themeIndex := id - ui.UnifiedMenuThemeBase
		themes := c.uiManager.GetAvailableThemes()
		if themeIndex >= 0 && themeIndex < len(themes) {
			themeName := themes[themeIndex]
			c.logger.Info("Theme selected from unified menu", "theme", themeName)
			c.uiManager.LoadTheme(themeName)
			// Save to config
			c.saveThemeConfig(themeName)
		}
	}
}

// HandleShowContextMenu handles context menu request from TSF (bridge interface)
func (c *Coordinator) HandleShowContextMenu(screenX, screenY int) {
	c.handleShowUnifiedMenu(screenX, screenY, 0)
}

// resetAndResync restarts the Go service process
// It starts a new process and exits the current one
func (c *Coordinator) resetAndResync() {
	c.logger.Info("Restarting Go service process...")

	// Clear current state and hide UI
	c.mu.Lock()
	c.clearState()
	c.hideUI()
	c.mu.Unlock()

	// Request process restart through the restart manager
	RequestRestart()
}

// syncToolbarState synchronizes the current state to the toolbar
// Note: This should be called with lock held, or use broadcastState() instead
func (c *Coordinator) syncToolbarState() {
	c.syncToolbarStateNoLock()
}

// saveToolbarPosition saves the toolbar position to config
func (c *Coordinator) saveToolbarPosition(x, y int) {
	if c.config == nil {
		return
	}

	c.config.Toolbar.PositionX = x
	c.config.Toolbar.PositionY = y

	if err := config.Save(c.config); err != nil {
		c.logger.Error("Failed to save toolbar position", "error", err)
	}
}
