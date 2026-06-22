<script setup lang="ts">
import type { ModuleContribution } from '../core/module/types'

defineProps<{
  contributions: ModuleContribution[]
  activeId: string | null
  activeViewId: string | null
}>()

defineEmits<{
  selectView: [viewId: string]
}>()
</script>

<template>
  <div class="sidebar" data-testid="sidebar">
    <template v-for="c in contributions" :key="c.moduleId">
      <div v-if="c.moduleId === activeId" class="sidebar__group">
        <div class="sidebar__title">{{ c.activity.title }}</div>
        <button
          v-for="v in c.views"
          :key="v.id"
          class="sidebar__item"
          :class="{ 'is-active': v.id === activeViewId }"
          :title="v.title"
          :data-testid="'sidebar-view-' + v.id"
          type="button"
          @click="$emit('selectView', v.id)"
        >
          {{ v.title }}
        </button>
      </div>
    </template>
    <div v-if="!activeId" class="sidebar__empty">选择一个模块</div>
  </div>
</template>

<style scoped>
.sidebar {
  --sidebar-font-size: 13px;
  --sidebar-line-height: 20px;
  width: 200px;
  flex: 0 0 200px;
  background: var(--app-surface, #252526);
  border-right: 1px solid var(--app-border, #2d2d2d);
  padding: 8px 0;
  overflow-y: auto;
}
.sidebar__title {
  font-size: var(--sidebar-font-size);
  line-height: var(--sidebar-line-height);
  text-transform: uppercase;
  color: var(--app-text-muted, #858585);
  padding: 4px 16px;
  font-weight: 600;
}
.sidebar__item {
  display: block;
  width: 100%;
  padding: 6px 16px;
  border: 0;
  background: transparent;
  color: var(--app-text, #d4d4d4);
  cursor: pointer;
  font: inherit;
  font-size: var(--sidebar-font-size);
  line-height: var(--sidebar-line-height);
  text-align: left;
}
.sidebar__item:hover,
.sidebar__item.is-active {
  background: var(--app-active, #094771);
}
.sidebar__empty {
  padding: 16px;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
</style>
