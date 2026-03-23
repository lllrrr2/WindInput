<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-23 -->

# pkg/dictfile

## Purpose
词库文件数据类型定义和读写逻辑。定义 `phrases.yaml`、`shadow.yaml`、用户词库文件的数据结构，供服务端（`internal/dict`）和外部工具共用。

## Key Files
| File | Description |
|------|-------------|
| `types.go` | 核心类型：`PhraseEntry`、`PhraseConfig`、`PhrasesConfig`、`ShadowPinConfig`、`ShadowCodeConfig`、`ShadowConfig`、`UserWord`、`UserDictData` |
| `phrase.go` | `PhrasesConfig` 的 YAML 读写函数 |
| `shadow.go` | `ShadowConfig` 的 YAML 读写及操作函数（`PinWord`、`DeleteWord`、`RemoveShadowRule`、`GetRuleCount`） |
| `userdict.go` | `UserDictData` 的 JSON 读写函数 |

## For AI Agents

### Working In This Directory
- `PhraseConfig.Type` 值：空字符串（普通短语）或 `"command"`（内置命令）
- `PhraseConfig.Handler` 内置命令名：`date`、`time`、`datetime`、`week`、`uuid`、`timestamp`
- **Shadow 架构（pin+delete）**：旧的 `ShadowAction` 枚举（top/delete/reweight）已废弃，现为：
  - `ShadowCodeConfig.Pinned []ShadowPinConfig`：固定位置规则，每条含 `Word` 和 `Position`
  - `ShadowCodeConfig.Deleted []string`：隐藏词列表
  - `ShadowConfig.Rules map[string]*ShadowCodeConfig`：按编码索引的规则集
- `shadow.go` 提供高层操作函数：`PinWord(cfg, code, word, position)`、`DeleteWord(cfg, code, word)`、`RemoveShadowRule(cfg, code, word)`
- `PinWord` 会自动从 Deleted 中移除、`DeleteWord` 会自动从 Pinned 中移除（互斥操作）
- `UserDictData` 使用 JSON 格式（非 YAML），路径在 `pkg/config` 中定义
- 短语模板变量（在 `internal/dict/phrase.go` 中展开）：`{year}`、`{month}`、`{day}`、`{hour}`、`{minute}`、`{second}`、`{week}`

### Testing Requirements
- 纯 Go 逻辑，可做序列化往返单元测试
- Shadow 操作函数的互斥行为（pin/delete 互斥）可做纯函数单元测试

### Common Patterns
- 文件格式示例见 `configs/phrases.example.yaml` 和 `configs/shadow.example.yaml`
- 用户词库采用 JSON 而非 YAML，便于程序读写（YAML 适合人工编辑）
- Shadow 文件通过 `fileutil.AtomicWrite` 保存，保证写入安全

## Dependencies
### Internal
- `pkg/config` — 获取 Shadow 文件路径（`GetShadowPath()`）
- `pkg/fileutil` — 原子写入（`AtomicWrite`）

### External
- `gopkg.in/yaml.v3` — 短语和 Shadow 文件的 YAML 解析

<!-- MANUAL: -->
