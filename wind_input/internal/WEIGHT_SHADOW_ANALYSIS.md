# Weight / 排序 / Shadow 全局逻辑梳理

## 一、当前实现

### 1. Weight 数值范围

| 来源 | 范围 | 说明 |
|------|------|------|
| **五笔码表(有词频)** | 原值 | 直接使用码表文件中的频率 |
| **五笔码表(无词频)** | `1000000 - entryOrder` | 越靠前越高，基数100万 |
| **五笔前缀匹配** | 原值 - 2,000,000 | 确保精确匹配绝对优先 |
| **UserDict** | 默认100 | 可手动设置 |
| **Shadow top(CompositeDict)** | 999,999 | CompositeDict 内部 |
| **Shadow top(拼音引擎)** | 200,000,000 | applyShadowRules 中 |
| **拼音 rimeScore** | 0 ~ 5,000,000 | exp(nw)+iq+coverage 乘以1000000 |
| **拼音命令** | ~101,000,000 | iq=100.0 |

### 2. 排序器

| 排序器 | 规则 | 使用者 |
|--------|------|--------|
| **Better** | Weight降序 > Text升序 > Code升序 > ConsumedLength降序 | 五笔(frequency模式), 拼音, UserDict |
| **BetterNatural** | NaturalOrder升序 > fallback到Better | 五笔(natural模式), CompositeDict(natural模式) |

### 3. CompositeDict 层级

```
Lv0 Logic(PhraseLayer) → Lv2 User(UserDict) → Lv4 System(CodeTable/PinyinDict)
按层顺序遍历，seen[Text] 去重（先到先得），最终按 sortMode 排序。
Shadow 通过 ShadowProvider 接口独立注入，不作为 DictLayer。
```

### 4. Shadow 规则

| Action | 存储 | 应用位置 | 五笔 | 拼音 |
|--------|------|---------|------|------|
| top | code+word | CompositeDict(w=999999) + 拼音引擎(w=200000000) | ✅ | ✅ |
| delete | code+word | CompositeDict(单字禁删) + 拼音引擎(单字禁删) | ✅ | ✅ |
| order | code+word+offset | 五笔 Phase5.5 ApplyOrderOffsets | ✅ | ❌ |

### 5. 五笔引擎流程
```
Phase 1: CompositeDict.Search(input) → 精确匹配(含Shadow top/delete)
Phase 2: CompositeDict.SearchPrefix(input) → 前缀匹配
Phase 3: 前缀候选 Weight -= 2000000
Phase 4: 合并 + 去重
Phase 5: 排序(Better/BetterNatural)
Phase 5.5: Shadow order 偏移
Phase 6: 过滤 + 截断
```

### 6. 拼音引擎流程
```
步骤 0: 命令(iq=100)
步骤 0b: Poet造句(iq=4.0)
步骤 1: 精确匹配(iq=4.0/2.0)
步骤 1b: 多切分(iq=3.5)
步骤 2: 子词组(iq=3.0/2.0)
步骤 3: 前缀匹配(iq=2.0)
步骤 4: 单字(iq=0.5~4.0)
步骤 5: partial展开(iq=0.0~1.5)
步骤 6: 简拼(iq=1.0/3.0)
排序 → applyShadowRules(只有top/delete) → 过滤 → 截断
```

---

## 二、已知问题

1. **Shadow Order 覆盖 Top**: Order()方法会把已有的top规则覆盖为order
2. **五笔 top 无效**: BetterNatural 模式下 w=999999 不一定排首位
3. **前缀匹配候选 order 无效**: 五笔 Phase 5.5 只读当前 input 的规则，但前缀候选的 code 不同
4. **order 偏移累加无边界**: 多次前移可能产生 offset=-10 导致越界
5. **多 order 规则连锁**: ApplyOrderOffsets 逐个移动导致下标错位
6. **拼音 Shadow 查询过宽**: 查所有候选 Code 的规则可能误删
7. **拼音 top weight 和五笔 top weight 不统一**: 200000000 vs 999999

---

## 三、重构目标

### 架构原则：引擎与呈现层分离

```
引擎层（Phase 1-5）：只负责"找词+打分"，不关心 Shadow
    ↓ 已排序的候选列表
呈现层（Phase 6 Shadow 拦截器）：只负责"重排+过滤"，不修改 weight
    ↓ 最终候选列表
UI 层：显示 + 菜单状态控制
```

**铁律**：Shadow 不修改任何候选的 weight。pin 机制完全在呈现层操作。

### 五笔
1. **置顶**: pin(position=0) — 固定在首位
2. **前移/后移**: pin(当前位置 ± 1) — 精确到第N位
3. **删除**: 隐藏指定多字词（单字禁删）
4. **恢复默认**: 移除该词的所有 Shadow 规则
5. **调词**（未来）: 通过 UserDict.Add(newCode, word, weight) 实现，不属于 Shadow 层

### 拼音
1. **置顶**: pin(position=0)
2. **删除**: 隐藏多字词（单字禁删）
3. **恢复默认**: 移除规则
4. **不支持前移/后移**（与主流拼音输入法一致）

### 统一原则
- Shadow 规则按 `(当前输入编码, 词)` 存储和查询
- **Shadow 与 Weight 彻底切断**：不注入 999999 或 200000000 等魔术数字
- 操作结果只影响当前编码，不影响其他编码（前缀匹配隔离）
- 拼音和五笔共用 Shadow 存储格式，但支持的操作集不同
- 调词 ≠ Shadow，调词通过 UserDict 层实现

### Shadow 存储方案

```yaml
# shadow_wubi86.yaml / shadow_pinyin.yaml
# 只记录被编辑过的条目，未记录的按引擎原始顺序排列
rules:
  sf:
    pinned:                     # 固定位置的词（稀疏槽位，数组顺序=时间戳）
      - word: "标"
        position: 0             # 第0位（首位）
      - word: "材"
        position: 2
    deleted:                    # 隐藏的词（仅多字词）
      - "恭喜发财"
  nihao:
    pinned:
      - word: "你好"
        position: 0
    deleted:
      - "逆号"
```

### 槽位碰撞处理（LIFO 后发先至）

多个词竞争同一 position 时，按 YAML 数组顺序（即操作时间）决定优先级：

```
场景：用户先 pin "标" 到 position=0，再 pin "材" 到 position=0
YAML: pinned: [{word:"材", pos:0}, {word:"标", pos:0}]  # 材后写入排前
结果："材"占位置0，"标"顺延到位置1，其余候选从位置2开始
```

顺延规则：
1. 按 YAML 数组顺序处理 pin 规则（前面的优先级高）
2. 优先级高的占据目标位置
3. 被挤的 pin 词顺延到下一个空位
4. 未被 pin 的候选填充剩余位置

### 应用逻辑（Phase 6 Shadow 拦截器）

```
输入：已排序的候选列表 + 当前编码的 Shadow 规则
输出：最终候选列表

1. 移除 deleted 中的词（单字跳过）
2. 从候选列表中提取有 pin 规则的词（记录原始候选信息）
3. 创建结果数组，按 position 放置 pin 词：
   a. 按 YAML 顺序遍历 pinned 规则
   b. 目标位置已被占 → 占据者顺延
   c. pin 的词不在候选列表中 → 跳过（词库变更后自然失效）
4. 剩余候选按原始顺序填充空位
5. 返回最终列表
```

### UI 操作映射

| 用户操作 | Shadow 变更 | 说明 |
|---------|------------|------|
| 置顶 | pin(word, position=0) 插入 pinned 数组头部 | LIFO 保证最后置顶的排最前 |
| 前移 | pin(word, position=当前位置-1) | 当前位置从候选列表实时获取 |
| 后移 | pin(word, position=当前位置+1) | 同上 |
| 删除 | 加入 deleted | 仅多字词，单字 UI 禁用 |
| 恢复默认 | 从 pinned 和 deleted 中移除 | 恢复引擎原始排序 |

### 菜单状态控制

| 条件 | 禁用项 |
|------|--------|
| 候选在位置 0（首位） | 前移 |
| 候选在最后一位 | 后移 |
| 只有 1 个候选 | 前移、后移 |
| 单字候选 | 删除 |
| 拼音模式 | 前移、后移 |
| 已在 pinned[0]（置顶状态） | 置顶（显示为"已置顶"） |

### 实施步骤

```
Step 1: 重写 ShadowRule 数据结构（pin + delete，废弃 top/order）
Step 2: 重写 Shadow YAML 读写（pinned 数组 + deleted 数组）
Step 3: 实现 Phase 6 Shadow 拦截器（applyShadowPins）
Step 4: 五笔引擎接入（替换 Phase 5.5 + CompositeDict 中的 top 逻辑）
Step 5: 拼音引擎接入（替换 applyShadowRules）
Step 6: UI 回调适配（handle_ui_callbacks.go）
Step 7: 菜单状态联动（候选窗口 UI）
Step 8: 清理旧代码（ShadowActionTop/Order/ApplyOrderOffsets 等）
Step 9: 词频功能实现（五笔 OnCandidateSelected + 防线）
```

---

## 四、词频（自动调频）

### 与 pin 的关系

```
词频 = 引擎层的打分输入（影响 Phase 1-5 的候选排序）
pin  = 呈现层的位置覆盖（Phase 6 强制重排，无视打分）

没有 pin 规则时 → 词频决定排序（高频词自然排前面）
有 pin 规则时   → pin 优先，词频只影响未被 pin 的候选的相对顺序
```

用户"恢复默认"只移除 pin 规则，不清除词频数据。

### 实现机制

通过 UserDict 层实现（已有 Add/IncreaseWeight 基础设施）：

| 场景 | 动作 |
|------|------|
| 用户选词 | `UserDict.IncreaseWeight(code, word, +10)`，词不存在则 `Add(code, word, 初始权重)` |
| 引擎查询 | UserDict(Lv2) 候选优先于 System(Lv4)，高频词通过 weight 自然排前 |
| 开关 | 五笔和拼音各自的 `learning.mode`（auto=启用, manual=关闭） |

### 防线一：权重上限（防 Weight Inflation）

用户高频输入同一个词，weight 可能无限膨胀，最终无法被其他词超越。

```go
const MaxDynamicWeight = 2000 // UserDict 动态权重硬上限

func (ud *UserDict) IncreaseWeight(code, word string, delta int) {
    // ...
    newWeight := current + delta
    if newWeight > MaxDynamicWeight {
        newWeight = MaxDynamicWeight
    }
    ud.entries[code][i].Weight = newWeight
}
```

### 防线二：误选保护（防 Ghost Word）

用户手滑选错生僻词，该词不应立即跳到首位。

策略：**引入选中计数阈值**——首次选中只记录 count=1 不大幅提权，
count >= 3 后才正式提升 weight 参与排序竞争。

```go
type UserWord struct {
    Text      string
    Weight    int
    Count     int       // 选中次数
    CreatedAt time.Time
}

func (ud *UserDict) OnSelected(code, word string) {
    // 已存在：count++ 并按阈值决定是否提权
    if existing {
        existing.Count++
        if existing.Count >= 3 {
            existing.Weight += 10 // 达到阈值后才提权
        }
        return
    }
    // 不存在：添加但初始 weight 较低
    ud.Add(code, word, 50) // 低于系统默认，不会立即越级
}
```

### 五笔"首选保护"策略（推荐）

纯固定五笔打偏僻词痛苦（每次翻页），纯动态五笔首位老变（破坏肌肉记忆）。
折中方案：**自动词频只在第 N 位之后生效，前 N 位由码表固定**。

```
learning:
  mode: auto
  protect_top_n: 3    # 前3位锁定码表原始顺序，自动调频只影响第4位及以后
```

实现：OnCandidateSelected 时，检查被选中的词在码表中的原始排名。
如果原始排名 <= protect_top_n，不写入 UserDict（已经在前列，无需调频）。
如果原始排名 > protect_top_n，才写入 UserDict 提升权重。

只有用户显式的 pin 操作才能改变前 N 位的顺序。

### 配置示例

```yaml
# wubi86.schema.yaml
learning:
  mode: manual          # 五笔默认关闭自动调频（保护肌肉记忆）
  # mode: auto          # 可选开启
  # protect_top_n: 3    # 首选保护：前3位锁定

# pinyin.schema.yaml
learning:
  mode: manual          # 拼音当前也关闭（待评分体系稳定后可开启 auto）
  unigram_path: "dict/pinyin/unigram.txt"
```
