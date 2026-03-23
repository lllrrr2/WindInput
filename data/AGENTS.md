<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-23 | Updated: 2026-03-23 -->

# data

## Purpose
输入法的数据资源目录，包含 Schema 方案定义、词库源数据和配置示例文件。这些文件在构建时被复制到 `build/` 目录，运行时由 `wind_input` 服务加载。

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `schemas/` | 输入方案定义文件（`*.schema.yaml`），驱动引擎创建和词库配置 |
| `dict/` | 词库源数据（拼音 unigram、常用字表等）(see `dict/AGENTS.md`) |
| `examples/` | 用户数据示例文件（短语、Shadow 规则） |

## Key Files

### schemas/
| File | Description |
|------|-------------|
| `pinyin.schema.yaml` | 拼音输入方案定义（引擎类型、词库路径、学习策略） |
| `wubi86.schema.yaml` | 五笔86输入方案定义（码表配置、自动上屏规则） |

### examples/
| File | Description |
|------|-------------|
| `phrases.example.yaml` | 用户短语示例（自定义短语格式参考） |
| `shadow.example.yaml` | Shadow 规则示例（pin/delete 操作格式参考） |

## For AI Agents

### Working In This Directory
- Schema 文件是方案驱动架构的核心配置，修改后需确保 `internal/schema` 包能正确解析
- 词库源数据较大（unigram.txt ~25MB），不要在 AI 上下文中完整读取
- examples/ 文件供用户参考，修改时保持格式清晰易懂

### Testing Requirements
- 修改 Schema 文件后运行 `cd wind_input && go test ./internal/schema/...`
- 词库数据变更后需重新生成二进制词库（`cmd/gen_bindict`、`cmd/gen_wubi_wdb`）

## Dependencies

### Internal
- `wind_input/internal/schema` — Schema 加载和解析
- `wind_input/internal/dict` — 词库加载
- `wind_input/cmd/gen_*` — 词库生成工具

### External
- 拼音词库源: [雾凇拼音 rime-ice](https://github.com/iDvel/rime-ice)
- 五笔词库源: Rime 生态五笔86词库

<!-- MANUAL: -->
