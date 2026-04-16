<template>
  <div class="freq-panel">
    <!-- 工具栏 -->
    <div class="dict-toolbar">
      <label class="toolbar-checkbox-wrap">
        <input type="checkbox" :checked="allSelected" @change="toggleAll" />
        <span>全选</span>
      </label>
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
        @input="onSearchInput"
      />
      <span class="toolbar-total">共 {{ total }} 条</span>
      <button class="btn btn-sm btn-danger-outline" @click="handleClear">
        清空
      </button>
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
            <col class="col-count" />
            <col class="col-boost" />
            <col class="col-time" />
            <col class="col-action" />
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th>编码</th>
              <th>词条</th>
              <th>次数</th>
              <th>提升</th>
              <th>最后使用</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, idx) in freqList"
              :key="idx"
              :class="{ selected: selectedKeys.has(itemKey(item)) }"
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
              <td class="td-weight">{{ item.count }}</td>
              <td class="td-weight">{{ item.boost }}</td>
              <td class="td-meta">{{ formatLastUsed(item.last_used) }}</td>
              <td>
                <button
                  class="btn-icon btn-delete"
                  @click="handleDelete(item)"
                  title="删除"
                >
                  &times;
                </button>
              </td>
            </tr>
            <tr v-if="freqList.length === 0">
              <td :colspan="7" class="td-empty">
                {{ searchQuery ? "未找到匹配词频记录" : "暂无词频记录" }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="dict-pager" v-if="total > pageSize">
        <button
          class="btn btn-sm"
          :disabled="page === 0"
          @click="
            page--;
            loadData();
          "
        >
          上一页
        </button>
        <span class="dict-pager-info">
          {{ page * pageSize + 1 }}-{{
            Math.min((page + 1) * pageSize, total)
          }}
          / {{ total }}
        </span>
        <button
          class="btn btn-sm"
          :disabled="(page + 1) * pageSize >= total"
          @click="
            page++;
            loadData();
          "
        >
          下一页
        </button>
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
import { ref, computed, onMounted } from "vue";
import { useToast } from "../../composables/useToast";
import { useConfirm } from "../../composables/useConfirm";
import { getFreqList, deleteFreq, clearFreq } from "../../api/wails";
import type { FreqItem } from "../../api/wails";

const props = defineProps<{
  schemaId: string;
  schemaName: string;
}>();

const emit = defineEmits<{
  (e: "loading", val: boolean): void;
}>();

defineExpose({ loadData });

const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

// 状态
const freqList = ref<FreqItem[]>([]);
const searchQuery = ref("");
const total = ref(0);
const page = ref(0);
const pageSize = 100;
const selectedKeys = ref(new Set<string>());
const loading = ref(false);

// 防抖
let searchTimer: ReturnType<typeof setTimeout> | null = null;

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    page.value = 0;
    loadData();
  }, 300);
}

const allSelected = computed(
  () =>
    freqList.value.length > 0 &&
    freqList.value.every((item) => selectedKeys.value.has(itemKey(item))),
);

function itemKey(item: FreqItem) {
  return `${item.code}|${item.text}`;
}

function toggleAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(freqList.value.map(itemKey));
  }
}

function toggleSelect(item: FreqItem) {
  const k = itemKey(item);
  const next = new Set(selectedKeys.value);
  if (next.has(k)) {
    next.delete(k);
  } else {
    next.add(k);
  }
  selectedKeys.value = next;
}

function formatLastUsed(ts: number): string {
  if (!ts) return "-";
  const d = new Date(ts * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

async function loadData() {
  loading.value = true;
  emit("loading", true);
  try {
    const result = await getFreqList(
      props.schemaId,
      searchQuery.value.trim(),
      pageSize,
      page.value * pageSize,
    );
    freqList.value = result.entries;
    total.value = result.total;
    selectedKeys.value = new Set();
  } catch (e) {
    toast("加载词频失败", "error");
  } finally {
    loading.value = false;
    emit("loading", false);
  }
}

async function handleDelete(item: FreqItem) {
  const ok = await confirm(`确定删除词频记录「${item.text}」？`);
  if (!ok) return;
  try {
    await deleteFreq(props.schemaId, item.code, item.text);
    toast("已删除", "success");
    await loadData();
  } catch (e) {
    toast("删除失败", "error");
  }
}

async function handleBatchDelete() {
  if (selectedKeys.value.size === 0) return;
  const ok = await confirm(
    `确定删除选中的 ${selectedKeys.value.size} 条词频记录？`,
  );
  if (!ok) return;
  let failed = 0;
  for (const item of freqList.value) {
    if (selectedKeys.value.has(itemKey(item))) {
      try {
        await deleteFreq(props.schemaId, item.code, item.text);
      } catch {
        failed++;
      }
    }
  }
  if (failed > 0) {
    toast(`删除完成，${failed} 个失败`, "error");
  } else {
    toast("已删除选中词频记录", "success");
  }
  await loadData();
}

async function handleClear() {
  const ok = await confirm(
    `确定清空方案「${props.schemaName || props.schemaId}」的所有词频记录？`,
  );
  if (!ok) return;
  try {
    const count = await clearFreq(props.schemaId);
    toast(`已清空 ${count} 条词频记录`, "success");
    page.value = 0;
    await loadData();
  } catch (e) {
    toast("清空失败", "error");
  }
}

onMounted(() => {
  loadData();
});
</script>

<style>
@import "./dict-shared.css";
</style>

<style scoped>
.freq-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
</style>
