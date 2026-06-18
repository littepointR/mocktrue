<script setup lang="ts">
import { computed } from 'vue'
import { useSerialStore } from '../stores/serialStore'
import DataDisplay from './DataDisplay.vue'
import SendPanel from './SendPanel.vue'
import StatsPanel from './StatsPanel.vue'

const props = defineProps<{
  handleId: string
}>()

const store = useSerialStore()
const handle = computed(() => store.handles.get(props.handleId))
</script>

<template>
  <div class="serial-tab-content">
    <div class="serial-tab-content__display">
      <DataDisplay :handle-id="handleId" />
    </div>
    <div class="serial-tab-content__send">
      <SendPanel :handle-id="handleId" />
    </div>
    <div class="serial-tab-content__stats">
      <StatsPanel :handle-id="handleId" />
    </div>
  </div>
</template>

<style scoped>
.serial-tab-content {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.serial-tab-content__display {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.serial-tab-content__send {
  flex: 0 0 auto;
  border-top: 1px solid #2d2d2d;
  background: #252526;
}
.serial-tab-content__stats {
  flex: 0 0 auto;
  border-top: 1px solid #2d2d2d;
  background: #252526;
}
</style>
