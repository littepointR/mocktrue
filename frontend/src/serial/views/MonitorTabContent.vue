<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NButton, NInput, NInputNumber, NSelect, NSwitch } from 'naive-ui'
import { useMonitorStore, defaultMonitorAutoSave, type MonitorDisplayMode } from '../stores/monitorStore'
import type { AutoSaveOptions, Frame } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/monitor/models.js'

const props = defineProps<{
  monitorId: string
}>()

const store = useMonitorStore()
const session = computed(() => store.sessions.get(props.monitorId) ?? null)
const frames = computed(() => store.framesByMonitor.get(props.monitorId) ?? [])
const total = computed(() => store.frameTotals.get(props.monitorId) ?? frames.value.length)
const filter = computed(() => store.filterFor(props.monitorId))
const selectedSeq = ref<number | null>(null)
const exportPath = ref('')
const exportFormat = ref('csv')
const autoSavePath = ref('')
const autoSaveFormat = ref('csv')
const autoSaveSplitMode = ref('none')
const autoSaveSizeKB = ref(1024)
const autoSaveSeconds = ref(60)
const savingAutoSave = ref(false)

const directionOptions = [
  { label: '全部方向', value: 'all' },
  { label: 'A → B', value: 'a_to_b' },
  { label: 'B → A', value: 'b_to_a' },
]
const displayOptions = [
  { label: '文本', value: 'text' },
  { label: 'HEX', value: 'hex' },
  { label: 'DEC', value: 'dec' },
  { label: 'OCT', value: 'oct' },
  { label: 'BIN', value: 'bin' },
]
const formatOptions = [
  { label: 'CSV', value: 'csv' },
  { label: 'TXT', value: 'txt' },
  { label: 'HTML', value: 'html' },
  { label: 'PCAPNG', value: 'pcapng' },
]
const splitOptions = [
  { label: '不分割', value: 'none' },
  { label: '按大小', value: 'size' },
  { label: '按时间', value: 'time' },
]

const selectedFrame = computed(() => {
  if (selectedSeq.value === null) return frames.value[0] ?? null
  return frames.value.find(frame => frame.Seq === selectedSeq.value) ?? frames.value[0] ?? null
})

onMounted(() => {
  if (session.value?.Status === 'running') {
    store.refreshFrames(props.monitorId)
  }
})

watch(
  () => props.monitorId,
  id => {
    selectedSeq.value = null
    if (session.value?.Status === 'running') {
      store.refreshFrames(id)
    }
  }
)

watch(
  session,
  value => {
    const options = value?.AutoSave ?? defaultMonitorAutoSave()
    autoSavePath.value = options.Path || options.Directory
    autoSaveFormat.value = options.Format || 'csv'
    autoSaveSplitMode.value = options.SplitMode || 'none'
    autoSaveSizeKB.value = options.SplitSizeKB || 1024
    autoSaveSeconds.value = options.SplitIntervalSeconds || 60
  },
  { immediate: true }
)

function setDirection(value: string) {
  store.refreshFrames(props.monitorId, { direction: value })
}

function setSearch(value: string) {
  store.refreshFrames(props.monitorId, { search: value })
}

function setDisplayMode(value: MonitorDisplayMode) {
  store.setFilter(props.monitorId, { displayMode: value })
}

function setModbusFunction(value: number | null) {
  store.refreshFrames(props.monitorId, { modbusFunction: value ?? 0 })
}

function frameDisplay(frame: Frame): string {
  switch (filter.value.displayMode) {
    case 'text':
      return frame.DisplayText
    case 'dec':
      return frame.DisplayDec
    case 'oct':
      return frame.DisplayOct
    case 'bin':
      return frame.DisplayBin
    default:
      return frame.DisplayHex
  }
}

function shortTime(value: unknown): string {
  if (!value) return ''
  const date = new Date(String(value))
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleTimeString(undefined, { hour12: false, fractionalSecondDigits: 3 })
}

async function stopMonitor() {
  await store.stopMonitor(props.monitorId)
}

async function clearFrames() {
  selectedSeq.value = null
  await store.clearFrames(props.monitorId)
}

async function exportCapture() {
  if (!exportPath.value) return
  await store.exportMonitor({
    MonitorID: props.monitorId,
    Format: exportFormat.value,
    Path: exportPath.value,
    Encoding: session.value?.Encoding ?? 'utf-8',
    Direction: filter.value.direction,
    Search: filter.value.search,
  })
}

async function toggleAutoSave(enabled: boolean) {
  savingAutoSave.value = true
  try {
    await store.setAutoSave(props.monitorId, autoSaveOptions(enabled))
  } finally {
    savingAutoSave.value = false
  }
}

async function applyAutoSave() {
  savingAutoSave.value = true
  try {
    await store.setAutoSave(props.monitorId, autoSaveOptions(session.value?.AutoSave.Enabled ?? false))
  } finally {
    savingAutoSave.value = false
  }
}

function autoSaveOptions(enabled: boolean): AutoSaveOptions {
  const path = autoSavePath.value
  return {
    Enabled: enabled,
    Path: autoSaveSplitMode.value === 'none' ? path : '',
    Directory: autoSaveSplitMode.value === 'none' ? '' : path,
    BaseName: 'serial-monitor',
    Format: autoSaveFormat.value,
    SplitMode: autoSaveSplitMode.value,
    SplitSizeKB: autoSaveSizeKB.value,
    SplitIntervalSeconds: autoSaveSeconds.value,
    Encoding: session.value?.Encoding ?? 'utf-8',
  }
}
</script>

<template>
  <div class="monitor-tab">
    <div class="monitor-tab__header">
      <div class="monitor-tab__title">
        <strong>{{ session?.Name ?? monitorId }}</strong>
        <span>{{ session?.PortA }} ⇄ {{ session?.PortB }}</span>
      </div>
      <div class="monitor-tab__stats">
        <span>{{ session?.Status ?? 'stopped' }}</span>
        <span>TX {{ session?.TxBytes ?? 0 }}</span>
        <span>RX {{ session?.RxBytes ?? 0 }}</span>
        <span>{{ total }} 帧</span>
      </div>
      <NButton size="small" secondary :disabled="session?.Status !== 'running'" @click="stopMonitor">停止</NButton>
      <NButton size="small" secondary @click="clearFrames">清空</NButton>
    </div>

    <div class="monitor-tab__toolbar">
      <NSelect
        class="monitor-tab__control"
        :value="filter.direction"
        :options="directionOptions"
        size="small"
        @update:value="setDirection"
      />
      <NInput
        class="monitor-tab__search"
        :value="filter.search"
        placeholder="搜索文本、HEX、端口或 Modbus"
        size="small"
        clearable
        @update:value="setSearch"
      />
      <NInputNumber
        class="monitor-tab__function"
        :value="filter.modbusFunction || null"
        placeholder="功能码"
        size="small"
        clearable
        :min="0"
        :max="255"
        @update:value="setModbusFunction"
      />
      <div class="monitor-tab__display">
        <button
          v-for="option in displayOptions"
          :key="option.value"
          :data-testid="`monitor-display-${option.value}`"
          type="button"
          :class="{ 'is-active': filter.displayMode === option.value }"
          @click="setDisplayMode(option.value as MonitorDisplayMode)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <div class="monitor-tab__export">
      <NSelect class="monitor-tab__format" v-model:value="exportFormat" :options="formatOptions" size="small" />
      <NInput v-model:value="exportPath" class="monitor-tab__path" placeholder="导出文件路径" size="small" />
      <NButton size="small" type="primary" :disabled="!exportPath" @click="exportCapture">导出</NButton>
      <span class="monitor-tab__autosave-label">自动保存</span>
      <NSwitch
        :value="session?.AutoSave.Enabled ?? false"
        :loading="savingAutoSave"
        @update:value="toggleAutoSave"
      />
      <NSelect class="monitor-tab__format" v-model:value="autoSaveFormat" :options="formatOptions" size="small" />
      <NSelect class="monitor-tab__format" v-model:value="autoSaveSplitMode" :options="splitOptions" size="small" />
      <NInput v-model:value="autoSavePath" class="monitor-tab__path" placeholder="自动保存文件或目录" size="small" />
      <NInputNumber
        v-if="autoSaveSplitMode === 'size'"
        v-model:value="autoSaveSizeKB"
        class="monitor-tab__number"
        size="small"
        :min="1"
      />
      <NInputNumber
        v-if="autoSaveSplitMode === 'time'"
        v-model:value="autoSaveSeconds"
        class="monitor-tab__number"
        size="small"
        :min="1"
      />
      <NButton size="small" :disabled="!autoSavePath" @click="applyAutoSave">应用</NButton>
    </div>

    <div class="monitor-tab__body">
      <table class="monitor-table">
        <thead>
          <tr>
            <th>#</th>
            <th>时间</th>
            <th>方向</th>
            <th>端口</th>
            <th>长度</th>
            <th>数据</th>
            <th>Modbus</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="frame in frames"
            :key="frame.Seq"
            :class="{ 'is-selected': selectedFrame?.Seq === frame.Seq }"
            @click="selectedSeq = frame.Seq"
          >
            <td>{{ frame.Seq }}</td>
            <td>{{ shortTime(frame.Timestamp) }}</td>
            <td>{{ frame.Direction === 'a_to_b' ? 'A → B' : 'B → A' }}</td>
            <td>{{ frame.Port }}</td>
            <td>{{ frame.Length }}</td>
            <td><code>{{ frameDisplay(frame) }}</code></td>
            <td>{{ frame.Modbus?.Summary ?? '' }} <span v-if="frame.Modbus?.Error">{{ frame.Modbus.Error }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="selectedFrame" class="monitor-detail">
      <div><strong>HEX</strong><code>{{ selectedFrame.DisplayHex }}</code></div>
      <div><strong>文本</strong><code>{{ selectedFrame.DisplayText }}</code></div>
      <div v-if="selectedFrame.Modbus"><strong>Modbus</strong><code>{{ selectedFrame.Modbus.Summary }} {{ selectedFrame.Modbus.Error }}</code></div>
    </div>
  </div>
</template>

<style scoped>
.monitor-tab {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;
  color: #d4d4d4;
  background: #1e1e1e;
}
.monitor-tab__header,
.monitor-tab__toolbar,
.monitor-tab__export {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid #2d2d2d;
  background: #252526;
}
.monitor-tab__title {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}
.monitor-tab__title span,
.monitor-tab__stats {
  color: #858585;
  font-size: 12px;
}
.monitor-tab__stats {
  display: flex;
  gap: 10px;
  white-space: nowrap;
}
.monitor-tab__control {
  width: 120px;
}
.monitor-tab__search {
  flex: 1;
  min-width: 180px;
}
.monitor-tab__function {
  width: 100px;
}
.monitor-tab__display {
  display: inline-flex;
  height: 28px;
  overflow: hidden;
  border: 1px solid #3c3c3c;
  border-radius: 4px;
}
.monitor-tab__display button {
  width: 44px;
  border: 0;
  border-right: 1px solid #3c3c3c;
  background: #2d2d2d;
  color: #cccccc;
  cursor: pointer;
}
.monitor-tab__display button:last-child {
  border-right: 0;
}
.monitor-tab__display button.is-active {
  background: #0e639c;
  color: #ffffff;
}
.monitor-tab__export {
  flex-wrap: wrap;
}
.monitor-tab__format {
  width: 94px;
}
.monitor-tab__path {
  width: min(32vw, 360px);
}
.monitor-tab__number {
  width: 92px;
}
.monitor-tab__autosave-label {
  color: #858585;
  font-size: 12px;
  white-space: nowrap;
}
.monitor-tab__body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.monitor-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 12px;
}
.monitor-table th,
.monitor-table td {
  overflow: hidden;
  padding: 5px 8px;
  border-bottom: 1px solid #2d2d2d;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.monitor-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #252526;
  color: #858585;
  text-align: left;
}
.monitor-table tr.is-selected td {
  background: #094771;
}
.monitor-table code,
.monitor-detail code {
  font-family: var(--serial-terminal-font, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}
.monitor-detail {
  display: grid;
  flex: 0 0 92px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding: 8px 10px;
  overflow: auto;
  border-top: 1px solid #2d2d2d;
  background: #252526;
  font-size: 12px;
}
.monitor-detail div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}
.monitor-detail strong {
  color: #858585;
}
.monitor-detail code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
