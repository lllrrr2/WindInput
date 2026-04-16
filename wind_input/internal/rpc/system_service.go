package rpc

import (
	"fmt"
	"log/slog"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/store"
	"github.com/huanfeng/wind_input/pkg/config"
	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// SystemService 系统管理 RPC 服务
type SystemService struct {
	dm     *dict.DictManager
	store  *store.Store
	server *Server
	logger *slog.Logger
}

// Ping 心跳检测
func (s *SystemService) Ping(args *rpcapi.Empty, reply *rpcapi.Empty) error {
	return nil
}

// GetStatus 获取系统状态
func (s *SystemService) GetStatus(args *rpcapi.Empty, reply *rpcapi.SystemStatusReply) error {
	reply.Running = true
	reply.StoreEnabled = s.dm.UseStore()
	reply.SchemaID = s.dm.GetActiveSchemaID()

	stats := s.dm.GetStats()
	reply.UserWords = stats["user_words"]
	reply.TempWords = stats["temp_words"]
	reply.Phrases = stats["phrases"]
	reply.ShadowRules = stats["shadow_rules"]

	s.server.mu.Lock()
	provider := s.server.statusProvider
	s.server.mu.Unlock()

	if provider != nil {
		reply.EngineType = provider.GetEngineType()
		reply.ChineseMode = provider.IsChineseMode()
	}

	return nil
}

// ReloadPhrases 重载短语
func (s *SystemService) ReloadPhrases(args *rpcapi.Empty, reply *rpcapi.Empty) error {
	s.logger.Info("RPC System.ReloadPhrases")
	return s.dm.ReloadPhrases()
}

// ReloadAll 重载所有
func (s *SystemService) ReloadAll(args *rpcapi.Empty, reply *rpcapi.Empty) error {
	s.logger.Info("RPC System.ReloadAll")
	return s.dm.ReloadPhrases()
}

// ResetDB 重置数据库（清除用户词库、临时词库、Shadow 规则、词频数据）
func (s *SystemService) ResetDB(args *rpcapi.SystemResetDBArgs, reply *rpcapi.SystemResetDBReply) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}

	if args.SchemaID != "" {
		s.logger.Info("RPC System.ResetDB: clearing schema", "schemaID", args.SchemaID)
		if err := s.store.ClearSchema(args.SchemaID); err != nil {
			return fmt.Errorf("clear schema: %w", err)
		}
	} else {
		s.logger.Info("RPC System.ResetDB: clearing all schemas")
		if err := s.store.ClearAllSchemas(); err != nil {
			return fmt.Errorf("clear all schemas: %w", err)
		}
	}

	reply.Success = true
	return nil
}

// DeleteSchema 彻底删除方案的 bucket（用于清理残留方案）
func (s *SystemService) DeleteSchema(args *rpcapi.SystemResetDBArgs, reply *rpcapi.SystemResetDBReply) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}
	if args.SchemaID == "" {
		return fmt.Errorf("schema_id is required")
	}

	s.logger.Info("RPC System.DeleteSchema", "schemaID", args.SchemaID)
	if err := s.store.DeleteSchema(args.SchemaID); err != nil {
		return fmt.Errorf("delete schema: %w", err)
	}

	reply.Success = true
	return nil
}

// ListSchemas 列出所有方案及其状态
func (s *SystemService) ListSchemas(args *rpcapi.Empty, reply *rpcapi.ListSchemasReply) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}

	// 获取 bbolt 中已有数据的方案
	storeIDs, err := s.store.ListSchemaIDs()
	if err != nil {
		return fmt.Errorf("list schema IDs: %w", err)
	}

	// 获取配置中启用的方案
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	enabledSet := make(map[string]bool, len(cfg.Schema.Available))
	for _, id := range cfg.Schema.Available {
		enabledSet[id] = true
	}

	// 已处理的方案集合
	processed := make(map[string]bool)

	// 处理 store 中的方案
	for _, id := range storeIDs {
		status := "orphaned"
		if enabledSet[id] {
			status = "enabled"
		}

		entry := rpcapi.SchemaStatus{
			SchemaID: id,
			Status:   status,
		}
		entry.UserWords, _ = s.store.UserWordCount(id)
		entry.TempWords, _ = s.store.TempWordCount(id)
		entry.ShadowRules, _ = s.store.ShadowRuleCount(id)

		// 词频记录数
		freqEntries, _ := s.store.SearchFreqPrefix(id, "", 0)
		entry.FreqRecords = len(freqEntries)

		// 跳过数据全为空的 orphaned 方案（已被清除的残留 bucket）
		if status == "orphaned" && entry.UserWords == 0 && entry.TempWords == 0 && entry.ShadowRules == 0 && entry.FreqRecords == 0 {
			processed[id] = true
			continue
		}

		reply.Schemas = append(reply.Schemas, entry)
		processed[id] = true
	}

	// 添加配置中启用但 store 中没有数据的方案
	for _, id := range cfg.Schema.Available {
		if processed[id] {
			continue
		}
		reply.Schemas = append(reply.Schemas, rpcapi.SchemaStatus{
			SchemaID: id,
			Status:   "enabled",
		})
	}

	s.logger.Info("RPC System.ListSchemas", "count", len(reply.Schemas))
	return nil
}
