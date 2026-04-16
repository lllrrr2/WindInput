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
        :disabled="tempDict.length === 0"
        @click="handlePromoteAll"
      >
        全部转正
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
      <span class="toolbar-total">共 {{ tempDict.length }} 条</span>
      <button class="btn btn-sm btn-danger-outline" @click="handleClear">
        清空
      </button>
    </div>

    <!-- 内容区 -->
    <div class="dict-content-area">
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
            <col class="col-count" />
            <col class="col-actions-wide" />
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th>编码</th>
              <th>词条</th>
              <th>权重</th>
              <th>次数</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in filteredTempDict"
              :key="item.code + '|' + item.text"
              :class="{
                selected: selectedKeys.has(item.code + '|' + item.text),
              }"
            >
              <td>
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedKeys.has(item.code + '|' + item.text)"
                  @change="toggleSelect(item)"
                />
              </td>
              <td>
                <span class="dict-item-code">{{ item.code }}</span>
              </td>
              <td>{{ item.text }}</td>
              <td class="td-weight">{{ item.weight }}</td>
              <td class="td-weight">{{ item.count }}</td>
              <td>
                <div style="display: flex; gap: 4px; align-items: center">
                  <button
                    class="btn btn-sm"
                    style="font-size: 12px; padding: 2px 8px"
                    @click="handlePromote(item)"
                  >
                    转正
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
            <tr v-if="filteredTempDict.length === 0">
              <td :colspan="6" class="td-empty">
                {{ searchQuery ? "未找到匹配词条" : "暂无临时词条" }}
              </td>
            </tr>
          </tbody>
        </table>
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
  getTempDictBySchema,
  removeTempWordForSchema,
  promoteTempWordForSchema,
  promoteAllTempWordsForSchema,
  clearTempDictForSchema,
  type TempWordItem,
} from "../../api/wails";

const props = defineProps<{
  schemaId: string;
}>();

const emit = defineEmits<{
  (e: "loading", value: boolean): void;
  (e: "schema-changed"): void;
}>();

defineExpose({ loadData });

const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

const tempDict = ref<TempWordItem[]>([]);
const searchQuery = ref("");
const selectedKeys = ref<Set<string>>(new Set());
const loading = ref(false);

const filteredTempDict = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return tempDict.value;
  return tempDict.value.filter(
    (item) =>
      item.code.toLowerCase().includes(q) ||
      item.text.toLowerCase().includes(q),
  );
});

const allSelected = computed(
  () =>
    filteredTempDict.value.length > 0 &&
    filteredTempDict.value.every((item) =>
      selectedKeys.value.has(item.code + "|" + item.text),
    ),
);

function itemKey(item: TempWordItem) {
  return item.code + "|" + item.text;
}

function toggleAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(tempDict.value.map(itemKey));
  }
}

function toggleSelect(item: TempWordItem) {
  const key = itemKey(item);
  const next = new Set(selectedKeys.value);
  if (next.has(key)) {
    next.delete(key);
  } else {
    next.add(key);
  }
  selectedKeys.value = next;
}

async function loadData() {
  loading.value = true;
  emit("loading", true);
  try {
    tempDict.value = await getTempDictBySchema(props.schemaId);
    selectedKeys.value = new Set();
  } finally {
    loading.value = false;
    emit("loading", false);
  }
}

async function handlePromote(item: TempWordItem) {
  try {
    await promoteTempWordForSchema(props.schemaId, item.code, item.text);
    toast(`已将「${item.text}」转正`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`转正失败: ${e}`, "error");
  }
}

async function handlePromoteAll() {
  const ok = await confirm(
    `确定将全部 ${tempDict.value.length} 条临时词条转正？`,
  );
  if (!ok) return;
  try {
    const count = await promoteAllTempWordsForSchema(props.schemaId);
    toast(`已将 ${count} 条词条转正`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`转正失败: ${e}`, "error");
  }
}

async function handleClear() {
  const ok = await confirm("确定清空当前方案的所有临时词库？此操作不可撤销。");
  if (!ok) return;
  try {
    const count = await clearTempDictForSchema(props.schemaId);
    toast(`已清空 ${count} 条临时词条`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`清空失败: ${e}`, "error");
  }
}

async function handleRemove(item: TempWordItem) {
  const ok = await confirm(`确定删除临时词条「${item.text}」？`);
  if (!ok) return;
  try {
    await removeTempWordForSchema(props.schemaId, item.code, item.text);
    toast(`已删除「${item.text}」`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

async function handleBatchRemove() {
  const count = selectedKeys.value.size;
  const ok = await confirm(`确定删除选中的 ${count} 条临时词条？`);
  if (!ok) return;
  try {
    const items = tempDict.value.filter((item) =>
      selectedKeys.value.has(itemKey(item)),
    );
    for (const item of items) {
      await removeTempWordForSchema(props.schemaId, item.code, item.text);
    }
    toast(`已删除 ${count} 条词条`);
    await loadData();
    emit("schema-changed");
  } catch (e) {
    toast(`删除失败: ${e}`, "error");
  }
}

onMounted(() => {
  loadData();
});
</script>

<style>
@import "./dict-shared.css";
</style>
