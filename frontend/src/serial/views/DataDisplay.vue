<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { NSelect, NSpace, NSwitch } from 'naive-ui'
import { useBufferStore } from '../stores/bufferStore'

const props = defineProps<{
  handleId: string
}>()

type ViewMode = 'ascii' | 'hexClassic' | 'hexTable'
type LayoutMode = 'combined' | 'split'

const bufferStore = useBufferStore()
const viewMode = ref<ViewMode>('ascii')
const layoutMode = ref<LayoutMode>('combined')
const showTimestamp = ref(true)

const buffer = computed(() => bufferStore.getBuffer(props.handleId))

const displayText = computed(() => {
  const decoder = new TextDecoder()
  return decoder.decode(buffer.value)
})

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
</script>

<template>
  <div class="data-display">
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
        <span style="font-size: 12px; color: #858585">时间戳</span>
        <NSwitch v-model:value="showTimestamp" size="small" />
      </NSpace>
    </div>

    <div class="data-display__content" :class="{ 'split': layoutMode === 'split' }">
      <!-- ASCII View -->
      <div v-if="viewMode === 'ascii'" class="data-view">
        <pre class="ascii-content">{{ displayText }}</pre>
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
.data-display__content {
  flex: 1;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
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
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  color: #4ec9b0;
  white-space: pre-wrap;
  word-wrap: break-word;
}
.hex-row {
  padding: 2px 0;
  color: #4ec9b0;
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
