<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useRegistry } from '../core/registry'

const registry = useRegistry()
const activeModuleId = computed(() => registry.active.value)

const SerialView = defineAsyncComponent(() => import('../serial/views/SerialView.vue'))

const moduleComponents: Record<string, any> = {
  serial: SerialView,
}

const activeComponent = computed(() => {
  return activeModuleId.value ? moduleComponents[activeModuleId.value] : null
})
</script>

<template>
  <div class="editor-groups">
    <component :is="activeComponent" v-if="activeComponent" />
    <div v-else class="editor-groups__placeholder">
      <p>选择左侧模块开始</p>
    </div>
  </div>
</template>

<style scoped>
.editor-groups {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
  background: #1e1e1e;
}
.editor-groups__placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #858585;
  font-size: 14px;
}
</style>
