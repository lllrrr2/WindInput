# 拼音候选质量优化

## 概述

本文档记录拼音引擎候选生成过程中发现的质量问题及其修复方案。

## 问题 1：模糊音配置热更新不生效

### 现象
用户在设置中关闭模糊音后保存，输入法仍然按模糊音匹配（如 `linwai` 输出"另外"）。

### 根因
1. **混输方案未处理**：`reload_handler.go` 的 `reloadActiveSchemaConfig()` 中，switch 语句只处理了 `EngineTypePinyin` 和 `EngineTypeCodeTable`，缺少 `EngineTypeMixed` 分支。用户使用的五笔拼音（`wubi86_pinyin`）是混输方案，配置保存后 `UpdatePinyinOptions` 不会被调用。
2. **线程安全缺失**：`Engine.config.Fuzzy` 的读写无同步机制，热更新时存在数据竞争。
3. **字段映射不完整**：`FuzzyConfig` 的 `IanIang` 和 `UanUang` 在 3 处配置映射中遗漏。

### 修复
- 新增 `EngineTypeMixed` case，从次方案获取拼音配置
- `Engine` 新增 `fuzzyPtr atomic.Pointer[FuzzyConfig]`，读写通过原子操作
- 补全所有配置映射中的 `IanIang`/`UanUang` 字段

## 问题 2：候选词排序不稳定

### 现象
同一输入（如 `linwai`）多次查询，候选顺序不确定（如"林外/临外/林歪"顺序变化）。

### 根因
三层叠加：
1. **Map 迭代随机**：候选词先存入 `map[string]*Candidate`，Go 的 map 遍历顺序不确定
2. **非稳定排序**：多处使用 `sort.Slice`，对相等元素不保证顺序
3. **比较函数非全序**：`Better()` 在 Weight/Code/NaturalOrder/ConsumedLength 都相同时无法区分不同文本的候选

### 修复
- `Better()` 增加 `Text` 最终兜底比较，确保全序关系
- 关键排序路径统一改用 `sort.SliceStable`

## 问题 3：Viterbi 造句产生高频单字拼凑伪词组

### 现象
输入 `qinta` 出现"前他"、`linwai` 出现"林歪"等词库中不存在的单字组合。

### 根因
Viterbi 造句在找不到多字词路径时，退化为高频单字组合：
1. **Lattice 单字回退无惩罚**：`BuildLattice` 为确保通路，为每个音节添加单字节点，但 LogProb 与正常词库词完全一样
2. **Bigram 回退无惩罚**：`BigramModel.LogProb` 找不到词对时直接返回 Unigram 概率，高频单字组合得分可超过低频真实词组
3. **无过滤机制**：纯单字拼凑的 Viterbi 结果直接进入候选列表

### 修复
- **过滤**：短输入（≤3 音节）的纯单字 Viterbi 结果直接丢弃
- **Lattice 惩罚**：单字回退节点施加 `singleCharPenalty = -3.0`
- **Bigram 回退惩罚**：未命中词对时施加 `backoffPenalty = -4.0`
- **长句兜底**：≥4 音节保留单字回退，但降低 `initialQuality`

## 问题 4：非首音节单字候选导致输入丢失

### 现象
输入 `linwai` 时，候选末尾出现"外/歪/崴"（仅对应第二音节 `wai`），选中后 `lin` 被丢弃。

### 根因
`convertCore` 步骤4中，非首音节单字（如 `wai→外`）被加入初始候选列表，`ConsumedLength` 设为消耗到该音节结束位置（整个 `linwai`），但文本只有一个字。上屏时引擎认为全部输入已处理完毕，前面未确认的音节被丢弃。

### 修复
移除非首音节单字候选的生成。这些候选应在用户部分上屏（确认首音节）后自然出现，不需要在初始列表中预生成。

## 问题 5：DAG 贪心切分导致音节丢失

### 现象
输入 `henihejiele` 时，DAG 贪心选择最长匹配 "hen"，导致后续 "i" 无法匹配，"ni" 被拆散。实际应切分为 "he"+"ni"+"he"+"jie"+"le" 以覆盖全部输入。

### 根因
`MaximumMatch()` 使用正向贪心——每步选当前位置最长音节。这在存在歧义时无法回溯：`hen`(3字符) > `he`(2字符)，但选了 `hen` 后 `ihejiele` 的 `i` 不是有效音节起点，后续覆盖率反而更低。

### 修复
将 `MaximumMatch()` 从贪心改为 DP 全局最优：
- `dp[i]` 记录从位置 0 到位置 i 的最大覆盖字符数
- 遍历所有 DAG 节点更新最优路径
- 回溯构建最终音节序列

同时 `Parser.parseSegment` 的余部处理也增加 DP 回退，避免贪心 `MatchPrefixAt` 产生歧义切分。

## 问题 6：前导 partial 音节导致候选消耗不匹配

### 现象
输入 `lwai`（l 为 partial，wai 为 exact）时：
1. 步骤 4a 的 `dict.Lookup("li")` 返回"立案"等多字词，但 `ConsumedLength` 只设为 1（partial "l" 的长度），消耗与候选不匹配
2. 简拼模式下 "lw" 匹配"龙王"（long+wang），但实际输入的 "wai" ≠ "w"，选中后 "ai" 丢失

### 根因
1. **步骤 4a/4b 未过滤多字词**：注释写"单字候选"但 `Lookup` 返回所有条目
2. **简拼匹配未区分纯 partial**：混合输入（partial+exact）中全拼音节的首字母不应被当作简拼

### 修复
1. 步骤 4a/4b 增加 `charCount != 1` 过滤，仅保留单字候选
2. 简拼匹配增加 `isPureAbbrev` 守卫：`syllableCount == 0 && len(allSyllables) >= 2`，仅在所有音节都是 partial 时启用

## 问题 7：ContiguousCompleted 防护体系

### 背景
问题 4-6 的根源是同一个：候选的 `ConsumedLength` 跨过了输入中未被候选覆盖的音节（partial 间隔），导致用户选词后这些音节被静默丢弃。

### 设计
引入 `ContiguousCompletedFromStart()` 方法：从输入起始位置返回连续的完成音节（遇到 partial 立即中断）。

```
"nihao"  → [ni(E),hao(E)]       → contiguous=["ni","hao"], end=5
"nihdao" → [ni(E),h(P),dao(E)]  → contiguous=["ni"],       end=2
"lwai"   → [l(P),wai(E)]        → contiguous=[],            end=0
```

所有候选生成步骤（0b/1/1b/2/3/4/5/6）统一使用 `contiguousCount`/`contiguousSyllables` 替代 `syllableCount`/`completedSyllables`：

| 步骤 | 触发条件 | 说明 |
|------|---------|------|
| 0b Viterbi | `contiguousCount >= 2` | 仅对连续完成音节造句 |
| 1 精确匹配 | `contiguousCount > 0` | 仅匹配连续音节编码 |
| 1b 多切分 | `contiguousCount > 0` | 同上 |
| 2 子词组 | `contiguousCount > 1` | 传入 contiguousSyllables |
| 3 前缀匹配 | `contiguousCount > 0` | 同上 |
| 4 首音节单字 | `contiguousCount > 0` | 首连续音节 |
| 4a leading partial | `contiguousCount == 0 && syllableCount > 0` | 首段 partial 展开 |
| 4b 多 partial 首字 | `contiguousCount == 0 && len(allSyllables) > 1` | 纯 partial 首字 |
| 5 partial 前缀 | `partial != "" && (contiguousCount > 0 \|\| single)` | 尾部 partial 安全条件 |
| 6 简拼 | `syllableCount == 0 && len(allSyllables) >= 2` | 纯 partial 简拼 |

### 子词组 ConsumedLength 精确化

原逻辑中非首位子词组（start>0）的 `ConsumedLength = len(parsed.Input)` 消耗整个输入。修正为 `ConsumedBytesForCompletedN(start + length)`——精确到子词组最后一个音节的结束位置。非首位子词组的 `initialQuality` 降级为 2.0（首位为 3.0）。

## 问题 8：多字词 Unigram 估算缺失

### 现象
`ruguo` 首候选为"入锅"而非"如果"。Viterbi 中词库合法词组（如"和解"）被单字组合碾压。

### 根因
多字词不在 Unigram 模型中时直接返回 `minProb`（极低值），导致所有未登录多字词在 Viterbi 路径比较中完败于高频单字组合。

### 修复
- `UnigramModel.LogProb`：多字词 fallback 改用 `CharBasedScore`（字符平均 LogProb），常见字组成的词得分更高
- `BinaryUnigramModel.LogProb`：同步应用 `CharBasedScore` fallback
- `Lattice.calcLogProb`：多字词不在 Unigram 中时使用 `CharBasedScore` 替代原始权重归一化
- `BigramModel` 回退惩罚从 `-4.0` 调整为 `-1.0`，避免过度惩罚
