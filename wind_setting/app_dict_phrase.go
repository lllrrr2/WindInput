package main

import (
	"fmt"

	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// ========== 短语管理（通过 RPC）==========

// PhraseItem 短语条目（前端用）
type PhraseItem struct {
	Code     string `json:"code"`
	Text     string `json:"text,omitempty"`
	Texts    string `json:"texts,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	Enabled  bool   `json:"enabled"`
	IsSystem bool   `json:"is_system"`
}

// GetPhrases 获取所有短语（通过 RPC）
func (a *App) GetPhrases() ([]PhraseItem, error) {
	reply, err := a.rpcClient.PhraseList()
	if err != nil {
		return nil, fmt.Errorf("获取短语列表失败: %w", err)
	}
	items := make([]PhraseItem, len(reply.Phrases))
	for i, p := range reply.Phrases {
		items[i] = PhraseItem{
			Code: p.Code, Text: p.Text, Texts: p.Texts, Name: p.Name,
			Type: p.Type, Position: p.Position, Enabled: p.Enabled, IsSystem: p.IsSystem,
		}
	}
	return items, nil
}

// AddPhrase 添加短语
func (a *App) AddPhrase(code, text, texts, name, pType string, position int) error {
	return a.rpcClient.PhraseAdd(rpcapi.PhraseAddArgs{
		Code: code, Text: text, Texts: texts, Name: name, Type: pType, Position: position,
	})
}

// UpdatePhrase 更新短语
func (a *App) UpdatePhrase(code, text, name, newText string, newPosition int, enabled *bool) error {
	return a.rpcClient.PhraseUpdate(rpcapi.PhraseUpdateArgs{
		Code: code, Text: text, Name: name, NewText: newText, NewPosition: newPosition, Enabled: enabled,
	})
}

// RemovePhrase 删除短语
func (a *App) RemovePhrase(code, text, name string) error {
	return a.rpcClient.PhraseRemove(code, text, name)
}

// SetPhraseEnabled 设置短语启用/禁用状态
func (a *App) SetPhraseEnabled(code, text, name string, enabled bool) error {
	return a.rpcClient.PhraseUpdate(rpcapi.PhraseUpdateArgs{
		Code: code, Text: text, Name: name, Enabled: &enabled,
	})
}

// ResetPhrasesToDefault 重置短语为默认值
func (a *App) ResetPhrasesToDefault() error {
	return a.rpcClient.PhraseResetDefaults()
}

// ========== 词频管理 ==========

// FreqItem 词频条目
type FreqItem struct {
	Code     string `json:"code"`
	Text     string `json:"text"`
	Count    int    `json:"count"`
	LastUsed int64  `json:"last_used"`
	Streak   int    `json:"streak"`
	Boost    int    `json:"boost"`
}

// GetFreqList 搜索词频记录
func (a *App) GetFreqList(schemaID, prefix string, limit, offset int) (map[string]interface{}, error) {
	reply, err := a.rpcClient.FreqSearch(schemaID, prefix, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]FreqItem, len(reply.Entries))
	for i, e := range reply.Entries {
		items[i] = FreqItem{
			Code: e.Code, Text: e.Text, Count: e.Count,
			LastUsed: e.LastUsed, Streak: e.Streak, Boost: e.Boost,
		}
	}
	return map[string]interface{}{"entries": items, "total": reply.Total}, nil
}

// DeleteFreq 删除单条词频记录
func (a *App) DeleteFreq(schemaID, code, text string) error {
	return a.rpcClient.FreqDelete(schemaID, code, text)
}

// ClearFreq 清空指定方案的所有词频数据
func (a *App) ClearFreq(schemaID string) (int, error) {
	return a.rpcClient.FreqClear(schemaID)
}

// ========== 方案列表 ==========

// SchemaStatusItem 方案状态信息
type SchemaStatusItem struct {
	SchemaID    string `json:"schema_id"`
	SchemaName  string `json:"schema_name"`
	Status      string `json:"status"`
	UserWords   int    `json:"user_words"`
	TempWords   int    `json:"temp_words"`
	ShadowRules int    `json:"shadow_rules"`
	FreqRecords int    `json:"freq_records"`
}

// GetAllSchemaStatuses 获取所有方案状态
func (a *App) GetAllSchemaStatuses() ([]SchemaStatusItem, error) {
	reply, err := a.rpcClient.SystemListSchemas()
	if err != nil {
		return nil, err
	}

	// 从已有方法获取方案显示名称
	nameMap := make(map[string]string)
	if stats, err := a.GetEnabledSchemasWithDictStats(); err == nil {
		for _, s := range stats {
			nameMap[s.SchemaID] = s.SchemaName
		}
	}

	items := make([]SchemaStatusItem, len(reply.Schemas))
	for i, s := range reply.Schemas {
		name := nameMap[s.SchemaID]
		if name == "" {
			name = s.SchemaID
		}
		items[i] = SchemaStatusItem{
			SchemaID: s.SchemaID, SchemaName: name, Status: s.Status,
			UserWords: s.UserWords, TempWords: s.TempWords,
			ShadowRules: s.ShadowRules, FreqRecords: s.FreqRecords,
		}
	}
	return items, nil
}

// ========== 短语文件变化检测（已迁移到 RPC，保留空实现兼容前端）==========

// CheckPhrasesModified 检查短语是否被外部修改（RPC 模式下不再适用）
func (a *App) CheckPhrasesModified() (bool, error) {
	return false, nil
}

// ReloadPhrases 重新加载短语（RPC 模式下由服务端管理）
func (a *App) ReloadPhrases() error {
	return nil
}
