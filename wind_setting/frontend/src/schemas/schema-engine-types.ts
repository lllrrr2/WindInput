// Schema 类型定义 — SchemaConfig（方案引擎设置）驱动的 SchemaFieldDef 体系
//
// 与 types.ts 的区别：
//   - 操作对象是 SchemaConfig 而非全局 Config
//   - 增加 engines / tab 两个过滤维度
//   - key 路径如 "engine.codetable.auto_commit_unique"、"learning.freq.enabled"

import type { SchemaConfig } from '@/api/wails'
import { getPath, setPath } from './types'

export type EngineType = 'codetable' | 'pinyin' | 'mixed'

type SCFn<T> = (cfg: SchemaConfig) => T

// ── 布局标记 ──────────────────────────────────────────────────

export interface EngineSectionDef {
  type: 'section'
  label: string
  engines?: EngineType[]
  /** 仅码表引擎有双 tab，undefined 表示所有 tab 都显示 */
  tab?: 'basic' | 'advanced'
}

// ── 基础字段属性 ──────────────────────────────────────────────

interface SchemaBaseField {
  key: string
  label: string
  hint?: string | SCFn<string>
  engines?: EngineType[]
  tab?: 'basic' | 'advanced'
  dependsOn?: SCFn<boolean>
  hidden?: SCFn<boolean>
}

// ── 具体字段类型 ──────────────────────────────────────────────

export interface SchemaToggleField extends SchemaBaseField {
  type: 'toggle'
}

export interface SchemaSelectOption {
  value: string
  label: string
}

export interface SchemaSelectField extends SchemaBaseField {
  type: 'select'
  options: SchemaSelectOption[]
  width?: string
}

// ── 联合类型 ──────────────────────────────────────────────────

export type SchemaFieldDef = EngineSectionDef | SchemaToggleField | SchemaSelectField
export type EngineSchema = SchemaFieldDef[]

// ── 过滤工具 ──────────────────────────────────────────────────

export function filterEngineSchema(
  schema: EngineSchema,
  engineType: EngineType,
  activeTab: 'basic' | 'advanced',
  cfg: SchemaConfig,
): SchemaFieldDef[] {
  return schema.filter((field) => {
    // engines 过滤
    if ('engines' in field && field.engines && !field.engines.includes(engineType)) return false
    // tab 过滤：有 tab 属性时，只对码表引擎按 tab 过滤
    if ('tab' in field && field.tab) {
      if (engineType === 'codetable' && field.tab !== activeTab) return false
    }
    // hidden 过滤
    if ('hidden' in field && field.hidden?.(cfg)) return false
    return true
  })
}

// 重新导出工具函数，避免引用多个路径
export { getPath, setPath }
