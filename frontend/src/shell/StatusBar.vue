<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { getRuntimeMetrics, type RuntimeMetrics } from '../core/runtimeMetrics'

const props = defineProps<{
  activeId: string | null
  runtimeMetrics?: RuntimeMetrics | null
}>()

const sampledMetrics = ref<RuntimeMetrics | null>(props.runtimeMetrics ?? null)
let pollTimer: number | null = null

const metrics = computed(() => props.runtimeMetrics ?? sampledMetrics.value)
const cpuText = computed(() => `${formatPercent(metrics.value?.CPUPercent ?? 0)}`)
const memoryText = computed(() => formatBytes(metrics.value?.MemoryBytes ?? 0))

watch(
  () => props.runtimeMetrics,
  value => {
    if (value) sampledMetrics.value = value
  }
)

onMounted(() => {
  if (props.runtimeMetrics) return
  refreshMetrics()
  pollTimer = window.setInterval(refreshMetrics, 1000)
})

onUnmounted(() => {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
  }
})

async function refreshMetrics() {
  sampledMetrics.value = await getRuntimeMetrics()
}

function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '0.0%'
  return `${Math.max(0, value).toFixed(1)}%`
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }

  return unit === 0 ? `${Math.round(size)} ${units[unit]}` : `${size.toFixed(1)} ${units[unit]}`
}
</script>

<template>
  <div class="status-bar">
    <span class="status-bar__left">MockTrue v0.1.0</span>
    <span class="status-bar__right">
      <span class="status-bar__metrics">CPU {{ cpuText }} · 内存 {{ memoryText }}</span>
      <span class="status-bar__active">{{ activeId ?? '—' }}</span>
    </span>
  </div>
</template>

<style scoped>
.status-bar {
  height: 24px;
  flex: 0 0 24px;
  background: #007acc;
  color: #fff;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  user-select: none;
}
.status-bar__right {
  display: inline-flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}
.status-bar__metrics {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.status-bar__active {
  white-space: nowrap;
}
</style>
