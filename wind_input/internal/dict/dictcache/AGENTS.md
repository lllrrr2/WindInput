<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-13 -->

# internal/dict/dictcache

## Purpose
词库缓存管理。负责将文本格式的码表和字典转换为高效的二进制 `wdb` 格式，并缓存到本地（`%LOCALAPPDATA%\WindInput\cache`）。提供缓存有效性检测（通过文件 mtime 比较）和自动重新生成机制。

## Key Files
| File | Description |
|------|-------------|
| `cache.go` | 缓存路径管理（`GetCacheDir`、`CachePath`）和有效性检测（`NeedsRegenerate`） |
| `convert.go` | 码表转换逻辑（`ConvertCodeTableToWdb`）和元数据管理（`CodeTableMeta`） |

## For AI Agents

### Working In This Directory
- 缓存目录：`%LOCALAPPDATA%\WindInput\cache\<name>.wdb`
- 元数据文件：`<wdb_path>.meta.json`，存储码表的 Header 信息
- `NeedsRegenerate(srcPaths, wdbPath)` 判断缓存是否过期（源文件 mtime > wdb mtime）
- `ConvertCodeTableToWdb(srcPath, wdbPath)` 执行转换，并生成 meta.json sidecar

### Testing Requirements
- 缓存有效性检测可做单元测试（文件 mtime 比较逻辑）
- 转换逻辑可通过读写往返验证

### Common Patterns
- 启动时调用 `NeedsRegenerate` 检测是否需要重新生成二进制文件
- 转换失败时回退到文本格式加载（见 `internal/dict.LoadCodeTable`）

## Dependencies
### Internal
- `internal/dict` — LoadCodeTable、CodeTable 接口
- `internal/dict/binformat` — DictWriter（码表写入）

### External
- 无（仅标准库）

<!-- MANUAL: -->
