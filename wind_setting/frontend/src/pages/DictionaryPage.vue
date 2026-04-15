<template>
  <section class="section dict-page">
    <!-- ===== 紧凑标题 ===== -->
    <div class="dict-header">
      <h2>词库管理</h2>
      <span class="dict-header-desc">管理您的词库数据（修改即时生效）</span>
    </div>

    <!-- 非 Wails 环境提示 -->
    <div v-if="!isWailsEnv" class="settings-card">
      <div class="dict-note-center">
        <p>词库管理功能需要在桌面应用中使用</p>
        <p class="dict-note">请使用 <code>wails dev</code> 或编译后的应用</p>
      </div>
    </div>

    <template v-else>
      <!-- ===== 词库类型选择行 ===== -->
      <div class="type-selector-row">
        <span class="type-label">词库:</span>
        <button
          :class="['type-btn', { active: dictMode === 'phrases' }]"
          @click="switchToPhrase"
        >
          快捷短语
        </button>
        <div class="schema-dropdown-wrap" @click.stop>
          <button
            :class="['type-btn schema-btn', { active: dictMode === 'schema' }]"
            @click="schemaDropdownOpen = !schemaDropdownOpen"
          >
            {{ selectedSchemaName ? selectedSchemaName : "选择方案" }}
            ▾
          </button>
          <div v-if="schemaDropdownOpen" class="schema-dropdown">
            <div
              v-for="s in allSchemaStatuses"
              :key="s.schema_id"
              :class="[
                'schema-dropdown-item',
                {
                  active:
                    dictMode === 'schema' && selectedSchemaID === s.schema_id,
                },
              ]"
              @click="selectSchema(s)"
            >
              <span class="schema-dd-name">
                <span
                  v-if="s.status === 'enabled'"
                  class="schema-dd-dot dot-enabled"
                ></span>
                <span
                  v-else-if="s.status === 'orphaned'"
                  class="schema-dd-dot dot-orphaned"
                ></span>
                <span v-else class="schema-dd-dot dot-disabled"></span>
                {{ s.schema_name || s.schema_id }}
                <span v-if="s.status === 'orphaned'" class="tag-orphan"
                  >(残留)</span
                >
              </span>
              <span v-if="s.status !== 'orphaned'" class="schema-dd-count">
                {{ s.user_words }}词
              </span>
            </div>
            <div v-if="allSchemaStatuses.length === 0" class="schema-dd-empty">
              暂无方案
            </div>
          </div>
        </div>
      </div>

      <!-- ===== 快捷短语模式 ===== -->
      <template v-if="dictMode === 'phrases'">
        <!-- 工具栏 -->
        <div class="dict-toolbar">
          <label class="toolbar-checkbox-wrap">
            <input
              type="checkbox"
              :checked="allPhraseSelected"
              @change="toggleAllPhrases"
            />
            <span>全选</span>
          </label>
          <button class="btn btn-primary btn-sm" @click="openAddPhraseDialog">
            + 添加
          </button>
          <button
            class="btn btn-sm btn-danger-outline"
            :disabled="selectedPhraseKeys.size === 0"
            @click="handleBatchRemovePhrases"
          >
            删除{{
              selectedPhraseKeys.size > 0 ? ` (${selectedPhraseKeys.size})` : ""
            }}
          </button>
          <div class="toolbar-spacer"></div>
          <input
            type="text"
            v-model="phraseSearchQuery"
            class="input input-sm toolbar-search"
            placeholder="搜索..."
          />
          <button
            class="btn btn-sm"
            @click="handleResetPhrasesToDefault"
            title="恢复所有短语为系统默认"
          >
            恢复默认
          </button>
        </div>

        <!-- loading 遮罩 -->
        <div class="dict-content-area" style="position: relative">
          <div v-if="dictLoading" class="content-loading-overlay">
            <div class="spinner"></div>
          </div>

          <!-- 短语列表 -->
          <div
            v-if="filteredPhrases.length > 0"
            class="dict-list dict-list-scrollable"
          >
            <div
              class="dict-list-item"
              v-for="(item, idx) in filteredPhrases"
              :key="idx"
              :class="{
                selected: selectedPhraseKeys.has(phraseKey(item)),
                'item-disabled': !item.enabled,
              }"
            >
              <input
                type="checkbox"
                class="item-checkbox"
                :checked="selectedPhraseKeys.has(phraseKey(item))"
                @change="togglePhraseSelect(item)"
              />
              <label class="item-switch-mini" @click.stop>
                <input
                  type="checkbox"
                  :checked="item.enabled"
                  @change="handleTogglePhraseEnabled(item)"
                />
              </label>
              <div class="dict-item-main">
                <span class="dict-item-code">{{ item.code }}</span>
                <span v-if="item.type === 'array'" class="dict-item-text">{{
                  item.name || item.code
                }}</span>
                <span v-else class="dict-item-text">{{ item.text }}</span>
                <span v-if="item.type === 'array'" class="tag-mapping"
                  >数组</span
                >
                <span v-else-if="item.type === 'dynamic'" class="tag-dynamic"
                  >动态</span
                >
                <span v-if="item.is_system" class="tag-system">系统</span>
                <span class="dict-item-weight">{{ item.position }}</span>
              </div>
              <div class="dict-item-actions">
                <button
                  class="btn-icon"
                  @click="openEditPhraseDialog(item)"
                  title="编辑"
                >
                  ✎
                </button>
                <button
                  class="btn-icon btn-delete"
                  @click="handleRemovePhrase(item)"
                  title="删除"
                >
                  &times;
                </button>
              </div>
            </div>
          </div>
          <div v-else-if="phraseSearchQuery" class="dict-empty">
            未找到匹配短语
          </div>
          <div v-else class="dict-empty">暂无短语</div>
        </div>
      </template>

      <!-- ===== 方案模式 ===== -->
      <template v-if="dictMode === 'schema' && selectedSchemaID">
        <!-- 残留方案警告 -->
        <div v-if="selectedSchemaOrphaned" class="orphan-banner">
          ⚠ 此方案数据为历史残留
        </div>

        <!-- 方案子标签页 -->
        <div class="schema-sub-tabs">
          <button
            :class="['sub-tab', { active: schemaSubTab === 'userdict' }]"
            @click="switchSchemaSubTab('userdict')"
          >
            用户词库
          </button>
          <button
            :class="['sub-tab', { active: schemaSubTab === 'freq' }]"
            @click="switchSchemaSubTab('freq')"
          >
            词频
          </button>
          <button
            :class="['sub-tab', { active: schemaSubTab === 'temp' }]"
            @click="switchSchemaSubTab('temp')"
          >
            临时词库
          </button>
          <button
            :class="['sub-tab', { active: schemaSubTab === 'shadow' }]"
            @click="switchSchemaSubTab('shadow')"
          >
            候选调整
          </button>
        </div>

        <!-- ===== 用户词库子页 ===== -->
        <template v-if="schemaSubTab === 'userdict'">
          <div class="dict-toolbar">
            <label class="toolbar-checkbox-wrap">
              <input
                type="checkbox"
                :checked="allWordSelected"
                @change="toggleAllWords"
              />
              <span>全选</span>
            </label>
            <button class="btn btn-primary btn-sm" @click="openAddWordDialog">
              + 添加
            </button>
            <button
              class="btn btn-sm btn-danger-outline"
              :disabled="selectedWordKeys.size === 0"
              @click="handleBatchRemoveWords"
            >
              删除{{
                selectedWordKeys.size > 0 ? ` (${selectedWordKeys.size})` : ""
              }}
            </button>
            <div class="toolbar-spacer"></div>
            <input
              type="text"
              v-model="wordSearchQuery"
              class="input input-sm toolbar-search"
              placeholder="搜索..."
            />
            <div
              class="toolbar-more"
              @click.stop="showWordMenu = !showWordMenu"
            >
              <button class="btn btn-sm">...</button>
              <div v-if="showWordMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item"
                  @click.stop="
                    handleImportUserDict();
                    showWordMenu = false;
                  "
                >
                  导入用户词库
                </div>
                <div
                  class="dropdown-item"
                  @click.stop="
                    handleExportUserDict();
                    showWordMenu = false;
                  "
                >
                  导出用户词库
                </div>
              </div>
            </div>
            <!-- 重置下拉 -->
            <div
              class="toolbar-more"
              @click.stop="showResetMenu = !showResetMenu"
            >
              <button class="btn btn-sm btn-danger-outline">重置 ▾</button>
              <div v-if="showResetMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetCurrentSchema();
                    showResetMenu = false;
                  "
                >
                  重置当前方案
                </div>
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetAllSchemas();
                    showResetMenu = false;
                  "
                >
                  重置所有方案
                </div>
              </div>
            </div>
          </div>

          <div class="dict-content-area" style="position: relative">
            <div v-if="dictLoading" class="content-loading-overlay">
              <div class="spinner"></div>
            </div>

            <div v-if="filteredUserDict.length > 0" class="dict-table-wrap">
              <table class="dict-table">
                <colgroup>
                  <col class="col-check" />
                  <col class="col-code" />
                  <col />
                  <col class="col-weight" />
                  <col class="col-action" />
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
                    :class="{
                      selected: selectedWordKeys.has(wordKey(item)),
                    }"
                    @dblclick="openEditWordDialog(item)"
                  >
                    <td>
                      <input
                        type="checkbox"
                        class="item-checkbox"
                        :checked="selectedWordKeys.has(wordKey(item))"
                        @change="toggleWordSelect(item)"
                      />
                    </td>
                    <td>
                      <span class="dict-item-code">{{ item.code }}</span>
                    </td>
                    <td>{{ item.text }}</td>
                    <td class="td-weight">{{ item.weight || 0 }}</td>
                    <td>
                      <button
                        class="btn-icon btn-delete"
                        @click="handleRemoveUserWord(item)"
                        title="删除"
                      >
                        &times;
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else-if="wordSearchQuery" class="dict-empty">
              未找到匹配词条
            </div>
            <div v-else class="dict-empty">暂无用户词条</div>
          </div>
        </template>

        <!-- ===== 词频子页 ===== -->
        <template v-if="schemaSubTab === 'freq'">
          <div class="dict-toolbar">
            <label class="toolbar-checkbox-wrap">
              <input
                type="checkbox"
                :checked="allFreqSelected"
                @change="toggleAllFreqs"
              />
              <span>全选</span>
            </label>
            <button
              class="btn btn-sm btn-danger-outline"
              :disabled="selectedFreqKeys.size === 0"
              @click="handleBatchRemoveFreqs"
            >
              删除{{
                selectedFreqKeys.size > 0 ? ` (${selectedFreqKeys.size})` : ""
              }}
            </button>
            <div class="toolbar-spacer"></div>
            <input
              type="text"
              v-model="freqSearchQuery"
              class="input input-sm toolbar-search"
              placeholder="搜索..."
              @input="debouncedLoadFreq"
            />
            <button
              class="btn btn-sm btn-danger-outline"
              @click="handleClearAllFreq"
            >
              清空
            </button>
            <!-- 重置下拉 -->
            <div
              class="toolbar-more"
              @click.stop="showResetMenu = !showResetMenu"
            >
              <button class="btn btn-sm btn-danger-outline">重置 ▾</button>
              <div v-if="showResetMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetCurrentSchema();
                    showResetMenu = false;
                  "
                >
                  重置当前方案
                </div>
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetAllSchemas();
                    showResetMenu = false;
                  "
                >
                  重置所有方案
                </div>
              </div>
            </div>
          </div>

          <div class="dict-content-area" style="position: relative">
            <div v-if="dictLoading" class="content-loading-overlay">
              <div class="spinner"></div>
            </div>

            <div
              v-if="freqList.length > 0"
              class="dict-list dict-list-scrollable"
            >
              <div
                class="dict-list-item"
                v-for="(item, idx) in freqList"
                :key="idx"
                :class="{
                  selected: selectedFreqKeys.has(freqKey(item)),
                }"
              >
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedFreqKeys.has(freqKey(item))"
                  @change="toggleFreqSelect(item)"
                />
                <div class="dict-item-main">
                  <span class="dict-item-code">{{ item.code }}</span>
                  <span class="dict-item-text">{{ item.text }}</span>
                  <span class="dict-item-weight">×{{ item.count }}</span>
                  <span class="dict-item-weight">boost:{{ item.boost }}</span>
                  <span class="dict-item-weight">{{
                    formatLastUsed(item.last_used)
                  }}</span>
                </div>
                <div class="dict-item-actions">
                  <button
                    class="btn-icon btn-delete"
                    @click="handleRemoveFreq(item)"
                    title="删除"
                  >
                    &times;
                  </button>
                </div>
              </div>
            </div>
            <div v-else-if="freqSearchQuery" class="dict-empty">
              未找到匹配词频记录
            </div>
            <div v-else class="dict-empty">暂无词频记录</div>

            <!-- 词频分页 -->
            <div v-if="freqTotal > freqPageSize" class="freq-pager">
              <button
                class="btn btn-sm"
                :disabled="freqPage === 0"
                @click="
                  freqPage--;
                  loadFreqData();
                "
              >
                上一页
              </button>
              <span class="freq-pager-info">
                {{ freqPage * freqPageSize + 1 }}-{{
                  Math.min((freqPage + 1) * freqPageSize, freqTotal)
                }}
                / {{ freqTotal }}
              </span>
              <button
                class="btn btn-sm"
                :disabled="(freqPage + 1) * freqPageSize >= freqTotal"
                @click="
                  freqPage++;
                  loadFreqData();
                "
              >
                下一页
              </button>
            </div>
          </div>
        </template>

        <!-- ===== 临时词库子页 ===== -->
        <template v-if="schemaSubTab === 'temp'">
          <div class="dict-toolbar">
            <label class="toolbar-checkbox-wrap">
              <input
                type="checkbox"
                :checked="allTempSelected"
                @change="toggleAllTemps"
              />
              <span>全选</span>
            </label>
            <button
              class="btn btn-primary btn-sm"
              @click="handlePromoteAllTemp"
              :disabled="tempDict.length === 0"
            >
              全部转正
            </button>
            <button
              class="btn btn-sm btn-danger-outline"
              :disabled="selectedTempKeys.size === 0"
              @click="handleBatchRemoveTemps"
            >
              删除{{
                selectedTempKeys.size > 0 ? ` (${selectedTempKeys.size})` : ""
              }}
            </button>
            <div class="toolbar-spacer"></div>
            <div
              class="toolbar-more"
              @click.stop="showTempMenu = !showTempMenu"
            >
              <button class="btn btn-sm">...</button>
              <div v-if="showTempMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleClearTempDict();
                    showTempMenu = false;
                  "
                >
                  清空临时词库
                </div>
              </div>
            </div>
            <!-- 重置下拉 -->
            <div
              class="toolbar-more"
              @click.stop="showResetMenu = !showResetMenu"
            >
              <button class="btn btn-sm btn-danger-outline">重置 ▾</button>
              <div v-if="showResetMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetCurrentSchema();
                    showResetMenu = false;
                  "
                >
                  重置当前方案
                </div>
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetAllSchemas();
                    showResetMenu = false;
                  "
                >
                  重置所有方案
                </div>
              </div>
            </div>
          </div>

          <div class="dict-content-area" style="position: relative">
            <div v-if="dictLoading" class="content-loading-overlay">
              <div class="spinner"></div>
            </div>

            <div
              v-if="tempDict.length > 0"
              class="dict-list dict-list-scrollable"
            >
              <div
                class="dict-list-item"
                v-for="(item, idx) in tempDict"
                :key="idx"
                :class="{
                  selected: selectedTempKeys.has(tempKey(item)),
                }"
              >
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedTempKeys.has(tempKey(item))"
                  @change="toggleTempSelect(item)"
                />
                <div class="dict-item-main">
                  <span class="dict-item-code">{{ item.code }}</span>
                  <span class="dict-item-text">{{ item.text }}</span>
                  <span class="dict-item-weight">w:{{ item.weight }}</span>
                  <span class="dict-item-weight">×{{ item.count }}</span>
                </div>
                <div class="dict-item-actions">
                  <button
                    class="btn btn-sm"
                    @click="handlePromoteTempWord(item)"
                    title="转为永久词条"
                  >
                    转正
                  </button>
                  <button
                    class="btn-icon btn-delete"
                    @click="handleRemoveTempWord(item)"
                    title="删除"
                  >
                    &times;
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="dict-empty">
              <p>暂无临时词条</p>
              <p class="dict-note">自动学习的词条将在此显示</p>
            </div>
          </div>
        </template>

        <!-- ===== 候选调整子页 ===== -->
        <template v-if="schemaSubTab === 'shadow'">
          <div class="dict-toolbar">
            <label class="toolbar-checkbox-wrap">
              <input
                type="checkbox"
                :checked="allShadowSelected"
                @change="toggleAllShadows"
              />
              <span>全选</span>
            </label>
            <button class="btn btn-primary btn-sm" @click="openShadowDialog()">
              + 添加
            </button>
            <button
              class="btn btn-sm btn-danger-outline"
              :disabled="selectedShadowKeys.size === 0"
              @click="handleBatchRemoveShadows"
            >
              删除{{
                selectedShadowKeys.size > 0
                  ? ` (${selectedShadowKeys.size})`
                  : ""
              }}
            </button>
            <div class="toolbar-spacer"></div>
            <!-- 重置下拉 -->
            <div
              class="toolbar-more"
              @click.stop="showResetMenu = !showResetMenu"
            >
              <button class="btn btn-sm btn-danger-outline">重置 ▾</button>
              <div v-if="showResetMenu" class="toolbar-dropdown">
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetCurrentSchema();
                    showResetMenu = false;
                  "
                >
                  重置当前方案
                </div>
                <div
                  class="dropdown-item dropdown-danger"
                  @click.stop="
                    handleResetAllSchemas();
                    showResetMenu = false;
                  "
                >
                  重置所有方案
                </div>
              </div>
            </div>
          </div>

          <div class="dict-content-area" style="position: relative">
            <div v-if="dictLoading" class="content-loading-overlay">
              <div class="spinner"></div>
            </div>

            <div
              v-if="shadowRules.length > 0"
              class="dict-list dict-list-scrollable"
            >
              <div
                class="dict-list-item"
                v-for="(item, idx) in shadowRules"
                :key="idx"
                :class="{
                  selected: selectedShadowKeys.has(shadowKey(item)),
                }"
              >
                <input
                  type="checkbox"
                  class="item-checkbox"
                  :checked="selectedShadowKeys.has(shadowKey(item))"
                  @change="toggleShadowSelect(item)"
                />
                <div class="dict-item-main">
                  <span class="dict-item-code">{{ item.code }}</span>
                  <span class="dict-item-text">{{ item.word }}</span>
                  <span class="dict-item-tag" :class="'tag-' + item.type">{{
                    getShadowActionLabel(item.type)
                  }}</span>
                  <span class="dict-item-weight" v-if="item.type === 'pin'"
                    >位置: {{ item.position }}</span
                  >
                </div>
                <div class="dict-item-actions">
                  <button
                    class="btn-icon"
                    @click="openShadowDialog(item)"
                    title="编辑"
                  >
                    ✎
                  </button>
                  <button
                    class="btn-icon btn-delete"
                    @click="handleRemoveShadowRule(item)"
                    title="删除"
                  >
                    &times;
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="dict-empty">暂无调整规则</div>
          </div>
        </template>
      </template>
    </template>

    <!-- ========== 添加/编辑短语对话框 ========== -->
    <div
      v-if="addPhraseDialogVisible"
      class="dialog-overlay"
      @click.self="addPhraseDialogVisible = false"
    >
      <div class="dialog-box">
        <div class="dialog-title">
          {{ editingPhrase ? "编辑短语" : "添加短语" }}
        </div>
        <div class="form-row">
          <label>编码</label>
          <input
            type="text"
            v-model="newPhrase.code"
            class="input"
            placeholder="如: dz"
            :disabled="!!editingPhrase"
          />
        </div>
        <div class="form-row">
          <label>类型</label>
          <label class="radio-inline">
            <input type="radio" :value="false" v-model="phraseIsArray" /> 普通
          </label>
          <label class="radio-inline">
            <input type="radio" :value="true" v-model="phraseIsArray" /> 数组
          </label>
        </div>
        <template v-if="phraseIsArray">
          <div class="form-row">
            <label>名称</label>
            <input
              type="text"
              v-model="newPhrase.name"
              class="input"
              placeholder="如: 标点符号"
            />
          </div>
          <div class="form-row">
            <label>字符列表</label>
            <textarea
              v-model="newPhrase.texts"
              class="input"
              rows="3"
              placeholder="每个字符展开为独立候选，如: ①②③④⑤"
            ></textarea>
          </div>
        </template>
        <template v-else>
          <div class="form-row">
            <label>文本</label>
            <input
              type="text"
              v-model="newPhrase.text"
              class="input"
              placeholder="如: 我的地址是xxx 或 $Y-$MM-$DD"
            />
          </div>
        </template>
        <div class="form-row">
          <label>位置</label>
          <input
            type="number"
            v-model.number="newPhrase.position"
            class="input input-sm"
            min="1"
          />
        </div>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="addPhraseDialogVisible = false">
            取消
          </button>
          <button class="btn btn-primary btn-sm" @click="handleSavePhrase">
            {{ editingPhrase ? "保存" : "添加" }}
          </button>
        </div>
      </div>
    </div>

    <!-- ========== 添加/编辑词条对话框 ========== -->
    <AddWordPage
      v-if="addWordDialogVisible"
      :initialText="editWordText"
      :initialCode="editWordCode"
      :initialSchema="selectedSchemaID"
      @close="handleAddWordDialogClose"
    />

    <!-- ========== Shadow 规则对话框 ========== -->
    <div
      v-if="shadowDialogVisible"
      class="dialog-overlay"
      @click.self="shadowDialogVisible = false"
    >
      <div class="dialog-box">
        <div class="dialog-title">
          {{ shadowDialogEditing ? "编辑规则" : "添加规则" }}
        </div>
        <div class="form-row">
          <label>编码</label>
          <input
            type="text"
            v-model="shadowForm.code"
            class="input"
            placeholder="如: sf"
            :disabled="shadowDialogEditing"
          />
        </div>
        <div class="form-row">
          <label>词条</label>
          <input
            type="text"
            v-model="shadowForm.word"
            class="input"
            placeholder="如: 村"
            :disabled="shadowDialogEditing"
          />
        </div>
        <div class="form-row">
          <label>操作</label>
          <select v-model="shadowForm.action" class="select">
            <option value="pin">固定位置</option>
            <option value="delete">隐藏词条</option>
          </select>
        </div>
        <div class="form-row" v-if="shadowForm.action === 'pin'">
          <label>目标位置</label>
          <input
            type="number"
            v-model.number="shadowForm.position"
            class="input input-sm"
            min="0"
            placeholder="0=首位"
          />
        </div>
        <div class="dialog-actions">
          <button class="btn btn-sm" @click="shadowDialogVisible = false">
            取消
          </button>
          <button class="btn btn-primary btn-sm" @click="handleSaveShadowRule">
            {{ shadowDialogEditing ? "保存" : "添加" }}
          </button>
        </div>
      </div>
    </div>

    <!-- ========== 确认对话框 ========== -->
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
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import * as wailsApi from "../api/wails";
import { useToast } from "../composables/useToast";
import { useConfirm } from "../composables/useConfirm";
import AddWordPage from "./AddWordPage.vue";
import type {
  PhraseItem,
  UserWordItem,
  ShadowRuleItem,
  TempWordItem,
  FreqItem,
  SchemaStatusItem,
  DictEvent,
} from "../api/wails";

const props = defineProps<{
  isWailsEnv: boolean;
}>();

// ===== 全局 Toast & Confirm =====
const { toast } = useToast();
const { confirmVisible, confirmMessage, confirm, handleConfirm, handleCancel } =
  useConfirm();

// ===== 模式与通用状态 =====
const dictMode = ref<"phrases" | "schema">("phrases");
const dictLoading = ref(false);

// ===== 方案状态列表 =====
const allSchemaStatuses = ref<SchemaStatusItem[]>([]);
const selectedSchemaID = ref("");
const schemaDropdownOpen = ref(false);

const selectedSchemaName = computed(() => {
  const s = allSchemaStatuses.value.find(
    (s) => s.schema_id === selectedSchemaID.value,
  );
  return s ? s.schema_name || s.schema_id : "";
});

const selectedSchemaOrphaned = computed(() => {
  const s = allSchemaStatuses.value.find(
    (s) => s.schema_id === selectedSchemaID.value,
  );
  return s?.status === "orphaned";
});

// ===== 方案子标签页 =====
const schemaSubTab = ref<"userdict" | "freq" | "temp" | "shadow">("userdict");

// ===== 短语 =====
const allPhrases = ref<PhraseItem[]>([]);
const phraseSearchQuery = ref("");
const addPhraseDialogVisible = ref(false);
const newPhrase = ref({ code: "", text: "", texts: "", name: "", position: 1 });
const editingPhrase = ref<PhraseItem | null>(null);
const phraseIsArray = ref(false);

const filteredPhrases = computed(() => {
  if (!phraseSearchQuery.value.trim()) return allPhrases.value;
  const q = phraseSearchQuery.value.trim().toLowerCase();
  return allPhrases.value.filter(
    (p) =>
      p.code.toLowerCase().includes(q) ||
      (p.text && p.text.toLowerCase().includes(q)) ||
      (p.name && p.name.toLowerCase().includes(q)),
  );
});

// 短语多选
const selectedPhraseKeys = ref(new Set<string>());
const allPhraseSelected = computed(
  () =>
    filteredPhrases.value.length > 0 &&
    filteredPhrases.value.every((p) =>
      selectedPhraseKeys.value.has(phraseKey(p)),
    ),
);
function phraseKey(item: PhraseItem) {
  return `${item.code}||${item.text || ""}||${item.name || ""}`;
}
function togglePhraseSelect(item: PhraseItem) {
  const k = phraseKey(item);
  if (selectedPhraseKeys.value.has(k)) selectedPhraseKeys.value.delete(k);
  else selectedPhraseKeys.value.add(k);
  selectedPhraseKeys.value = new Set(selectedPhraseKeys.value);
}
function toggleAllPhrases() {
  if (allPhraseSelected.value) {
    selectedPhraseKeys.value = new Set();
  } else {
    selectedPhraseKeys.value = new Set(filteredPhrases.value.map(phraseKey));
  }
}

// ===== 用户词库 =====
const userDict = ref<UserWordItem[]>([]);
const addWordDialogVisible = ref(false);
const wordSearchQuery = ref("");
const sortKey = ref<"code" | "text" | "weight">("code");
const sortAsc = ref(true);

const filteredUserDict = computed(() => {
  let list = userDict.value;
  if (wordSearchQuery.value.trim()) {
    const q = wordSearchQuery.value.trim().toLowerCase();
    list = list.filter(
      (w) =>
        w.code.toLowerCase().includes(q) || w.text.toLowerCase().includes(q),
    );
  }
  return [...list].sort((a, b) => {
    const ka = sortKey.value;
    const av = a[ka];
    const bv = b[ka];
    if (typeof av === "number" && typeof bv === "number") {
      return sortAsc.value ? av - bv : bv - av;
    }
    const as = String(av ?? "");
    const bs = String(bv ?? "");
    return sortAsc.value ? as.localeCompare(bs) : bs.localeCompare(as);
  });
});

function toggleSort(key: "code" | "text" | "weight") {
  if (sortKey.value === key) sortAsc.value = !sortAsc.value;
  else {
    sortKey.value = key;
    sortAsc.value = true;
  }
}

// 词条多选
const selectedWordKeys = ref(new Set<string>());
const uniqueFilteredWordKeys = computed(
  () => new Set(filteredUserDict.value.map(wordKey)),
);
const allWordSelected = computed(
  () =>
    uniqueFilteredWordKeys.value.size > 0 &&
    [...uniqueFilteredWordKeys.value].every((k) =>
      selectedWordKeys.value.has(k),
    ),
);
function wordKey(item: UserWordItem) {
  return `${item.code}||${item.text}`;
}
function toggleWordSelect(item: UserWordItem) {
  const k = wordKey(item);
  if (selectedWordKeys.value.has(k)) selectedWordKeys.value.delete(k);
  else selectedWordKeys.value.add(k);
  selectedWordKeys.value = new Set(selectedWordKeys.value);
}
function toggleAllWords() {
  if (allWordSelected.value) {
    selectedWordKeys.value = new Set();
  } else {
    selectedWordKeys.value = new Set(filteredUserDict.value.map(wordKey));
  }
}

// ===== 词频 =====
const freqList = ref<FreqItem[]>([]);
const freqSearchQuery = ref("");
const freqTotal = ref(0);
const freqPage = ref(0);
const freqPageSize = 100;

// 词频多选
const selectedFreqKeys = ref(new Set<string>());
const allFreqSelected = computed(
  () =>
    freqList.value.length > 0 &&
    freqList.value.every((f) => selectedFreqKeys.value.has(freqKey(f))),
);
function freqKey(item: FreqItem) {
  return `${item.code}||${item.text}`;
}
function toggleFreqSelect(item: FreqItem) {
  const k = freqKey(item);
  if (selectedFreqKeys.value.has(k)) selectedFreqKeys.value.delete(k);
  else selectedFreqKeys.value.add(k);
  selectedFreqKeys.value = new Set(selectedFreqKeys.value);
}
function toggleAllFreqs() {
  if (allFreqSelected.value) {
    selectedFreqKeys.value = new Set();
  } else {
    selectedFreqKeys.value = new Set(freqList.value.map(freqKey));
  }
}

let freqDebounceTimer: ReturnType<typeof setTimeout> | null = null;
function debouncedLoadFreq() {
  if (freqDebounceTimer) clearTimeout(freqDebounceTimer);
  freqDebounceTimer = setTimeout(() => {
    freqPage.value = 0;
    loadFreqData();
  }, 300);
}

function formatLastUsed(ts: number): string {
  if (!ts) return "";
  const d = new Date(ts * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// ===== 临时词库 =====
const tempDict = ref<TempWordItem[]>([]);

// 临时词库多选
const selectedTempKeys = ref(new Set<string>());
const allTempSelected = computed(
  () =>
    tempDict.value.length > 0 &&
    tempDict.value.every((t) => selectedTempKeys.value.has(tempKey(t))),
);
function tempKey(item: TempWordItem) {
  return `${item.code}||${item.text}`;
}
function toggleTempSelect(item: TempWordItem) {
  const k = tempKey(item);
  if (selectedTempKeys.value.has(k)) selectedTempKeys.value.delete(k);
  else selectedTempKeys.value.add(k);
  selectedTempKeys.value = new Set(selectedTempKeys.value);
}
function toggleAllTemps() {
  if (allTempSelected.value) {
    selectedTempKeys.value = new Set();
  } else {
    selectedTempKeys.value = new Set(tempDict.value.map(tempKey));
  }
}

// ===== 候选调整 =====
const shadowRules = ref<ShadowRuleItem[]>([]);
const shadowDialogVisible = ref(false);
const shadowDialogEditing = ref(false);
const shadowForm = ref({ code: "", word: "", action: "pin", position: 0 });

// Shadow 多选
const selectedShadowKeys = ref(new Set<string>());
const allShadowSelected = computed(
  () =>
    shadowRules.value.length > 0 &&
    shadowRules.value.every((s) => selectedShadowKeys.value.has(shadowKey(s))),
);
function shadowKey(item: ShadowRuleItem) {
  return `${item.code}||${item.word}`;
}
function toggleShadowSelect(item: ShadowRuleItem) {
  const k = shadowKey(item);
  if (selectedShadowKeys.value.has(k)) selectedShadowKeys.value.delete(k);
  else selectedShadowKeys.value.add(k);
  selectedShadowKeys.value = new Set(selectedShadowKeys.value);
}
function toggleAllShadows() {
  if (allShadowSelected.value) {
    selectedShadowKeys.value = new Set();
  } else {
    selectedShadowKeys.value = new Set(shadowRules.value.map(shadowKey));
  }
}

// ===== 下拉菜单 =====
const showWordMenu = ref(false);
const showTempMenu = ref(false);
const showResetMenu = ref(false);

// ===== 工具函数 =====
function showDictMessage(msg: string, type: "success" | "error") {
  toast(msg, type);
}

function getShadowActionLabel(type: string): string {
  const labels: Record<string, string> = {
    pin: "固定位置",
    delete: "隐藏",
  };
  return labels[type] || type;
}

// ===== 数据加载 =====
async function loadSchemaStatuses() {
  try {
    const list = await wailsApi.getAllSchemaStatuses();
    allSchemaStatuses.value = list || [];
    // 如果当前选中的方案不在列表中，选第一个
    if (
      selectedSchemaID.value &&
      !allSchemaStatuses.value.find(
        (s) => s.schema_id === selectedSchemaID.value,
      )
    ) {
      const first = allSchemaStatuses.value.find((s) => s.status === "enabled");
      selectedSchemaID.value = first?.schema_id || "";
    }
    if (!selectedSchemaID.value && allSchemaStatuses.value.length > 0) {
      const first = allSchemaStatuses.value.find((s) => s.status === "enabled");
      selectedSchemaID.value =
        first?.schema_id || allSchemaStatuses.value[0].schema_id;
    }
  } catch (e) {
    console.error("加载方案状态失败", e);
  }
}

async function loadPhraseData() {
  dictLoading.value = true;
  try {
    const data = await wailsApi.getPhraseList();
    allPhrases.value = data || [];
    selectedPhraseKeys.value = new Set();
  } catch (e) {
    console.error("加载短语数据失败", e);
  } finally {
    dictLoading.value = false;
  }
}

async function loadUserDictData() {
  if (!selectedSchemaID.value) return;
  dictLoading.value = true;
  try {
    const data = await wailsApi.getUserDictBySchema(selectedSchemaID.value);
    userDict.value = data || [];
    selectedWordKeys.value = new Set();
  } catch (e) {
    console.error("加载用户词库失败", e);
    showDictMessage("加载用户词库失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

async function loadFreqData() {
  if (!selectedSchemaID.value) return;
  dictLoading.value = true;
  try {
    const result = await wailsApi.getFreqList(
      selectedSchemaID.value,
      freqSearchQuery.value.trim(),
      freqPageSize,
      freqPage.value * freqPageSize,
    );
    freqList.value = result.entries || [];
    freqTotal.value = result.total || 0;
    selectedFreqKeys.value = new Set();
  } catch (e) {
    console.error("加载词频失败", e);
    showDictMessage("加载词频失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

async function loadTempData() {
  if (!selectedSchemaID.value) return;
  dictLoading.value = true;
  try {
    const data = await wailsApi.getTempDictBySchema(selectedSchemaID.value);
    tempDict.value = data || [];
    selectedTempKeys.value = new Set();
  } catch (e) {
    console.error("加载临时词库失败", e);
    showDictMessage("加载临时词库失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

async function loadShadowData() {
  if (!selectedSchemaID.value) return;
  dictLoading.value = true;
  try {
    const data = await wailsApi.getShadowBySchema(selectedSchemaID.value);
    shadowRules.value = data || [];
    selectedShadowKeys.value = new Set();
  } catch (e) {
    console.error("加载候选调整失败", e);
    showDictMessage("加载候选调整失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

async function loadCurrentSchemaTabData() {
  switch (schemaSubTab.value) {
    case "userdict":
      await loadUserDictData();
      break;
    case "freq":
      await loadFreqData();
      break;
    case "temp":
      await loadTempData();
      break;
    case "shadow":
      await loadShadowData();
      break;
  }
}

async function loadAllData() {
  if (!props.isWailsEnv) return;
  try {
    await loadSchemaStatuses();
    if (dictMode.value === "phrases") {
      await loadPhraseData();
    } else {
      await loadCurrentSchemaTabData();
    }
  } catch (e) {
    console.error("初始化数据失败", e);
  }
}

// ===== 模式/标签页切换 =====
function switchToPhrase() {
  dictMode.value = "phrases";
  schemaDropdownOpen.value = false;
  loadPhraseData();
}

function selectSchema(s: SchemaStatusItem) {
  selectedSchemaID.value = s.schema_id;
  dictMode.value = "schema";
  schemaDropdownOpen.value = false;
  schemaSubTab.value = "userdict";
  loadCurrentSchemaTabData();
}

async function switchSchemaSubTab(
  tab: "userdict" | "freq" | "temp" | "shadow",
) {
  schemaSubTab.value = tab;
  await loadCurrentSchemaTabData();
}

// ===== 短语操作 =====
function openAddPhraseDialog() {
  editingPhrase.value = null;
  newPhrase.value = { code: "", text: "", texts: "", name: "", position: 1 };
  phraseIsArray.value = false;
  addPhraseDialogVisible.value = true;
}

function openEditPhraseDialog(item: PhraseItem) {
  editingPhrase.value = item;
  phraseIsArray.value = item.type === "array";
  newPhrase.value = {
    code: item.code,
    text: item.text || "",
    texts: item.texts || "",
    name: item.name || "",
    position: item.position,
  };
  addPhraseDialogVisible.value = true;
}

async function handleSavePhrase() {
  const isArr = phraseIsArray.value;
  if (
    !newPhrase.value.code ||
    (!isArr && !newPhrase.value.text) ||
    (isArr && !newPhrase.value.texts)
  ) {
    showDictMessage(
      isArr ? "请填写编码和字符列表" : "请填写编码和文本",
      "error",
    );
    return;
  }
  try {
    if (editingPhrase.value) {
      // 编辑模式：updatePhrase
      const orig = editingPhrase.value;
      await wailsApi.updatePhrase(
        orig.code,
        orig.text || "",
        orig.name || "",
        isArr ? "" : newPhrase.value.text,
        newPhrase.value.position,
        null,
      );
      showDictMessage("修改成功", "success");
    } else {
      // 新增
      const typeStr = isArr ? "array" : "static";
      await wailsApi.addPhrase(
        newPhrase.value.code,
        isArr ? "" : newPhrase.value.text,
        isArr ? newPhrase.value.texts : "",
        isArr ? newPhrase.value.name : "",
        typeStr,
        newPhrase.value.position,
      );
      showDictMessage("添加成功", "success");
    }
    addPhraseDialogVisible.value = false;
    editingPhrase.value = null;
    await loadPhraseData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "操作失败", "error");
  }
}

async function handleTogglePhraseEnabled(item: PhraseItem) {
  try {
    await wailsApi.setPhraseEnabled(
      item.code,
      item.text || "",
      item.name || "",
      !item.enabled,
    );
    await loadPhraseData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "操作失败", "error");
  }
}

async function handleRemovePhrase(item: PhraseItem) {
  if (
    !(await confirm(
      `确定删除短语 "${item.text || item.name || item.code}" 吗？`,
    ))
  )
    return;
  try {
    await wailsApi.removePhrase(item.code, item.text || "", item.name || "");
    selectedPhraseKeys.value.delete(phraseKey(item));
    selectedPhraseKeys.value = new Set(selectedPhraseKeys.value);
    showDictMessage("删除成功", "success");
    await loadPhraseData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "删除失败", "error");
  }
}

async function handleBatchRemovePhrases() {
  if (selectedPhraseKeys.value.size === 0) return;
  if (
    !(await confirm(
      `确定删除选中的 ${selectedPhraseKeys.value.size} 条短语吗？`,
    ))
  )
    return;
  const toDelete = allPhrases.value.filter((p) =>
    selectedPhraseKeys.value.has(phraseKey(p)),
  );
  try {
    for (const item of toDelete) {
      await wailsApi.removePhrase(item.code, item.text || "", item.name || "");
    }
    selectedPhraseKeys.value = new Set();
    showDictMessage(`已删除 ${toDelete.length} 条短语`, "success");
    await loadPhraseData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "批量删除失败", "error");
  }
}

async function handleResetPhrasesToDefault() {
  if (
    !(await confirm(
      "确定恢复所有短语为系统默认设置？\n这将删除所有自定义短语并恢复系统默认。",
    ))
  )
    return;
  try {
    await wailsApi.resetPhrasesToDefault();
    showDictMessage("已恢复为系统默认", "success");
    await loadPhraseData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "操作失败", "error");
  }
}

// ===== 用户词库操作 =====
const editWordText = ref("");
const editWordCode = ref("");

function openAddWordDialog() {
  editWordText.value = "";
  editWordCode.value = "";
  addWordDialogVisible.value = true;
}

function openEditWordDialog(item: UserWordItem) {
  editWordText.value = item.text;
  editWordCode.value = item.code;
  addWordDialogVisible.value = true;
}

async function handleAddWordDialogClose() {
  addWordDialogVisible.value = false;
  editWordText.value = "";
  editWordCode.value = "";
  await loadUserDictData();
  await loadSchemaStatuses();
}

async function handleRemoveUserWord(item: UserWordItem) {
  if (!(await confirm(`确定删除词条 "${item.text}" 吗？`))) return;
  try {
    await wailsApi.removeUserWordForSchema(
      selectedSchemaID.value,
      item.code,
      item.text,
    );
    selectedWordKeys.value.delete(wordKey(item));
    selectedWordKeys.value = new Set(selectedWordKeys.value);
    showDictMessage("删除成功", "success");
    await loadUserDictData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "删除失败", "error");
  }
}

async function handleBatchRemoveWords() {
  if (selectedWordKeys.value.size === 0) return;
  if (
    !(await confirm(`确定删除选中的 ${selectedWordKeys.value.size} 条词条吗？`))
  )
    return;

  const seen = new Set<string>();
  const toDelete: { code: string; text: string }[] = [];
  for (const item of userDict.value) {
    const k = wordKey(item);
    if (selectedWordKeys.value.has(k) && !seen.has(k)) {
      seen.add(k);
      toDelete.push({ code: item.code, text: item.text });
    }
  }

  let deleted = 0;
  let failed = 0;
  for (const item of toDelete) {
    try {
      await wailsApi.removeUserWordForSchema(
        selectedSchemaID.value,
        item.code,
        item.text,
      );
      deleted++;
    } catch {
      failed++;
    }
  }
  selectedWordKeys.value = new Set();
  if (failed > 0) {
    showDictMessage(`已删除 ${deleted} 条，${failed} 条失败`, "error");
  } else {
    showDictMessage(`已删除 ${deleted} 条词条`, "success");
  }
  await loadUserDictData();
  await loadSchemaStatuses();
}

async function handleImportUserDict() {
  dictLoading.value = true;
  try {
    const result = await wailsApi.importUserDictForSchema(
      selectedSchemaID.value,
    );
    if (result.cancelled) return;
    showDictMessage(
      `导入成功，新增 ${result.count} 条，共 ${result.total} 条`,
      "success",
    );
    await loadUserDictData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "导入失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

async function handleExportUserDict() {
  dictLoading.value = true;
  try {
    const result = await wailsApi.exportUserDictForSchema(
      selectedSchemaID.value,
    );
    if (result.cancelled) return;
    showDictMessage(`导出成功，共 ${result.count} 条`, "success");
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "导出失败", "error");
  } finally {
    dictLoading.value = false;
  }
}

// ===== 词频操作 =====
async function handleRemoveFreq(item: FreqItem) {
  if (!(await confirm(`确定删除词频记录 "${item.text}" 吗？`))) return;
  try {
    await wailsApi.deleteFreq(selectedSchemaID.value, item.code, item.text);
    selectedFreqKeys.value.delete(freqKey(item));
    selectedFreqKeys.value = new Set(selectedFreqKeys.value);
    showDictMessage("删除成功", "success");
    await loadFreqData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "删除失败", "error");
  }
}

async function handleBatchRemoveFreqs() {
  if (selectedFreqKeys.value.size === 0) return;
  if (
    !(await confirm(
      `确定删除选中的 ${selectedFreqKeys.value.size} 条词频记录吗？`,
    ))
  )
    return;
  const toDelete = freqList.value.filter((f) =>
    selectedFreqKeys.value.has(freqKey(f)),
  );
  let deleted = 0;
  let failed = 0;
  for (const item of toDelete) {
    try {
      await wailsApi.deleteFreq(selectedSchemaID.value, item.code, item.text);
      deleted++;
    } catch {
      failed++;
    }
  }
  selectedFreqKeys.value = new Set();
  if (failed > 0) {
    showDictMessage(`已删除 ${deleted} 条，${failed} 条失败`, "error");
  } else {
    showDictMessage(`已删除 ${deleted} 条词频记录`, "success");
  }
  await loadFreqData();
}

async function handleClearAllFreq() {
  if (!(await confirm("确定清空当前方案的所有词频数据吗？此操作不可恢复。")))
    return;
  try {
    const count = await wailsApi.clearFreq(selectedSchemaID.value);
    showDictMessage(`已清空 ${count} 条词频记录`, "success");
    await loadFreqData();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "清空失败", "error");
  }
}

// ===== 临时词库操作 =====
async function handlePromoteTempWord(item: TempWordItem) {
  try {
    await wailsApi.promoteTempWordForSchema(
      selectedSchemaID.value,
      item.code,
      item.text,
    );
    showDictMessage(`已将 "${item.text}" 转为永久词条`, "success");
    await loadTempData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "转正失败", "error");
  }
}

async function handlePromoteAllTemp() {
  if (!(await confirm("确定将所有临时词条转为永久词条吗？"))) return;
  try {
    const count = await wailsApi.promoteAllTempWordsForSchema(
      selectedSchemaID.value,
    );
    showDictMessage(`已转正 ${count} 条词条`, "success");
    await loadTempData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "转正失败", "error");
  }
}

async function handleClearTempDict() {
  if (!(await confirm("确定清空临时词库吗？此操作不可恢复。"))) return;
  try {
    const count = await wailsApi.clearTempDictForSchema(selectedSchemaID.value);
    showDictMessage(`已清空 ${count} 条临时词条`, "success");
    await loadTempData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "清空失败", "error");
  }
}

async function handleBatchRemoveTemps() {
  if (selectedTempKeys.value.size === 0) return;
  if (
    !(await confirm(
      `确定删除选中的 ${selectedTempKeys.value.size} 条临时词条吗？`,
    ))
  )
    return;
  try {
    for (const key of selectedTempKeys.value) {
      const [code, text] = key.split("||");
      await wailsApi.removeTempWordForSchema(
        selectedSchemaID.value,
        code,
        text,
      );
    }
    selectedTempKeys.value = new Set();
    showDictMessage("批量删除成功", "success");
    await loadTempData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "批量删除失败", "error");
  }
}

async function handleRemoveTempWord(item: TempWordItem) {
  if (!(await confirm(`确定删除临时词条 "${item.text}" 吗？`))) return;
  try {
    await wailsApi.removeTempWordForSchema(
      selectedSchemaID.value,
      item.code,
      item.text,
    );
    showDictMessage("删除成功", "success");
    await loadTempData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "删除失败", "error");
  }
}

// ===== 候选调整操作 =====
function openShadowDialog(item?: ShadowRuleItem) {
  if (item) {
    shadowDialogEditing.value = true;
    shadowForm.value = {
      code: item.code,
      word: item.word,
      action: item.type,
      position: item.position || 0,
    };
  } else {
    shadowDialogEditing.value = false;
    shadowForm.value = { code: "", word: "", action: "pin", position: 0 };
  }
  shadowDialogVisible.value = true;
}

async function handleSaveShadowRule() {
  if (!shadowForm.value.code || !shadowForm.value.word) {
    showDictMessage("请填写编码和词条", "error");
    return;
  }
  try {
    if (shadowDialogEditing.value) {
      await wailsApi.removeShadowRuleForSchema(
        selectedSchemaID.value,
        shadowForm.value.code,
        shadowForm.value.word,
      );
    }
    if (shadowForm.value.action === "pin") {
      await wailsApi.pinShadowWordForSchema(
        selectedSchemaID.value,
        shadowForm.value.code,
        shadowForm.value.word,
        shadowForm.value.position,
      );
    } else {
      await wailsApi.deleteShadowWordForSchema(
        selectedSchemaID.value,
        shadowForm.value.code,
        shadowForm.value.word,
      );
    }
    showDictMessage(
      shadowDialogEditing.value ? "保存成功" : "添加成功",
      "success",
    );
    shadowDialogVisible.value = false;
    await loadShadowData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "操作失败", "error");
  }
}

async function handleRemoveShadowRule(item: ShadowRuleItem) {
  if (!(await confirm(`确定删除规则 "${item.word}" 吗？`))) return;
  try {
    await wailsApi.removeShadowRuleForSchema(
      selectedSchemaID.value,
      item.code,
      item.word,
    );
    selectedShadowKeys.value.delete(shadowKey(item));
    selectedShadowKeys.value = new Set(selectedShadowKeys.value);
    showDictMessage("删除成功", "success");
    await loadShadowData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "删除失败", "error");
  }
}

async function handleBatchRemoveShadows() {
  if (selectedShadowKeys.value.size === 0) return;
  if (
    !(await confirm(
      `确定删除选中的 ${selectedShadowKeys.value.size} 条规则吗？`,
    ))
  )
    return;
  const toDelete = shadowRules.value.filter((s) =>
    selectedShadowKeys.value.has(shadowKey(s)),
  );
  try {
    for (const item of toDelete) {
      await wailsApi.removeShadowRuleForSchema(
        selectedSchemaID.value,
        item.code,
        item.word,
      );
    }
    selectedShadowKeys.value = new Set();
    showDictMessage(`已删除 ${toDelete.length} 条规则`, "success");
    await loadShadowData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "批量删除失败", "error");
  }
}

// ===== 重置操作 =====
async function handleResetCurrentSchema() {
  const name = selectedSchemaName.value;
  if (
    !(await confirm(
      `确定重置「${name}」的所有用户数据吗？\n\n将清除：用户词库、临时词库、候选调整、词频数据\n\n此操作不可恢复！`,
    ))
  )
    return;
  try {
    await wailsApi.resetUserData(selectedSchemaID.value);
    showDictMessage(`已重置「${name}」的所有用户数据`, "success");
    await loadCurrentSchemaTabData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "重置失败", "error");
  }
}

async function handleResetAllSchemas() {
  if (
    !(await confirm(
      "确定重置所有方案的用户数据吗？\n\n将清除所有方案的：用户词库、临时词库、候选调整、词频数据\n\n此操作不可恢复！",
    ))
  )
    return;
  try {
    await wailsApi.resetUserData("");
    showDictMessage("已重置所有方案的用户数据", "success");
    await loadCurrentSchemaTabData();
    await loadSchemaStatuses();
  } catch (e: unknown) {
    showDictMessage((e as Error).message || "重置失败", "error");
  }
}

// ===== 事件监听 =====
function handleDictEvent(event: DictEvent) {
  if (!event) return;

  if (event.type === "phrase") {
    if (dictMode.value === "phrases") {
      loadPhraseData();
    }
  } else if (dictMode.value === "schema") {
    const matchesSchema =
      !event.schema_id || event.schema_id === selectedSchemaID.value;
    if (!matchesSchema) {
      // 仅刷新方案状态计数
      loadSchemaStatuses();
      return;
    }
    switch (event.type) {
      case "userdict":
        if (schemaSubTab.value === "userdict") loadUserDictData();
        break;
      case "freq":
        if (schemaSubTab.value === "freq") loadFreqData();
        break;
      case "temp":
        if (schemaSubTab.value === "temp") loadTempData();
        break;
      case "shadow":
        if (schemaSubTab.value === "shadow") loadShadowData();
        break;
    }
    loadSchemaStatuses();
  }
}

// ===== 初始化 =====
function closeDropdowns() {
  showWordMenu.value = false;
  showTempMenu.value = false;
  showResetMenu.value = false;
  schemaDropdownOpen.value = false;
}

onMounted(() => {
  loadAllData();
  document.addEventListener("click", closeDropdowns);
  wailsApi.onDictEvent(handleDictEvent);
});

onUnmounted(() => {
  document.removeEventListener("click", closeDropdowns);
  wailsApi.offDictEvent();
});
</script>

<style scoped>
/* ===== 整体布局 ===== */
.dict-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* ===== 紧凑标题 ===== */
.dict-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 12px 0 8px;
}
.dict-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
}
.dict-header-desc {
  font-size: 13px;
  color: #9ca3af;
}

/* ===== 类型选择行 ===== */
.type-selector-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-shrink: 0;
}
.type-label {
  font-size: 13px;
  color: #6b7280;
  font-weight: 500;
}
.type-btn {
  padding: 5px 14px;
  font-size: 13px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: #fff;
  color: #374151;
  cursor: pointer;
  transition: all 0.15s;
}
.type-btn:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
}
.type-btn.active {
  background: #2563eb;
  color: #fff;
  border-color: #2563eb;
}
.schema-dropdown-wrap {
  position: relative;
}
.schema-btn {
  min-width: 120px;
  text-align: left;
}
.schema-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  min-width: 220px;
  z-index: 50;
  overflow: hidden;
  max-height: 320px;
  overflow-y: auto;
}
.schema-dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.1s;
}
.schema-dropdown-item:hover {
  background: #f3f4f6;
}
.schema-dropdown-item.active {
  background: #eff6ff;
  color: #1d4ed8;
}
.schema-dd-name {
  display: flex;
  align-items: center;
  gap: 6px;
}
.schema-dd-dot {
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
.tag-orphan {
  font-size: 11px;
  color: #dc2626;
  margin-left: 4px;
}
.schema-dd-count {
  font-size: 11px;
  color: #9ca3af;
}
.schema-dd-empty {
  padding: 16px;
  text-align: center;
  color: #9ca3af;
  font-size: 13px;
}

/* ===== 残留方案警告 ===== */
.orphan-banner {
  background: #fef3c7;
  border: 1px solid #fbbf24;
  border-radius: 6px;
  padding: 6px 14px;
  font-size: 13px;
  color: #92400e;
  margin-bottom: 8px;
  flex-shrink: 0;
}

/* ===== 方案子标签页 ===== */
.schema-sub-tabs {
  display: flex;
  gap: 0;
  margin-bottom: 8px;
  flex-shrink: 0;
  border-bottom: 1px solid #e5e7eb;
}
.sub-tab {
  padding: 6px 16px;
  font-size: 13px;
  border: none;
  background: none;
  color: #6b7280;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.15s;
  margin-bottom: -1px;
}
.sub-tab:hover {
  color: #374151;
}
.sub-tab.active {
  color: #2563eb;
  border-bottom-color: #2563eb;
  font-weight: 500;
}

/* ===== 内容区域（flex 填充） ===== */
.dict-content-area {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

/* ===== 工具栏 ===== */
.dict-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
  flex-wrap: nowrap;
}
.toolbar-spacer {
  flex: 1;
  min-width: 4px;
}
.toolbar-search {
  width: 100px !important;
  min-width: 60px;
  flex-shrink: 1;
}
.toolbar-more {
  position: relative;
  display: inline-block;
}
.toolbar-dropdown {
  position: absolute;
  right: 0;
  top: 100%;
  margin-top: 4px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  min-width: 120px;
  z-index: 20;
  overflow: hidden;
}
.dropdown-item {
  padding: 8px 14px;
  font-size: 13px;
  color: #374151;
  cursor: pointer;
  white-space: nowrap;
}
.dropdown-item:hover {
  background: #f3f4f6;
}
.dropdown-danger {
  color: #dc2626;
}
.dropdown-danger:hover {
  background: #fef2f2;
}
.toolbar-checkbox-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #6b7280;
  cursor: pointer;
  user-select: none;
}

/* loading 遮罩 */
.content-loading-overlay {
  position: absolute;
  inset: 0;
  background: rgba(255, 255, 255, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
  border-radius: 8px;
}

/* ===== 滚动条 ===== */
.dict-list-scrollable::-webkit-scrollbar,
.dict-table-wrap::-webkit-scrollbar,
.schema-dropdown::-webkit-scrollbar {
  width: 6px;
}
.dict-list-scrollable::-webkit-scrollbar-track,
.dict-table-wrap::-webkit-scrollbar-track,
.schema-dropdown::-webkit-scrollbar-track {
  background: transparent;
  margin: 4px 0;
}
.dict-list-scrollable::-webkit-scrollbar-thumb,
.dict-table-wrap::-webkit-scrollbar-thumb,
.schema-dropdown::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}
.dict-list-scrollable::-webkit-scrollbar-thumb:hover,
.dict-table-wrap::-webkit-scrollbar-thumb:hover,
.schema-dropdown::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

/* ===== 列表 ===== */
.dict-list {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
}
.dict-list-scrollable {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}
.dict-list-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 14px;
  border-bottom: 1px solid #f3f4f6;
  transition: background 0.15s;
}
.dict-list-item:last-child {
  border-bottom: none;
}
.dict-list-item:hover {
  background: #f9fafb;
}
.dict-list-item.selected {
  background: #eff6ff;
}

.item-checkbox {
  width: 15px;
  height: 15px;
  cursor: pointer;
  accent-color: #2563eb;
  flex-shrink: 0;
}

.dict-item-main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
.dict-item-code {
  font-family: "Consolas", "Monaco", monospace;
  font-size: 13px;
  color: #6b7280;
  background: #f3f4f6;
  padding: 2px 8px;
  border-radius: 4px;
  min-width: 50px;
  flex-shrink: 0;
}
.dict-item-text {
  font-size: 14px;
  color: #1f2937;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dict-item-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #e0e7ff;
  color: #3730a3;
  flex-shrink: 0;
}
.dict-item-tag.tag-pin {
  background: #dcfce7;
  color: #166534;
}
.dict-item-tag.tag-delete {
  background: #fee2e2;
  color: #991b1b;
}
.dict-item-weight {
  font-size: 12px;
  color: #9ca3af;
  flex-shrink: 0;
}
.dict-item-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

/* 标签 */
.tag-dynamic {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: #dbeafe;
  color: #1e40af;
  flex-shrink: 0;
}
.tag-system {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: #f3f4f6;
  color: #6b7280;
  flex-shrink: 0;
}
.tag-mapping {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: #dcfce7;
  color: #166534;
  flex-shrink: 0;
}

/* 禁用状态 */
.item-disabled {
  opacity: 0.5;
}
.item-switch-mini {
  flex-shrink: 0;
  cursor: pointer;
}
.item-switch-mini input {
  cursor: pointer;
}

/* ===== 词库表格 ===== */
.dict-table-wrap {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}
.dict-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  table-layout: fixed;
}
.dict-table col.col-check {
  width: 32px;
}
.dict-table col.col-code {
  width: 100px;
}
.dict-table col.col-weight {
  width: 60px;
}
.dict-table col.col-action {
  width: 50px;
}
.dict-table thead {
  position: sticky;
  top: 0;
  background: #f9fafb;
  z-index: 1;
}
.dict-table th {
  padding: 7px 12px;
  text-align: left;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 1px solid #e5e7eb;
}
.sortable-th {
  cursor: pointer;
  user-select: none;
}
.sortable-th:hover {
  color: #374151;
}
.sort-icon {
  font-size: 11px;
  margin-left: 4px;
  color: #9ca3af;
}
.dict-table td {
  padding: 7px 12px;
  border-bottom: 1px solid #f3f4f6;
  color: #1f2937;
  background: #fff;
}
.dict-table tr:last-child td {
  border-bottom: none;
}
.dict-table tbody tr:hover td {
  background: #f9fafb;
}
.dict-table tbody tr.selected td {
  background: #eff6ff;
}
.dict-table tbody tr {
  cursor: default;
}
.td-weight {
  color: #9ca3af;
  font-size: 12px;
}

/* ===== 词频分页 ===== */
.freq-pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 8px 0;
  flex-shrink: 0;
}
.freq-pager-info {
  font-size: 12px;
  color: #6b7280;
}

/* ===== 空状态 ===== */
.dict-empty {
  text-align: center;
  padding: 36px 24px;
  color: #9ca3af;
  background: #f9fafb;
  border-radius: 8px;
  flex: 1;
}

/* ===== 危险按钮 ===== */
.btn-danger-outline {
  color: #dc2626;
  border-color: #fca5a5;
}
.btn-danger-outline:hover {
  background: #fee2e2;
  border-color: #dc2626;
}

/* ===== 其他 ===== */
.dict-note {
  font-size: 12px;
  color: #9ca3af;
  font-style: italic;
  margin-top: 6px;
}
.dict-note-center {
  text-align: center;
  padding: 32px;
  color: #6b7280;
}
.dict-note-center code {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
}
</style>
