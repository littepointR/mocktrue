<script setup lang="ts">
import ModbusPanel from './ModbusPanel.vue'

defineProps<{
  sessionId: string
}>()
</script>

<template>
  <ModbusPanel variant="tab" :session-id="sessionId" />
</template>
