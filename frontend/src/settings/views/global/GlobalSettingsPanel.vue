<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInputGroup, NSelect } from 'naive-ui'
import { useSettingsStore } from '../../stores/settingsStore'
import { useWorkspaceFileStore } from '../../../workspace/stores/workspaceFileStore'
import { useRegistry } from '../../../core/registry'

const store = useSettingsStore()
const workspaceFile = useWorkspaceFileStore()
const registry = useRegistry()
const exampleOptions = workspaceFile.listDemos().map(demo => ({
  label: demo.title,
  value: demo.id,
}))
const selectedExampleId = computed(() => workspaceFile.selectedDemoId)

const themeOptions = [
  { label: '深色', value: 'dark' },
  { label: '浅色', value: 'light' },
  { label: '跟随系统', value: 'system' },
]

const theme = computed({
  get: () => store.global.Theme,
  set: value => store.updateGlobal({ Theme: value }),
})
const pendingAction = ref<string | null>(null)

const isBusy = computed(() => pendingAction.value !== null)

async function runWorkspaceAction(action: string, callback: () => Promise<unknown>) {
  pendingAction.value = action
  try {
    await callback()
  } finally {
    pendingAction.value = null
  }
}

function updateSelectedExample(value: string | number | null) {
  workspaceFile.setSelectedDemo(value)
}

async function loadExampleWorkspace() {
  if (!selectedExampleId.value) return
  await runWorkspaceAction('demo', async () => {
    await workspaceFile.loadDemo(selectedExampleId.value)
    if (registry.list().some(item => item.moduleId === 'serial')) {
      registry.setActive('serial')
    }
  })
}
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
      <NFormItem label="示例配置">
        <NInputGroup>
          <NSelect
            class="settings-panel__demo-select"
            :value="selectedExampleId"
            :options="exampleOptions"
            filterable
            :virtual-scroll="false"
            placeholder="搜索或选择示例"
            @update:value="updateSelectedExample"
          />
          <NButton
            data-testid="load-demo"
            size="small"
            :loading="pendingAction === 'demo'"
            :disabled="!selectedExampleId || (isBusy && pendingAction !== 'demo')"
            @click="loadExampleWorkspace"
          >
            加载示例
          </NButton>
        </NInputGroup>
        <p class="settings-panel__hint">示例会作为只读拓扑图标签页打开，修改后可在拓扑图工具栏另存为文件。</p>
      </NFormItem>
      <NAlert v-if="workspaceFile.lastError" type="error" closable @close="workspaceFile.setError(null)">
        {{ workspaceFile.lastError }}
      </NAlert>
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
.settings-panel__demo-select {
  flex: 1 1 auto;
  min-width: 0;
}
.settings-panel__hint {
  margin: 8px 0 0;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
</style>
