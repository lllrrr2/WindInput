package main

import (
	"fmt"
	"os"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/huanfeng/wind_input/pkg/dictfile"
)

// ========== 短语管理 ==========

// PhraseItem 短语项（用于前端）
type PhraseItem struct {
	Code     string `json:"code"`
	Text     string `json:"text"`
	Texts    string `json:"texts,omitempty"` // 数组映射：每个字符展开为独立候选
	Name     string `json:"name,omitempty"`  // 组显示名称
	Position int    `json:"position"`
	IsSystem bool   `json:"is_system"` // 是否为系统短语
	Disabled bool   `json:"disabled"`  // 是否被禁用
}

// GetPhrases 获取所有短语（用户短语）
func (a *App) GetPhrases() ([]PhraseItem, error) {
	if a.phraseEditor == nil {
		return nil, fmt.Errorf("phrase editor not initialized")
	}

	cfg := a.phraseEditor.GetPhrases()
	if cfg == nil {
		return []PhraseItem{}, nil
	}

	items := make([]PhraseItem, len(cfg.Phrases))
	for i, p := range cfg.Phrases {
		items[i] = PhraseItem{
			Code:     p.Code,
			Text:     p.Text,
			Texts:    p.Texts,
			Name:     p.Name,
			Position: p.Position,
		}
	}

	return items, nil
}

// GetSystemPhrases 获取系统短语
// 优先从用户目录的副本读取，不存在则用程序目录的原始文件
func (a *App) GetSystemPhrases() ([]PhraseItem, error) {
	// 优先使用用户目录副本
	editor := a.systemUserPhraseEditor
	if editor != nil {
		cfg := editor.GetPhrases()
		if cfg != nil && len(cfg.Phrases) > 0 {
			return a.phraseCfgToItems(cfg, true), nil
		}
	}
	// 降级到原始系统文件
	if a.systemPhraseEditor == nil {
		return []PhraseItem{}, nil
	}
	cfg := a.systemPhraseEditor.GetPhrases()
	if cfg == nil {
		return []PhraseItem{}, nil
	}
	return a.phraseCfgToItems(cfg, true), nil
}

// phraseCfgToItems 将短语配置转换为前端项
func (a *App) phraseCfgToItems(cfg *dictfile.PhrasesConfig, isSystem bool) []PhraseItem {
	items := make([]PhraseItem, len(cfg.Phrases))
	for i, p := range cfg.Phrases {
		items[i] = PhraseItem{
			Code:     p.Code,
			Text:     p.Text,
			Texts:    p.Texts,
			Name:     p.Name,
			Position: p.Position,
			IsSystem: isSystem,
			Disabled: p.Disabled,
		}
	}
	return items
}

// SavePhrases 保存用户短语配置
func (a *App) SavePhrases(items []PhraseItem) error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}

	cfg := &dictfile.PhrasesConfig{
		Phrases: make([]dictfile.PhraseConfig, len(items)),
	}
	for i, item := range items {
		cfg.Phrases[i] = dictfile.PhraseConfig{
			Code:     item.Code,
			Text:     item.Text,
			Texts:    item.Texts,
			Name:     item.Name,
			Position: item.Position,
		}
	}

	a.phraseEditor.SetPhrases(cfg)

	if err := a.phraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.phraseEditor.GetFilePath())
	go a.NotifyReload("phrases")

	return nil
}

// AddPhrase 添加用户短语
func (a *App) AddPhrase(code, text string, position int) error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}

	a.phraseEditor.AddPhrase(code, text, position)

	if err := a.phraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.phraseEditor.GetFilePath())
	go a.NotifyReload("phrases")

	return nil
}

// RemovePhrase 删除短语
func (a *App) RemovePhrase(code, text string) error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}

	if !a.phraseEditor.RemovePhrase(code, text) {
		return fmt.Errorf("phrase not found")
	}

	if err := a.phraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.phraseEditor.GetFilePath())
	go a.NotifyReload("phrases")

	return nil
}

// UpdatePhrase 编辑用户短语（修改 code/text/position）
func (a *App) UpdatePhrase(oldCode, oldText, newCode, newText string, newPosition int) error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}

	// 移除旧条目
	a.phraseEditor.RemovePhrase(oldCode, oldText)
	// 添加新条目
	a.phraseEditor.AddPhrase(newCode, newText, newPosition)

	if err := a.phraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.phraseEditor.GetFilePath())
	go a.NotifyReload("phrases")
	return nil
}

// OverrideSystemPhrase 覆盖系统短语（保存到覆盖文件，不修改系统文件和用户短语文件）
// 可用于修改 position 或禁用系统短语
func (a *App) OverrideSystemPhrase(code, text string, position int, disabled bool) error {
	if a.systemUserPhraseEditor == nil {
		return fmt.Errorf("system user phrase editor not initialized")
	}

	// 首次修改时，将完整系统文件复制到用户目录
	if err := a.ensureSystemPhraseCopy(); err != nil {
		return fmt.Errorf("复制系统短语失败: %w", err)
	}

	cfg := a.systemUserPhraseEditor.GetPhrases()
	if cfg == nil {
		return fmt.Errorf("system phrases not loaded")
	}

	// 在副本中查找并修改
	found := false
	for i, p := range cfg.Phrases {
		matched := p.Code == code && p.Text == text
		if !matched && p.Texts != "" {
			matched = p.Code == code && p.Texts != ""
		}
		if matched {
			cfg.Phrases[i].Position = position
			cfg.Phrases[i].Disabled = disabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("系统短语未找到: %s", code)
	}

	a.systemUserPhraseEditor.SetPhrases(cfg)

	if err := a.systemUserPhraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.systemUserPhraseEditor.GetFilePath())
	go a.NotifyReload("phrases")
	return nil
}

// ensureSystemPhraseCopy 确保用户目录有系统短语的副本
func (a *App) ensureSystemPhraseCopy() error {
	cfg := a.systemUserPhraseEditor.GetPhrases()
	if cfg != nil && len(cfg.Phrases) > 0 {
		return nil // 已有副本
	}

	// 从系统文件复制
	sysCfg := a.systemPhraseEditor.GetPhrases()
	if sysCfg == nil {
		return fmt.Errorf("system phrases not available")
	}

	a.systemUserPhraseEditor.SetPhrases(sysCfg)
	return a.systemUserPhraseEditor.Save()
}

// RemoveSystemPhraseOverride 移除对系统短语的覆盖（恢复为系统默认）
func (a *App) RemoveSystemPhraseOverride(code, text string) error {
	if a.systemUserPhraseEditor == nil {
		return fmt.Errorf("override phrase editor not initialized")
	}

	// 从覆盖文件中移除同 code+text 的覆盖
	a.systemUserPhraseEditor.RemovePhrase(code, text)

	if err := a.systemUserPhraseEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.systemUserPhraseEditor.GetFilePath())
	go a.NotifyReload("phrases")
	return nil
}

// ResetAllSystemPhraseOverrides 删除用户目录的系统短语副本（恢复全部为系统默认）
func (a *App) ResetAllSystemPhraseOverrides() error {
	if a.systemUserPhraseEditor == nil {
		return fmt.Errorf("system user phrase editor not initialized")
	}

	// 删除用户目录的副本文件
	filePath := a.systemUserPhraseEditor.GetFilePath()
	_ = os.Remove(filePath)

	// 清空编辑器内存数据
	a.systemUserPhraseEditor.SetPhrases(&dictfile.PhrasesConfig{})
	a.fileWatcher.UpdateState(filePath)
	go a.NotifyReload("phrases")
	return nil
}

// CheckPhrasesModified 检查短语是否被外部修改
func (a *App) CheckPhrasesModified() (bool, error) {
	if a.phraseEditor == nil {
		return false, nil
	}
	return a.phraseEditor.HasChanged()
}

// ReloadPhrases 重新加载短语
func (a *App) ReloadPhrases() error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}
	return a.phraseEditor.Reload()
}

// ExportUserPhrases 导出用户短语文件
func (a *App) ExportUserPhrases() (string, error) {
	if a.phraseEditor == nil {
		return "", fmt.Errorf("phrase editor not initialized")
	}
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出用户短语",
		DefaultFilename: "user.phrases.yaml",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "YAML 文件", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}
	data, err := os.ReadFile(a.phraseEditor.GetFilePath())
	if err != nil {
		return "", fmt.Errorf("读取用户短语失败: %w", err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("导出失败: %w", err)
	}
	return savePath, nil
}

// ImportUserPhrases 导入用户短语文件
func (a *App) ImportUserPhrases() error {
	if a.phraseEditor == nil {
		return fmt.Errorf("phrase editor not initialized")
	}
	openPath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "导入用户短语",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "YAML 文件", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil || openPath == "" {
		return err
	}
	data, err := os.ReadFile(openPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	if err := os.WriteFile(a.phraseEditor.GetFilePath(), data, 0644); err != nil {
		return fmt.Errorf("导入失败: %w", err)
	}
	a.phraseEditor.Reload()
	a.fileWatcher.UpdateState(a.phraseEditor.GetFilePath())
	go a.NotifyReload("phrases")
	return nil
}

// ExportSystemPhrases 导出完整系统短语（优先用户副本，否则原始系统文件）
func (a *App) ExportSystemPhrases() (string, error) {
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出系统短语",
		DefaultFilename: "system.phrases.yaml",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "YAML 文件", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}
	// 优先用户副本
	var data []byte
	if a.systemUserPhraseEditor != nil {
		data, err = os.ReadFile(a.systemUserPhraseEditor.GetFilePath())
	}
	if len(data) == 0 && a.systemPhraseEditor != nil {
		data, err = os.ReadFile(a.systemPhraseEditor.GetFilePath())
	}
	if err != nil {
		return "", fmt.Errorf("读取系统短语失败: %w", err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("导出失败: %w", err)
	}
	return savePath, nil
}

// ImportSystemPhrases 导入系统短语文件（替换用户目录的副本）
func (a *App) ImportSystemPhrases() error {
	if a.systemUserPhraseEditor == nil {
		return fmt.Errorf("system user phrase editor not initialized")
	}
	openPath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "导入系统短语",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "YAML 文件", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil || openPath == "" {
		return err
	}
	data, err := os.ReadFile(openPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	if err := os.WriteFile(a.systemUserPhraseEditor.GetFilePath(), data, 0644); err != nil {
		return fmt.Errorf("导入失败: %w", err)
	}
	a.systemUserPhraseEditor.Reload()
	a.fileWatcher.UpdateState(a.systemUserPhraseEditor.GetFilePath())
	go a.NotifyReload("phrases")
	return nil
}
