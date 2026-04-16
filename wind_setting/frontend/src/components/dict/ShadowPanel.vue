<template>
  <div class="dict-panel-wrap">
    <!-- 工具栏 -->
    <div class="dict-toolbar">
      <label class="toolbar-checkbox-wrap">
        <input type="checkbox" :checked="allSelected" @change="toggleAll" />
        <span>全选</span>
      </label>
      <button
        class="btn btn-primary btn-sm"
        :disabled="readonly"
        @click="openDialog()"
      >
        + 添加
      </button>
      <button
        class="btn btn-sm btn-danger-outline"
        :disabled="selectedKeys.size === 0"
        @click="handleBatchRemove"
      >
        删除{{ selectedKeys.size > 0 ? ` (${selectedKeys.size})` : "" }}
      </button>
      <div class="toolbar-spacer"></div>
      <input
        type="text"
        v-model="searchQuery"
        class="input input-sm toolbar-search"
        placeholder="搜索..."
      />
      <span class="toolbar-total">共 {{ shadowRules.length }} 条</span>
      <button
        class="btn btn-sm btn-danger-outline"
        :disabled="shadowRules.length === 0"
        @click="handleClearAll"
      >
        清空
      </button>
    </div>

    <!-- 内容区 -->
    <div class="dict-content-area" style="position: relative">
      <div v-if="loading" class="content-loading-overlay">
        <div class="spinner"></div>
      </div>

      <!-- 表格 -->
      <div class="dict-table-wrap">
        <table class="dict-table">
          <colgroup>
            <col class="col-check" />
            <col class="col-code" />
            <col />
            <col class="col-tag" />
            <col class="col-pos" />
            <col class="col-actions-wide" />
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th>编码</th>
              <th>词条</th>
              <th>操作类型</th>
              <th>位置</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in filteredShadowRules"
              :key="item.code + '|' + item.word"
              :class="{
                selected: selectedKeys.has(item.code + '|' + item.word),
              }"
            >
              <td>
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedKeys.has(item.code + '|' + item.word)"
                  @change="toggleSelect(item)"
                />
              </td>
              <td>
                <span class="dict-item-code">{{ item.code }}</span>
              </td>
              <td>{{ item.word }}</td>
              <td>
                <span :class="item.type === 'pin' ? 'tag-pin' : 'tag-delete'">
                  {{ getShadowActionLabel(item.type) }}
                </span>
              </td>
              <td class="td-weight">
                {{ item.type === "pin" ? item.position : "" }}
              </td>
              <td>
                <div style="display: flex; gap: 4px; align-items: center">
                  <button
                    class="btn-icon"
                    @click="openDialog(item)"
                    title="编辑"
                  >
                    ✎
                  </button>
                  <button
                    class="btn-icon btn-delete"
                    @click="handleRemove(item)"
                  >
                    ×
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="filteredShadowRules.length === 0">
              <td :colspan="6" class="td-empty">
                {{ searchQuery ? "未找到匹配规则" : "暂无调整规则" }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 添加/编辑对话框 -->
    <div
      v-if="dialogVisible"
      class="dialog-overlay"
      @click.self="dialogVisible = false"
    >
      <div class="dialog-box">
        <div class="dialog-title">
          {{ dialogEditing ? "编辑规则" : "添加规则" }}
        </div>
        <div class="form-row">
          <label>编码</label>
          <input
            type="text"
            v-model="form.code"
            class="input"
            placeholder="如: sf"
            :disabled="dialogEditing"
          />
        </div>
        <div class="form-row">
          <label>词条</label>
          <input
            type="text"
            v-model="form.word"
            class="input"
            placeholder="如: 村"
            :disabled="dialogEditing"
          />
        </div>
        <div class="form-row">
          <label>操作</label>
          <select v-model="form.action" class="select">
            <option value="pin">固定位置</option>
            <option value="delete">隐藏词条</option>
          </select>
        </div>
        <div class="form-row" v-if="form.action === 'pin'">
          <label>目标位置</label>
          <input
            type="number"
            v-model.number="form.position"
            class="input input-sm"
            min="0"
            placeholder="0=首位"
          />
        </div>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="dialogVisible = false">
            取消
          </button>
          <button class="btn btn-primary btn-sm" @click="handleSave">
            {{ dialogEditing ? "保存" : "添加" }}
          </button>
        </div>
      </div>
    </div>

    <!-- 确认对话框 -->
    <div
      v-if="confirmVisible"
      class="dialog-overlay"
      @click.self="handleCancel"
    >
      <div class="dialog-box">
        <div class="dialog-title">确认操作</div>
        <p style="font-size: 14px; color: #374151; margin-bottom: 0">
          {{ confirmMessage }}
        </p>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="handleCancel">取消</button>
          <button class="btn btn-primary btn-sm" @click="handleConfirm">
            确认
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useToast } from "../../composables/useToast";
import { useConfirm } from "../../composables/useConfirm";
import {
  getShadowBySchema,
  pinShadowWordForSchema,
  deleteShadowWordForSchema,
  removeShadowRuleForSchema,
  type ShadowRuleItem,
} from "../../api/wails";

const props = defineProps<{
  schemaId: string;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (e: "loading", value: boolean): void;
  (e: "schema-changed"): void;
}>();

defineExpose({ loadData });

const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

const shadowRules = ref<ShadowRuleItem[]>([]);
const searchQuery = ref("");
const selectedKeys = ref<Set<string>>(new Set());
const loading = ref(false);

const filteredShadowRules = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return shadowRules.value;
  return shadowRules.value.filter(
    (item) =>
      item.code.toLowerCase().includes(q) ||
      item.word.toLowerCase().includes(q),
  );
});

const dialogVisible = ref(false);
const dialogEditing = ref(false);
const form = ref({
  code: "",
  word: "",
  action: "pin" as "pin" | "delete",
  position: 0,
});

const allSelected = computed(
  () =>
    filteredShadowRules.value.length > 0 &&
    filteredShadowRules.value.every((item) =>
      selectedKeys.value.has(item.code + "|" + item.word),
    ),
);

function itemKey(item: ShadowRuleItem) {
  return item.code + "|" + item.word;
}

function toggleAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(shadowRules.value.map(itemKey));
  }
}

function toggleSelect(item: ShadowRuleItem) {
  const key = itemKey(item);
  const next = new Set(selectedKeys.value);
  if (next.has(key)) {
    next.delete(key);
  } else {
    next.add(key);
  }
  selectedKeys.value = next;
}

function getShadowActionLabel(type: string): string {
  if (type === "pin") return "固定位置";
  if (type === "delete") return "隐藏";
  return type;
}

function openDialog(item?: ShadowRuleItem) {
  if (item) {
    dialogEditing.value = true;
    form.value = {
      code: item.code,
      word: item.word,
      action: item.type === "pin" ? "pin" : "delete",
      position: item.position ?? 0,
    };
  } else {
    dialogEditing.value = false;
    form.value = { code: "", word: "", action: "pin", position: 0 };
  }
  dialogVisible.value = true;
}

async function handleSave() {
  if (!form.value.code.trim() || !form.value.word.trim()) {
    toast("编码和词条不能为空", "error");
    return;
  }
  try {
    if (dialogEditing.value) {
      // 先删除旧规则，再添加新规则
      await removeShadowRuleForSchema(
        props.schemaId,
        form.value.code,
        form.value.word,
      );
    }
    if (form.value.action === "pin") {
      await pinShadowWordForSchema(
        props.schemaId,
        form.value.code,
        form.value.word,
        form.value.position,
      );
    } else {
      await deleteShadowWordForSchema(
        props.schemaId,
        form.value.code,
        form.value.word,
      );
    }
    toast(dialogEditing.value ? "规则已保存" : "规则已添加");
    dialogVisible.value = false;
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`保存失败: ${e}`, "error");
  }
}

async function loadData() {
  loading.value = true;
  emit("loading", true);
  try {
    shadowRules.value = (await getShadowBySchema(props.schemaId)) || [];
    selectedKeys.value = new Set();
  } finally {
    loading.value = false;
    emit("loading", false);
  }
}

async function handleRemove(item: ShadowRuleItem) {
  const ok = await confirm(`确定删除「${item.word}」的调整规则？`);
  if (!ok) return;
  try {
    await removeShadowRuleForSchema(props.schemaId, item.code, item.word);
    toast(`已删除「${item.word}」的规则`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

async function handleBatchRemove() {
  const count = selectedKeys.value.size;
  const ok = await confirm(`确定删除选中的 ${count} 条调整规则？`);
  if (!ok) return;
  try {
    const items = shadowRules.value.filter((item) =>
      selectedKeys.value.has(itemKey(item)),
    );
    for (const item of items) {
      await removeShadowRuleForSchema(props.schemaId, item.code, item.word);
    }
    toast(`已删除 ${count} 条规则`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

async function handleClearAll() {
  if (shadowRules.value.length === 0) return;
  const ok = await confirm(
    "确定清空当前方案的所有候选调整规则吗？此操作不可撤销。",
  );
  if (!ok) return;
  try {
    for (const item of shadowRules.value) {
      await removeShadowRuleForSchema(props.schemaId, item.code, item.word);
    }
    toast(`已清空 ${shadowRules.value.length} 条规则`, "success");
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`清空失败: ${e}`, "error");
  }
}

onMounted(() => {
  loadData();
});
</script>

<style>
@import "./dict-shared.css";
</style>
