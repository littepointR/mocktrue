<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NForm, NFormItem, NSelect, NButton, NAlert } from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import { useSettingsStore } from '../../settings/stores/settingsStore'

const store = useSerialStore()
const settingsStore = useSettingsStore()
const emit = defineEmits<{
  opened: [handleId: string]
}>()

const selectedPort = ref('')
const baudRate = ref(settingsStore.serial.BaudRate)
const dataBits = ref(settingsStore.serial.DataBits)
const stopBits = ref(settingsStore.serial.StopBits)
const parity = ref(settingsStore.serial.Parity)
const flowMode = ref(settingsStore.serial.FlowMode)
const loading = ref(false)

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 4000000].map(v => ({
  label: String(v),
  value: v,
}))

const dataBitsOptions = [5, 6, 7, 8].map(v => ({
  label: String(v),
  value: v,
}))

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
  store.ports.map(p => ({
    label: p.FriendlyName || p.Name,
    value: p.Name,
  }))
)

onMounted(() => {
  store.refreshPorts()
})

async function handleOpen() {
  if (!selectedPort.value) return

  loading.value = true
  try {
    const handleId = await store.openPort(selectedPort.value, baudRate.value, {
      dataBits: dataBits.value,
      stopBits: stopBits.value,
      parity: parity.value,
      flowMode: flowMode.value,
      readBufKB: settingsStore.serial.ReadBufKB,
    })
    // 重置表单
    selectedPort.value = ''
    emit('opened', handleId)
  } catch (e) {
    console.error('Failed to open port:', e)
  } finally {
    loading.value = false
  }
}

async function handleRefresh() {
  loading.value = true
  try {
    await store.refreshPorts()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="port-config-form">
    <NAlert v-if="store.error" type="error" closable @close="store.clearError()" style="margin: 12px">
      {{ store.error }}
    </NAlert>

    <div class="port-config-form__content">
      <NForm label-placement="top" size="small">
        <NFormItem label="端口">
          <div style="display: flex; gap: 8px; width: 100%;">
            <NSelect
              v-model:value="selectedPort"
              :options="portOptions"
              placeholder="选择串口"
              :disabled="loading"
              style="flex: 1;"
            />
            <NButton @click="handleRefresh" :loading="loading" size="small">
              刷新
            </NButton>
          </div>
        </NFormItem>

        <NFormItem label="波特率">
          <NSelect
            v-model:value="baudRate"
            :options="baudOptions"
            :disabled="loading"
          />
        </NFormItem>

        <NFormItem label="数据位">
          <NSelect
            v-model:value="dataBits"
            :options="dataBitsOptions"
            :disabled="loading"
          />
        </NFormItem>

        <NFormItem label="停止位">
          <NSelect
            v-model:value="stopBits"
            :options="stopBitsOptions"
            :disabled="loading"
          />
        </NFormItem>

        <NFormItem label="校验">
          <NSelect
            v-model:value="parity"
            :options="parityOptions"
            :disabled="loading"
          />
        </NFormItem>

        <NFormItem label="流控">
          <NSelect
            v-model:value="flowMode"
            :options="flowOptions"
            :disabled="loading"
          />
        </NFormItem>

        <NButton
          type="primary"
          block
          @click="handleOpen"
          :loading="loading"
          :disabled="!selectedPort"
        >
          打开串口
        </NButton>
      </NForm>
    </div>
  </div>
</template>

<style scoped>
.port-config-form {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.port-config-form__content {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}
</style>
