<script setup lang="ts">
import type { ModuleContribution } from '../core/module/types'

defineProps<{ contributions: ModuleContribution[]; activeId: string | null }>()
</script>

<template>
  <div class="sidebar">
    <template v-for="c in contributions" :key="c.moduleId">
      <div v-if="c.moduleId === activeId" class="sidebar__group">
        <div class="sidebar__title">{{ c.activity.title }}</div>
        <div
          v-for="v in c.views"
          :key="v.id"
          class="sidebar__item"
          :title="v.title"
        >
          {{ v.title }}
        </div>
      </div>
    </template>
    <div v-if="!activeId" class="sidebar__empty">选择一个模块</div>
  </div>
</template>

<style scoped>
.sidebar {
  width: 200px;
  flex: 0 0 200px;
  background: #252526;
  border-right: 1px solid #2d2d2d;
  padding: 8px 0;
  overflow-y: auto;
}
.sidebar__title {
  font-size: 11px;
  text-transform: uppercase;
  color: #858585;
  padding: 4px 16px;
}
.sidebar__item {
  padding: 6px 16px;
  color: #d4d4d4;
  cursor: pointer;
}
.sidebar__item:hover {
  background: #094771;
}
.sidebar__empty {
  padding: 16px;
  color: #858585;
  font-size: 12px;
}
</style>
