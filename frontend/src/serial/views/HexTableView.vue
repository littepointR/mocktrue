<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { NDataTable } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import { useSerialStore } from '../stores/serialStore'
import { QueryPage } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js'
import { toByteArray } from '../utils/bytes'

interface RowData {
  offset: number
  hex: string
  ascii: string
  direction: string
  timestamp: string
}

const store = useSerialStore()
const rows = ref<RowData[]>([])
const maxRows = 1000

const columns = [
  { title: 'Offset', key: 'offset', width: 100 },
  { title: '方向', key: 'direction', width: 60 },
  { title: 'HEX', key: 'hex', width: 400 },
  { title: 'ASCII', key: 'ascii', width: 200 },
  { title: '时间', key: 'timestamp', width: 120 },
]

function formatTimestamp(): string {
  const now = new Date()
  return `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}`
}

function formatHex(data: number[]): string {
  return data.map((b: number) => b.toString(16).padStart(2, '0')).join(' ')
}

function formatAscii(data: number[]): string {
  return data.map((b: number) => (b >= 32 && b <= 126) ? String.fromCharCode(b) : '.').join('')
}

let cancelDataEvent: (() => void) | null = null

onMounted(() => {
  cancelDataEvent = Events.On('serial:data', (event: any) => {
    const evt = event.data
    if (!evt || evt.PortID !== store.activePortId) return

    const data = toByteArray(evt.Data)
    const ts = formatTimestamp()
    const hex = formatHex(data)
    const ascii = formatAscii(data)

    rows.value.push({
      offset: rows.value.length * 16,
      hex,
      ascii,
      direction: 'RX',
      timestamp: ts,
    })

    // Trim to maxRows
    if (rows.value.length > maxRows) {
      rows.value = rows.value.slice(-maxRows)
    }
  })
})

onUnmounted(() => {
  cancelDataEvent?.()
})

watch(() => store.activePortId, () => {
  rows.value = []
})
</script>

<template>
  <div class="hex-table">
    <NDataTable
      :columns="columns"
      :data="rows"
      :max-height="400"
      :scroll-x="800"
      size="small"
      :bordered="false"
      :single-line="false"
    />
  </div>
</template>

<style scoped>
.hex-table {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  background: #1e1e1e;
}
</style>
