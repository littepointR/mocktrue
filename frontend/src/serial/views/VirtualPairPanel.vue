<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NCard, NList, NListItem, NThing, NButton, NSpace, NEmpty,
  NForm, NFormItem, NInput, NDivider, NAlert, NPopconfirm, NTag,
} from 'naive-ui'
import { useVirtualStore } from '../stores/virtualStore'

const store = useVirtualStore()

const newPortId = ref('')
const newPortName = ref('')
const loading = ref(false)

onMounted(() => {
  store.refreshBackendStatus()
  store.refreshVirtualPorts()
})

async function handleCreateVirtualPort() {
  if (!newPortId.value || !newPortName.value) return
  if (store.backendStatus && !store.backendStatus.Available) return

  loading.value = true
  try {
    await store.createVirtualPort(newPortId.value, newPortName.value)
    newPortId.value = ''
    newPortName.value = ''
  } finally {
    loading.value = false
  }
}

async function handleDeleteVirtualPort(id: string) {
  await store.deleteVirtualPort(id)
}

function generateDefaultId() {
  const timestamp = Date.now().toString(36).slice(-4)
  newPortId.value = `vport-${timestamp}`
  newPortName.value = `tty${timestamp}`
}

function refreshPanel() {
  store.refreshBackendStatus()
  store.refreshVirtualPorts()
}
</script>

<template>
  <div class="virtual-pair-panel">
    <div class="panel-header">
      <h3>虚拟串口</h3>
      <NButton size="small" @click="refreshPanel">刷新</NButton>
    </div>

    <NAlert
      v-if="store.backendStatus"
      :type="store.backendStatus.Available ? 'success' : 'warning'"
      style="margin: 8px 0"
    >
      <div class="backend-status">
        <span>{{ store.backendStatus.Name }}</span>
        <span>{{ store.backendStatus.Message }}</span>
        <span v-if="store.backendStatus.RequiresAdmin">需要管理员权限</span>
      </div>
      <div v-if="store.backendStatus.Reason" class="backend-reason">
        {{ store.backendStatus.Reason }}
      </div>
    </NAlert>

    <NAlert v-if="store.error" type="error" closable @close="store.clearError()" style="margin: 8px 0">
      {{ store.error }}
    </NAlert>

    <NCard size="small" title="新建虚拟串口" style="margin-bottom: 12px">
      <NForm size="small" label-placement="left" label-width="60">
        <NFormItem label="ID">
          <NInput v-model:value="newPortId" placeholder="例: vport-1" :disabled="loading" />
        </NFormItem>
        <NFormItem label="端口">
          <NInput v-model:value="newPortName" placeholder="例: ttyV0" :disabled="loading" />
        </NFormItem>
        <NSpace>
          <NButton size="small" @click="generateDefaultId">自动生成</NButton>
          <NButton
            type="primary"
            size="small"
            :loading="loading"
            :disabled="!newPortId || !newPortName || Boolean(store.backendStatus && !store.backendStatus.Available)"
            @click="handleCreateVirtualPort"
          >
            创建
          </NButton>
        </NSpace>
      </NForm>
    </NCard>

    <NDivider style="margin: 12px 0" />

    <div class="pair-list-header">
      <span>已有虚拟串口 ({{ store.virtualPorts.length }})</span>
    </div>

    <NEmpty v-if="store.virtualPorts.length === 0" description="暂无虚拟串口" size="small" />

    <NList v-else size="small" hoverable>
      <NListItem v-for="vport in store.virtualPorts" :key="vport.ID">
        <NThing>
          <template #header>
            <NTag size="small" type="info">{{ vport.ID }}</NTag>
          </template>
          <template #description>
            <div class="port-list">
              <code>{{ vport.Port }}</code>
            </div>
          </template>
          <template #action>
            <NPopconfirm @positive-click="handleDeleteVirtualPort(vport.ID)">
              <template #trigger>
                <NButton size="tiny" type="error" tertiary>删除</NButton>
              </template>
              确定删除虚拟串口 {{ vport.ID }} 吗？
            </NPopconfirm>
          </template>
        </NThing>
      </NListItem>
    </NList>
  </div>
</template>

<style scoped>
.virtual-pair-panel {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #d4d4d4;
}
.pair-list-header {
  font-size: 12px;
  color: #858585;
  margin-bottom: 8px;
}
.port-list {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: #d4d4d4;
}
.port-list code {
  background: #1e1e1e;
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid #2d2d2d;
}
.backend-status {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
}
.backend-reason {
  margin-top: 4px;
  font-size: 11px;
  color: #858585;
  word-break: break-word;
}
</style>
