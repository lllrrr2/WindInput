package rpc

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/store"
	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// DictService 词库管理 RPC 服务
type DictService struct {
	store  *store.Store
	dm     *dict.DictManager
	logger *slog.Logger
}

func (d *DictService) resolveSchemaID(id string) string {
	if id != "" {
		return id
	}
	return d.dm.GetActiveSchemaID()
}

// Search 搜索用户词库（前缀匹配，支持分页）
func (d *DictService) Search(args *rpcapi.DictSearchArgs, reply *rpcapi.DictSearchReply) error {
	if d.store == nil {
		return fmt.Errorf("store not available")
	}

	schemaID := d.resolveSchemaID(args.SchemaID)
	prefix := strings.ToLower(args.Prefix)
	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}

	allWords, err := d.store.SearchUserWordsPrefix(schemaID, prefix, 0)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	reply.Total = len(allWords)

	start := args.Offset
	if start > len(allWords) {
		start = len(allWords)
	}
	end := start + limit
	if end > len(allWords) {
		end = len(allWords)
	}

	pageWords := allWords[start:end]
	reply.Words = make([]rpcapi.WordEntry, len(pageWords))
	for i, w := range pageWords {
		reply.Words[i] = rpcapi.WordEntry{
			Text:      w.Text,
			Weight:    w.Weight,
			Count:     w.Count,
			CreatedAt: w.CreatedAt,
		}
	}

	return nil
}

// SearchByCode 精确编码查询
func (d *DictService) SearchByCode(args *rpcapi.DictSearchArgs, reply *rpcapi.DictSearchReply) error {
	if d.store == nil {
		return fmt.Errorf("store not available")
	}

	schemaID := d.resolveSchemaID(args.SchemaID)
	code := strings.ToLower(args.Prefix)

	words, err := d.store.GetUserWords(schemaID, code)
	if err != nil {
		return fmt.Errorf("search by code: %w", err)
	}

	reply.Total = len(words)
	reply.Words = make([]rpcapi.WordEntry, len(words))
	for i, w := range words {
		reply.Words[i] = rpcapi.WordEntry{
			Code:      code,
			Text:      w.Text,
			Weight:    w.Weight,
			Count:     w.Count,
			CreatedAt: w.CreatedAt,
		}
	}

	return nil
}

// Add 添加用户词条
func (d *DictService) Add(args *rpcapi.DictAddArgs, reply *rpcapi.Empty) error {
	if d.store == nil {
		return fmt.Errorf("store not available")
	}
	if args.Code == "" || args.Text == "" {
		return fmt.Errorf("code and text are required")
	}

	schemaID := d.resolveSchemaID(args.SchemaID)
	weight := args.Weight
	if weight <= 0 {
		weight = 1200
	}

	d.logger.Info("RPC Dict.Add", "schemaID", schemaID, "codeLen", len(args.Code), "textLen", len([]rune(args.Text)))
	return d.store.AddUserWord(schemaID, args.Code, args.Text, weight)
}

// Remove 删除用户词条
func (d *DictService) Remove(args *rpcapi.DictRemoveArgs, reply *rpcapi.Empty) error {
	if d.store == nil {
		return fmt.Errorf("store not available")
	}
	if args.Code == "" || args.Text == "" {
		return fmt.Errorf("code and text are required")
	}

	schemaID := d.resolveSchemaID(args.SchemaID)
	d.logger.Info("RPC Dict.Remove", "schemaID", schemaID, "codeLen", len(args.Code), "textLen", len([]rune(args.Text)))
	return d.store.RemoveUserWord(schemaID, args.Code, args.Text)
}

// Update 更新词条权重
func (d *DictService) Update(args *rpcapi.DictUpdateArgs, reply *rpcapi.Empty) error {
	if d.store == nil {
		return fmt.Errorf("store not available")
	}
	if args.Code == "" || args.Text == "" {
		return fmt.Errorf("code and text are required")
	}

	schemaID := d.resolveSchemaID(args.SchemaID)
	return d.store.UpdateUserWordWeight(schemaID, args.Code, args.Text, args.NewWeight)
}

// GetStats 获取词库统计
func (d *DictService) GetStats(args *rpcapi.Empty, reply *rpcapi.DictStatsReply) error {
	reply.Stats = d.dm.GetStats()
	return nil
}
