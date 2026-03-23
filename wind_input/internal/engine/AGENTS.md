<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-23 -->

# internal/engine

## Purpose
引擎管理层。定义 `Engine`/`ExtendedEngine` 接口和 `ConvertResult` 数据结构，通过 `Manager` 统一管理所有输入方案引擎的加载、切换和调用。引擎以方案 ID（SchemaID）为键，支持运行时动态切换方案（`SwitchSchema`/`ToggleSchema`）和临时方案激活（`ActivateTempSchema`/`ActivateTempPinyin`）。

原 `manager_init.go` 和 `manager_userfreq.go` 已删除，初始化逻辑移至 `internal/schema/factory.go`，用户词频保存逻辑整合进 `manager.go`。

## Key Files
| File | Description |
|------|-------------|
| `engine.go` | `Engine`、`ExtendedEngine` 接口定义，`ConvertResult` 结构体（含拼音专用字段） |
| `manager.go` | `Manager`：Schema 驱动的引擎注册表；`SwitchSchema`（切换/懒加载方案引擎）、`ToggleSchema`（循环切换）、`ActivateTempSchema`/`DeactivateTempSchema`（临时方案）、`ActivateTempPinyin`/`DeactivateTempPinyin`（临时拼音词库层注入）；`Convert`/`ConvertEx`/`HandleTopCode`/`OnCandidateSelected`/`SaveUserFreqs` 等调度方法；兼容旧 API `RegisterEngine`/`SwitchEngine`/`ToggleEngine` |
| `manager_config.go` | 配置热更新：`UpdateFilterMode`、`UpdateWubiOptions`、`UpdatePinyinOptions`（含五笔反查码表懒加载） |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `pinyin/` | 拼音输入引擎（DAG、Viterbi、音节 Trie、模糊拼音、连续评分模型等） |
| `wubi/` | 五笔输入引擎（码表查询、顶码、词频学习） |

## For AI Agents

### Working In This Directory
- `Manager` 使用 `sync.RWMutex` 保护引擎注册表，读操作（Convert）用读锁，切换用写锁
- 引擎以 **SchemaID**（字符串）为键，不再使用固定的 `"pinyin"`/`"wubi"` 常量（但保留兼容方法）
- `SwitchSchema` 懒加载：首次切换某方案时调用 `schema.CreateEngineFromSchema` 创建引擎并缓存；后续切换已加载的方案直接复用缓存
- 切换方案时通过 `systemLayers` 缓存各方案的系统词库层，重新激活缓存引擎时通过 `reRegisterSystemLayer` 重新注册
- `ActivateTempPinyin`/`DeactivateTempPinyin` 操作 `DictManager` 的 `CompositeDict`，向其注入/卸载拼音词库层，不切换 `currentEngine`
- `SaveUserFreqs` 遍历所有已加载引擎，仅对开启了 `EnableUserFreq` 的拼音引擎保存词频
- `GetCurrentType()` 返回 `EngineType(currentID)`，仍可与旧代码兼容

### Testing Requirements
- `go test ./internal/engine/...`（会递归测试 pinyin/ 和 wubi/ 子目录）
- Manager 层无独立测试文件，逻辑通过集成测试覆盖

### Common Patterns
- `EngineType` 常量保留 `"pinyin"`/`"wubi"`，但新代码应使用 SchemaID
- 引擎接口设计为无状态（拼音引擎确实无状态），`Reset()` 为预留接口
- `ConvertEx` 对拼音引擎返回 `PreeditDisplay`/`CompletedSyllables`/`PartialSyllable`；对五笔引擎返回 `ShouldCommit`/`CommitText`/`ShouldClear`/`ToEnglish`

## Dependencies
### Internal
- `internal/candidate` — Candidate 类型、CandidateSortMode
- `internal/dict` — DictManager、CompositeDict、DictLayer
- `internal/engine/pinyin` — 拼音引擎实现
- `internal/engine/wubi` — 五笔引擎实现
- `internal/schema` — SchemaManager、CreateEngineFromSchema、SavePinyinUserFreqs
- `pkg/config` — PinyinConfig（热更新参数）

### External
- 无

<!-- MANUAL: -->
