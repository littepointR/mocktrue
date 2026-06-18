<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NModal, NCard, NForm, NFormItem, NSelect, NButton, NSpace, NAlert } from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}>()

const store = useSerialStore()

const selectedPort = ref('')
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
    await store.openPort(selectedPort.value, baudRate.value, {
      dataBits: dataBits.value,
      stopBits: stopBits.value,
      parity: parity.value,
      flowMode: flowMode.value,
    })
    emit('update:show', false)
    emit('success')
  } catch (e) {
    console.error('Failed to open port:', e)
  } finally {
    loading.value = false
  }
}

function handleCancel() {
  emit('update:show', false)
}
</script>

<template>
  <NModal :show="show" @update:show="emit('update:show', $event)">
    <NCard
      style="width: 600px"
      title="打开串口"
      :bordered="false"
      size="small"
      role="dialog"
      aria-modal="true"
    >
      <NAlert v-if="store.error" type="error" closable @close="store.clearError()" style="margin-bottom: 16px">
        {{ store.error }}
      </NAlert>

      <NForm label-placement="left" label-width="100">
        <NFormItem label="端口">
          <NSelect
            v-model:value="selectedPort"
            :options="portOptions"
            placeholder="选择串口"
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
      </NForm>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="handleCancel" :disabled="loading">取消</NButton>
          <NButton
            type="primary"
            @click="handleOpen"
            :loading="loading"
            :disabled="!selectedPort"
          >
            打开
          </NButton>
        </NSpace>
      </template>
    </NCard>
  </NModal>
</template>

<style scoped>
</style>
