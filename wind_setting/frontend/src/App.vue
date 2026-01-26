<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted, nextTick } from 'vue';
import * as api from './api/settings';
import type { Config, Status, EngineInfo, LogEntry } from './api/settings';

// 状态
const loading = ref(true);
const error = ref('');
const connected = ref(false);
const activeTab = ref('general');
const saving = ref(false);
const saveMessage = ref('');
const saveMessageType = ref<'success' | 'error'>('success');

// 数据
const config = ref<Config | null>(null);
const status = ref<Status | null>(null);
const engines = ref<EngineInfo[]>([]);

// 表单数据（用于编辑）
const formData = ref<Partial<Config>>({});

// 测试页面状态
const testInput = ref('');
const testCandidates = ref<any[]>([]);
const testEngine = ref('current');
const testFilterMode = ref('current');
const testLoading = ref(false);

// 日志页面状态
const logs = ref<LogEntry[]>([]);
const logLevel = ref('all');
const logFilter = ref('');
const logAutoScroll = ref(true);
const logTotal = ref(0);
const logContentRef = ref<HTMLElement | null>(null);
let logTimer: number | null = null;

// 标签页
const tabs = [
  { id: 'general', label: '常规', icon: '⚙' },
  { id: 'engine', label: '引擎', icon: '🔧' },
  { id: 'ui', label: '界面', icon: '🎨' },
  { id: 'hotkey', label: '快捷键', icon: '⌨' },
  { id: 'test', label: '测试', icon: '🧪' },
  { id: 'log', label: '日志', icon: '📋' },
  { id: 'about', label: '关于', icon: 'ℹ' },
];

// 是否显示底部操作栏
const showActionBar = computed(() => {
  return !['about', 'test', 'log'].includes(activeTab.value);
});

// 加载数据
async function loadData() {
  loading.value = true;
  error.value = '';

  try {
    const healthRes = await api.checkHealth();
    if (!healthRes.success) {
      connected.value = false;
      error.value = '无法连接到输入法服务，请确保 WindInput 正在运行';
      loading.value = false;
      return;
    }
    connected.value = true;

    const configRes = await api.getConfig();
    if (configRes.success && configRes.data) {
      // Ensure toolbar and input fields exist with defaults
      const cfg = configRes.data;
      if (!cfg.toolbar) {
        cfg.toolbar = { visible: false, position_x: 100, position_y: 100 };
      }
      if (!cfg.input) {
        cfg.input = { full_width: false, chinese_punctuation: true };
      }
      config.value = cfg;
      formData.value = JSON.parse(JSON.stringify(cfg));
    }

    const statusRes = await api.getStatus();
    if (statusRes.success && statusRes.data) {
      status.value = statusRes.data;
    }

    const enginesRes = await api.getEngineList();
    if (enginesRes.success && enginesRes.data) {
      engines.value = enginesRes.data.engines;
    }
  } catch (e) {
    error.value = '加载数据失败';
  } finally {
    loading.value = false;
  }
}

// 保存配置
async function saveConfig() {
  if (!formData.value) return;

  saving.value = true;
  saveMessage.value = '';

  try {
    const res = await api.updateConfig(formData.value);
    if (res.success && res.data) {
      saveMessageType.value = 'success';
      saveMessage.value = '保存成功';
      if (res.data.needReload.length > 0) {
        saveMessage.value += '（部分设置需要重载生效）';
      }
      // Update local config without reloading everything (preserves scroll position)
      config.value = JSON.parse(JSON.stringify(formData.value)) as Config;
    } else {
      saveMessageType.value = 'error';
      saveMessage.value = res.error || '保存失败';
    }
  } catch (e) {
    saveMessageType.value = 'error';
    saveMessage.value = '保存失败';
  } finally {
    saving.value = false;
    setTimeout(() => { saveMessage.value = ''; }, 3000);
  }
}

// 切换引擎
async function handleSwitchEngine(type: string) {
  try {
    const res = await api.switchEngine(type);
    if (res.success) {
      await loadData();
    }
  } catch (e) {
    console.error('切换引擎失败', e);
  }
}

// 重载配置
async function handleReload() {
  try {
    const res = await api.reloadConfig();
    if (res.success) {
      saveMessageType.value = 'success';
      saveMessage.value = '重载成功';
      // Only reload config data, not the full page
      const configRes = await api.getConfig();
      if (configRes.success && configRes.data) {
        const cfg = configRes.data;
        if (!cfg.toolbar) {
          cfg.toolbar = { visible: false, position_x: 100, position_y: 100 };
        }
        if (!cfg.input) {
          cfg.input = { full_width: false, chinese_punctuation: true };
        }
        config.value = cfg;
        formData.value = JSON.parse(JSON.stringify(cfg));
      }
    } else {
      saveMessageType.value = 'error';
      saveMessage.value = res.error || '重载失败';
    }
  } catch (e) {
    saveMessageType.value = 'error';
    saveMessage.value = '重载失败';
  }
  setTimeout(() => { saveMessage.value = ''; }, 3000);
}

// 仅刷新状态信息（用于关于页面）
async function refreshStatus() {
  try {
    const statusRes = await api.getStatus();
    if (statusRes.success && statusRes.data) {
      status.value = statusRes.data;
    }
  } catch (e) {
    console.error('刷新状态失败', e);
  }
}

// 重置为当前配置
function resetToDefault() {
  if (config.value) {
    formData.value = JSON.parse(JSON.stringify(config.value));
    saveMessageType.value = 'success';
    saveMessage.value = '已重置为当前配置';
    setTimeout(() => { saveMessage.value = ''; }, 2000);
  }
}

// 测试输入
async function handleTestInput() {
  if (!testInput.value.trim()) {
    testCandidates.value = [];
    return;
  }

  testLoading.value = true;
  try {
    const res = await api.testConvert(testInput.value, testEngine.value, testFilterMode.value);
    if (res.success && res.data) {
      testCandidates.value = res.data.candidates || [];
    }
  } catch (e) {
    console.error('测试失败', e);
  } finally {
    testLoading.value = false;
  }
}

// 监听测试输入变化
watch(testInput, () => {
  handleTestInput();
});

watch([testEngine, testFilterMode], () => {
  if (testInput.value) {
    handleTestInput();
  }
});

// 加载日志
async function loadLogs() {
  try {
    const res = await api.getLogs(logLevel.value, logFilter.value);
    if (res.success && res.data) {
      logs.value = res.data.logs || [];
      logTotal.value = res.data.total || 0;
      // 自动滚动到底部
      if (logAutoScroll.value) {
        nextTick(() => {
          if (logContentRef.value) {
            logContentRef.value.scrollTop = logContentRef.value.scrollHeight;
          }
        });
      }
    }
  } catch (e) {
    console.error('加载日志失败', e);
  }
}

// 清空日志
async function handleClearLogs() {
  try {
    await api.clearLogs();
    logs.value = [];
    logTotal.value = 0;
  } catch (e) {
    console.error('清空日志失败', e);
  }
}

// 监听日志过滤变化
watch([logLevel, logFilter], () => {
  loadLogs();
});

// 监听标签页切换
watch(activeTab, (tab) => {
  if (tab === 'log') {
    loadLogs();
    // 启动定时刷新
    if (logTimer) clearInterval(logTimer);
    logTimer = window.setInterval(loadLogs, 2000);
  } else {
    // 停止定时刷新
    if (logTimer) {
      clearInterval(logTimer);
      logTimer = null;
    }
  }
});

onMounted(() => {
  loadData();
});

onUnmounted(() => {
  if (logTimer) {
    clearInterval(logTimer);
  }
});
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="logo">
        <span class="logo-icon">🌬</span>
        <span class="logo-text">WindInput</span>
      </div>
      <nav class="nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['nav-item', { active: activeTab === tab.id }]"
          @click="activeTab = tab.id"
        >
          <span class="nav-icon">{{ tab.icon }}</span>
          <span class="nav-label">{{ tab.label }}</span>
        </button>
      </nav>
      <div class="sidebar-footer">
        <div :class="['status-badge', connected ? 'connected' : 'disconnected']">
          <span class="status-dot"></span>
          {{ connected ? '已连接' : '未连接' }}
        </div>
      </div>
    </aside>

    <main class="main">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>加载中...</p>
      </div>

      <div v-else-if="error" class="error-panel">
        <div class="error-icon">⚠</div>
        <p>{{ error }}</p>
        <button class="btn btn-primary" @click="loadData">重试</button>
      </div>

      <div v-else class="content">
        <!-- 常规设置 -->
        <section v-if="activeTab === 'general'" class="section">
          <div class="section-header">
            <h2>常规设置</h2>
            <p class="section-desc">基本的输入法行为设置</p>
          </div>

          <div class="settings-card">
            <div class="setting-item">
              <div class="setting-info">
                <label>启动模式</label>
                <p class="setting-hint">启用后，每次激活输入法时默认为中文输入状态</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.general!.start_in_chinese_mode" />
                  <span class="slider"></span>
                </label>
                <span class="switch-label">{{ formData.general?.start_in_chinese_mode ? '中文' : '英文' }}</span>
              </div>
            </div>

            <div class="setting-item">
              <div class="setting-info">
                <label>日志级别</label>
                <p class="setting-hint">更改日志级别需要重启服务才能生效</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.general!.log_level" class="select">
                  <option value="debug">Debug（调试）</option>
                  <option value="info">Info（信息）</option>
                  <option value="warn">Warn（警告）</option>
                  <option value="error">Error（错误）</option>
                </select>
              </div>
            </div>
          </div>
        </section>

        <!-- 引擎设置 -->
        <section v-if="activeTab === 'engine'" class="section">
          <div class="section-header">
            <h2>引擎设置</h2>
            <p class="section-desc">输入法引擎和过滤设置</p>
          </div>

          <div class="settings-card">
            <div class="card-title">当前引擎</div>
            <div class="engine-cards">
              <div
                v-for="engine in engines"
                :key="engine.type"
                :class="['engine-card', { active: engine.isActive }]"
                @click="handleSwitchEngine(engine.type)"
              >
                <div class="engine-icon">{{ engine.type === 'pinyin' ? '拼' : '五' }}</div>
                <div class="engine-info">
                  <div class="engine-name">{{ engine.type === 'pinyin' ? '拼音输入' : '五笔输入' }}</div>
                  <div class="engine-desc">{{ engine.description }}</div>
                </div>
                <div v-if="engine.isActive" class="engine-check">✓</div>
              </div>
            </div>
          </div>

          <div class="settings-card">
            <div class="card-title">过滤设置</div>
            <div class="setting-item">
              <div class="setting-info">
                <label>字符过滤模式</label>
                <p class="setting-hint">控制候选词中显示的字符范围</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.engine!.filter_mode" class="select">
                  <option value="smart">智能模式（推荐）</option>
                  <option value="general">仅通用字</option>
                  <option value="gb18030">全部字符</option>
                </select>
              </div>
            </div>
          </div>

          <div v-if="formData.engine?.type === 'pinyin'" class="settings-card">
            <div class="card-title">拼音设置</div>
            <div class="setting-item">
              <div class="setting-info">
                <label>五笔编码提示</label>
                <p class="setting-hint">在候选词旁边显示对应的五笔编码</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.engine!.pinyin.show_wubi_hint" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
          </div>

          <div v-if="formData.engine?.type === 'wubi'" class="settings-card">
            <div class="card-title">五笔设置</div>
            <div class="setting-item">
              <div class="setting-info">
                <label>自动上屏</label>
                <p class="setting-hint">候选词满足条件时自动上屏</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.engine!.wubi.auto_commit" class="select">
                  <option value="none">不自动上屏</option>
                  <option value="unique">候选唯一时</option>
                  <option value="unique_at_4">四码唯一时</option>
                  <option value="unique_full_match">完整匹配唯一时</option>
                </select>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>空码处理</label>
                <p class="setting-hint">输入无匹配候选时的处理方式</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.engine!.wubi.empty_code" class="select">
                  <option value="none">继续输入</option>
                  <option value="clear">清空编码</option>
                  <option value="clear_at_4">四码时清空</option>
                  <option value="to_english">转为英文</option>
                </select>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>五码顶字</label>
                <p class="setting-hint">输入第五码时自动上屏首选</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.engine!.wubi.top_code_commit" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>标点顶字</label>
                <p class="setting-hint">输入标点时自动上屏首选</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.engine!.wubi.punct_commit" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- 界面设置 -->
        <section v-if="activeTab === 'ui'" class="section">
          <div class="section-header">
            <h2>界面设置</h2>
            <p class="section-desc">候选窗口外观设置</p>
          </div>

          <div class="settings-card">
            <div class="setting-item">
              <div class="setting-info">
                <label>字体大小</label>
                <p class="setting-hint">候选窗口的字体大小</p>
              </div>
              <div class="setting-control range-control">
                <input type="range" min="12" max="36" step="1" v-model.number="formData.ui!.font_size" />
                <span class="range-value">{{ formData.ui?.font_size }}px</span>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>每页候选数</label>
                <p class="setting-hint">每页显示的候选词数量</p>
              </div>
              <div class="setting-control range-control">
                <input type="range" min="3" max="9" step="1" v-model.number="formData.ui!.candidates_per_page" />
                <span class="range-value">{{ formData.ui?.candidates_per_page }}</span>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>自定义字体</label>
                <p class="setting-hint">留空使用系统默认字体</p>
              </div>
              <div class="setting-control">
                <input type="text" v-model="formData.ui!.font_path" class="input" placeholder="C:\Windows\Fonts\msyh.ttc" />
              </div>
            </div>
          </div>

          <div class="settings-card">
            <div class="card-title">工具栏</div>
            <div class="setting-item">
              <div class="setting-info">
                <label>显示工具栏</label>
                <p class="setting-hint">在屏幕上显示可拖动的输入法工具栏</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.toolbar!.visible" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
          </div>

          <div class="settings-card">
            <div class="card-title">输入选项</div>
            <div class="setting-item">
              <div class="setting-info">
                <label>全角字符</label>
                <p class="setting-hint">启用后输出全角字符（如：ＡＢＣ１２３）</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.input!.full_width" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>中文标点</label>
                <p class="setting-hint">启用后输出中文标点符号（如：，。！？）</p>
              </div>
              <div class="setting-control">
                <label class="switch">
                  <input type="checkbox" v-model="formData.input!.chinese_punctuation" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- 快捷键设置 -->
        <section v-if="activeTab === 'hotkey'" class="section">
          <div class="section-header">
            <h2>快捷键设置</h2>
            <p class="section-desc">输入法快捷键配置</p>
          </div>

          <div class="settings-card">
            <div class="setting-item">
              <div class="setting-info">
                <label>切换中英文</label>
                <p class="setting-hint">在中文和英文模式间切换的快捷键</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.hotkeys!.toggle_mode" class="select">
                  <option value="shift">Shift</option>
                  <option value="ctrl+space">Ctrl + Space</option>
                </select>
              </div>
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <label>切换引擎</label>
                <p class="setting-hint">在拼音和五笔引擎间切换的快捷键</p>
              </div>
              <div class="setting-control">
                <select v-model="formData.hotkeys!.switch_engine" class="select">
                  <option value="ctrl+`">Ctrl + `</option>
                  <option value="ctrl+shift+e">Ctrl + Shift + E</option>
                </select>
              </div>
            </div>
          </div>
        </section>

        <!-- 测试页面 -->
        <section v-if="activeTab === 'test'" class="section">
          <div class="section-header">
            <h2>码表测试</h2>
            <p class="section-desc">在此测试输入法候选词，无需切换系统输入法</p>
          </div>

          <div class="settings-card">
            <div class="test-options">
              <div class="test-option">
                <label>引擎</label>
                <select v-model="testEngine" class="select select-sm">
                  <option value="current">当前引擎</option>
                  <option value="pinyin">拼音</option>
                  <option value="wubi">五笔</option>
                </select>
              </div>
              <div class="test-option">
                <label>过滤</label>
                <select v-model="testFilterMode" class="select select-sm">
                  <option value="current">当前设置</option>
                  <option value="smart">智能模式</option>
                  <option value="general">仅通用字</option>
                  <option value="gb18030">全部字符</option>
                </select>
              </div>
            </div>

            <div class="test-input-wrap">
              <input
                type="text"
                v-model="testInput"
                class="test-input"
                placeholder="输入编码进行测试..."
                @keydown.enter.prevent
              />
              <span v-if="testLoading" class="test-loading">加载中...</span>
            </div>

            <div class="test-candidates-wrap" v-if="testCandidates.length > 0">
              <div class="test-candidates">
                <div class="test-candidate" v-for="(cand, idx) in testCandidates" :key="idx">
                  <span class="cand-index">{{ idx + 1 }}.</span>
                  <span class="cand-text">{{ cand.text }}</span>
                  <span class="cand-code" v-if="cand.code">{{ cand.code }}</span>
                  <span class="cand-common" v-if="cand.isCommon">通用</span>
                  <span class="cand-rare" v-else>生僻</span>
                </div>
              </div>
            </div>
            <div class="test-empty" v-else-if="testInput && !testLoading">
              无匹配候选词
            </div>
          </div>
        </section>

        <!-- 日志页面 -->
        <section v-if="activeTab === 'log'" class="section section-full">
          <div class="section-header">
            <h2>调试日志</h2>
            <p class="section-desc">查看服务端运行日志（共 {{ logTotal }} 条）</p>
          </div>

          <div class="settings-card log-card">
            <div class="log-toolbar">
              <select v-model="logLevel" class="select select-sm">
                <option value="all">全部级别</option>
                <option value="DEBUG">DEBUG</option>
                <option value="INFO">INFO</option>
                <option value="WARN">WARN</option>
                <option value="ERROR">ERROR</option>
              </select>
              <input type="text" v-model="logFilter" class="input input-sm" placeholder="搜索日志..." />
              <label class="checkbox-inline">
                <input type="checkbox" v-model="logAutoScroll" />
                自动滚动
              </label>
              <button class="btn btn-sm" @click="loadLogs">刷新</button>
              <button class="btn btn-sm" @click="handleClearLogs">清空</button>
            </div>
            <div class="log-content" ref="logContentRef">
              <div v-for="(log, idx) in logs" :key="idx" :class="['log-line', 'log-' + log.level.toLowerCase()]">
                <span class="log-time">{{ log.time }}</span>
                <span class="log-level">{{ log.level }}</span>
                <span class="log-msg">{{ log.message }}</span>
              </div>
              <div v-if="logs.length === 0" class="log-empty">暂无日志</div>
            </div>
          </div>
        </section>

        <!-- 关于 -->
        <section v-if="activeTab === 'about'" class="section">
          <div class="section-header">
            <h2>关于</h2>
            <p class="section-desc">WindInput 输入法信息</p>
          </div>

          <div class="settings-card about-card" v-if="status">
            <div class="about-header">
              <div class="about-icon-wrap">🌬</div>
              <div class="about-title">
                <h3>{{ status.service.name }}</h3>
                <p>版本 {{ status.service.version }}</p>
              </div>
            </div>
            <div class="about-stats">
              <div class="stat-item">
                <span class="stat-label">运行时间</span>
                <span class="stat-value">{{ status.service.uptime }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">当前引擎</span>
                <span class="stat-value">{{ status.engine.info }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">内存使用</span>
                <span class="stat-value">{{ status.memory.allocMB }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">系统内存</span>
                <span class="stat-value">{{ status.memory.sysMB }}</span>
              </div>
            </div>
          </div>

          <div class="about-actions">
            <button class="btn" @click="handleReload">重载配置</button>
            <button class="btn" @click="refreshStatus">刷新状态</button>
          </div>
        </section>

        <!-- 底部操作栏 -->
        <div class="action-bar" v-if="showActionBar">
          <div class="action-message">
            <span v-if="saveMessage" :class="['message', saveMessageType]">
              {{ saveMessage }}
            </span>
          </div>
          <div class="action-buttons">
            <button class="btn" @click="resetToDefault">重置</button>
            <button class="btn btn-primary" @click="saveConfig" :disabled="saving">
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Microsoft YaHei UI', sans-serif;
  background: #f0f2f5;
  color: #1f2937;
  line-height: 1.5;
  font-size: 14px;
}

.app { display: flex; min-height: 100vh; height: 100vh; }

/* Sidebar */
.sidebar {
  width: 200px;
  min-width: 200px;
  background: #fff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
}

.logo {
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid #e5e7eb;
}
.logo-icon { font-size: 24px; }
.logo-text { font-size: 16px; font-weight: 600; color: #1f2937; }

.nav { flex: 1; padding: 12px; overflow-y: auto; }
.nav-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: none;
  background: none;
  cursor: pointer;
  border-radius: 8px;
  font-size: 14px;
  color: #6b7280;
  transition: all 0.15s;
  text-align: left;
}
.nav-item:hover { background: #f3f4f6; color: #374151; }
.nav-item.active { background: #eff6ff; color: #2563eb; font-weight: 500; }
.nav-icon { font-size: 16px; width: 20px; text-align: center; }

.sidebar-footer { padding: 16px; border-top: 1px solid #e5e7eb; }
.status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 20px;
}
.status-badge.connected { background: #dcfce7; color: #166534; }
.status-badge.disconnected { background: #fee2e2; color: #991b1b; }
.status-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

/* Main */
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; min-width: 0; }
.content { flex: 1; padding: 24px 32px; overflow-y: auto; padding-bottom: 80px; }

/* Section */
.section { max-width: 700px; width: 100%; }
.section-full { max-width: none; }
.section-header { margin-bottom: 20px; }
.section-header h2 { font-size: 20px; font-weight: 600; color: #111827; margin-bottom: 4px; text-align: left; }
.section-desc { font-size: 13px; color: #6b7280; text-align: left; }

/* Settings Card */
.settings-card {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}
.card-title { font-size: 13px; font-weight: 600; color: #374151; margin-bottom: 16px; text-transform: uppercase; letter-spacing: 0.5px; text-align: left; }

/* Setting Item */
.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid #f3f4f6;
  gap: 16px;
}
.setting-item:last-child { border-bottom: none; padding-bottom: 0; }
.setting-item:first-child { padding-top: 0; }
.setting-info { flex: 1; min-width: 0; }
.setting-info label { font-size: 14px; font-weight: 500; color: #1f2937; display: block; text-align: left; }
.setting-hint { font-size: 12px; color: #9ca3af; margin-top: 2px; text-align: left; }
.setting-control { flex-shrink: 0; display: flex; align-items: center; gap: 10px; }

/* Form Controls */
.select {
  padding: 8px 32px 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  background: #fff url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e") no-repeat right 8px center;
  background-size: 16px;
  cursor: pointer;
  color: #1f2937;
  min-width: 160px;
  -webkit-appearance: none;
  appearance: none;
}
.select:focus { outline: none; border-color: #2563eb; box-shadow: 0 0 0 3px rgba(37,99,235,0.1); }
.select-sm { padding: 6px 28px 6px 10px; font-size: 13px; min-width: 100px; }

.input {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  width: 240px;
  color: #1f2937;
}
.input:focus { outline: none; border-color: #2563eb; box-shadow: 0 0 0 3px rgba(37,99,235,0.1); }
.input::placeholder { color: #9ca3af; }
.input-sm { padding: 6px 10px; font-size: 13px; width: 160px; }

/* Switch */
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}
.switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute;
  cursor: pointer;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: #d1d5db;
  transition: .2s;
  border-radius: 24px;
}
.slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: .2s;
  border-radius: 50%;
}
input:checked + .slider { background-color: #2563eb; }
input:checked + .slider:before { transform: translateX(20px); }
.switch-label { font-size: 13px; color: #6b7280; min-width: 30px; }

/* Range */
.range-control { flex-direction: column; align-items: flex-end; gap: 4px; }
.range-control input[type="range"] {
  width: 160px;
  height: 4px;
  -webkit-appearance: none;
  background: #e5e7eb;
  border-radius: 2px;
}
.range-control input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 16px;
  height: 16px;
  background: #2563eb;
  border-radius: 50%;
  cursor: pointer;
}
.range-value { font-size: 13px; font-weight: 500; color: #374151; }

/* Engine Cards */
.engine-cards { display: flex; gap: 12px; margin-bottom: 8px; flex-wrap: wrap; }
.engine-card {
  flex: 1;
  min-width: 200px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 2px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
}
.engine-card:hover { border-color: #93c5fd; }
.engine-card.active { border-color: #2563eb; background: #eff6ff; }
.engine-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #2563eb;
  color: #fff;
  border-radius: 8px;
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0;
}
.engine-info { flex: 1; min-width: 0; text-align: left; }
.engine-name { font-weight: 500; color: #1f2937; }
.engine-desc { font-size: 12px; color: #6b7280; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.engine-check { color: #2563eb; font-size: 18px; flex-shrink: 0; }

/* Action Bar */
.action-bar {
  position: fixed;
  bottom: 0;
  left: 200px;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 32px;
  background: #fff;
  border-top: 1px solid #e5e7eb;
  z-index: 100;
}
.action-message { flex: 1; }
.message { font-size: 13px; padding: 6px 12px; border-radius: 6px; }
.message.success { background: #dcfce7; color: #166534; }
.message.error { background: #fee2e2; color: #991b1b; }
.action-buttons { display: flex; gap: 10px; }

.btn {
  padding: 8px 16px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  background: #fff;
  color: #374151;
}
.btn:hover { background: #f9fafb; border-color: #9ca3af; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: #2563eb; color: #fff; border-color: #2563eb; }
.btn-primary:hover:not(:disabled) { background: #1d4ed8; }
.btn-sm { padding: 5px 10px; font-size: 12px; }

/* Test Page */
.test-options { display: flex; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
.test-option { display: flex; align-items: center; gap: 8px; }
.test-option label { font-size: 13px; color: #6b7280; }

.test-input-wrap { position: relative; margin-bottom: 16px; }
.test-input {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  font-size: 16px;
  color: #1f2937;
}
.test-input:focus { outline: none; border-color: #2563eb; }
.test-loading { position: absolute; right: 16px; top: 50%; transform: translateY(-50%); font-size: 12px; color: #9ca3af; }

.test-candidates-wrap { max-height: 400px; overflow-y: auto; }
.test-candidates { border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; }
.test-candidate {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid #f3f4f6;
}
.test-candidate:last-child { border-bottom: none; }
.test-candidate:hover { background: #f9fafb; }
.cand-index { color: #9ca3af; font-size: 13px; min-width: 24px; flex-shrink: 0; }
.cand-text { font-size: 16px; color: #1f2937; word-break: break-all; }
.cand-code { font-size: 12px; color: #6b7280; background: #f3f4f6; padding: 2px 6px; border-radius: 4px; flex-shrink: 0; }
.cand-common { font-size: 11px; color: #059669; background: #d1fae5; padding: 2px 6px; border-radius: 4px; flex-shrink: 0; }
.cand-rare { font-size: 11px; color: #dc2626; background: #fee2e2; padding: 2px 6px; border-radius: 4px; flex-shrink: 0; }
.test-empty { text-align: center; padding: 32px; color: #9ca3af; }

/* Log Page */
.log-card { display: flex; flex-direction: column; height: calc(100vh - 200px); min-height: 300px; }
.log-toolbar { display: flex; align-items: center; gap: 12px; padding-bottom: 12px; border-bottom: 1px solid #e5e7eb; flex-wrap: wrap; }
.checkbox-inline { display: flex; align-items: center; gap: 6px; font-size: 13px; color: #6b7280; cursor: pointer; white-space: nowrap; }
.checkbox-inline input { cursor: pointer; }
.log-content { flex: 1; overflow-y: auto; font-family: 'Consolas', 'Monaco', 'Courier New', monospace; font-size: 12px; padding: 12px 0; }
.log-line { display: flex; gap: 12px; padding: 4px 0; text-align: left; align-items: flex-start; }
.log-time { color: #9ca3af; flex-shrink: 0; }
.log-level { font-weight: 600; min-width: 50px; flex-shrink: 0; }
.log-msg { color: #374151; word-break: break-all; flex: 1; }
.log-debug .log-level { color: #6b7280; }
.log-info .log-level { color: #2563eb; }
.log-warn .log-level { color: #d97706; }
.log-error .log-level { color: #dc2626; }
.log-empty { color: #9ca3af; text-align: center; padding: 32px; }

/* About Page */
.about-card { padding: 24px; }
.about-header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.about-icon-wrap { font-size: 48px; width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; background: #eff6ff; border-radius: 12px; flex-shrink: 0; }
.about-title { text-align: left; }
.about-title h3 { font-size: 18px; font-weight: 600; margin: 0; color: #1f2937; }
.about-title p { color: #6b7280; margin: 4px 0 0; }
.about-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; }
.stat-item { padding: 12px 16px; background: #f9fafb; border-radius: 8px; text-align: left; }
.stat-label { display: block; font-size: 12px; color: #9ca3af; margin-bottom: 4px; }
.stat-value { font-size: 14px; font-weight: 500; color: #1f2937; }
.about-actions { display: flex; gap: 12px; margin-top: 16px; }

/* Loading & Error */
.loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: #6b7280;
}
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top-color: #2563eb;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.error-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  text-align: center;
  padding: 32px;
}
.error-icon { font-size: 48px; }
.error-panel p { color: #6b7280; max-width: 300px; }

/* Responsive */
@media (max-width: 768px) {
  .sidebar { width: 60px; min-width: 60px; }
  .logo-text { display: none; }
  .nav-label { display: none; }
  .nav-item { justify-content: center; padding: 12px; }
  .action-bar { left: 60px; }
  .content { padding: 16px; }
}
</style>
