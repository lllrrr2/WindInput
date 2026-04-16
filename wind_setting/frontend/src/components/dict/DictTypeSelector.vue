<template>
  <div class="dict-type-selector" @click.stop>
    <span class="selector-label">词库类型:</span>
    <div class="selector-btn-wrap">
      <button class="selector-btn" @click="toggle">
        <span class="selector-btn-text">{{ displayLabel }}</span>
        <span class="selector-arrow">▾</span>
      </button>
      <div v-if="open" class="selector-dropdown">
        <div
          class="selector-item"
          :class="{ active: modelValue.mode === 'phrases' }"
          @click="selectPhrases"
        >
          <span class="item-name">快捷短语</span>
        </div>
        <div v-if="schemas.length > 0" class="selector-divider"></div>
        <div
          v-for="s in schemas"
          :key="s.schema_id"
          class="selector-item"
          :class="{
            active:
              modelValue.mode === 'schema' &&
              modelValue.schemaId === s.schema_id,
          }"
          @click="selectSchema(s)"
        >
          <span class="status-dot" :class="dotClass(s)"></span>
          <span class="item-name">方案：{{ s.schema_name }}</span>
          <span class="item-engine-tag">{{ engineLabel(s) }}</span>
          <span v-if="s.status === 'orphaned'" class="orphan-tag">(残留)</span>
        </div>
        <div v-if="schemas.length === 0" class="selector-empty">暂无方案</div>
      </div>
    </div>
    <slot name="actions"></slot>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import type { SchemaStatusItem } from "../../api/wails";

interface ModelValue {
  mode: "phrases" | "schema";
  schemaId: string;
}

const props = defineProps<{
  modelValue: ModelValue;
  schemas: SchemaStatusItem[];
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: ModelValue): void;
}>();

const open = ref(false);
let openedAt = 0;

const displayLabel = computed(() => {
  if (props.modelValue.mode === "phrases") {
    return "快捷短语";
  }
  const found = props.schemas.find(
    (s) => s.schema_id === props.modelValue.schemaId,
  );
  return found ? `方案：${found.schema_name}` : "选择方案";
});

function toggle() {
  if (open.value) {
    open.value = false;
  } else {
    open.value = true;
    openedAt = Date.now();
  }
}

function selectPhrases() {
  emit("update:modelValue", { mode: "phrases", schemaId: "" });
  open.value = false;
}

function selectSchema(s: SchemaStatusItem) {
  emit("update:modelValue", { mode: "schema", schemaId: s.schema_id });
  open.value = false;
}

function dotClass(s: SchemaStatusItem) {
  if (s.status === "enabled") return "dot-enabled";
  if (s.status === "disabled") return "dot-disabled";
  return "dot-orphaned";
}

function engineLabel(s: SchemaStatusItem): string {
  switch (s.engine_type) {
    case "codetable":
      return "码表";
    case "pinyin":
      return "拼音";
    case "mixed":
      return "混输";
    default:
      return "";
  }
}

function handleOutsideClick() {
  if (Date.now() - openedAt > 100) {
    open.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", handleOutsideClick);
});

onUnmounted(() => {
  document.removeEventListener("click", handleOutsideClick);
});
</script>

<style scoped>
.dict-type-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0 10px;
  flex-shrink: 0;
}

.selector-label {
  font-size: 13px;
  color: #6b7280;
  font-weight: 500;
  flex-shrink: 0;
}

.selector-btn-wrap {
  position: relative;
  width: 50%;
  min-width: 180px;
}

.selector-btn {
  width: 100%;
  padding: 6px 12px;
  font-size: 13px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #374151;
  transition: border-color 0.15s;
}

.selector-btn:hover {
  border-color: #9ca3af;
}

.selector-btn-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selector-arrow {
  font-size: 11px;
  opacity: 0.6;
  flex-shrink: 0;
  margin-left: 4px;
}

.selector-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  min-width: 100%;
  z-index: 50;
  max-height: 320px;
  overflow-y: auto;
  padding: 4px 0;
}

.selector-item {
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #374151;
}

.selector-item:hover {
  background: #f3f4f6;
}

.selector-item.active {
  background: #eff6ff;
  color: #1d4ed8;
}

.item-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-engine-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: #f3f4f6;
  color: #6b7280;
  flex-shrink: 0;
}

.selector-divider {
  height: 1px;
  background: #e5e7eb;
  margin: 4px 0;
}

.selector-empty {
  padding: 8px 14px;
  font-size: 13px;
  color: #9ca3af;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-enabled {
  background: #22c55e;
}

.dot-disabled {
  background: #d1d5db;
}

.dot-orphaned {
  background: #f97316;
}

.orphan-tag {
  font-size: 11px;
  color: #dc2626;
  flex-shrink: 0;
}

.selector-dropdown::-webkit-scrollbar {
  width: 6px;
}
.selector-dropdown::-webkit-scrollbar-track {
  background: transparent;
}
.selector-dropdown::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}
</style>
