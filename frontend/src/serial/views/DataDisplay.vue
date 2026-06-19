<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { NSelect, NSpace, NSwitch } from 'naive-ui'
import { useBufferStore } from '../stores/bufferStore'
import { useSettingsStore } from '../../settings/stores/settingsStore'
import { decodeSerialText } from '../utils/textEncoding'
import { serialTerminalStyle } from '../utils/terminalStyle'
import { useSerialWorkspaceStore, type ReceiveLayoutMode, type ReceiveViewMode } from '../stores/workspaceStore'

const props = defineProps<{
  handleId: string
}>()

type ViewMode = 'ascii' | 'hexClassic' | 'hexTable'
type LayoutMode = 'combined' | 'split'

const bufferStore = useBufferStore()
const settingsStore = useSettingsStore()
const workspaceStore = useSerialWorkspaceStore()
const contentEl = ref<HTMLElement | null>(null)

const viewMode = computed({
  get: () => workspaceStore.tabState(props.handleId).dataDisplay.viewMode,
  set: value => workspaceStore.updateTabState(props.handleId, { dataDisplay: { viewMode: value as ReceiveViewMode } }),
})
const layoutMode = computed({
  get: () => workspaceStore.tabState(props.handleId).dataDisplay.layoutMode,
  set: value => workspaceStore.updateTabState(props.handleId, { dataDisplay: { layoutMode: value as ReceiveLayoutMode } }),
})
const showTimestamp = computed({
  get: () => workspaceStore.tabState(props.handleId).dataDisplay.showTimestamp,
  set: value => workspaceStore.updateTabState(props.handleId, { dataDisplay: { showTimestamp: value } }),
})
const autoScroll = computed({
  get: () => workspaceStore.tabState(props.handleId).dataDisplay.autoScroll,
  set: value => workspaceStore.updateTabState(props.handleId, { dataDisplay: { autoScroll: value } }),
})

const buffer = computed(() => bufferStore.getBuffer(props.handleId))
const chunks = computed(() => bufferStore.getChunks(props.handleId))

const displayText = computed(() => {
  return decodeSerialText(buffer.value, settingsStore.serial.TextEncoding)
})

const terminalStyle = computed(() => ({
  ...serialTerminalStyle(
    settingsStore.serial.TerminalFontFamily,
    settingsStore.serial.TerminalFontSize,
  ),
}))

const asciiLines = computed(() => {
  if (!showTimestamp.value) {
    return [{ key: 'combined', text: displayText.value, timestamp: '' }]
  }
  return chunks.value.map((chunk, index) => ({
    key: `${chunk.timestamp}-${index}`,
    text: decodeSerialText(chunk.data, settingsStore.serial.TextEncoding),
    timestamp: formatTimestamp(chunk.timestamp),
  }))
})

function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp)
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  const ss = String(date.getSeconds()).padStart(2, '0')
  const ms = String(date.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

function formatHex(data: Uint8Array): string {
  return Array.from(data).map(b => b.toString(16).padStart(2, '0')).join(' ')
}

function formatAscii(data: Uint8Array): string {
  return Array.from(data).map(b => (b >= 32 && b <= 126) ? String.fromCharCode(b) : '.').join('')
}

const hexLines = computed(() => {
  const result: Array<{ offset: number; hex: string; ascii: string }> = []
  const arr = buffer.value
  for (let i = 0; i < arr.length; i += 16) {
    const chunk = arr.slice(i, i + 16)
    result.push({
      offset: i,
      hex: formatHex(chunk),
      ascii: formatAscii(chunk),
    })
  }
  return result
})

watch(
  () => buffer.value.length,
  async () => {
    if (!autoScroll.value) return
    await nextTick()
    const el = contentEl.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  }
)
</script>

<template>
  <div class="data-display" :style="terminalStyle">
    <div class="data-display__controls">
      <NSpace align="center" size="small">
        <NSelect
          v-model:value="viewMode"
          :options="[
            { label: 'ASCII', value: 'ascii' },
            { label: 'HEX 经典', value: 'hexClassic' },
            { label: 'HEX 表格', value: 'hexTable' },
          ]"
          size="tiny"
          style="width: 100px"
        />
        <NSelect
          v-model:value="layoutMode"
          :options="[
            { label: '合并', value: 'combined' },
            { label: '分栏', value: 'split' },
          ]"
          size="tiny"
          style="width: 80px"
        />
        <span class="data-display__switch-control data-display__timestamp-control">
          <span>时间戳</span>
          <NSwitch v-model:value="showTimestamp" size="small" />
        </span>
        <span class="data-display__switch-control data-display__autoscroll-control">
          <span>自动滚动</span>
          <NSwitch v-model:value="autoScroll" size="small" />
        </span>
      </NSpace>
    </div>

    <div
      ref="contentEl"
      class="data-display__content"
      :class="{ 'split': layoutMode === 'split' }"
    >
      <!-- ASCII View -->
      <div v-if="viewMode === 'ascii'" class="data-view">
        <div class="ascii-content">
          <div
            v-for="line in asciiLines"
            :key="line.key"
            class="ascii-line"
          >
            <span v-if="showTimestamp" class="ascii-timestamp">{{ line.timestamp }}</span>
            <span class="ascii-text">{{ line.text }}</span>
          </div>
        </div>
      </div>

      <!-- HEX Classic View -->
      <div v-if="viewMode === 'hexClassic'" class="data-view hex-classic">
        <div v-for="(line, idx) in hexLines" :key="idx" class="hex-row">
          <span class="hex-addr">{{ line.offset.toString(16).padStart(8, '0') }}</span>
          <span class="hex-data">{{ line.hex }}</span>
          <span class="hex-ascii">{{ line.ascii }}</span>
        </div>
      </div>

      <!-- HEX Table View placeholder -->
      <div v-if="viewMode === 'hexTable'" class="data-view">
        <div class="placeholder">HEX 表格视图（待集成 HexTableView）</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.data-display {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
}
.data-display__controls {
  padding: 4px 8px;
  border-bottom: 1px solid #2d2d2d;
  background: #252526;
}
.data-display__switch-control {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #858585;
  font-size: 12px;
}
.data-display__content {
  flex: 1;
  overflow-y: auto;
  font-family: var(--serial-terminal-font-family);
  font-size: var(--serial-terminal-font-size);
}
.data-display__content.split {
  display: flex;
}
.data-view {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
}
.ascii-content {
  margin: 0;
  color: #4ec9b0;
  font-family: var(--serial-terminal-font-family);
  font-size: var(--serial-terminal-font-size);
  white-space: pre-wrap;
  word-wrap: break-word;
}
.ascii-line {
  min-height: 18px;
}
.ascii-timestamp {
  display: inline-block;
  margin-right: 8px;
  color: #858585;
  user-select: none;
}
.ascii-text {
  white-space: pre-wrap;
  word-break: break-word;
}
.hex-row {
  padding: 2px 0;
  color: #4ec9b0;
  font-family: var(--serial-terminal-font-family);
  font-size: var(--serial-terminal-font-size);
}
.hex-addr {
  color: #858585;
  margin-right: 16px;
}
.hex-data {
  margin-right: 16px;
  white-space: pre;
}
.hex-ascii {
  white-space: pre;
}
.placeholder {
  color: #858585;
  padding: 20px;
  text-align: center;
}
</style>
