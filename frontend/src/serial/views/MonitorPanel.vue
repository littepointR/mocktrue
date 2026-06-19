<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import { useMonitorStore } from '../stores/monitorStore'
import { useSettingsStore } from '../../settings/stores/settingsStore'

const emit = defineEmits<{
  started: [id: string]
}>()

const serialStore = useSerialStore()
const monitorStore = useMonitorStore()
const settings = useSettingsStore()

const id = ref(defaultMonitorId())
const name = ref('')
const portA = ref('')
const baudRate = ref(115200)
const dataBits = ref(8)
const stopBits = ref('1')
const parity = ref('none')
const flowMode = ref('none')
const loading = ref(false)

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 4000000].map(v => ({
  label: String(v),
  value: v,
}))
const dataBitsOptions = [5, 6, 7, 8].map(v => ({ label: String(v), value: v }))
const stopBitsOptions = [
  { label: '1', value: '1' },
  { label: '1.5', value: '1.5' },
  { label: '2', value: '2' },
]
const parityOptions = [
  { label: '无', value: 'none' },
  { label: '偶', value: 'even' },
  { label: '奇', value: 'odd' },
  { label: 'Mark', value: 'mark' },
  { label: 'Space', value: 'space' },
]
const flowOptions = [
  { label: '无', value: 'none' },
  { label: '硬件 (RTS/CTS)', value: 'hw_rtscts' },
  { label: '软件 (XON/XOFF)', value: 'sw_xonxoff' },
]

const portOptions = computed(() =>
  serialStore.ports.map(p => ({
    label: p.FriendlyName || p.Name,
    value: p.Name,
  }))
)

const canStart = computed(() => Boolean(id.value && portA.value))

onMounted(() => {
  serialStore.refreshPorts()
})

async function startMonitor() {
  if (!canStart.value) return
  loading.value = true
  try {
    const monitorId = await monitorStore.startAutoVirtualMonitor({
      id: id.value,
      name: name.value || `${portA.value} monitor`,
      sourcePort: portA.value,
      baudRate: baudRate.value,
      dataBits: dataBits.value,
      stopBits: stopBits.value,
      parity: parity.value,
      flowMode: flowMode.value,
      encoding: settings.serial.TextEncoding,
    })
    emit('started', monitorId)
    id.value = defaultMonitorId()
    name.value = ''
  } finally {
    loading.value = false
  }
}

function defaultMonitorId(): string {
  return `mon-${Date.now().toString(36)}`
}
</script>

<template>
  <div class="monitor-panel">
    <div class="monitor-panel__header">
      <strong>串口监控</strong>
      <NButton size="tiny" secondary @click="serialStore.refreshPorts">刷新</NButton>
    </div>
    <NAlert v-if="monitorStore.error" type="error" closable @close="monitorStore.clearError()">
      {{ monitorStore.error }}
    </NAlert>
    <NForm class="monitor-panel__form" label-placement="top" size="small">
      <NFormItem label="会话 ID">
        <NInput v-model:value="id" :disabled="loading" />
      </NFormItem>
      <NFormItem label="名称">
        <NInput v-model:value="name" :disabled="loading" placeholder="可选" />
      </NFormItem>
      <NFormItem label="被监听端口">
        <NSelect
          v-model:value="portA"
          :options="portOptions"
          :disabled="loading"
          filterable
        />
      </NFormItem>
      <div class="monitor-panel__grid">
        <NFormItem label="波特率">
          <NSelect v-model:value="baudRate" :options="baudOptions" :disabled="loading" />
        </NFormItem>
        <NFormItem label="数据位">
          <NSelect v-model:value="dataBits" :options="dataBitsOptions" :disabled="loading" />
        </NFormItem>
        <NFormItem label="停止位">
          <NSelect v-model:value="stopBits" :options="stopBitsOptions" :disabled="loading" />
        </NFormItem>
        <NFormItem label="校验">
          <NSelect v-model:value="parity" :options="parityOptions" :disabled="loading" />
        </NFormItem>
      </div>
      <NFormItem label="流控">
        <NSelect v-model:value="flowMode" :options="flowOptions" :disabled="loading" />
      </NFormItem>
      <NSpace justify="end">
        <NButton
          type="primary"
          :loading="loading"
          :disabled="!canStart"
          @click="startMonitor"
        >
          开始监控
        </NButton>
      </NSpace>
    </NForm>
  </div>
</template>

<style scoped>
.monitor-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  padding: 12px;
  overflow: auto;
  color: #d4d4d4;
}
.monitor-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.monitor-panel__form {
  min-width: 0;
}
.monitor-panel__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 8px;
}
</style>
