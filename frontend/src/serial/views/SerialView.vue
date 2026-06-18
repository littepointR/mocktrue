<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { NTabs, NTabPane } from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import { useBufferStore } from '../stores/bufferStore'
import SerialTabContent from './SerialTabContent.vue'
import PortConfigPanel from './PortConfigPanel.vue'
import VirtualPairPanel from './VirtualPairPanel.vue'
import BridgePanel from './BridgePanel.vue'

const serialStore = useSerialStore()
const bufferStore = useBufferStore()

const sidebarTab = ref('config')

onMounted(() => {
  serialStore.initEventListeners()
  bufferStore.initEventListeners()
  serialStore.refreshHandles()
})

onUnmounted(() => {
  bufferStore.cleanup()
  serialStore.stopStatsPolling()
})

const tabs = computed(() => serialStore.openHandles.map(h => ({
  id: h.ID,
  name: h.Config.PortName,
})))

async function handleCloseTab(id: string) {
  await serialStore.closePort(id)
  bufferStore.clearBuffer(id)
}
</script>

<template>
  <div class="serial-view">
    <div class="serial-view__sidebar">
      <NTabs v-model:value="sidebarTab" type="line" size="small" justify-content="space-evenly">
        <NTabPane name="config" tab="串口">
          <PortConfigPanel />
        </NTabPane>
        <NTabPane name="virtual" tab="虚拟">
          <VirtualPairPanel />
        </NTabPane>
        <NTabPane name="bridge" tab="桥接">
          <BridgePanel />
        </NTabPane>
      </NTabs>
    </div>
    <div class="serial-view__main">
      <NTabs
        v-if="tabs.length > 0"
        :value="serialStore.activePortId ?? undefined"
        type="card"
        closable
        @update:value="serialStore.setActivePort($event)"
        @close="handleCloseTab($event)"
      >
        <NTabPane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.name"
        >
          <SerialTabContent :handle-id="tab.id" />
        </NTabPane>
      </NTabs>
      <div v-else class="serial-view__empty">
        <p>在左侧配置并打开串口</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.serial-view {
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.serial-view__sidebar {
  width: 320px;
  flex: 0 0 320px;
  border-right: 1px solid #2d2d2d;
  overflow: hidden;
  background: #252526;
  display: flex;
  flex-direction: column;
}
.serial-view__sidebar :deep(.n-tabs) {
  height: 100%;
}
.serial-view__sidebar :deep(.n-tabs-nav) {
  flex-shrink: 0;
}
.serial-view__sidebar :deep(.n-tabs-pane-wrapper) {
  flex: 1;
  overflow: hidden;
}
.serial-view__sidebar :deep(.n-tab-pane) {
  height: 100%;
  padding: 0;
}
.serial-view__main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.serial-view__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #858585;
  font-size: 14px;
}
</style>
