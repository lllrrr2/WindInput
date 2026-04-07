package coordinator

import (
	"log/slog"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/engine"
	"github.com/huanfeng/wind_input/internal/schema"
	"github.com/huanfeng/wind_input/pkg/config"
	pkgcontrol "github.com/huanfeng/wind_input/pkg/control"
)

// ReloadHandler 实现 control.ReloadHandler 接口，负责配置热重载和状态查询。
// 协调 schema/engine/dict 等子系统的配置变更。
type ReloadHandler struct {
	coord     *Coordinator
	cfg       *config.Config
	schemaMgr *schema.SchemaManager
	engineMgr *engine.Manager
	dictMgr   *dict.DictManager
	logger    *slog.Logger
}

// NewReloadHandler 创建配置重载处理器
func NewReloadHandler(coord *Coordinator, cfg *config.Config, schemaMgr *schema.SchemaManager, engineMgr *engine.Manager, dictMgr *dict.DictManager, logger *slog.Logger) *ReloadHandler {
	return &ReloadHandler{
		coord:     coord,
		cfg:       cfg,
		schemaMgr: schemaMgr,
		engineMgr: engineMgr,
		dictMgr:   dictMgr,
		logger:    logger,
	}
}

// ReloadConfig 重载配置（处理 config.yaml 变更和 schema 文件变更）
func (h *ReloadHandler) ReloadConfig() error {
	newCfg, err := config.Load()
	if err != nil {
		return err
	}

	// 检查活跃方案是否切换
	oldSchemaID := h.cfg.Schema.Active
	newSchemaID := newCfg.Schema.Active
	if newSchemaID != "" && newSchemaID != oldSchemaID {
		h.logger.Info("Schema changed via config reload", "from", oldSchemaID, "to", newSchemaID)
		if err := h.engineMgr.SwitchSchema(newSchemaID); err != nil {
			h.logger.Error("Failed to switch schema", "error", err)
		} else {
			h.schemaMgr.SetActive(newSchemaID)
			s := h.schemaMgr.GetSchema(newSchemaID)
			if s != nil && h.dictMgr != nil {
				h.dictMgr.SwitchSchemaFull(newSchemaID, s.UserData.ShadowFile, s.UserData.UserDictFile,
					s.UserData.TempDictFile, s.Learning.TempMaxEntries, s.Learning.TempPromoteCount)
			}
		}
	}

	// 重新加载活跃方案的 schema 文件，应用引擎选项热更新
	h.reloadActiveSchemaConfig()

	// 更新协调器的全局配置
	if h.coord != nil {
		h.coord.UpdateHotkeyConfig(&newCfg.Hotkeys)
		h.coord.UpdateStartupConfig(&newCfg.Startup)
		h.coord.UpdateUIConfig(&newCfg.UI)
		h.coord.UpdateToolbarConfig(&newCfg.Toolbar)
		h.coord.UpdateInputConfig(&newCfg.Input)
	}

	// 从全局配置更新候选过滤模式
	if newCfg.Input.FilterMode != "" {
		h.engineMgr.UpdateFilterMode(newCfg.Input.FilterMode)
	}

	// 更新保存的配置引用
	*h.cfg = *newCfg

	h.logger.Info("Config reloaded successfully",
		"schema", newCfg.Schema.Active,
		"toggleModeKeys", newCfg.Hotkeys.ToggleModeKeys)
	return nil
}

// reloadActiveSchemaConfig 从 schema 文件重新加载引擎选项并热更新
func (h *ReloadHandler) reloadActiveSchemaConfig() {
	if h.schemaMgr == nil {
		return
	}

	// 重新加载 schema 文件
	if err := h.schemaMgr.LoadSchemas(); err != nil {
		h.logger.Error("Failed to reload schemas", "error", err)
		return
	}

	activeID := h.schemaMgr.GetActiveID()
	s := h.schemaMgr.GetSchema(activeID)
	if s == nil {
		return
	}

	// 根据引擎类型应用配置
	switch s.Engine.Type {
	case schema.EngineTypeCodeTable:
		if spec := s.Engine.CodeTable; spec != nil {
			h.engineMgr.UpdateCodetableOptions(
				spec.AutoCommitUnique,
				spec.ClearOnEmptyMax,
				spec.TopCodeCommit,
				spec.PunctCommit,
				spec.ShowCodeHint,
				spec.SingleCodeInput,
				spec.CandidateSortMode,
			)
		}

	case schema.EngineTypePinyin:
		if spec := s.Engine.Pinyin; spec != nil {
			pinyinCfg := &config.PinyinConfig{
				ShowCodeHint:    spec.ShowCodeHint,
				UseSmartCompose: spec.UseSmartCompose,
				CandidateOrder:  spec.CandidateOrder,
			}
			if spec.Fuzzy != nil && spec.Fuzzy.Enabled {
				pinyinCfg.Fuzzy = config.FuzzyPinyinConfig{
					Enabled: true,
					ZhZ:     spec.Fuzzy.ZhZ,
					ChC:     spec.Fuzzy.ChC,
					ShS:     spec.Fuzzy.ShS,
					NL:      spec.Fuzzy.NL,
					FH:      spec.Fuzzy.FH,
					RL:      spec.Fuzzy.RL,
					AnAng:   spec.Fuzzy.AnAng,
					EnEng:   spec.Fuzzy.EnEng,
					InIng:   spec.Fuzzy.InIng,
				}
			}
			h.engineMgr.UpdatePinyinOptions(pinyinCfg)
		}
	}

	h.logger.Debug("Schema config reloaded", "schema", activeID, "engineType", s.Engine.Type)
}

// GetStatus 获取服务状态
func (h *ReloadHandler) GetStatus() *pkgcontrol.ServiceStatus {
	status := &pkgcontrol.ServiceStatus{
		Running: true,
	}

	if h.coord != nil {
		status.ChineseMode = h.coord.GetChineseMode()
		status.FullWidth = h.coord.GetFullWidth()
		status.ChinesePunct = h.coord.GetChinesePunctuation()
		status.EngineType = h.coord.GetCurrentEngineName()
	}

	return status
}
