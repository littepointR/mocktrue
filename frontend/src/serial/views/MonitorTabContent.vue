<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NButton, NInput, NSelect } from 'naive-ui'
import { useMonitorStore, type MonitorDisplayMode } from '../stores/monitorStore'
import ResizableTable, { type ResizableTableColumn } from '../components/ResizableTable.vue'
import type { Frame } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/monitor/models.js'

const props = defineProps<{
  monitorId: string
}>()

const store = useMonitorStore()
const session = computed(() => store.sessions.get(props.monitorId) ?? null)
const frames = computed(() => store.framesByMonitor.get(props.monitorId) ?? [])
const total = computed(() => store.frameTotals.get(props.monitorId) ?? frames.value.length)
const filter = computed(() => store.filterFor(props.monitorId))
const selectedSeq = ref<number | null>(null)

const directionOptions = [
  { label: '全部方向', value: 'all' },
  { label: '接收', value: 'a_to_b' },
  { label: '发送', value: 'b_to_a' },
]
const displayOptions = [
  { label: '文本', value: 'text' },
  { label: 'HEX', value: 'hex' },
  { label: 'DEC', value: 'dec' },
  { label: 'OCT', value: 'oct' },
  { label: 'BIN', value: 'bin' },
]
const frameColumns: ResizableTableColumn[] = [
  { key: 'seq', label: '#', width: 52, minWidth: 44, class: 'monitor-table__seq-column' },
  { key: 'time', label: '时间', width: 180, minWidth: 132, class: 'monitor-table__time-column' },
  { key: 'direction', label: '方向', width: 64, minWidth: 56, class: 'monitor-table__direction-column' },
  { key: 'port', label: '端口', width: 180, minWidth: 96, class: 'monitor-table__port-column' },
  { key: 'length', label: '长度', width: 56, minWidth: 50, class: 'monitor-table__length-column' },
  { key: 'data', label: '数据', width: 360, minWidth: 160, class: 'monitor-table__data-column' },
]
const selectedFrame = computed(() => {
  if (selectedSeq.value === null) return frames.value[0] ?? null
  return frames.value.find(frame => frame.Seq === selectedSeq.value) ?? frames.value[0] ?? null
})
const connectionLabel = computed(() => {
  const current = session.value
  if (!current) return ''
  if (current.ExternalPort) {
    return `监听 ${current.PortA}，外部 ${current.ExternalPort}`
  }
  return `监听 ${current.PortA}，对端 ${current.PortB}`
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

function setDirection(value: string) {
  store.refreshFrames(props.monitorId, { direction: value })
}

function setSearch(value: string) {
  store.refreshFrames(props.monitorId, { search: value })
}

function setDisplayMode(value: MonitorDisplayMode) {
  store.setFilter(props.monitorId, { displayMode: value })
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

function directionLabel(direction: string): string {
  if (direction === 'a_to_b') return '接收'
  if (direction === 'b_to_a') return '发送'
  return direction
}

function pad(value: number, size = 2): string {
  return String(value).padStart(size, '0')
}

function frameTime(value: unknown): string {
  if (!value) return ''
  const date = new Date(String(value))
  if (Number.isNaN(date.getTime())) return String(value)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} `
    + `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}`
}

async function stopMonitor() {
  await store.stopMonitor(props.monitorId)
}

async function clearFrames() {
  selectedSeq.value = null
  await store.clearFrames(props.monitorId)
}
</script>

<template>
  <div class="monitor-tab">
    <div class="monitor-tab__header">
      <div class="monitor-tab__title">
        <strong>{{ session?.Name ?? monitorId }}</strong>
        <span>{{ connectionLabel }}</span>
      </div>
      <div class="monitor-tab__stats">
        <span>{{ session?.Status ?? 'stopped' }}</span>
        <span>发送 {{ session?.TxBytes ?? 0 }}</span>
        <span>接收 {{ session?.RxBytes ?? 0 }}</span>
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
        placeholder="搜索文本、HEX 或端口"
        size="small"
        clearable
        @update:value="setSearch"
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

    <div class="monitor-tab__receive-container" data-testid="monitor-receive-container">
      <div class="monitor-tab__body" data-testid="monitor-frame-list">
        <ResizableTable :columns="frameColumns" table-class="monitor-table" :min-width="860">
          <tr
            v-for="frame in frames"
            :key="frame.Seq"
            :class="{ 'is-selected': selectedFrame?.Seq === frame.Seq }"
            @click="selectedSeq = frame.Seq"
          >
            <td>{{ frame.Seq }}</td>
            <td>{{ frameTime(frame.Timestamp) }}</td>
            <td>{{ directionLabel(frame.Direction) }}</td>
            <td>{{ frame.Port }}</td>
            <td>{{ frame.Length }}</td>
            <td><code>{{ frameDisplay(frame) }}</code></td>
          </tr>
        </ResizableTable>
      </div>

      <div v-if="selectedFrame" class="monitor-detail" data-testid="monitor-frame-detail">
        <div><strong>HEX</strong><code>{{ selectedFrame.DisplayHex }}</code></div>
        <div><strong>文本</strong><code>{{ selectedFrame.DisplayText }}</code></div>
      </div>
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
.monitor-tab__toolbar {
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
.monitor-tab__receive-container {
  display: flex;
  flex: 1 1 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}
.monitor-tab__body {
  flex: 1 1 0;
  min-height: 0;
  overflow: auto;
}
.monitor-tab :deep(.monitor-table) {
  width: 100%;
  min-width: 860px;
  font-size: 12px;
}
.monitor-tab :deep(.monitor-table th),
.monitor-tab :deep(.monitor-table td) {
  overflow: hidden;
  padding: 5px 8px;
  border-bottom: 1px solid #2d2d2d;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.monitor-tab :deep(.monitor-table th) {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #252526;
  color: #858585;
  text-align: left;
}
.monitor-tab :deep(.monitor-table tr.is-selected td) {
  background: #094771;
}
.monitor-tab :deep(.monitor-table code),
.monitor-detail code {
  font-family: var(--serial-terminal-font, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}
.monitor-tab :deep(.monitor-table td:last-child code) {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.monitor-detail {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  max-height: 120px;
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
