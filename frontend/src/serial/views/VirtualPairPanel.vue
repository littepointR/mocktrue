<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NCard, NList, NListItem, NThing, NButton, NSpace, NEmpty,
  NForm, NFormItem, NInput, NDivider, NAlert, NPopconfirm, NTag,
} from 'naive-ui'
import { useVirtualStore } from '../stores/virtualStore'

const store = useVirtualStore()

const newPairId = ref('')
const newPort1Name = ref('')
const newPort2Name = ref('')
const loading = ref(false)

onMounted(() => {
  store.refreshPairs()
})

async function handleCreatePair() {
  if (!newPairId.value || !newPort1Name.value || !newPort2Name.value) return

  loading.value = true
  try {
    await store.createPair(newPairId.value, newPort1Name.value, newPort2Name.value)
    // Reset form
    newPairId.value = ''
    newPort1Name.value = ''
    newPort2Name.value = ''
  } finally {
    loading.value = false
  }
}

async function handleDeletePair(id: string) {
  await store.deletePair(id)
}

function generateDefaultId() {
  const timestamp = Date.now().toString(36).slice(-4)
  newPairId.value = `pair-${timestamp}`
  newPort1Name.value = `tty${timestamp}A`
  newPort2Name.value = `tty${timestamp}B`
}
</script>

<template>
  <div class="virtual-pair-panel">
    <div class="panel-header">
      <h3>虚拟串口对</h3>
      <NButton size="small" @click="store.refreshPairs()">刷新</NButton>
    </div>

    <NAlert v-if="store.error" type="error" closable @close="store.clearError()" style="margin: 8px 0">
      {{ store.error }}
    </NAlert>

    <NCard size="small" title="新建虚拟串口对" style="margin-bottom: 12px">
      <NForm size="small" label-placement="left" label-width="60">
        <NFormItem label="ID">
          <NInput v-model:value="newPairId" placeholder="例: pair-1" :disabled="loading" />
        </NFormItem>
        <NFormItem label="端口1">
          <NInput v-model:value="newPort1Name" placeholder="例: ttyA0" :disabled="loading" />
        </NFormItem>
        <NFormItem label="端口2">
          <NInput v-model:value="newPort2Name" placeholder="例: ttyA1" :disabled="loading" />
        </NFormItem>
        <NSpace>
          <NButton size="small" @click="generateDefaultId">自动生成</NButton>
          <NButton
            type="primary"
            size="small"
            :loading="loading"
            :disabled="!newPairId || !newPort1Name || !newPort2Name"
            @click="handleCreatePair"
          >
            创建
          </NButton>
        </NSpace>
      </NForm>
    </NCard>

    <NDivider style="margin: 12px 0" />

    <div class="pair-list-header">
      <span>已有虚拟串口对 ({{ store.pairs.length }})</span>
    </div>

    <NEmpty v-if="store.pairs.length === 0" description="暂无虚拟串口对" size="small" />

    <NList v-else size="small" hoverable>
      <NListItem v-for="pair in store.pairs" :key="pair.ID">
        <NThing>
          <template #header>
            <NTag size="small" type="info">{{ pair.ID }}</NTag>
          </template>
          <template #description>
            <div class="port-list">
              <code>{{ pair.Port1 }}</code>
              <span>↔</span>
              <code>{{ pair.Port2 }}</code>
            </div>
          </template>
          <template #action>
            <NPopconfirm @positive-click="handleDeletePair(pair.ID)">
              <template #trigger>
                <NButton size="tiny" type="error" tertiary>删除</NButton>
              </template>
              确定删除虚拟串口对 {{ pair.ID }} 吗？
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
</style>
