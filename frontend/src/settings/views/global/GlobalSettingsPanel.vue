<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NInputGroup, NSelect, NSpace } from 'naive-ui'
import { useSettingsStore } from '../../stores/settingsStore'
import { useWorkspaceFileStore } from '../../../workspace/stores/workspaceFileStore'

const store = useSettingsStore()
const workspaceFile = useWorkspaceFileStore()

const themeOptions = [
  { label: '深色', value: 'dark' },
  { label: '浅色', value: 'light' },
  { label: '跟随系统', value: 'system' },
]

const theme = computed({
  get: () => store.global.Theme,
  set: value => store.updateGlobal({ Theme: value }),
})
const workspacePath = computed({
  get: () => workspaceFile.currentPath,
  set: value => workspaceFile.setPath(value),
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

async function selectWorkspacePath() {
  await runWorkspaceAction('select', () => workspaceFile.selectOpenPath())
}

async function saveWorkspace() {
  await runWorkspaceAction('save', () => workspaceFile.save())
}

async function saveWorkspaceAs() {
  await runWorkspaceAction('saveAs', () => workspaceFile.saveAs())
}

async function importWorkspace() {
  await runWorkspaceAction('import', () => workspaceFile.importSelected())
}

async function exportWorkspaceCopy() {
  await runWorkspaceAction('export', () => workspaceFile.exportCopy())
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
      <NFormItem label="配置文件路径">
        <NInputGroup>
          <NInput v-model:value="workspacePath" placeholder="留空时保存到默认配置文件" />
          <NButton
            size="small"
            :loading="pendingAction === 'select'"
            :disabled="isBusy && pendingAction !== 'select'"
            @click="selectWorkspacePath"
          >
            选择
          </NButton>
        </NInputGroup>
      </NFormItem>
      <NFormItem label="配置文件">
        <NSpace>
          <NButton
            size="small"
            type="primary"
            :loading="pendingAction === 'save'"
            :disabled="isBusy && pendingAction !== 'save'"
            @click="saveWorkspace"
          >
            保存
          </NButton>
          <NButton
            size="small"
            :loading="pendingAction === 'saveAs'"
            :disabled="isBusy && pendingAction !== 'saveAs'"
            @click="saveWorkspaceAs"
          >
            另存为
          </NButton>
          <NButton
            size="small"
            :loading="pendingAction === 'import'"
            :disabled="isBusy && pendingAction !== 'import'"
            @click="importWorkspace"
          >
            导入
          </NButton>
          <NButton
            size="small"
            :loading="pendingAction === 'export'"
            :disabled="isBusy && pendingAction !== 'export'"
            @click="exportWorkspaceCopy"
          >
            导出副本
          </NButton>
        </NSpace>
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
</style>
