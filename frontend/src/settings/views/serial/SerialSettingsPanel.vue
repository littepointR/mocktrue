<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInputNumber, NSelect } from 'naive-ui'
import { useSettingsStore } from '../../stores/settingsStore'
import { ListSystemFonts } from '../../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'
import { serialTerminalFontFamily } from '../../../serial/utils/terminalStyle'

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
const fallbackFontFamilies = ['Consolas', 'Monaco', 'Courier New', 'Menlo', 'SF Mono', 'monospace']
const systemFontFamilies = ref<string[]>([])
const fontOptions = computed(() => {
  const families = uniqueFontFamilies([
    ...fallbackFontFamilies,
    ...systemFontFamilies.value,
    store.serial.TerminalFontFamily,
  ])
  return families.map(value => ({ label: value, value }))
})
const encodingOptions = [
  { label: 'ASCII', value: 'ascii' },
  { label: 'UTF-8', value: 'utf-8' },
  { label: 'UTF-16LE', value: 'utf-16le' },
  { label: 'UTF-16BE', value: 'utf-16be' },
  { label: 'GB2312', value: 'gb2312' },
  { label: 'GBK', value: 'gbk' },
  { label: 'Big5', value: 'big5' },
  { label: 'Shift_JIS', value: 'shift_jis' },
  { label: 'Windows-1251', value: 'windows-1251' },
  { label: 'Windows-1252', value: 'windows-1252' },
]
const enterStringOptions = [
  { label: 'CR (\\r)', value: '\r' },
  { label: 'LF (\\n)', value: '\n' },
  { label: 'CRLF (\\r\\n)', value: '\r\n' },
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
const terminalFontFamily = computed({
  get: () => store.serial.TerminalFontFamily,
  set: value => store.updateSerial({ TerminalFontFamily: value }),
})
const terminalFontSize = computed({
  get: () => store.serial.TerminalFontSize,
  set: value => store.updateSerial({ TerminalFontSize: value ?? 14 }),
})
const textEncoding = computed({
  get: () => store.serial.TextEncoding,
  set: value => store.updateSerial({ TextEncoding: value }),
})
const enterString = computed({
  get: () => store.serial.EnterString,
  set: value => store.updateSerial({ EnterString: value }),
})

onMounted(async () => {
  try {
    systemFontFamilies.value = await ListSystemFonts() ?? []
  } catch {
    systemFontFamilies.value = []
  }
})

function uniqueFontFamilies(values: string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const value of values) {
    const trimmed = value.trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    result.push(trimmed)
  }
  return result
}

function renderFontLabel(option: { label?: string | number; value?: string | number }) {
  const label = String(option.label ?? option.value ?? '')
  const family = String(option.value ?? option.label ?? '')
  return h('span', { style: { fontFamily: serialTerminalFontFamily(family) } }, label)
}
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
      <NFormItem label="收发区域字体">
        <NSelect v-model:value="terminalFontFamily" :options="fontOptions" :render-label="renderFontLabel" filterable />
      </NFormItem>
      <NFormItem label="收发区域字号">
        <NInputNumber v-model:value="terminalFontSize" :min="10" :max="24" />
      </NFormItem>
      <NFormItem label="文本编码">
        <NSelect v-model:value="textEncoding" :options="encodingOptions" />
      </NFormItem>
      <NFormItem label="回车字符串">
        <NSelect v-model:value="enterString" :options="enterStringOptions" />
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
