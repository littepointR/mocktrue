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
    <button
      class="stats-panel__reset"
      type="button"
      @click="store.resetCounters(handleId)"
    >
      复位计数
    </button>
  </div>
</template>

<style scoped>
.stats-panel {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 8px 16px;
  font-size: 12px;
  color: #858585;
}
.stats-panel__item {
  white-space: nowrap;
}
.stats-panel__reset {
  margin-left: auto;
  padding: 2px 8px;
  border: 1px solid #3c3c3c;
  border-radius: 3px;
  background: transparent;
  color: #cccccc;
  cursor: pointer;
  font: inherit;
}
.stats-panel__reset:hover {
  background: #2d2d2d;
}
</style>
