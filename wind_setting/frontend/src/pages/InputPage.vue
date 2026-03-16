<template>
  <section class="section">
    <div class="section-header">
      <h2>输入习惯</h2>
      <p class="section-desc">定制您的打字体验</p>
    </div>

    <!-- 字符与标点 -->
    <div class="settings-card">
      <div class="card-title">字符与标点</div>
      <div class="setting-item">
        <div class="setting-info">
          <label>候选字符范围</label>
          <p class="setting-hint">控制候选词中显示的字符范围</p>
        </div>
        <div class="setting-control">
          <select v-model="formData.engine.filter_mode" class="select">
            <option value="smart">智能模式（推荐）</option>
            <option value="general">仅常用字</option>
            <option value="gb18030">大字符集</option>
          </select>
        </div>
      </div>
      <div class="setting-item">
        <div class="setting-info">
          <label>标点随中英文切换</label>
          <p class="setting-hint">切换到英文模式时自动切换英文标点</p>
        </div>
        <div class="setting-control">
          <label class="switch">
            <input
              type="checkbox"
              v-model="formData.input.punct_follow_mode"
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
    </div>

    <!-- 默认状态 -->
    <div class="settings-card">
      <div class="card-title">默认状态</div>
      <div class="setting-item">
        <div class="setting-info">
          <label>记忆前次状态</label>
          <p class="setting-hint">
            启用后将使用上次退出时的状态，忽略以下默认设置
          </p>
        </div>
        <div class="setting-control">
          <label class="switch">
            <input
              type="checkbox"
              v-model="formData.startup.remember_last_state"
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      <div
        class="setting-item"
        :class="{ 'item-disabled': formData.startup.remember_last_state }"
      >
        <div class="setting-info">
          <label>初始语言模式</label>
          <p class="setting-hint">每次激活输入法时的默认语言</p>
        </div>
        <div class="setting-control">
          <div class="segmented-control">
            <button
              :class="{ active: formData.startup.default_chinese_mode }"
              @click="formData.startup.default_chinese_mode = true"
              :disabled="formData.startup.remember_last_state"
            >
              中文
            </button>
            <button
              :class="{ active: !formData.startup.default_chinese_mode }"
              @click="formData.startup.default_chinese_mode = false"
              :disabled="formData.startup.remember_last_state"
            >
              英文
            </button>
          </div>
        </div>
      </div>
      <div
        class="setting-item"
        :class="{ 'item-disabled': formData.startup.remember_last_state }"
      >
        <div class="setting-info">
          <label>初始字符宽度</label>
          <p class="setting-hint">每次激活输入法时的默认字符宽度</p>
        </div>
        <div class="setting-control">
          <div class="segmented-control">
            <button
              :class="{ active: !formData.startup.default_full_width }"
              @click="formData.startup.default_full_width = false"
              :disabled="formData.startup.remember_last_state"
            >
              半角
            </button>
            <button
              :class="{ active: formData.startup.default_full_width }"
              @click="formData.startup.default_full_width = true"
              :disabled="formData.startup.remember_last_state"
            >
              全角
            </button>
          </div>
        </div>
      </div>
      <div
        class="setting-item"
        :class="{ 'item-disabled': formData.startup.remember_last_state }"
      >
        <div class="setting-info">
          <label>初始标点模式</label>
          <p class="setting-hint">每次激活输入法时的默认标点类型</p>
        </div>
        <div class="setting-control">
          <div class="segmented-control">
            <button
              :class="{ active: formData.startup.default_chinese_punct }"
              @click="formData.startup.default_chinese_punct = true"
              :disabled="formData.startup.remember_last_state"
            >
              中文标点
            </button>
            <button
              :class="{ active: !formData.startup.default_chinese_punct }"
              @click="formData.startup.default_chinese_punct = false"
              :disabled="formData.startup.remember_last_state"
            >
              英文标点
            </button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Config } from "../api/settings";

defineProps<{
  formData: Config;
}>();
</script>
