<script setup lang="ts">
import { computed } from 'vue'
import { useSerialStore } from '../stores/serialStore'

const props = defineProps<{
  handleId: string
}>()

const store = useSerialStore()
const handle = computed(() => store.handles.get(props.handleId))

const stats = computed(() => {
  if (!handle.value) return null
  return {
    rxBytes: handle.value.RxBytes,
    txBytes: handle.value.TxBytes,
  }
})
</script>

<template>
  <div v-if="stats" class="stats-panel">
    <span class="stats-panel__item">RX: {{ stats.rxBytes }} 字节</span>
    <span class="stats-panel__item">TX: {{ stats.txBytes }} 字节</span>
  </div>
</template>

<style scoped>
.stats-panel {
  display: flex;
  gap: 24px;
  padding: 8px 16px;
  font-size: 12px;
  color: #858585;
}
.stats-panel__item {
  white-space: nowrap;
}
</style>
