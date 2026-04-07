package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/huanfeng/wind_input/pkg/config"

	"wind_setting/internal/editor"
)

// ========== 按方案操作词库（左右分栏 UI） ==========

// SchemaDictStats 方案词库统计信息
type SchemaDictStats struct {
	SchemaID      string `json:"schema_id"`
	SchemaName    string `json:"schema_name"`
	IconLabel     string `json:"icon_label"`
	WordCount     int    `json:"word_count"`
	ShadowCount   int    `json:"shadow_count"`
	TempWordCount int    `json:"temp_word_count"`
}

// TempWordItem 临时词条（前端展示用）
type TempWordItem struct {
	Code   string `json:"code"`
	Text   string `json:"text"`
	Weight int    `json:"weight"`
	Count  int    `json:"count"` // 选择次数
}

// getOrCreateSchemaUserDictEditor 获取或创建指定方案的用户词库编辑器
func (a *App) getOrCreateSchemaUserDictEditor(schemaID string) (*editor.UserDictEditor, error) {
	if ed, ok := a.schemaUserDicts[schemaID]; ok {
		return ed, nil
	}

	cfg, err := a.GetSchemaConfig(schemaID)
	if err != nil {
		return nil, fmt.Errorf("获取方案配置失败: %w", err)
	}

	if cfg.UserData.UserDictFile == "" {
		return nil, fmt.Errorf("方案 %s 未配置用户词库文件", schemaID)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	path := filepath.Join(configDir, cfg.UserData.UserDictFile)
	ed := editor.NewUserDictEditorWithPath(path)
	ed.Load() // 文件不存在是正常的

	a.schemaUserDicts[schemaID] = ed
	return ed, nil
}

// getOrCreateSchemaShadowEditor 获取或创建指定方案的 Shadow 编辑器
func (a *App) getOrCreateSchemaShadowEditor(schemaID string) (*editor.ShadowEditor, error) {
	if ed, ok := a.schemaShadows[schemaID]; ok {
		return ed, nil
	}

	cfg, err := a.GetSchemaConfig(schemaID)
	if err != nil {
		return nil, fmt.Errorf("获取方案配置失败: %w", err)
	}

	if cfg.UserData.ShadowFile == "" {
		return nil, fmt.Errorf("方案 %s 未配置 Shadow 文件", schemaID)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	path := filepath.Join(configDir, cfg.UserData.ShadowFile)
	ed := editor.NewShadowEditorWithPath(path)
	ed.Load() // 文件不存在是正常的

	a.schemaShadows[schemaID] = ed
	return ed, nil
}

// getOrCreateSchemaTempDictEditor 获取或创建指定方案的临时词库编辑器
func (a *App) getOrCreateSchemaTempDictEditor(schemaID string) (*editor.UserDictEditor, error) {
	if ed, ok := a.schemaTempDicts[schemaID]; ok {
		return ed, nil
	}

	cfg, err := a.GetSchemaConfig(schemaID)
	if err != nil {
		return nil, fmt.Errorf("获取方案配置失败: %w", err)
	}

	tempFile := cfg.UserData.TempDictFile
	if tempFile == "" {
		// 默认临时词库文件名
		tempFile = "temp_words_" + schemaID + ".txt"
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	path := filepath.Join(configDir, tempFile)
	ed := editor.NewUserDictEditorWithPath(path)
	ed.Load() // 文件不存在是正常的

	a.schemaTempDicts[schemaID] = ed
	return ed, nil
}

// GetEnabledSchemasWithDictStats 获取所有已启用方案及其词库统计
func (a *App) GetEnabledSchemasWithDictStats() ([]SchemaDictStats, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	schemas, err := a.GetAvailableSchemas()
	if err != nil {
		return nil, err
	}

	// 建立 ID→SchemaInfo 映射
	schemaMap := make(map[string]SchemaInfo)
	for _, s := range schemas {
		schemaMap[s.ID] = s
	}

	// 获取引用关系，用于过滤引用式混输方案（其用户数据继承自主方案）
	refs, _ := a.GetSchemaReferences()

	var result []SchemaDictStats
	for _, schemaID := range cfg.Schema.Available {
		info, ok := schemaMap[schemaID]
		if !ok {
			continue
		}

		// 引用式混输方案的用户数据与主方案共享，不在词库管理中重复显示
		if ref, ok := refs[schemaID]; ok && (ref.PrimarySchema != "" || ref.SecondarySchema != "") {
			continue
		}

		stats := SchemaDictStats{
			SchemaID:   schemaID,
			SchemaName: info.Name,
			IconLabel:  info.IconLabel,
		}

		if ud, err := a.getOrCreateSchemaUserDictEditor(schemaID); err == nil {
			stats.WordCount = ud.GetWordCount()
		}

		if sd, err := a.getOrCreateSchemaShadowEditor(schemaID); err == nil {
			stats.ShadowCount = sd.GetRuleCount()
		}

		if td, err := a.getOrCreateSchemaTempDictEditor(schemaID); err == nil {
			stats.TempWordCount = td.GetWordCount()
		}

		result = append(result, stats)
	}

	// 添加被引用但未启用的方案（如混输引用的 pinyin 方案有独立的 userfreq）
	addedIDs := make(map[string]bool)
	for _, s := range result {
		addedIDs[s.SchemaID] = true
	}
	refIDs, _ := a.GetReferencedSchemaIDs()
	for _, refID := range refIDs {
		if addedIDs[refID] {
			continue
		}
		info, ok := schemaMap[refID]
		if !ok {
			continue
		}
		stats := SchemaDictStats{
			SchemaID:   refID,
			SchemaName: info.Name,
			IconLabel:  info.IconLabel,
		}
		if ud, err := a.getOrCreateSchemaUserDictEditor(refID); err == nil {
			stats.WordCount = ud.GetWordCount()
		}
		if sd, err := a.getOrCreateSchemaShadowEditor(refID); err == nil {
			stats.ShadowCount = sd.GetRuleCount()
		}
		if td, err := a.getOrCreateSchemaTempDictEditor(refID); err == nil {
			stats.TempWordCount = td.GetWordCount()
		}
		result = append(result, stats)
	}

	return result, nil
}

// GetUserDictBySchema 获取指定方案的用户词库
func (a *App) GetUserDictBySchema(schemaID string) ([]UserWordItem, error) {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return nil, err
	}

	data := ed.GetUserDict()
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

// AddUserWordForSchema 向指定方案添加用户词条
func (a *App) AddUserWordForSchema(schemaID, code, text string, weight int) error {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return err
	}

	ed.AddWord(code, text, weight)

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("userdict")
	return nil
}

// RemoveUserWordForSchema 从指定方案删除用户词条
func (a *App) RemoveUserWordForSchema(schemaID, code, text string) error {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return err
	}

	if !ed.RemoveWord(code, text) {
		return fmt.Errorf("word not found")
	}

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("userdict")
	return nil
}

// SearchUserDictBySchema 搜索指定方案的用户词库
func (a *App) SearchUserDictBySchema(schemaID, query string, limit int) ([]UserWordItem, error) {
	ed, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return nil, err
	}

	words := ed.SearchWords(query, limit)
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

// GetShadowBySchema 获取指定方案的 Shadow 规则
func (a *App) GetShadowBySchema(schemaID string) ([]ShadowRuleItem, error) {
	ed, err := a.getOrCreateSchemaShadowEditor(schemaID)
	if err != nil {
		return nil, err
	}

	cfg := ed.GetShadowConfig()
	if cfg == nil {
		return []ShadowRuleItem{}, nil
	}

	var items []ShadowRuleItem
	for code, cc := range cfg.Rules {
		for _, p := range cc.Pinned {
			items = append(items, ShadowRuleItem{
				Code:     code,
				Word:     p.Word,
				Type:     "pin",
				Position: p.Position,
			})
		}
		for _, d := range cc.Deleted {
			items = append(items, ShadowRuleItem{
				Code: code,
				Word: d,
				Type: "delete",
			})
		}
	}

	return items, nil
}

// PinShadowWordForSchema 在指定方案中固定词到指定位置
func (a *App) PinShadowWordForSchema(schemaID, code, word string, position int) error {
	ed, err := a.getOrCreateSchemaShadowEditor(schemaID)
	if err != nil {
		return err
	}

	ed.PinWord(code, word, position)

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("shadow")
	return nil
}

// DeleteShadowWordForSchema 在指定方案中隐藏词条
func (a *App) DeleteShadowWordForSchema(schemaID, code, word string) error {
	ed, err := a.getOrCreateSchemaShadowEditor(schemaID)
	if err != nil {
		return err
	}

	ed.DeleteWord(code, word)

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("shadow")
	return nil
}

// RemoveShadowRuleForSchema 在指定方案中删除规则
func (a *App) RemoveShadowRuleForSchema(schemaID, code, word string) error {
	ed, err := a.getOrCreateSchemaShadowEditor(schemaID)
	if err != nil {
		return err
	}

	if !ed.RemoveRule(code, word) {
		return fmt.Errorf("rule not found")
	}

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("shadow")
	return nil
}

// GetTempDictBySchema 获取指定方案的临时词库
func (a *App) GetTempDictBySchema(schemaID string) ([]TempWordItem, error) {
	ed, err := a.getOrCreateSchemaTempDictEditor(schemaID)
	if err != nil {
		return nil, err
	}

	data := ed.GetUserDict()
	if data == nil {
		return []TempWordItem{}, nil
	}

	items := make([]TempWordItem, len(data.Words))
	for i, w := range data.Words {
		items[i] = TempWordItem{
			Code:   w.Code,
			Text:   w.Text,
			Weight: w.Weight,
			Count:  0, // dictfile.UserWord 没有 Count 字段，暂为 0
		}
	}
	return items, nil
}

// ClearTempDictForSchema 清空指定方案的临时词库
func (a *App) ClearTempDictForSchema(schemaID string) (int, error) {
	ed, err := a.getOrCreateSchemaTempDictEditor(schemaID)
	if err != nil {
		return 0, err
	}

	data := ed.GetUserDict()
	count := 0
	if data != nil {
		count = len(data.Words)
	}

	// 清空并保存
	ed.SetUserDict(nil)
	if err := ed.Save(); err != nil {
		return 0, err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("tempdict")
	return count, nil
}

// PromoteTempWordForSchema 将临时词条晋升到用户词库
func (a *App) PromoteTempWordForSchema(schemaID, code, text string) error {
	tempEd, err := a.getOrCreateSchemaTempDictEditor(schemaID)
	if err != nil {
		return err
	}

	userEd, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return err
	}

	// 查找词条获取权重
	data := tempEd.GetUserDict()
	if data == nil {
		return fmt.Errorf("临时词库为空")
	}

	weight := 100
	for _, w := range data.Words {
		if w.Code == code && w.Text == text {
			weight = w.Weight
			break
		}
	}

	// 添加到用户词库
	userEd.AddWord(code, text, weight)
	if err := userEd.Save(); err != nil {
		return fmt.Errorf("保存用户词库失败: %w", err)
	}

	// 从临时词库移除
	tempEd.RemoveWord(code, text)
	if err := tempEd.Save(); err != nil {
		return fmt.Errorf("保存临时词库失败: %w", err)
	}

	a.fileWatcher.UpdateState(userEd.GetFilePath())
	a.fileWatcher.UpdateState(tempEd.GetFilePath())
	go a.NotifyReload("userdict")
	go a.NotifyReload("tempdict")
	return nil
}

// PromoteAllTempWordsForSchema 将所有临时词条晋升到用户词库
func (a *App) PromoteAllTempWordsForSchema(schemaID string) (int, error) {
	tempEd, err := a.getOrCreateSchemaTempDictEditor(schemaID)
	if err != nil {
		return 0, err
	}

	userEd, err := a.getOrCreateSchemaUserDictEditor(schemaID)
	if err != nil {
		return 0, err
	}

	data := tempEd.GetUserDict()
	if data == nil || len(data.Words) == 0 {
		return 0, nil
	}

	count := 0
	for _, w := range data.Words {
		userEd.AddWord(w.Code, w.Text, w.Weight)
		count++
	}

	if err := userEd.Save(); err != nil {
		return 0, fmt.Errorf("保存用户词库失败: %w", err)
	}

	// 清空临时词库
	ed, _ := a.getOrCreateSchemaTempDictEditor(schemaID)
	ed.SetUserDict(nil)
	if err := ed.Save(); err != nil {
		return count, fmt.Errorf("清空临时词库失败: %w", err)
	}

	go a.NotifyReload("userdict")
	go a.NotifyReload("tempdict")
	return count, nil
}

// RemoveTempWordForSchema 从临时词库删除词条
func (a *App) RemoveTempWordForSchema(schemaID, code, text string) error {
	ed, err := a.getOrCreateSchemaTempDictEditor(schemaID)
	if err != nil {
		return err
	}

	if !ed.RemoveWord(code, text) {
		return fmt.Errorf("word not found")
	}

	if err := ed.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(ed.GetFilePath())
	go a.NotifyReload("tempdict")
	return nil
}
