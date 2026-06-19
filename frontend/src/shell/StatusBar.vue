<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { getRuntimeMetrics, type RuntimeMetrics } from '../core/runtimeMetrics'
import { onMCPStatus, type MCPStatus } from '../core/mcpStatus'

const props = defineProps<{
  activeId: string | null
  runtimeMetrics?: RuntimeMetrics | null
  mcpStatus?: MCPStatus | null
  dirty?: boolean
  configPath?: string
}>()

const sampledMetrics = ref<RuntimeMetrics | null>(props.runtimeMetrics ?? null)
const sampledMCPStatus = ref<MCPStatus | null>(props.mcpStatus ?? null)
let pollTimer: number | null = null
let cancelMCPStatus: (() => void) | null = null

const metrics = computed(() => props.runtimeMetrics ?? sampledMetrics.value)
const mcpStatus = computed(() => props.mcpStatus ?? sampledMCPStatus.value)
const cpuText = computed(() => `${formatPercent(metrics.value?.CPUPercent ?? 0)}`)
const memoryText = computed(() => formatBytes(metrics.value?.MemoryBytes ?? 0))
const configPathText = computed(() => props.configPath || '未指定')
const mcpText = computed(() => {
  const status = mcpStatus.value
  if (!status?.Enabled) return null
  if (status.Running) return `MCP ${status.Address}${status.Path}`
  return status.Error ? `MCP 错误` : 'MCP 停止'
})

watch(
  () => props.runtimeMetrics,
  value => {
    if (value) sampledMetrics.value = value
  }
)

onMounted(() => {
  if (!props.runtimeMetrics) {
    refreshMetrics()
    pollTimer = window.setInterval(refreshMetrics, 1000)
  }
  if (!props.mcpStatus) {
    cancelMCPStatus = onMCPStatus(status => {
      sampledMCPStatus.value = status
    })
  }
})

onUnmounted(() => {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
  }
  cancelMCPStatus?.()
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
    <span class="status-bar__left">MockTrue{{ dirty ? '*' : '' }} v0.1.0</span>
    <span class="status-bar__right">
      <span class="status-bar__metrics">CPU {{ cpuText }} · 内存 {{ memoryText }}</span>
      <span class="status-bar__config" :title="configPathText">配置 {{ configPathText }}</span>
      <span v-if="mcpText" class="status-bar__mcp">{{ mcpText }}</span>
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
.status-bar__config {
  max-width: min(34vw, 520px);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.status-bar__mcp {
  white-space: nowrap;
}
.status-bar__active {
  white-space: nowrap;
}
</style>
