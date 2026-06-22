<script setup lang="ts">
import type { ModuleContribution } from '../core/module/types'

defineProps<{ contributions: ModuleContribution[]; activeId: string | null }>()
const emit = defineEmits<{ (e: 'select', id: string): void }>()

function iconGlyph(icon: string): string {
  // Stage 0: map a handful of icon keys to simple glyphs.
  const map: Record<string, string> = { serial: '⇄', settings: '⚙' }
  return map[icon] ?? '◆'
}

function isBottomActivity(moduleId: string): boolean {
  return moduleId === 'settings'
}

function handleSelect(moduleId: string) {
  emit('select', moduleId)
}
</script>

<template>
  <div class="activity-bar">
    <button
      v-for="c in contributions"
      :key="c.moduleId"
      class="activity-bar__item"
      :class="{
        'is-active': c.moduleId === activeId,
        'activity-bar__item--bottom': isBottomActivity(c.moduleId),
      }"
      :title="c.activity.title"
      :data-testid="'activity-' + c.moduleId"
      @click="handleSelect(c.moduleId)"
    >
      <span class="activity-bar__icon">{{ iconGlyph(c.activity.icon) }}</span>
    </button>
  </div>
</template>

<style scoped>
.activity-bar {
  width: 48px;
  flex: 0 0 48px;
  background: var(--app-surface, #252526);
  border-right: 1px solid var(--app-border, #2d2d2d);
  display: flex;
  flex-direction: column;
  padding: 8px 0;
}
.activity-bar__item {
  width: 48px;
  height: 48px;
  border: none;
  background: transparent;
  color: var(--app-text-muted, #858585);
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.activity-bar__item--bottom {
  margin-top: auto;
}
.activity-bar__item:hover {
  color: var(--app-text, #d4d4d4);
}
.activity-bar__item.is-active {
  color: var(--app-text, #d4d4d4);
  border-left: 2px solid #007acc;
}
</style>
