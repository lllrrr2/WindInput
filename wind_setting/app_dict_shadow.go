package main

import "fmt"

// ========== Shadow 管理 ==========

// ShadowRuleItem Shadow 规则项（用于前端）
type ShadowRuleItem struct {
	Code     string `json:"code"`
	Word     string `json:"word"`
	Type     string `json:"type"`     // "pin" 或 "delete"
	Position int    `json:"position"` // 仅 pin 有效
}

// GetShadowRules 获取所有 Shadow 规则
func (a *App) GetShadowRules() ([]ShadowRuleItem, error) {
	if a.shadowEditor == nil {
		return nil, fmt.Errorf("shadow editor not initialized")
	}

	cfg := a.shadowEditor.GetShadowConfig()
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

// PinShadowWord 固定词到指定位置
func (a *App) PinShadowWord(code, word string, position int) error {
	if a.shadowEditor == nil {
		return fmt.Errorf("shadow editor not initialized")
	}

	a.shadowEditor.PinWord(code, word, position)

	if err := a.shadowEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.shadowEditor.GetFilePath())
	go a.NotifyReload("shadow")

	return nil
}

// DeleteShadowWord 隐藏词条
func (a *App) DeleteShadowWord(code, word string) error {
	if a.shadowEditor == nil {
		return fmt.Errorf("shadow editor not initialized")
	}

	a.shadowEditor.DeleteWord(code, word)

	if err := a.shadowEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.shadowEditor.GetFilePath())
	go a.NotifyReload("shadow")

	return nil
}

// RemoveShadowRule 删除 Shadow 规则
func (a *App) RemoveShadowRule(code, word string) error {
	if a.shadowEditor == nil {
		return fmt.Errorf("shadow editor not initialized")
	}

	if !a.shadowEditor.RemoveRule(code, word) {
		return fmt.Errorf("rule not found")
	}

	if err := a.shadowEditor.Save(); err != nil {
		return err
	}

	a.fileWatcher.UpdateState(a.shadowEditor.GetFilePath())
	go a.NotifyReload("shadow")

	return nil
}
