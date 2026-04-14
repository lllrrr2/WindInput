package rpc

import (
	"log/slog"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"path/filepath"
	"testing"

	"github.com/huanfeng/wind_input/internal/dict"
	"github.com/huanfeng/wind_input/internal/store"
	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// setupTestRPC 创建测试用 RPC 客户端（通过 net.Pipe 模拟管道连接）
func setupTestRPC(t *testing.T) *rpc.Client {
	t.Helper()
	dir := t.TempDir()

	s, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	dm := dict.NewDictManager(dir, dir, nil)
	if err := dm.Initialize(); err != nil {
		t.Fatal(err)
	}
	// 切换到测试方案（确保 GetActiveSchemaID 返回非空）
	dm.SwitchSchema("test", "", "")

	logger := slog.Default()
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("Dict", &DictService{store: s, dm: dm, logger: logger})
	rpcServer.RegisterName("Shadow", &ShadowService{store: s, dm: dm, logger: logger})
	rpcServer.RegisterName("System", &SystemService{dm: dm, server: &Server{}, logger: logger})

	serverConn, clientConn := net.Pipe()
	go rpcServer.ServeCodec(jsonrpc.NewServerCodec(serverConn))
	t.Cleanup(func() { clientConn.Close() })

	return jsonrpc.NewClient(clientConn)
}

func TestDictAddAndSearch(t *testing.T) {
	client := setupTestRPC(t)

	// 添加词条
	err := client.Call("Dict.Add", &rpcapi.DictAddArgs{
		Code: "ggtt", Text: "王国", Weight: 1200,
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Dict.Add: %v", err)
	}

	err = client.Call("Dict.Add", &rpcapi.DictAddArgs{
		Code: "ggtt", Text: "国王", Weight: 600,
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Dict.Add 2: %v", err)
	}

	// 精确查询
	var reply rpcapi.DictSearchReply
	err = client.Call("Dict.SearchByCode", &rpcapi.DictSearchArgs{
		Prefix: "ggtt",
	}, &reply)
	if err != nil {
		t.Fatalf("Dict.SearchByCode: %v", err)
	}
	if reply.Total != 2 {
		t.Errorf("expected 2 words, got %d", reply.Total)
	}

	// 前缀搜索
	err = client.Call("Dict.Add", &rpcapi.DictAddArgs{
		Code: "gg", Text: "王", Weight: 800,
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	var prefixReply rpcapi.DictSearchReply
	err = client.Call("Dict.Search", &rpcapi.DictSearchArgs{
		Prefix: "gg", Limit: 10,
	}, &prefixReply)
	if err != nil {
		t.Fatalf("Dict.Search: %v", err)
	}
	if prefixReply.Total != 3 {
		t.Errorf("expected 3 words with prefix 'gg', got %d", prefixReply.Total)
	}
}

func TestDictRemoveAndUpdate(t *testing.T) {
	client := setupTestRPC(t)

	// 添加
	client.Call("Dict.Add", &rpcapi.DictAddArgs{Code: "ab", Text: "测试", Weight: 100}, &rpcapi.Empty{})

	// 更新权重
	err := client.Call("Dict.Update", &rpcapi.DictUpdateArgs{
		Code: "ab", Text: "测试", NewWeight: 500,
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Dict.Update: %v", err)
	}

	var reply rpcapi.DictSearchReply
	client.Call("Dict.SearchByCode", &rpcapi.DictSearchArgs{Prefix: "ab"}, &reply)
	if reply.Words[0].Weight != 500 {
		t.Errorf("expected weight=500, got %d", reply.Words[0].Weight)
	}

	// 删除
	err = client.Call("Dict.Remove", &rpcapi.DictRemoveArgs{Code: "ab", Text: "测试"}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Dict.Remove: %v", err)
	}

	var reply2 rpcapi.DictSearchReply
	client.Call("Dict.SearchByCode", &rpcapi.DictSearchArgs{Prefix: "ab"}, &reply2)
	if reply2.Total != 0 {
		t.Errorf("expected 0 after remove, got %d", reply2.Total)
	}
}

func TestDictSearchPagination(t *testing.T) {
	client := setupTestRPC(t)

	// 添加 5 个词条
	for i, text := range []string{"词一", "词二", "词三", "词四", "词五"} {
		client.Call("Dict.Add", &rpcapi.DictAddArgs{
			Code: "ci", Text: text, Weight: 100 + i*10,
		}, &rpcapi.Empty{})
	}

	// 第一页（limit=2）
	var page1 rpcapi.DictSearchReply
	client.Call("Dict.Search", &rpcapi.DictSearchArgs{Prefix: "ci", Limit: 2, Offset: 0}, &page1)
	if page1.Total != 5 {
		t.Errorf("expected total=5, got %d", page1.Total)
	}
	if len(page1.Words) != 2 {
		t.Errorf("expected 2 words on page 1, got %d", len(page1.Words))
	}

	// 第二页
	var page2 rpcapi.DictSearchReply
	client.Call("Dict.Search", &rpcapi.DictSearchArgs{Prefix: "ci", Limit: 2, Offset: 2}, &page2)
	if len(page2.Words) != 2 {
		t.Errorf("expected 2 words on page 2, got %d", len(page2.Words))
	}

	// 第三页
	var page3 rpcapi.DictSearchReply
	client.Call("Dict.Search", &rpcapi.DictSearchArgs{Prefix: "ci", Limit: 2, Offset: 4}, &page3)
	if len(page3.Words) != 1 {
		t.Errorf("expected 1 word on page 3, got %d", len(page3.Words))
	}
}

func TestSystemPing(t *testing.T) {
	client := setupTestRPC(t)

	err := client.Call("System.Ping", &rpcapi.Empty{}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("System.Ping: %v", err)
	}
}

func TestSystemGetStatus(t *testing.T) {
	client := setupTestRPC(t)

	var reply rpcapi.SystemStatusReply
	err := client.Call("System.GetStatus", &rpcapi.Empty{}, &reply)
	if err != nil {
		t.Fatalf("System.GetStatus: %v", err)
	}
	if !reply.Running {
		t.Error("expected running=true")
	}
}

func TestShadowPinAndGetRules(t *testing.T) {
	client := setupTestRPC(t)

	// Pin
	err := client.Call("Shadow.Pin", &rpcapi.ShadowPinArgs{
		Code: "gg", Word: "王", Position: 0,
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Shadow.Pin: %v", err)
	}

	// Delete
	err = client.Call("Shadow.Delete", &rpcapi.ShadowDeleteArgs{
		Code: "gg", Word: "王国",
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Shadow.Delete: %v", err)
	}

	// GetRules
	var reply rpcapi.ShadowRulesReply
	err = client.Call("Shadow.GetRules", &rpcapi.ShadowGetRulesArgs{Code: "gg"}, &reply)
	if err != nil {
		t.Fatalf("Shadow.GetRules: %v", err)
	}
	if len(reply.Pinned) != 1 || reply.Pinned[0].Word != "王" {
		t.Errorf("unexpected pinned: %+v", reply.Pinned)
	}
	if len(reply.Deleted) != 1 || reply.Deleted[0] != "王国" {
		t.Errorf("unexpected deleted: %+v", reply.Deleted)
	}

	// RemoveRule
	err = client.Call("Shadow.RemoveRule", &rpcapi.ShadowDeleteArgs{
		Code: "gg", Word: "王",
	}, &rpcapi.Empty{})
	if err != nil {
		t.Fatalf("Shadow.RemoveRule: %v", err)
	}

	var reply2 rpcapi.ShadowRulesReply
	client.Call("Shadow.GetRules", &rpcapi.ShadowGetRulesArgs{Code: "gg"}, &reply2)
	if len(reply2.Pinned) != 0 {
		t.Errorf("expected 0 pinned after remove, got %d", len(reply2.Pinned))
	}
}

func TestDictGetStats(t *testing.T) {
	client := setupTestRPC(t)

	var reply rpcapi.DictStatsReply
	err := client.Call("Dict.GetStats", &rpcapi.Empty{}, &reply)
	if err != nil {
		t.Fatalf("Dict.GetStats: %v", err)
	}
	if reply.Stats == nil {
		t.Error("expected non-nil stats")
	}
}
