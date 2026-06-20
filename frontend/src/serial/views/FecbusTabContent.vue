<script setup lang="ts">
import FecbusPanel from './FecbusPanel.vue'

defineProps<{
  sessionId: string
}>()
</script>

<template>
  <FecbusPanel variant="tab" :session-id="sessionId" />
</template>
