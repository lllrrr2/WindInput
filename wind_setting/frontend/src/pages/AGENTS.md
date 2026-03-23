<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-23 -->

# pages

## Purpose
各设置页面的 Vue 3 单文件组件（SFC）。每个页面对应侧边栏一个标签项，由 `App.vue` 通过 `v-show` 控制显隐（组件始终挂载，不销毁）。页面组件接收 `formData`（全局配置对象引用）作为 prop，直接修改其属性；保存操作由 `App.vue` 统一触发。

## Key Files
| 文件 | 标签 | 说明 |
|------|------|------|
| `GeneralPage.vue` | 常用 | 引擎类型切换（五笔/拼音）、启动状态默认值（中文模式、全角、中文标点） |
| `InputPage.vue` | 输入 | 引擎输入行为：五笔（四码自动上屏、顶字等）、拼音（模糊音设置）、过滤模式、标点跟随模式 |
| `HotkeyPage.vue` | 按键 | 中英文切换键、引擎切换、全角切换、标点切换、候选选择键组、翻页键；负责检测快捷键冲突并 emit `update:hotkeyConflicts` |
| `AppearancePage.vue` | 外观 | 主题选择（含实时预览 ThemePreview）、字体、候选页大小、候选排列（横/竖）、状态指示器、工具栏位置 |
| `DictionaryPage.vue` | 词库 | 短语（Phrase）管理、用户词库（UserDict）管理、Shadow 候选调整管理；接收 `isWailsEnv` prop，直接调用 `wailsApi`，不通过 `formData` |
| `AdvancedPage.vue` | 高级 | 日志级别配置、打开日志目录（emit `openLogFolder`）、服务状态查看 |
| `AboutPage.vue` | 关于 | 应用版本、服务运行状态、GitHub 链接（emit `openExternalLink`） |

## For AI Agents
### Working In This Directory
- **页面组件不调用保存**：配置类页面（General、Input、Hotkey、Appearance、Advanced）只修改 `formData` prop，由 `App.vue` 的"保存设置"按钮统一提交
- **词库页面例外**：`DictionaryPage.vue` 直接调用 `wailsApi`（增删短语/词条/Shadow 规则），因为词库操作是独立的增量写入，不走 `formData` 流程
- 新增页面步骤：创建 `XxxPage.vue` -> 在 `App.vue` 中 import 并注册 `tabs` 条目 -> 添加 `<XxxPage v-show="activeTab === 'xxx'" ... />` 绑定
- Props 接收 `formData` 时使用 `defineProps<{ formData: Config }>()` 并标注类型
- `HotkeyPage` 使用 `defineEmits(['update:hotkeyConflicts'])` 向父组件上报冲突
- **无 components/ 子目录**：所有组件逻辑在页面级 SFC 内实现，不抽取独立子组件

### DictionaryPage.vue 详细说明（2026-03-13 重写）
`DictionaryPage.vue` 是最复杂的页面，结构如下：
- **Props**：`{ isWailsEnv: boolean }`（不接收 `formData`）
- **子标签页**：`phrases`（用户短语）、`userdict`（用户词库）、`shadow`（候选调整）
- **Shadow 操作**（使用 pin + delete 架构）：
  - 通过内联对话框（`shadowDialogVisible`）进行新增/编辑，支持 `pin`（固定位置）和 `delete`（隐藏）两种操作
  - `openShadowDialog(item?)` — 无参数为添加模式，有参数为编辑模式（先 `removeShadowRule` 再写入新规则）
  - `handleSaveShadowRule()` — 编辑时先调用 `removeShadowRule`，再根据 action 调用 `pinShadowWord` 或 `deleteShadowWord`
  - `handleRemoveShadowRule(item)` — 直接调用 `removeShadowRule` 彻底删除规则
- **用户词库切换**：按方案 ID（`wubi86` / `pinyin`）切换，调用 `switchUserDictSchema(schemaID)`
- **文件变化检测**：`checkFileChanges()` 调用 `checkAllFilesModified()`，检测到变化时显示警告条（`showFileChangeAlert`）
- **统计显示**：`dictStats`（`DictStats` 类型）在子标签页标题中实时显示各类数据的数量

### Testing Requirements
- `pnpm run build`（TypeScript 类型检查覆盖所有页面）
- 在 Wails 环境中逐一验证每个页面的表单交互和数据持久化
- `DictionaryPage.vue` 需要在有真实词库文件的环境中测试 CRUD 操作，尤其是 Shadow pin/delete/remove 三种操作

### Common Patterns
```vue
<!-- 配置类页面标准模式 -->
<script setup lang="ts">
import type { Config } from '../api/settings'
const props = defineProps<{ formData: Config }>()
// 直接修改：props.formData.engine.type = 'pinyin'
</script>

<!-- 词库页面：直接调用 wailsApi -->
<script setup lang="ts">
import * as wailsApi from '../api/wails'
async function addWord() {
  await wailsApi.addUserWord(code, text, weight)
  await loadDictData()  // 刷新列表
}
// Shadow 操作（pin + delete 架构）
async function handleSaveShadowRule() {
  if (shadowDialogEditing.value) {
    await wailsApi.removeShadowRule(code, word)  // 先移除旧规则
  }
  if (action === 'pin') {
    await wailsApi.pinShadowWord(code, word, position)
  } else {
    await wailsApi.deleteShadowWord(code, word)
  }
}
</script>
```

## Dependencies
### Internal
- `../api/wails` — Wails IPC API（DictionaryPage 直接使用：Shadow pin/delete/remove、短语、用户词库；AppearancePage 通过父组件 props 传入 theme 数据）
- `../api/settings` — Config 类型定义

### External
- Vue 3（`ref`、`computed`、`defineProps`、`defineEmits`、`onMounted`、`watch`）

<!-- MANUAL: -->
