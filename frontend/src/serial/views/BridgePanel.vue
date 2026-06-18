<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NCard, NList, NListItem, NThing, NButton, NSpace, NEmpty,
  NForm, NFormItem, NSelect, NInputNumber, NInput, NDivider, NAlert, NPopconfirm, NTag,
} from 'naive-ui'
import { useVirtualStore } from '../stores/virtualStore'
import { useSerialStore } from '../stores/serialStore'

const virtualStore = useVirtualStore()
const serialStore = useSerialStore()

const newBridgeId = ref('')
const port1 = ref('')
const port2 = ref('')
const baudRate = ref(115200)
const loading = ref(false)

onMounted(() => {
  virtualStore.refreshBridges()
  virtualStore.refreshPairs()
  serialStore.refreshPorts()
})

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600].map(v => ({
  label: String(v),
  value: v,
}))

// 可用端口：物理串口 + 虚拟串口对中的所有端口
const portOptions = computed(() => {
  const physicalPorts = serialStore.ports.map(p => ({
    label: p.FriendlyName || p.Name,
    value: p.Name,
  }))

  const virtualPorts: Array<{ label: string; value: string }> = []
  virtualStore.pairs.forEach(pair => {
    virtualPorts.push({ label: `${pair.Port1} (虚拟)`, value: pair.Port1 })
    virtualPorts.push({ label: `${pair.Port2} (虚拟)`, value: pair.Port2 })
  })

  return [...physicalPorts, ...virtualPorts]
})

async function handleCreateBridge() {
  if (!newBridgeId.value || !port1.value || !port2.value) return
  if (port1.value === port2.value) {
    virtualStore.error = '不能将端口桥接到自己'
    return
  }

  loading.value = true
  try {
    await virtualStore.createBridge(newBridgeId.value, port1.value, port2.value, baudRate.value)
    // Reset form
    newBridgeId.value = ''
    port1.value = ''
    port2.value = ''
  } finally {
    loading.value = false
  }
}

async function handleDeleteBridge(id: string) {
  await virtualStore.deleteBridge(id)
}

function generateDefaultId() {
  const timestamp = Date.now().toString(36).slice(-4)
  newBridgeId.value = `bridge-${timestamp}`
}

async function handleRefresh() {
  await Promise.all([
    virtualStore.refreshBridges(),
    virtualStore.refreshPairs(),
    serialStore.refreshPorts(),
  ])
}
</script>

<template>
  <div class="bridge-panel">
    <div class="panel-header">
      <h3>串口桥接</h3>
      <NButton size="small" @click="handleRefresh">刷新</NButton>
    </div>

    <NAlert v-if="virtualStore.error" type="error" closable @close="virtualStore.clearError()" style="margin: 8px 0">
      {{ virtualStore.error }}
    </NAlert>

    <NCard size="small" title="新建桥接" style="margin-bottom: 12px">
      <NForm size="small" label-placement="left" label-width="60">
        <NFormItem label="ID">
          <div style="display: flex; gap: 8px; width: 100%;">
            <NInput
              v-model:value="newBridgeId"
              placeholder="例: bridge-1"
              :disabled="loading"
              style="flex: 1"
            />
            <NButton size="small" @click="generateDefaultId">自动</NButton>
          </div>
        </NFormItem>
        <NFormItem label="端口1">
          <NSelect
            v-model:value="port1"
            :options="portOptions"
            placeholder="选择端口1"
            :disabled="loading"
          />
        </NFormItem>
        <NFormItem label="端口2">
          <NSelect
            v-model:value="port2"
            :options="portOptions"
            placeholder="选择端口2"
            :disabled="loading"
          />
        </NFormItem>
        <NFormItem label="波特率">
          <NSelect
            v-model:value="baudRate"
            :options="baudOptions"
            :disabled="loading"
          />
        </NFormItem>
        <NButton
          type="primary"
          size="small"
          block
          :loading="loading"
          :disabled="!newBridgeId || !port1 || !port2 || port1 === port2"
          @click="handleCreateBridge"
        >
          创建桥接
        </NButton>
      </NForm>
    </NCard>

    <NDivider style="margin: 12px 0" />

    <div class="bridge-list-header">
      <span>已有桥接 ({{ virtualStore.bridges.length }})</span>
    </div>

    <NEmpty v-if="virtualStore.bridges.length === 0" description="暂无桥接" size="small" />

    <NList v-else size="small" hoverable>
      <NListItem v-for="bridge in virtualStore.bridges" :key="bridge.ID">
        <NThing>
          <template #header>
            <NSpace>
              <NTag size="small" type="success">{{ bridge.ID }}</NTag>
              <NTag size="small">{{ bridge.BaudRate }} bps</NTag>
            </NSpace>
          </template>
          <template #description>
            <div class="port-list">
              <code>{{ bridge.Port1 }}</code>
              <span>⇌</span>
              <code>{{ bridge.Port2 }}</code>
            </div>
          </template>
          <template #action>
            <NPopconfirm @positive-click="handleDeleteBridge(bridge.ID)">
              <template #trigger>
                <NButton size="tiny" type="error" tertiary>断开</NButton>
              </template>
              确定断开桥接 {{ bridge.ID }} 吗？
            </NPopconfirm>
          </template>
        </NThing>
      </NListItem>
    </NList>
  </div>
</template>

<style scoped>
.bridge-panel {
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
.bridge-list-header {
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
</style>
