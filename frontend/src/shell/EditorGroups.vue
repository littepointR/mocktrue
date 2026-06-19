<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useRegistry } from '../core/registry'

defineProps<{
  activeViewId: string | null
  activeViewVersion: number
}>()

const registry = useRegistry()
const activeModuleId = computed(() => registry.active.value)

const SerialView = defineAsyncComponent(() => import('../serial/views/SerialView.vue'))
const SettingsView = defineAsyncComponent(() => import('../settings/views/SettingsView.vue'))

const moduleComponents: Record<string, any> = {
  serial: SerialView,
  settings: SettingsView,
}

const activeComponent = computed(() => {
  return activeModuleId.value ? moduleComponents[activeModuleId.value] : null
})
</script>

<template>
  <div class="editor-groups">
    <component
      :is="activeComponent"
      v-if="activeComponent"
      :active-view-id="activeViewId"
      :active-view-version="activeViewVersion"
    />
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
  background: var(--app-bg, #1e1e1e);
}
.editor-groups__placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted, #858585);
  font-size: 14px;
}
</style>
