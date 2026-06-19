<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NForm, NFormItem, NInputNumber, NSelect } from 'naive-ui'
import { useSettingsStore } from '../../stores/settingsStore'

const store = useSettingsStore()

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 4000000].map(value => ({
  label: String(value),
  value,
}))
const dataBitsOptions = [5, 6, 7, 8].map(value => ({ label: String(value), value }))
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

const baudRate = computed({
  get: () => store.serial.BaudRate,
  set: value => store.updateSerial({ BaudRate: value }),
})
const dataBits = computed({
  get: () => store.serial.DataBits,
  set: value => store.updateSerial({ DataBits: value }),
})
const stopBits = computed({
  get: () => store.serial.StopBits,
  set: value => store.updateSerial({ StopBits: value }),
})
const parity = computed({
  get: () => store.serial.Parity,
  set: value => store.updateSerial({ Parity: value }),
})
const flowMode = computed({
  get: () => store.serial.FlowMode,
  set: value => store.updateSerial({ FlowMode: value }),
})
const readBufKB = computed({
  get: () => store.serial.ReadBufKB,
  set: value => store.updateSerial({ ReadBufKB: value ?? 32 }),
})
</script>

<template>
  <section class="settings-panel">
    <header class="settings-panel__header">
      <h2>串口</h2>
      <NButton size="small" secondary @click="store.resetSerial()">恢复默认</NButton>
    </header>

    <NForm class="settings-panel__form" label-placement="top" size="small">
      <NFormItem label="默认波特率">
        <NSelect v-model:value="baudRate" :options="baudOptions" />
      </NFormItem>
      <NFormItem label="默认数据位">
        <NSelect v-model:value="dataBits" :options="dataBitsOptions" />
      </NFormItem>
      <NFormItem label="默认停止位">
        <NSelect v-model:value="stopBits" :options="stopBitsOptions" />
      </NFormItem>
      <NFormItem label="默认校验">
        <NSelect v-model:value="parity" :options="parityOptions" />
      </NFormItem>
      <NFormItem label="默认流控">
        <NSelect v-model:value="flowMode" :options="flowOptions" />
      </NFormItem>
      <NFormItem label="接收缓冲区 KB">
        <NInputNumber v-model:value="readBufKB" :min="4" :max="262144" />
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
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 4px 18px;
}
</style>
