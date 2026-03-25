<template>
  <section class="section">
    <div class="section-header">
      <h2>高级设置</h2>
      <p class="section-desc">故障排查与调试工具</p>
    </div>

    <div class="settings-card">
      <div class="card-title">配置文件</div>
      <div class="setting-item">
        <div class="setting-info">
          <label>配置文件目录</label>
          <p class="setting-hint">%APPDATA%\WindInput</p>
        </div>
        <div class="setting-control">
          <button class="btn btn-sm" @click="$emit('openConfigFolder')">打开文件夹</button>
        </div>
      </div>
    </div>

    <div class="settings-card">
      <div class="card-title">日志设置</div>
      <div class="setting-item">
        <div class="setting-info">
          <label>日志级别</label>
          <p class="setting-hint">重启输入法服务后生效</p>
        </div>
        <div class="setting-control">
          <select v-model="formData.advanced.log_level" class="select">
            <option value="debug">Debug（调试）</option>
            <option value="info">Info（信息）</option>
            <option value="warn">Warn（警告）</option>
            <option value="error">Error（错误）</option>
          </select>
        </div>
      </div>
      <div class="setting-item">
        <div class="setting-info">
          <label>日志文件位置</label>
          <p class="setting-hint">{{ logPath }}</p>
        </div>
        <div class="setting-control">
          <button class="btn btn-sm" @click="$emit('openLogFolder')">打开文件夹</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Config } from "../api/settings";

const props = defineProps<{
  formData: Config;
  isWailsEnv: boolean;
}>();

const emit = defineEmits<{
  openLogFolder: [];
  openConfigFolder: [];
}>();

const logPath = "%LOCALAPPDATA%\\WindInput\\logs\\";
</script>

<style scoped>
</style>
