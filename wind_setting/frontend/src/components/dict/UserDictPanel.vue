<template>
  <div class="user-dict-panel">
    <!-- 工具栏 -->
    <div class="dict-toolbar">
      <label class="toolbar-checkbox-wrap">
        <input type="checkbox" :checked="allSelected" @change="toggleAll" />
        <span>全选</span>
      </label>
      <button
        class="btn btn-primary btn-sm"
        :disabled="readonly"
        @click="openAddDialog"
      >
        + 添加
      </button>
      <button
        class="btn btn-sm btn-danger-outline"
        :disabled="selectedKeys.size === 0"
        @click="handleBatchDelete"
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
      <span class="toolbar-total">共 {{ userDict.length }} 条</span>
    </div>

    <!-- 内容区域 -->
    <div class="dict-content-area" style="position: relative">
      <div v-if="loading" class="content-loading-overlay">
        <div class="spinner"></div>
      </div>

      <div class="dict-table-wrap">
        <table class="dict-table">
          <colgroup>
            <col class="col-check" />
            <col class="col-code" />
            <col />
            <col class="col-weight" />
            <col class="col-actions-wide" />
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th @click="toggleSort('code')" class="sortable-th">
                编码
                <span class="sort-icon">{{
                  sortKey === "code" ? (sortAsc ? "▲" : "▼") : "⇅"
                }}</span>
              </th>
              <th @click="toggleSort('text')" class="sortable-th">
                词条
                <span class="sort-icon">{{
                  sortKey === "text" ? (sortAsc ? "▲" : "▼") : "⇅"
                }}</span>
              </th>
              <th @click="toggleSort('weight')" class="sortable-th">
                权重
                <span class="sort-icon">{{
                  sortKey === "weight" ? (sortAsc ? "▲" : "▼") : "⇅"
                }}</span>
              </th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, idx) in filteredUserDict"
              :key="idx"
              :class="{ selected: selectedKeys.has(itemKey(item)) }"
              @dblclick="openEditDialog(item)"
            >
              <td>
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedKeys.has(itemKey(item))"
                  @change="toggleSelect(item)"
                />
              </td>
              <td>
                <span class="dict-item-code">{{ item.code }}</span>
              </td>
              <td>{{ item.text }}</td>
              <td class="td-weight">{{ item.weight || 0 }}</td>
              <td>
                <div style="display: flex; gap: 2px">
                  <button
                    class="btn-icon"
                    @click="openEditDialog(item)"
                    title="编辑"
                  >
                    ✎
                  </button>
                  <button
                    class="btn-icon btn-delete"
                    @click="handleDelete(item)"
                    title="删除"
                  >
                    &times;
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="filteredUserDict.length === 0">
              <td :colspan="5" class="td-empty">
                {{ searchQuery ? "未找到匹配词条" : "暂无用户词条" }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- AddWordPage 对话框 -->
    <AddWordPage
      v-if="addWordVisible"
      :initialText="editText"
      :initialCode="editCode"
      :initialSchema="schemaId"
      @close="handleAddWordClose"
    />

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
import { ref, computed, onMounted } from "vue";
import { useToast } from "../../composables/useToast";
import { useConfirm } from "../../composables/useConfirm";
import AddWordPage from "../../pages/AddWordPage.vue";
import { getUserDictBySchema, removeUserWordForSchema } from "../../api/wails";

interface UserWordItem {
  code: string;
  text: string;
  weight: number;
  created_at?: string;
}

interface ImportExportResult {
  cancelled: boolean;
  count: number;
  total?: number;
  path?: string;
}

const props = defineProps<{
  schemaId: string;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (e: "loading", val: boolean): void;
  (e: "schema-changed"): void;
}>();

defineExpose({ loadData });

const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

// 状态
const userDict = ref<UserWordItem[]>([]);
const searchQuery = ref("");
const sortKey = ref<"code" | "text" | "weight">("code");
const sortAsc = ref(true);
const selectedKeys = ref(new Set<string>());
const loading = ref(false);
const addWordVisible = ref(false);
const editText = ref("");
const editCode = ref("");

// 过滤+排序
const filteredUserDict = computed(() => {
  let list = userDict.value;
  const q = searchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter(
      (item) =>
        item.code.toLowerCase().includes(q) ||
        item.text.toLowerCase().includes(q),
    );
  }
  const key = sortKey.value;
  const asc = sortAsc.value;
  list = [...list].sort((a, b) => {
    const av = a[key];
    const bv = b[key];
    if (typeof av === "number" && typeof bv === "number") {
      return asc ? av - bv : bv - av;
    }
    const as = String(av ?? "");
    const bs = String(bv ?? "");
    return asc ? as.localeCompare(bs) : bs.localeCompare(as);
  });
  return list;
});

const allSelected = computed(
  () =>
    filteredUserDict.value.length > 0 &&
    filteredUserDict.value.every((item) =>
      selectedKeys.value.has(itemKey(item)),
    ),
);

function itemKey(item: UserWordItem) {
  return `${item.code}|${item.text}`;
}

function toggleAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(filteredUserDict.value.map(itemKey));
  }
}

function toggleSelect(item: UserWordItem) {
  const k = itemKey(item);
  const next = new Set(selectedKeys.value);
  if (next.has(k)) {
    next.delete(k);
  } else {
    next.add(k);
  }
  selectedKeys.value = next;
}

function toggleSort(key: "code" | "text" | "weight") {
  if (sortKey.value === key) {
    sortAsc.value = !sortAsc.value;
  } else {
    sortKey.value = key;
    sortAsc.value = true;
  }
}

async function loadData() {
  loading.value = true;
  emit("loading", true);
  try {
    userDict.value = (await getUserDictBySchema(
      props.schemaId,
    )) as UserWordItem[];
    selectedKeys.value = new Set();
  } catch (e) {
    toast("加载用户词库失败", "error");
  } finally {
    loading.value = false;
    emit("loading", false);
  }
}

function openAddDialog() {
  editText.value = "";
  editCode.value = "";
  addWordVisible.value = true;
}

function openEditDialog(item: UserWordItem) {
  editText.value = item.text;
  editCode.value = item.code;
  addWordVisible.value = true;
}

async function handleAddWordClose() {
  addWordVisible.value = false;
  await loadData();
  emit("schema-changed");
}

async function handleDelete(item: UserWordItem) {
  const ok = await confirm(`确定删除词条「${item.text}」？`);
  if (!ok) return;
  try {
    await removeUserWordForSchema(props.schemaId, item.code, item.text);
    toast("已删除", "success");
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast("删除失败", "error");
  }
}

async function handleBatchDelete() {
  if (selectedKeys.value.size === 0) return;
  const ok = await confirm(
    `确定删除选中的 ${selectedKeys.value.size} 个词条？`,
  );
  if (!ok) return;
  let failed = 0;
  for (const item of filteredUserDict.value) {
    if (selectedKeys.value.has(itemKey(item))) {
      try {
        await removeUserWordForSchema(props.schemaId, item.code, item.text);
      } catch {
        failed++;
      }
    }
  }
  if (failed > 0) {
    toast(`删除完成，${failed} 个失败`, "error");
  } else {
    toast("已删除选中词条", "success");
  }
  await loadData();
  emit("schema-changed");
}

onMounted(() => {
  loadData();
});
</script>

<style>
@import "./dict-shared.css";
</style>

<style scoped>
.user-dict-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
</style>
