<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-23 -->

# internal

## Purpose
输入法核心逻辑的内部包集合，不对模块外部暴露。包含从 Schema 方案管理、IPC 通信到引擎计算、UI 渲染的完整实现链路。

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `schema/` | **输入方案管理**：Schema 定义、工厂、加载器、学习策略（方案驱动架构核心） |
| `bridge/` | Named Pipe IPC 服务端，与 C++ TSF 桥接层通信 |
| `candidate/` | 候选词数据结构和过滤/排序逻辑 |
| `control/` | 控制管道服务端，供设置应用调用 |
| `coordinator/` | 核心协调器，处理按键事件、模式切换、生命周期（14 个处理器文件） |
| `dict/` | 词库系统（Trie、CompositeDict、短语、Shadow pin+delete、用户词典、词频） |
| `engine/` | Schema 驱动的引擎管理器及拼音/五笔引擎实现 |
| `hotkey/` | 热键配置编译器 |
| `ipc/` | 底层 IPC 协议和 Named Pipe 服务端基础设施 |
| `state/` | IME 状态管理器 |
| `transform/` | 文本转换（全角/半角、中英文标点） |
| `ui/` | Windows 原生 UI 渲染（候选窗口、工具栏、Tooltip、DPI 感知） |

## For AI Agents

### Working In This Directory
- 这些包只能被 `cmd/` 和同级 `internal/` 包引用，不得被 `pkg/` 引用
- 核心数据流：`bridge` → `coordinator` → `schema` → `engine` → `dict` → `candidate` → `bridge`（响应）
- UI 更新：`coordinator` → `ui.Manager`（通过 channel 发送 UICommand）
- Schema 流程：`schema.Manager` 加载 `*.schema.yaml` → `schema.Factory` 创建 Engine + Dict

### Testing Requirements
- 各包独立测试：`go test ./internal/...`
- `engine/pinyin/`、`dict/` 和 `schema/` 有较多单元测试，修改时务必运行

### Common Patterns
- Windows 平台专属代码用 `_windows.go` 后缀（如 `binformat/mmap_windows.go`）
- 接口定义与实现分离（如 `engine.Engine`、`bridge.MessageHandler`）
- Shadow 规则使用 pin(position) + delete 二元架构

## Dependencies
### Internal
- `pkg/` 下的公共包
- `internal/` 包之间有依赖关系（见各子目录）

### External
- `golang.org/x/sys/windows`
- `github.com/Microsoft/go-winio`

<!-- MANUAL: -->
