<script setup lang="ts">
import { computed } from 'vue'
import GlobalSettingsPanel from './global/GlobalSettingsPanel.vue'
import SerialSettingsPanel from './serial/SerialSettingsPanel.vue'

const props = defineProps<{
  activeViewId: string | null
}>()

const activeComponent = computed(() => {
  switch (props.activeViewId) {
    case 'settings.serial':
      return SerialSettingsPanel
    case 'settings.global':
    default:
      return GlobalSettingsPanel
  }
})
</script>

<template>
  <div class="settings-view">
    <component :is="activeComponent" />
  </div>
</template>

<style scoped>
.settings-view {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: var(--app-bg, #1e1e1e);
}
</style>
