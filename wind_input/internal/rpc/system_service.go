package rpc

import (
	"log/slog"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// SystemService 系统管理 RPC 服务
type SystemService struct {
	dm     *dict.DictManager
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
