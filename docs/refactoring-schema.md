# Schema 架构重构记录

## Phase 1：Schema 包、数据结构与配置体系 ✅

**完成时间**：2026-03-15

### 变更内容

1. **新建 `wind_input/internal/schema/` 包**
   - `schema.go` — Schema 数据结构定义（SchemaInfo, EngineSpec, CodeTableSpec, PinyinSpec, ShuangpinSpec, FuzzySpec, DictSpec, UserDataSpec, LearningSpec）
   - `loader.go` — 方案文件发现与加载（LoadSchemaFile, DiscoverSchemas, validateSchema）
   - `manager.go` — SchemaManager（LoadSchemas, GetSchema, SetActive, ListSchemas）

2. **新建内置方案文件**
   - `schemas/wubi86.schema.yaml` — 五笔86方案
   - `schemas/pinyin.schema.yaml` — 全拼方案

3. **修改 `wind_input/pkg/config/config.go`**
   - 新增 `SchemaConfig` 结构（Active, Available）
   - Config 结构新增 `Schema` 字段
   - DefaultConfig 默认 active="wubi86", available=["wubi86", "pinyin"]
   - Engine/Dictionary 字段保留（标记 Deprecated，Phase 3 移除）

### 检查点

- [x] `go build ./...` 通过
- [x] `go fmt` 格式化完成
- [x] SchemaManager 能正确加载方案（数据结构和加载逻辑完备）
- [x] config.yaml 支持新的 schema 段

### 设计决策

- Phase 1 保留旧的 Engine/Dictionary 配置字段，避免破坏现有代码
- Schema 方案文件使用 YAML 格式，与 Rime 风格一致
- 双拼作为 PinyinSpec 的子配置（ShuangpinSpec.Layout）
- 方案文件优先级：用户目录 > 安装目录（同 ID 覆盖）

---

## Phase 2：DictManager 按方案隔离 + wdb 元数据嵌入 ✅

**完成时间**：2026-03-16

### 变更内容

1. **wdb 格式升级 V2**
   - `binformat/format.go` — `Reserved1` → `MetaOff`，Version 1→2，兼容 V1 读取
   - `binformat/writer.go` — 新增 `SetMeta()`，Write 末尾写入 MetaSection
   - `binformat/reader.go` — 新增 `ReadMeta()`、`HasMeta()`

2. **DictManager 全面重构**
   - `dict/manager.go` — 从硬编码 pinyin/wubi 改为 `map[string]*ShadowLayer` + `map[string]*UserDict`
   - `Initialize()` 仅加载全局 PhraseLayer
   - 新增 `SwitchSchema(schemaID, shadowFile, userDictFile)` 懒加载方案数据
   - `SetActiveEngine()` 保留为兼容包装器
   - `Save()`/`Close()` 遍历所有已加载方案

3. **dictcache/convert.go 更新**
   - `ConvertCodeTableToWdb` 嵌入 meta 到 wdb
   - 新增 `LoadCodeTableMetaFromWdb()` 从 wdb 读取 meta
   - sidecar 写入暂保留（Phase 3 移除）

4. **调用方适配**
   - `cmd/service/main.go` — Initialize() 无参数
   - `wubi_test.go` — 使用 SwitchSchema

5. **测试文件**
   - `schema/schema_test.go` — 方案加载、校验、发现、管理器测试
   - `binformat/meta_test.go` — wdb meta 写入/读取（含简拼+meta 组合）
   - `dict/manager_test.go` — SwitchSchema 隔离、shadow 隔离、保存重载、兼容性

### 检查点

- [x] wdb V2 格式正确，Reader 能读取 meta
- [x] V1 wdb 兼容（MetaOff=0 时正常读取）
- [x] DictManager.SwitchSchema 正确切换 Shadow/UserDict
- [x] 不同方案的 shadow 文件互不干扰（测试验证）
- [x] `go build ./...` 通过
- [x] 所有测试通过

---

## Phase 3：Schema 驱动引擎创建与方案切换

**状态**：待实施

---

## Phase 4：统一 DictLayer 接口，五笔走 CompositeDict

**状态**：待实施

---

## Phase 5：LearningStrategy 接口定义

**状态**：待实施
