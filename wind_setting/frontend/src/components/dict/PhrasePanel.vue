<template>
  <div class="phrase-panel">
    <!-- 工具栏 -->
    <div class="dict-toolbar">
      <label class="toolbar-checkbox-wrap">
        <input
          type="checkbox"
          class="item-checkbox"
          :checked="allSelected"
          @change="toggleAll"
        />
        <span>全选</span>
      </label>
      <button class="btn btn-primary btn-sm" @click="openAddDialog">
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
      <span class="toolbar-total">共 {{ allPhrases.length }} 条</span>
      <button class="btn btn-sm" @click="handleReset">恢复默认</button>
    </div>

    <!-- 内容区域 -->
    <div class="dict-content-area">
      <div v-if="loading" class="content-loading-overlay">
        <div class="spinner"></div>
      </div>

      <div class="dict-table-wrap">
        <table class="dict-table">
          <colgroup>
            <col class="col-check" />
            <col class="col-toggle" />
            <col class="col-code" />
            <col />
            <col class="col-tag" />
            <col class="col-pos" />
            <col class="col-actions-wide" />
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th>启用</th>
              <th>编码</th>
              <th>内容</th>
              <th>类型</th>
              <th>位置</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in filteredPhrases"
              :key="phraseKey(item)"
              :class="{
                selected: selectedKeys.has(phraseKey(item)),
                'row-disabled': !item.enabled,
              }"
            >
              <td>
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedKeys.has(phraseKey(item))"
                  @change="toggleSelect(item)"
                />
              </td>
              <td>
                <label class="toggle-switch" @click.stop>
                  <input
                    type="checkbox"
                    :checked="item.enabled"
                    @change="handleToggleEnabled(item)"
                  />
                  <span class="toggle-slider"></span>
                </label>
              </td>
              <td>
                <span class="dict-item-code">{{ item.code }}</span>
              </td>
              <td>
                <span v-if="item.type === 'array'">{{
                  item.name || item.code
                }}</span>
                <span v-else>{{ item.text }}</span>
              </td>
              <td>
                <span v-if="item.type === 'array'" class="tag-array">数组</span>
                <span v-else-if="item.type === 'dynamic'" class="tag-dynamic"
                  >动态</span
                >
              </td>
              <td class="td-meta">{{ item.position }}</td>
              <td>
                <div style="display: flex; gap: 2px">
                  <button
                    class="btn-icon"
                    title="编辑"
                    @click="openEditDialog(item)"
                  >
                    ✎
                  </button>
                  <button
                    class="btn-icon btn-delete"
                    title="删除"
                    @click="handleRemove(item)"
                  >
                    &times;
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="filteredPhrases.length === 0">
              <td :colspan="7" class="td-empty">
                {{ searchQuery ? "未找到匹配短语" : "暂无短语" }}
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
          {{ editingPhrase ? "编辑短语" : "添加短语" }}
        </div>
        <div class="form-row">
          <label>编码</label>
          <input
            type="text"
            class="input"
            v-model="newPhrase.code"
            :disabled="!!editingPhrase"
            placeholder="如: zdy"
            style="width: 100%"
          />
        </div>
        <div class="form-row">
          <label>类型</label>
          <label class="radio-inline">
            <input type="radio" v-model="phraseIsArray" :value="false" />
            普通
          </label>
          <label class="radio-inline">
            <input type="radio" v-model="phraseIsArray" :value="true" />
            数组
          </label>
        </div>
        <template v-if="phraseIsArray">
          <div class="form-row">
            <label>名称</label>
            <input
              type="text"
              class="input"
              v-model="newPhrase.name"
              placeholder="如: 特殊符号"
              style="width: 100%"
            />
          </div>
          <div class="form-row">
            <label>字符列表</label>
            <textarea
              class="input"
              v-model="newPhrase.texts"
              rows="4"
              placeholder="每行一个字符或词"
              style="width: 100%; resize: vertical"
            ></textarea>
          </div>
        </template>
        <template v-else>
          <div class="form-row">
            <label>文本</label>
            <textarea
              class="input"
              v-model="newPhrase.text"
              rows="3"
              placeholder="如: 我的地址是xxx 或 $Y-$MM-$DD&#10;支持多行文本"
              style="width: 100%; resize: vertical"
            ></textarea>
          </div>
        </template>
        <div class="form-row">
          <label>位置</label>
          <input
            type="number"
            class="input"
            v-model.number="newPhrase.position"
            min="1"
            style="width: 80px"
          />
        </div>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="dialogVisible = false">
            取消
          </button>
          <button class="btn btn-primary btn-sm" @click="handleSave">
            保存
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
      <div class="dialog-box" style="max-width: 360px">
        <div class="dialog-title">确认</div>
        <div
          style="
            padding: 8px 0 16px;
            font-size: 14px;
            color: #374151;
            white-space: pre-line;
          "
        >
          {{ confirmMessage }}
        </div>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="handleCancel">取消</button>
          <button class="btn btn-primary btn-sm" @click="handleConfirm">
            确定
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import {
  getPhraseList,
  addPhrase,
  updatePhrase,
  removePhrase,
  setPhraseEnabled,
  resetPhrasesToDefault,
  importPhrases,
  exportPhrases,
  type PhraseItem,
} from "../../api/wails";
import { useToast } from "../../composables/useToast";
import { useConfirm } from "../../composables/useConfirm";

const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

const emit = defineEmits<{
  (e: "loading", value: boolean): void;
}>();

// ── 状态 ──
const loading = ref(false);
const allPhrases = ref<PhraseItem[]>([]);
const searchQuery = ref("");
const selectedKeys = ref(new Set<string>());
const dialogVisible = ref(false);
const editingPhrase = ref<PhraseItem | null>(null);
const phraseIsArray = ref(false);
const newPhrase = ref({ code: "", text: "", texts: "", name: "", position: 1 });
const showMoreMenu = ref(false);

// ── 工具函数 ──
function phraseKey(item: PhraseItem): string {
  return `${item.code}||${item.text || ""}||${item.name || ""}`;
}

// ── 计算属性 ──
const filteredPhrases = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return allPhrases.value;
  return allPhrases.value.filter(
    (item) =>
      item.code.toLowerCase().includes(q) ||
      (item.text || "").toLowerCase().includes(q) ||
      (item.name || "").toLowerCase().includes(q),
  );
});

const allSelected = computed(() => {
  if (filteredPhrases.value.length === 0) return false;
  return filteredPhrases.value.every((item) =>
    selectedKeys.value.has(phraseKey(item)),
  );
});

// ── 选择操作 ──
function toggleSelect(item: PhraseItem) {
  const key = phraseKey(item);
  const next = new Set(selectedKeys.value);
  if (next.has(key)) {
    next.delete(key);
  } else {
    next.add(key);
  }
  selectedKeys.value = next;
}

function toggleAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(
      filteredPhrases.value.map((item) => phraseKey(item)),
    );
  }
}

// ── 数据加载 ──
async function loadData() {
  loading.value = true;
  emit("loading", true);
  try {
    allPhrases.value = await getPhraseList();
    selectedKeys.value = new Set();
  } catch (e) {
    toast(`加载短语失败: ${e}`, "error");
  } finally {
    loading.value = false;
    emit("loading", false);
  }
}

// ── 对话框 ──
function openAddDialog() {
  editingPhrase.value = null;
  phraseIsArray.value = false;
  newPhrase.value = { code: "", text: "", texts: "", name: "", position: 1 };
  dialogVisible.value = true;
}

function openEditDialog(item: PhraseItem) {
  editingPhrase.value = item;
  phraseIsArray.value = item.type === "array";
  newPhrase.value = {
    code: item.code,
    text: item.text || "",
    texts: item.texts || "",
    name: item.name || "",
    position: item.position,
  };
  dialogVisible.value = true;
}

// ── 保存 ──
async function handleSave() {
  const { code, text, texts, name, position } = newPhrase.value;
  if (!code.trim()) {
    toast("编码不能为空", "error");
    return;
  }
  const type = phraseIsArray.value ? "array" : "static";
  try {
    if (editingPhrase.value) {
      const oldText = editingPhrase.value.text || "";
      const oldName = editingPhrase.value.name || "";
      const newText = phraseIsArray.value ? texts : text;
      await updatePhrase(code, oldText, oldName, newText, position, null);
      toast("短语已更新");
    } else {
      await addPhrase(code, text, texts, name, type, position);
      toast("短语已添加");
    }
    dialogVisible.value = false;
    await loadData();
  } catch (e) {
    toast(`操作失败: ${e}`, "error");
  }
}

// ── 启用/禁用 ──
async function handleToggleEnabled(item: PhraseItem) {
  try {
    await setPhraseEnabled(
      item.code,
      item.text || "",
      item.name || "",
      !item.enabled,
    );
    await loadData();
  } catch (e) {
    toast(`操作失败: ${e}`, "error");
  }
}

// ── 删除单条 ──
async function handleRemove(item: PhraseItem) {
  const ok = await confirm(`确定删除短语「${item.code}」吗？`);
  if (!ok) return;
  try {
    await removePhrase(item.code, item.text || "", item.name || "");
    toast("短语已删除");
    await loadData();
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

// ── 批量删除 ──
async function handleBatchRemove() {
  const count = selectedKeys.value.size;
  if (count === 0) return;
  const ok = await confirm(`确定删除选中的 ${count} 条短语吗？`);
  if (!ok) return;
  const toDelete = filteredPhrases.value.filter((item) =>
    selectedKeys.value.has(phraseKey(item)),
  );
  try {
    for (const item of toDelete) {
      await removePhrase(item.code, item.text || "", item.name || "");
    }
    toast(`已删除 ${toDelete.length} 条短语`);
    await loadData();
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

// ── 恢复默认 ──
async function handleReset() {
  const ok = await confirm(
    "确定恢复所有短语为系统默认吗？\n自定义短语将会丢失。",
  );
  if (!ok) return;
  try {
    await resetPhrasesToDefault();
    toast("已恢复默认短语");
    await loadData();
  } catch (e) {
    toast(`操作失败: ${e}`, "error");
  }
}

// ── 导入 ──
async function handleImport() {
  try {
    const result = await importPhrases();
    if (result.cancelled) return;
    toast(`已导入 ${result.count} 条短语`);
    await loadData();
  } catch (e) {
    toast(`导入失败: ${e}`, "error");
  }
}

// ── 导出 ──
async function handleExport() {
  try {
    const result = await exportPhrases();
    if (result.cancelled) return;
    toast(`已导出 ${result.count} 条短语`);
  } catch (e) {
    toast(`导出失败: ${e}`, "error");
  }
}

// ── 点击外部关闭下拉 ──
function handleOutsideClick() {
  showMoreMenu.value = false;
}

onMounted(() => {
  document.addEventListener("click", handleOutsideClick);
  loadData();
});

onUnmounted(() => {
  document.removeEventListener("click", handleOutsideClick);
});

defineExpose({ loadData });
</script>

<style>
@import "./dict-shared.css";
</style>

<style scoped>
.phrase-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
