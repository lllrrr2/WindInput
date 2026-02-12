package dict

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestShadowLayer(t *testing.T) *ShadowLayer {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "shadow.yaml")
	return NewShadowLayer("test_shadow", filePath)
}

func TestShadowLayerTop(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Top a word
	sl.Top("nihao", "你好")

	if !sl.IsTopped("nihao", "你好") {
		t.Fatal("expected word to be topped")
	}
	if sl.IsDeleted("nihao", "你好") {
		t.Fatal("topped word should not be deleted")
	}

	// Save and reload to verify persistence
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	sl2 := NewShadowLayer("test_shadow2", sl.filePath)
	if err := sl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if !sl2.IsTopped("nihao", "你好") {
		t.Fatal("topped word should persist after reload")
	}
}

func TestShadowLayerDelete(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Delete a word
	sl.Delete("zhongguo", "中国")

	if !sl.IsDeleted("zhongguo", "中国") {
		t.Fatal("expected word to be deleted")
	}
	if sl.IsTopped("zhongguo", "中国") {
		t.Fatal("deleted word should not be topped")
	}

	// Save and reload to verify persistence
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	sl2 := NewShadowLayer("test_shadow2", sl.filePath)
	if err := sl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if !sl2.IsDeleted("zhongguo", "中国") {
		t.Fatal("deleted word should persist after reload")
	}
}

func TestShadowLayerReweight(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Reweight a word
	sl.Reweight("shijie", "世界", 500)

	rules := sl.GetShadowRules("shijie")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Action != ShadowActionReweight {
		t.Fatalf("expected reweight action, got %s", rules[0].Action)
	}
	if rules[0].NewWeight != 500 {
		t.Fatalf("expected weight 500, got %d", rules[0].NewWeight)
	}

	// Save and reload to verify persistence
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	sl2 := NewShadowLayer("test_shadow2", sl.filePath)
	if err := sl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	rules2 := sl2.GetShadowRules("shijie")
	if len(rules2) != 1 {
		t.Fatalf("expected 1 rule after reload, got %d", len(rules2))
	}
	if rules2[0].NewWeight != 500 {
		t.Fatalf("expected weight 500 after reload, got %d", rules2[0].NewWeight)
	}
}

func TestShadowLayerRemoveRule(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Add then remove
	sl.Top("nihao", "你好")
	if !sl.IsTopped("nihao", "你好") {
		t.Fatal("expected word to be topped")
	}

	sl.RemoveRule("nihao", "你好")
	if sl.IsTopped("nihao", "你好") {
		t.Fatal("rule should be removed")
	}

	rules := sl.GetShadowRules("nihao")
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after removal, got %d", len(rules))
	}

	// Verify the code key is cleaned up when all rules are removed
	if sl.GetRuleCount() != 0 {
		t.Fatalf("expected 0 total rules, got %d", sl.GetRuleCount())
	}
}

func TestShadowLayerOverwrite(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Top first, then change to delete
	sl.Top("nihao", "你好")
	if !sl.IsTopped("nihao", "你好") {
		t.Fatal("expected topped")
	}

	sl.Delete("nihao", "你好")
	if !sl.IsDeleted("nihao", "你好") {
		t.Fatal("expected deleted after overwrite")
	}
	if sl.IsTopped("nihao", "你好") {
		t.Fatal("should not be topped after overwrite to delete")
	}

	// Only 1 rule should exist (overwritten, not duplicated)
	rules := sl.GetShadowRules("nihao")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (overwrite), got %d", len(rules))
	}

	// Now reweight
	sl.Reweight("nihao", "你好", 200)
	rules = sl.GetShadowRules("nihao")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after reweight overwrite, got %d", len(rules))
	}
	if rules[0].Action != ShadowActionReweight || rules[0].NewWeight != 200 {
		t.Fatalf("expected reweight(200), got %s(%d)", rules[0].Action, rules[0].NewWeight)
	}

	// Save and reload
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	sl2 := NewShadowLayer("test_shadow2", sl.filePath)
	if err := sl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	rules2 := sl2.GetShadowRules("nihao")
	if len(rules2) != 1 {
		t.Fatalf("expected 1 rule after reload, got %d", len(rules2))
	}
	if rules2[0].Action != ShadowActionReweight || rules2[0].NewWeight != 200 {
		t.Fatalf("expected reweight(200) after reload, got %s(%d)", rules2[0].Action, rules2[0].NewWeight)
	}
}

func TestShadowLayerCaseInsensitive(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Code should be case-insensitive
	sl.Top("NiHao", "你好")
	if !sl.IsTopped("nihao", "你好") {
		t.Fatal("code lookup should be case-insensitive")
	}
	if !sl.IsTopped("NIHAO", "你好") {
		t.Fatal("code lookup should be case-insensitive (uppercase)")
	}
}

func TestShadowLayerLoadNonExistent(t *testing.T) {
	sl := NewShadowLayer("test", filepath.Join(t.TempDir(), "nonexistent.yaml"))

	// Loading non-existent file should not error
	if err := sl.Load(); err != nil {
		t.Fatalf("loading non-existent file should not error: %v", err)
	}

	if sl.GetRuleCount() != 0 {
		t.Fatal("rule count should be 0 for non-existent file")
	}
}

func TestShadowLayerDirtyFlag(t *testing.T) {
	sl := newTestShadowLayer(t)

	if sl.IsDirty() {
		t.Fatal("should not be dirty initially")
	}

	sl.Top("nihao", "你好")
	if !sl.IsDirty() {
		t.Fatal("should be dirty after modification")
	}

	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if sl.IsDirty() {
		t.Fatal("should not be dirty after save")
	}
}

func TestShadowLayerSaveSkipsWhenClean(t *testing.T) {
	sl := newTestShadowLayer(t)

	// Save without any modifications should be a no-op
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// File should not exist since nothing was dirty
	if _, err := os.Stat(sl.filePath); !os.IsNotExist(err) {
		t.Fatal("file should not exist when saving clean state")
	}
}

func TestShadowLayerMultipleCodesAndRules(t *testing.T) {
	sl := newTestShadowLayer(t)

	sl.Top("nihao", "你好")
	sl.Delete("nihao", "泥号")
	sl.Top("shijie", "世界")
	sl.Reweight("shijie", "时节", 300)

	if sl.GetRuleCount() != 4 {
		t.Fatalf("expected 4 total rules, got %d", sl.GetRuleCount())
	}

	// Verify nihao rules
	nihaoRules := sl.GetShadowRules("nihao")
	if len(nihaoRules) != 2 {
		t.Fatalf("expected 2 rules for nihao, got %d", len(nihaoRules))
	}

	// Verify shijie rules
	shijieRules := sl.GetShadowRules("shijie")
	if len(shijieRules) != 2 {
		t.Fatalf("expected 2 rules for shijie, got %d", len(shijieRules))
	}

	// Save and reload
	if err := sl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	sl2 := NewShadowLayer("test2", sl.filePath)
	if err := sl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if sl2.GetRuleCount() != 4 {
		t.Fatalf("expected 4 rules after reload, got %d", sl2.GetRuleCount())
	}
}
