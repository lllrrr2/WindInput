<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-13 -->

# themes/dark

## Purpose
深色主题定义。为清风输入法提供深色配色方案，包含候选窗口、工具栏和弹出菜单的颜色配置。适合深色系桌面环境使用。

## Key Files
| File | Description |
|------|-------------|
| `theme.yaml` | 深色主题配置文件，定义所有 UI 元素的颜色（`#RRGGBB` 或 `#RRGGBBAA` 格式） |

## For AI Agents

### Working In This Directory
- 主题文件结构（由 `pkg/theme.Theme` 定义）：
  - `meta` — 主题元信息（名称、描述、作者）
  - `candidate_window` — 候选窗口颜色组（背景、文字、边框、高亮等）
  - `toolbar` — 工具栏颜色组
  - `popup_menu` — 弹出菜单颜色组
- 颜色格式：6 位或 8 位十六进制（`#RRGGBB` 或 `#RRGGBBAA`）
- 修改颜色时需同步更新对应的 Go 结构体和 `pkg/theme/default_themes.go` 的内置默认数据

### Testing Requirements
- 颜色格式可通过 `pkg/theme.ParseColor` 验证
- 视觉效果需在 Windows 环境下手动验证

### Common Patterns
- 主题目录名作为主题标识符（该目录为 `dark`，对应主题名 `"dark"`）
- 用户通过 UI 右键菜单选择主题，配置保存到 `cfg.UI.Theme`
- 启动时由 `pkg/theme.Manager` 加载对应主题

## Dependencies
### Internal
- `pkg/theme` — 主题结构体定义和加载逻辑

### External
- 无

<!-- MANUAL: -->
