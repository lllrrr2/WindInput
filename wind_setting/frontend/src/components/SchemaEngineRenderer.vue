<script setup lang="ts">
// SchemaEngineRenderer — 方案引擎设置 Schema 渲染器
//
// 渲染 SchemaSettingsDialog 内部的引擎设置项。
// 根据 engineType 和 activeTab 过滤字段，直接 mutation localConfig。

import { computed } from 'vue'
import type { EngineSchema, SchemaFieldDef, EngineType } from '@/schemas/schema-engine-types'
import { filterEngineSchema, getPath, setPath } from '@/schemas/schema-engine-types'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

const props = defineProps<{
  schema: EngineSchema
  modelValue: Record<string, any>        // localConfig（SchemaConfig）
  engineType: EngineType
  activeTab?: 'basic' | 'advanced'
}>()

const visibleFields = computed(() =>
  filterEngineSchema(
    props.schema,
    props.engineType,
    props.activeTab ?? 'basic',
    props.modelValue as any,
  ),
)

function getValue(key: string): any {
  return getPath(props.modelValue, key)
}

function setValue(key: string, v: any): void {
  // 数字字段保持数字类型
  const orig = getPath(props.modelValue, key)
  if (typeof orig === 'number' && typeof v === 'string') {
    setPath(props.modelValue, key, Number(v))
  } else {
    setPath(props.modelValue, key, v)
  }
}

function selectVal(key: string): string {
  const v = getValue(key)
  return v == null ? '' : String(v)
}

function isDisabled(field: SchemaFieldDef): boolean {
  if (field.type === 'section') return false
  return field.dependsOn ? !field.dependsOn(props.modelValue as any) : false
}

function resolveHint(field: SchemaFieldDef): string {
  if (field.type === 'section') return ''
  const h = field.hint
  if (!h) return ''
  return typeof h === 'function' ? h(props.modelValue as any) : h
}
</script>

<template>
  <template v-for="field in visibleFields" :key="'key' in field ? field.key : field.label">
    <!-- 分节标题 -->
    <div v-if="field.type === 'section'" class="setting-section-title">{{ field.label }}</div>

    <!-- Toggle -->
    <div
      v-else-if="field.type === 'toggle'"
      class="setting-item"
      :class="{ 'item-disabled': isDisabled(field) }"
    >
      <div class="setting-info">
        <label>{{ field.label }}</label>
        <p v-if="resolveHint(field)" class="setting-hint">{{ resolveHint(field) }}</p>
      </div>
      <div class="setting-control">
        <Switch
          :checked="!!getValue(field.key)"
          :disabled="isDisabled(field)"
          @update:checked="setValue(field.key, $event)"
        />
      </div>
    </div>

    <!-- Select -->
    <div
      v-else-if="field.type === 'select'"
      class="setting-item"
      :class="{ 'item-disabled': isDisabled(field) }"
    >
      <div class="setting-info">
        <label>{{ field.label }}</label>
        <p v-if="resolveHint(field)" class="setting-hint">{{ resolveHint(field) }}</p>
      </div>
      <div class="setting-control">
        <Select
          :model-value="selectVal(field.key)"
          :disabled="isDisabled(field)"
          @update:model-value="setValue(field.key, $event)"
        >
          <SelectTrigger :class="`w-[${field.width || '140px'}]`">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="opt in field.options"
              :key="opt.value"
              :value="opt.value"
            >
              {{ opt.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  </template>
</template>
