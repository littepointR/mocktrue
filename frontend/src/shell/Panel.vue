<script setup lang="ts">
import { ref } from 'vue'

const visible = ref(true)
</script>

<template>
  <div class="panel" v-show="visible">
    <div class="panel__header">
      <span class="panel__title">面板</span>
      <button class="panel__close" @click="visible = false" title="折叠">✕</button>
    </div>
    <div class="panel__body">
      <div class="panel__placeholder">面板内容（占位）</div>
    </div>
  </div>
</template>

<style scoped>
.panel {
  height: 150px;
  flex: 0 0 150px;
  background: var(--app-surface, #252526);
  border-top: 1px solid var(--app-border, #2d2d2d);
  display: flex;
  flex-direction: column;
}
.panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
}
.panel__title {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--app-text-muted, #858585);
}
.panel__close {
  background: none;
  border: none;
  color: var(--app-text-muted, #858585);
  cursor: pointer;
  font-size: 12px;
}
.panel__body {
  flex: 1;
  padding: 8px 12px;
  overflow-y: auto;
}
.panel__placeholder {
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
</style>
