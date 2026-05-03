<script setup lang="ts">
// SchemaRenderer — 全局 Config schema 渲染器
//
// mode='full'：渲染完整的 settings-card 结构（含 card-title）
// mode='bare'：仅渲染 setting-item（嵌入现有 card 内部使用）

import { computed } from 'vue'
import type { FieldDef, PageSchema } from '@/schemas/types'
import FieldRenderer from './FieldRenderer.vue'

const props = defineProps<{
  schema: PageSchema
  formData: Record<string, any>
  mode?: 'full' | 'bare'
}>()

// bare 模式直接返回所有非 card 字段
const bareFields = computed(() =>
  props.schema.filter((f): f is Exclude<FieldDef, { type: 'card' }> => f.type !== 'card'),
)

// full 模式按 card 分组
interface CardGroup {
  label: string
  fields: FieldDef[]
}

const cardGroups = computed((): CardGroup[] => {
  const groups: CardGroup[] = []
  let current: CardGroup | null = null

  for (const field of props.schema) {
    if (field.type === 'card') {
      current = { label: field.label, fields: [] }
      groups.push(current)
    } else if (current) {
      current.fields.push(field)
    }
  }
  return groups
})

function isRenderable(field: FieldDef): boolean {
  if (field.type === 'card' || field.type === 'section') return true
  const f = field as any
  if (f.hidden?.(props.formData)) return false
  return true
}

type LeafField = Exclude<FieldDef, { type: 'card' } | { type: 'section' }>

function asLeafField(f: FieldDef): LeafField {
  return f as LeafField
}
</script>

<template>
  <!-- bare 模式：只输出 setting-item，供嵌入现有 card 使用 -->
  <template v-if="mode === 'bare'">
    <template v-for="field in bareFields" :key="'key' in field ? field.key : field.label">
      <template v-if="isRenderable(field)">
        <div v-if="field.type === 'section'" class="setting-section-title">{{ field.label }}</div>
        <FieldRenderer v-else :field="asLeafField(field)" :form-data="formData" />
      </template>
    </template>
  </template>

  <!-- full 模式：输出完整的 settings-card 结构 -->
  <template v-else>
    <div
      v-for="group in cardGroups"
      :key="group.label"
      class="settings-card"
    >
      <div class="card-title">{{ group.label }}</div>
      <template v-for="field in group.fields" :key="'key' in field ? field.key : field.label">
        <template v-if="isRenderable(field)">
          <div v-if="field.type === 'section'" class="setting-section-title">{{ field.label }}</div>
          <FieldRenderer v-else :field="asLeafField(field)" :form-data="formData" />
        </template>
      </template>
    </div>
  </template>
</template>
