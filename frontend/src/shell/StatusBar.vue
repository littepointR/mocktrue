<script setup lang="ts">
defineProps<{ activeId: string | null }>()
</script>

<template>
  <div class="status-bar">
    <span class="status-bar__left">MockTrue v0.1.0</span>
    <span class="status-bar__right">{{ activeId ?? '—' }}</span>
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
</style>
