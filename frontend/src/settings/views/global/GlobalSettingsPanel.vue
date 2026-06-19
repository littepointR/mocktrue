<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NForm, NFormItem, NSelect } from 'naive-ui'
import { useSettingsStore } from '../../stores/settingsStore'

const store = useSettingsStore()

const themeOptions = [
  { label: '深色', value: 'dark' },
  { label: '浅色', value: 'light' },
  { label: '跟随系统', value: 'system' },
]

const theme = computed({
  get: () => store.global.Theme,
  set: value => store.updateGlobal({ Theme: value }),
})
</script>

<template>
  <section class="settings-panel">
    <header class="settings-panel__header">
      <h2>全局设置</h2>
      <NButton size="small" secondary @click="store.resetGlobal()">恢复默认</NButton>
    </header>

    <NForm class="settings-panel__form" label-placement="top" size="small">
      <NFormItem label="主题">
        <NSelect v-model:value="theme" :options="themeOptions" />
      </NFormItem>
    </NForm>
  </section>
</template>

<style scoped>
.settings-panel {
  max-width: 720px;
  padding: 20px 24px;
}
.settings-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
.settings-panel__header h2 {
  margin: 0;
  color: var(--app-text, #d4d4d4);
  font-size: 18px;
  font-weight: 600;
}
.settings-panel__form {
  display: grid;
  gap: 4px;
}
</style>
