<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import type { Config, EngineInfo } from "../api/settings";
import * as wailsApi from "../api/wails";
import type { SchemaConfig, SchemaInfo } from "../api/wails";

const props = defineProps<{
  formData: Config;
  engines: EngineInfo[];
}>();

const emit = defineEmits<{
  switchEngine: [type: string];
}>();

// 方案选择下拉
const schemaDropdownOpen = ref(false);
const schemaDropdownRef = ref<HTMLElement | null>(null);

// 所有可用方案
const allSchemas = ref<SchemaInfo[]>([]);

// 已勾选方案的 ID 列表（有序）
const enabledSchemaIDs = ref<string[]>([]);

// 各方案的配置（schemaID -> config）
const schemaConfigs = ref<Record<string, SchemaConfig>>({});
const schemaLoading = ref(false);

// 模糊音对话框（记录当前编辑的方案 ID）
const showFuzzyDialog = ref(false);
const fuzzyEditSchemaID = ref("");

// 当前活跃方案 ID
const activeSchemaID = computed(() => props.formData.schema?.active || "");

// 加载所有方案信息和配置
async function loadAllSchemas() {
  schemaLoading.value = true;
  try {
    const schemas = await wailsApi.getAvailableSchemas();
    allSchemas.value = schemas || [];

    // 初始化已勾选列表（从 config.schema.available）
    const available = props.formData.schema?.available || [];
    if (available.length > 0) {
      enabledSchemaIDs.value = available.filter((id: string) =>
        schemas.some((s) => s.id === id),
      );
    } else {
      enabledSchemaIDs.value = schemas.map((s) => s.id);
    }

    // 加载每个已勾选方案的配置
    for (const id of enabledSchemaIDs.value) {
      await loadSchemaConfig(id);
    }
  } catch (e) {
    console.error("加载方案列表失败", e);
  } finally {
    schemaLoading.value = false;
  }
}

async function loadSchemaConfig(schemaID: string) {
  try {
    const cfg = await wailsApi.getSchemaConfig(schemaID);
    schemaConfigs.value[schemaID] = cfg;
  } catch (e) {
    console.error(`加载方案配置失败: ${schemaID}`, e);
  }
}

// 保存方案配置（防抖）
const saveTimers: Record<string, ReturnType<typeof setTimeout>> = {};
function onSchemaConfigChange(schemaID: string) {
  if (saveTimers[schemaID]) clearTimeout(saveTimers[schemaID]);
  saveTimers[schemaID] = setTimeout(() => {
    const cfg = schemaConfigs.value[schemaID];
    if (cfg) {
      wailsApi.saveSchemaConfig(schemaID, cfg).catch((e) => {
        console.error(`保存方案配置失败: ${schemaID}`, e);
      });
    }
  }, 800);
}

// 切换方案勾选状态
function toggleSchemaEnabled(schemaID: string) {
  const idx = enabledSchemaIDs.value.indexOf(schemaID);
  if (idx >= 0) {
    // 取消勾选：不允许取消最后一个，不允许取消当前活跃方案
    if (enabledSchemaIDs.value.length <= 1) return;
    if (schemaID === activeSchemaID.value) return;
    enabledSchemaIDs.value.splice(idx, 1);
    delete schemaConfigs.value[schemaID];
  } else {
    // 勾选：追加到末尾
    enabledSchemaIDs.value.push(schemaID);
    loadSchemaConfig(schemaID);
  }
  // 同步到 config
  props.formData.schema.available = [...enabledSchemaIDs.value];
}

function isSchemaEnabled(schemaID: string) {
  return enabledSchemaIDs.value.includes(schemaID);
}

function getSchemaInfo(schemaID: string): SchemaInfo | undefined {
  return allSchemas.value.find((s) => s.id === schemaID);
}

// 切换活跃方案
function onActiveSchemaSelect(schemaID: string) {
  if (schemaID === activeSchemaID.value) return;
  props.formData.schema.active = schemaID;
  props.engines.forEach((engine) => {
    engine.isActive = engine.type === schemaID;
  });
  emit("switchEngine", schemaID);
}

// 获取方案的引擎类型
function getEngineType(schemaID: string): string {
  return schemaConfigs.value[schemaID]?.engine?.type || "";
}

// 码表配置
function getCodetableConfig(schemaID: string) {
  const cfg = schemaConfigs.value[schemaID];
  if (!cfg) return {};
  if (!cfg.engine.codetable) cfg.engine.codetable = {};
  return cfg.engine.codetable;
}

// 拼音配置
function getPinyinConfig(schemaID: string) {
  const cfg = schemaConfigs.value[schemaID];
  if (!cfg) return {};
  if (!cfg.engine.pinyin) cfg.engine.pinyin = {};
  return cfg.engine.pinyin;
}

function getFuzzyConfig(schemaID: string) {
  const py = getPinyinConfig(schemaID);
  if (!py.fuzzy) py.fuzzy = {};
  return py.fuzzy;
}

// 模糊音
const fuzzyPairs = [
  { field: "zh_z", label: "zh ↔ z", example: "yi'zi → 一直" },
  { field: "ch_c", label: "ch ↔ c", example: "ci'chang → 持常" },
  { field: "sh_s", label: "sh ↔ s", example: "si'jian → 时间" },
  { field: "n_l", label: "n ↔ l", example: "ni → 里" },
  { field: "f_h", label: "f ↔ h", example: "fui → 灰" },
  { field: "r_l", label: "r ↔ l", example: "ren → 冷" },
  { field: "an_ang", label: "an ↔ ang", example: "shan → 上" },
  { field: "en_eng", label: "en ↔ eng", example: "fen → 风" },
  { field: "in_ing", label: "in ↔ ing", example: "xin → 星" },
];

function getFuzzyEnabledCount(schemaID: string) {
  const fuzzy = getFuzzyConfig(schemaID);
  return fuzzyPairs.filter((p) => (fuzzy as any)[p.field]).length;
}

function openFuzzyDialog(schemaID: string) {
  fuzzyEditSchemaID.value = schemaID;
  showFuzzyDialog.value = true;
}

function setAllFuzzyPairs(enabled: boolean) {
  const fuzzy = getFuzzyConfig(fuzzyEditSchemaID.value);
  fuzzyPairs.forEach((p) => {
    (fuzzy as any)[p.field] = enabled;
  });
  onSchemaConfigChange(fuzzyEditSchemaID.value);
}

// 点击外部关闭下拉
function handleDocumentClick(event: MouseEvent) {
  if (
    schemaDropdownRef.value &&
    !schemaDropdownRef.value.contains(event.target as Node)
  ) {
    schemaDropdownOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", handleDocumentClick);
  loadAllSchemas();
});

onUnmounted(() => {
  document.removeEventListener("click", handleDocumentClick);
  Object.values(saveTimers).forEach(clearTimeout);
});
</script>

<template>
  <section class="section">
    <div class="section-header">
      <h2>常用设置</h2>
      <p class="section-desc">输入方案选择与配置</p>
    </div>

    <!-- 方案管理 -->
    <div class="settings-card">
      <div class="card-title">输入方案</div>

      <!-- 当前活跃方案 -->
      <div class="setting-item">
        <div class="setting-info">
          <label>当前方案</label>
          <p class="setting-hint">按快捷键可循环切换</p>
        </div>
        <div class="setting-control">
          <div class="segmented-control">
            <button
              v-for="id in enabledSchemaIDs"
              :key="id"
              :class="{ active: id === activeSchemaID }"
              :disabled="id === activeSchemaID"
              @click="onActiveSchemaSelect(id)"
            >
              {{ getSchemaInfo(id)?.name || id }}
            </button>
          </div>
        </div>
      </div>

      <!-- 可用方案勾选 -->
      <div class="setting-item">
        <div class="setting-info">
          <label>可用方案</label>
          <p class="setting-hint">管理方案切换列表</p>
        </div>
        <div class="setting-control">
          <div
            class="theme-dropdown schema-dropdown"
            ref="schemaDropdownRef"
          >
            <button
              class="btn btn-sm"
              type="button"
              @click="schemaDropdownOpen = !schemaDropdownOpen"
            >
              管理 ({{ enabledSchemaIDs.length }}/{{ allSchemas.length }})
              <span class="theme-select-arrow">&#9662;</span>
            </button>
            <div v-if="schemaDropdownOpen" class="theme-options schema-options">
              <label
                v-for="schema in allSchemas"
                :key="schema.id"
                class="schema-check-item"
                :class="{
                  disabled:
                    schema.id === activeSchemaID &&
                    enabledSchemaIDs.length <= 1,
                }"
              >
                <input
                  type="checkbox"
                  :checked="isSchemaEnabled(schema.id)"
                  :disabled="
                    schema.id === activeSchemaID &&
                    enabledSchemaIDs.length <= 1
                  "
                  @change="toggleSchemaEnabled(schema.id)"
                />
                <div class="schema-check-info">
                  <span class="schema-check-name">{{ schema.name }}</span>
                  <span
                    v-if="schema.id === activeSchemaID"
                    class="theme-badge builtin"
                    >当前</span
                  >
                </div>
              </label>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 各方案配置（所有已勾选方案） -->
    <template v-for="schemaID in enabledSchemaIDs" :key="schemaID">
      <div v-if="schemaConfigs[schemaID]" class="settings-card">
        <div class="card-title">
          <span>{{ schemaConfigs[schemaID].schema?.name || schemaID }}</span>
          <span
            v-if="schemaID === activeSchemaID"
            class="theme-badge builtin"
            style="margin-left: 8px"
            >当前</span
          >
        </div>

        <!-- 码表类型 -->
        <template v-if="getEngineType(schemaID) === 'codetable'">
          <div class="setting-item">
            <div class="setting-info">
              <label>四码唯一自动上屏</label>
              <p class="setting-hint">输入四码且只有唯一候选时自动上屏</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).auto_commit_unique"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>四码空码清空</label>
              <p class="setting-hint">输入四码无匹配时自动清空</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).clear_on_empty_max"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>顶码上屏</label>
              <p class="setting-hint">输入第五码时自动上屏首选</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).top_code_commit"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>标点顶码上屏</label>
              <p class="setting-hint">输入标点时自动上屏首选</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).punct_commit"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>逐码模式</label>
              <p class="setting-hint">关闭前缀匹配，仅显示精确匹配</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).single_code_input"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>显示编码提示</label>
              <p class="setting-hint">在前缀匹配的候选词旁显示剩余编码</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getCodetableConfig(schemaID).show_code_hint"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>候选排序</label>
              <p class="setting-hint">候选词的排列方式</p>
            </div>
            <div class="setting-control">
              <select
                v-model="getCodetableConfig(schemaID).candidate_sort_mode"
                @change="onSchemaConfigChange(schemaID)"
              >
                <option value="frequency">词频优先</option>
                <option value="natural">原始顺序</option>
              </select>
            </div>
          </div>
        </template>

        <!-- 拼音类型 -->
        <template v-if="getEngineType(schemaID) === 'pinyin'">
          <div class="setting-item">
            <div class="setting-info">
              <label>五笔反查提示</label>
              <p class="setting-hint">在候选词旁显示对应的五笔编码</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getPinyinConfig(schemaID).show_wubi_hint"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>智能组句</label>
              <p class="setting-hint">使用语言模型优化多字词组匹配</p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getPinyinConfig(schemaID).use_smart_compose"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label>模糊音</label>
              <p class="setting-hint">
                允许近似发音输入（已启用
                {{ getFuzzyEnabledCount(schemaID) }} 组）
              </p>
            </div>
            <div class="setting-control">
              <label class="switch">
                <input
                  type="checkbox"
                  v-model="getFuzzyConfig(schemaID).enabled"
                  @change="onSchemaConfigChange(schemaID)"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
          <div v-if="getFuzzyConfig(schemaID).enabled" class="setting-item">
            <div class="setting-info">
              <label>模糊音配置</label>
              <p class="setting-hint">选择需要启用的模糊音组</p>
            </div>
            <div class="setting-control">
              <button class="btn btn-sm" @click="openFuzzyDialog(schemaID)">
                配置
              </button>
            </div>
          </div>
        </template>
      </div>
    </template>

    <!-- 模糊音配置对话框 -->
    <div
      class="fuzzy-dialog-overlay"
      v-if="showFuzzyDialog"
      @click.self="showFuzzyDialog = false"
    >
      <div class="fuzzy-dialog">
        <div class="fuzzy-dialog-header">
          <h3>模糊音配置</h3>
          <button class="fuzzy-dialog-close" @click="showFuzzyDialog = false">
            ×
          </button>
        </div>
        <div class="fuzzy-dialog-body">
          <div class="fuzzy-pairs-grid">
            <label
              class="fuzzy-pair-item"
              v-for="pair in fuzzyPairs"
              :key="pair.field"
            >
              <input
                type="checkbox"
                v-model="
                  (getFuzzyConfig(fuzzyEditSchemaID) as any)[pair.field]
                "
                @change="onSchemaConfigChange(fuzzyEditSchemaID)"
              />
              <span class="fuzzy-pair-label">{{ pair.label }}</span>
              <span class="fuzzy-pair-example">{{ pair.example }}</span>
            </label>
          </div>
        </div>
        <div class="fuzzy-dialog-footer">
          <button class="btn btn-sm" @click="setAllFuzzyPairs(true)">
            全选
          </button>
          <button class="btn btn-sm" @click="setAllFuzzyPairs(false)">
            全不选
          </button>
          <button
            class="btn btn-sm btn-primary"
            @click="showFuzzyDialog = false"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.schema-dropdown {
  position: relative;
}
.schema-options {
  min-width: 200px;
  right: 0;
  left: auto;
}
.schema-check-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  cursor: pointer;
  font-size: 13px;
  transition: background-color 0.15s;
}
.schema-check-item:hover {
  background-color: var(--bg-hover, #f3f4f6);
}
.schema-check-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.schema-check-item input {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: var(--accent-color, #2563eb);
  flex-shrink: 0;
}
.schema-check-info {
  display: flex;
  align-items: center;
  gap: 6px;
}
.schema-check-name {
  font-weight: 500;
}
</style>
