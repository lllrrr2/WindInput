package main

import (
	"fmt"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/huanfeng/wind_input/pkg/config"

	"wind_setting/internal/editor"
)

// ========== 用户词库管理 ==========

// UserWordItem 用户词条（用于前端）
type UserWordItem struct {
	Code      string `json:"code"`
	Text      string `json:"text"`
	Weight    int    `json:"weight"`
	CreatedAt string `json:"created_at"`
}

// ImportExportResult 导入导出操作结果
type ImportExportResult struct {
	Cancelled bool   `json:"cancelled"`
	Count     int    `json:"count"`
	Total     int    `json:"total,omitempty"`
	Path      string `json:"path,omitempty"`
}

// GetUserDict 获取用户词库
func (a *App) GetUserDict() ([]UserWordItem, error) {
	if a.userDictEditor == nil {
		return nil, fmt.Errorf("user dict editor not initialized")
	}

	data := a.userDictEditor.GetUserDict()
	if data == nil {
		return []UserWordItem{}, nil
	}

	items := make([]UserWordItem, len(data.Words))
	for i, w := range data.Words {
		items[i] = UserWordItem{
			Code:      w.Code,
			Text:      w.Text,
			Weight:    w.Weight,
			CreatedAt: w.CreatedAt.Format(time.RFC3339),
		}
	}

	return items, nil
}

// AddUserWord 添加用户词条
func (a *App) AddUserWord(code, text string, weight int) error {
	if a.userDictEditor == nil {
		return fmt.Errorf("user dict editor not initialized")
	}

	a.userDictEditor.AddWord(code, text, weight)

	if err := a.userDictEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.userDictEditor.GetFilePath())
	go a.NotifyReload("userdict")

	return nil
}

// RemoveUserWord 删除用户词条
func (a *App) RemoveUserWord(code, text string) error {
	if a.userDictEditor == nil {
		return fmt.Errorf("user dict editor not initialized")
	}

	if !a.userDictEditor.RemoveWord(code, text) {
		return fmt.Errorf("word not found")
	}

	if err := a.userDictEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.userDictEditor.GetFilePath())
	go a.NotifyReload("userdict")

	return nil
}

// SearchUserDict 搜索用户词库
func (a *App) SearchUserDict(query string, limit int) ([]UserWordItem, error) {
	if a.userDictEditor == nil {
		return nil, fmt.Errorf("user dict editor not initialized")
	}

	words := a.userDictEditor.SearchWords(query, limit)
	items := make([]UserWordItem, len(words))
	for i, w := range words {
		items[i] = UserWordItem{
			Code:      w.Code,
			Text:      w.Text,
			Weight:    w.Weight,
			CreatedAt: w.CreatedAt.Format(time.RFC3339),
		}
	}

	return items, nil
}

// GetUserDictStats 获取用户词库统计
func (a *App) GetUserDictStats() map[string]int {
	stats := make(map[string]int)

	if a.userDictEditor != nil {
		stats["word_count"] = a.userDictEditor.GetWordCount()
	}
	if a.phraseEditor != nil {
		stats["phrase_count"] = a.phraseEditor.GetPhraseCount()
	}
	if a.shadowEditor != nil {
		stats["shadow_count"] = a.shadowEditor.GetRuleCount()
	}

	return stats
}

// CheckUserDictModified 检查用户词库是否被外部修改
func (a *App) CheckUserDictModified() (bool, error) {
	if a.userDictEditor == nil {
		return false, nil
	}
	return a.userDictEditor.HasChanged()
}

// ReloadUserDict 重新加载用户词库
func (a *App) ReloadUserDict() error {
	if a.userDictEditor == nil {
		return fmt.Errorf("user dict editor not initialized")
	}
	return a.userDictEditor.Reload()
}

// GetUserDictSchemaID 获取当前用户词库对应的方案 ID
func (a *App) GetUserDictSchemaID() string {
	cfg, err := config.Load()
	if err != nil {
		return "wubi86"
	}
	if cfg.Schema.Active != "" {
		return cfg.Schema.Active
	}
	if len(cfg.Schema.Available) > 0 {
		return cfg.Schema.Available[0]
	}
	return "wubi86"
}

// SwitchUserDictSchema 切换用户词库到指定方案
func (a *App) SwitchUserDictSchema(schemaID string) error {
	// 先保存当前词库
	if a.userDictEditor != nil {
		a.userDictEditor.Save()
		// 取消旧文件的监控
		a.fileWatcher.Unwatch(a.userDictEditor.GetFilePath())
	}

	// 创建新方案的词库编辑器
	newEditor, err := editor.NewUserDictEditorForSchema(schemaID)
	if err != nil {
		return fmt.Errorf("failed to create user dict editor: %w", err)
	}

	if err := newEditor.Load(); err != nil {
		return fmt.Errorf("failed to load user dict: %w", err)
	}

	a.userDictEditor = newEditor
	a.fileWatcher.Watch(a.userDictEditor.GetFilePath())
	return nil
}

// ImportUserDict 从文件导入用户词库
func (a *App) ImportUserDict() (*ImportExportResult, error) {
	if a.userDictEditor == nil {
		return nil, fmt.Errorf("user dict editor not initialized")
	}

	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "导入用户词库",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "词库文件 (*.txt)",
				Pattern:     "*.txt",
			},
			{
				DisplayName: "所有文件 (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}

	if path == "" {
		return &ImportExportResult{Cancelled: true}, nil
	}

	count, err := a.userDictEditor.ImportFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("导入失败: %w", err)
	}

	if err := a.userDictEditor.Save(); err != nil {
		return nil, fmt.Errorf("保存失败: %w", err)
	}

	a.fileWatcher.UpdateState(a.userDictEditor.GetFilePath())
	go a.NotifyReload("userdict")

	return &ImportExportResult{
		Count: count,
		Total: a.userDictEditor.GetWordCount(),
	}, nil
}

// ExportUserDict 导出用户词库到文件
func (a *App) ExportUserDict() (*ImportExportResult, error) {
	if a.userDictEditor == nil {
		return nil, fmt.Errorf("user dict editor not initialized")
	}

	defaultFilename := fmt.Sprintf("user_dict_%s.txt", time.Now().Format("20060102"))

	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出用户词库",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "词库文件 (*.txt)",
				Pattern:     "*.txt",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开保存对话框失败: %w", err)
	}

	if path == "" {
		return &ImportExportResult{Cancelled: true}, nil
	}

	if err := a.userDictEditor.ExportToFile(path); err != nil {
		return nil, fmt.Errorf("导出失败: %w", err)
	}

	return &ImportExportResult{
		Count: a.userDictEditor.GetWordCount(),
		Path:  path,
	}, nil
}

// ========== 导入导出（按方案） ==========

// ImportUserDictForSchema 导入指定方案的用户词库
func (a *App) ImportUserDictForSchema(schemaID string) (*ImportExportResult, error) {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return nil, err
	}

	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "导入用户词库",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "词库文件 (*.txt)", Pattern: "*.txt"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if path == "" {
		return &ImportExportResult{Cancelled: true}, nil
	}

	count, err := ed.ImportFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("导入失败: %w", err)
	}

	if err := ed.Save(); err != nil {
		return nil, fmt.Errorf("保存失败: %w", err)
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("userdict")

	return &ImportExportResult{
		Count: count,
		Total: ed.GetWordCount(),
	}, nil
}

// ExportUserDictForSchema 导出指定方案的用户词库
func (a *App) ExportUserDictForSchema(schemaID string) (*ImportExportResult, error) {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return nil, err
	}

	defaultFilename := fmt.Sprintf("user_dict_%s_%s.txt", schemaID, time.Now().Format("20060102"))
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出用户词库",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "词库文件 (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开保存对话框失败: %w", err)
	}
	if path == "" {
		return &ImportExportResult{Cancelled: true}, nil
	}

	if err := ed.ExportToFile(path); err != nil {
		return nil, fmt.Errorf("导出失败: %w", err)
	}

	return &ImportExportResult{
		Count: ed.GetWordCount(),
		Path:  path,
	}, nil
}
