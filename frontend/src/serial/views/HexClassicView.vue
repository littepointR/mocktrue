<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { useSerialStore } from '../stores/serialStore'
import { useVirtualWindow } from '../../shared/composables/useVirtualWindow'
import { useBlobChunk } from '../../shared/composables/useBlobChunk'
import { QueryPage } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'

const ROW_BYTES = 16
const ROW_HEIGHT = 20

const store = useSerialStore()
const containerRef = ref<HTMLElement | null>(null)
const totalBytes = ref(0)
const rowCount = computed(() => Math.ceil(totalBytes.value / ROW_BYTES))

// Virtual scrolling
const { virtualItems, totalSize, scrollToIndex } = useVirtualWindow({
  parentRef: containerRef,
  rowCount,
  rowHeight: ROW_HEIGHT,
})

// Blob chunk cache
const { fetch: fetchChunk } = useBlobChunk(
  async (offset: number, length: number) => {
    if (!store.activePortId) return { Data: null }
    const snap = await QueryPage(store.activePortId, offset, length)
    return snap ?? { Data: null }
  }
)

// Visible rows data
const visibleRows = ref<Array<{ offset: number; hex: string; ascii: string }>>([])

async function updateVisibleRows() {
  const items = virtualItems.value
  if (items.length === 0) return

  const startOffset = items[0].index * ROW_BYTES
  const endOffset = (items[items.length - 1].index + 1) * ROW_BYTES
  const length = endOffset - startOffset

  try {
    const data = await fetchChunk(startOffset, length)
    const rows: Array<{ offset: number; hex: string; ascii: string }> = []

    for (const item of items) {
      const rowOffset = item.index * ROW_BYTES
      const rowStart = rowOffset - startOffset
      const rowData = data.slice(rowStart, rowStart + ROW_BYTES)

      // Format hex
      const hex = Array.from(rowData).map((b: number) => b.toString(16).padStart(2, '0')).join(' ')

      // Format ASCII
      const ascii = Array.from(rowData).map((b: number) => (b >= 32 && b <= 126) ? String.fromCharCode(b) : '.').join('')

      rows.push({ offset: rowOffset, hex, ascii })
    }

    visibleRows.value = rows
  } catch (e) {
    console.error('Failed to fetch rows:', e)
  }
}

// Update on scroll
watch(virtualItems, () => {
  updateVisibleRows()
}, { immediate: true })

// Listen for data events to update totalBytes
let cancelDataEvent: (() => void) | null = null

onMounted(() => {
  cancelDataEvent = Events.On('serial:data', (event: any) => {
    const evt = event.data
    if (!evt || evt.PortID !== store.activePortId) return
    // Update total from the event or query
    if (store.activePortId) {
      QueryPage(store.activePortId, 0, 0).then(snap => {
        if (snap && snap.Total !== undefined) {
          totalBytes.value = snap.Total
        }
      })
    }
  })
})

onUnmounted(() => {
  cancelDataEvent?.()
})

watch(() => store.activePortId, () => {
  totalBytes.value = 0
  visibleRows.value = []
})

function formatHexAddr(offset: number): string {
  return offset.toString(16).padStart(8, '0')
}
</script>

<template>
  <div
    ref="containerRef"
    class="hex-classic"
    style="overflow: auto; height: 100%;"
  >
    <div :style="{ height: `${totalSize}px`, position: 'relative' }">
      <div
        v-for="item in virtualItems"
        :key="item.index"
        :style="{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: `${item.size}px`,
          transform: `translateY(${item.start}px)`,
        }"
        class="hex-row"
      >
        <span class="hex-addr">{{ formatHexAddr(Number(item.index) * ROW_BYTES) }}</span>
        <span class="hex-data">{{ visibleRows[Number(item.key)]?.hex ?? '' }}</span>
        <span class="hex-ascii">{{ visibleRows[Number(item.key)]?.ascii ?? '' }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hex-classic {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 20px;
  background: #1e1e1e;
  color: #d4d4d4;
}
.hex-row {
  display: flex;
  padding: 0 8px;
}
.hex-addr {
  color: #858585;
  width: 80px;
  flex-shrink: 0;
}
.hex-data {
  color: #4ec9b0;
  width: 400px;
  flex-shrink: 0;
  white-space: pre;
}
.hex-ascii {
  color: #858585;
  white-space: pre;
}
</style>
