<script setup lang="ts">
import { computed } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import type { EditorLayoutNode, EditorTabInfo } from './editorLayout'
import SerialGraphPanel from './SerialGraphPanel.vue'

const props = defineProps<{
  node: EditorLayoutNode
  tabs: EditorTabInfo[]
  activeByGroup: Record<string, string | null>
}>()

const emit = defineEmits<{
  setActiveTab: [groupId: string, handleId: string]
  closeTab: [handleId: string]
  tabPointerDown: [event: PointerEvent, groupId: string, handleId: string]
}>()

const tabById = computed(() => new Map(props.tabs.map(tab => [tab.id, tab])))

function forwardSetActiveTab(groupId: string, handleId: string) {
  emit('setActiveTab', groupId, handleId)
}

function forwardCloseTab(handleId: string) {
  emit('closeTab', handleId)
}

function forwardTabPointerDown(event: PointerEvent, groupId: string, handleId: string) {
  emit('tabPointerDown', event, groupId, handleId)
}

function tabInfo(handleId: string) {
  return tabById.value.get(handleId)
}

function tabSourceId(handleId: string) {
  return tabInfo(handleId)?.sourceId ?? handleId
}
</script>

<template>
  <Splitpanes
    v-if="node.type === 'split'"
    class="editor-splitpanes"
    :horizontal="node.direction === 'vertical'"
  >
    <Pane
      v-for="child in node.children"
      :key="child.id"
      min-size="18"
    >
      <EditorLayoutNode
        :node="child"
        :tabs="tabs"
        :active-by-group="activeByGroup"
        @set-active-tab="forwardSetActiveTab"
        @close-tab="forwardCloseTab"
        @tab-pointer-down="forwardTabPointerDown"
      />
    </Pane>
  </Splitpanes>

  <div
    v-else
    class="editor-group"
    :data-editor-group-id="node.id"
  >
    <div class="editor-tabs">
      <button
        v-for="handleId in node.tabs"
        :key="handleId"
        class="editor-tab n-tabs-tab"
        :class="{ 'editor-tab--active': activeByGroup[node.id] === handleId }"
        :title="tabInfo(handleId)?.tooltip ?? tabInfo(handleId)?.name ?? handleId"
        :data-testid="'editor-tab-' + tabSourceId(handleId)"
        type="button"
        @click="emit('setActiveTab', node.id, handleId)"
        @pointerdown="emit('tabPointerDown', $event, node.id, handleId)"
      >
        <span class="editor-tab__label">{{ tabInfo(handleId)?.name ?? handleId }}</span>
        <span
          class="editor-tab__close n-tabs-tab__close"
          :data-testid="'close-graph-' + tabSourceId(handleId)"
          role="button"
          tabindex="0"
          @pointerdown.stop
          @click.stop="emit('closeTab', handleId)"
        >
          ×
        </span>
      </button>
    </div>
    <div class="editor-group__content">
      <div
        v-if="activeByGroup[node.id] && tabById.get(activeByGroup[node.id]!)?.kind === 'graph'"
        class="editor-group__panel"
        :data-testid="'graph-panel-' + tabById.get(activeByGroup[node.id]!)!.sourceId"
      >
        <SerialGraphPanel :graph-id="tabById.get(activeByGroup[node.id]!)!.sourceId" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor-splitpanes {
  flex: 1;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  background: var(--app-bg, #1e1e1e);
}
.editor-splitpanes :deep(.splitpanes__splitter) {
  background: var(--app-border, #2d2d2d);
  position: relative;
}
.editor-splitpanes.splitpanes--vertical :deep(.splitpanes__splitter) {
  width: 5px;
  min-width: 5px;
  cursor: col-resize;
}
.editor-splitpanes.splitpanes--horizontal :deep(.splitpanes__splitter) {
  height: 5px;
  min-height: 5px;
  cursor: row-resize;
}
.editor-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--app-bg, #1e1e1e);
}
.editor-tabs {
  display: flex;
  flex-shrink: 0;
  min-height: 35px;
  overflow-x: auto;
  overflow-y: hidden;
  background: var(--app-surface, #252526);
  border-bottom: 1px solid var(--app-border, #2d2d2d);
}
.editor-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 220px;
  min-width: 0;
  height: 35px;
  padding: 0 10px;
  border: 0;
  border-right: 1px solid var(--app-border, #2d2d2d);
  background: var(--app-hover-bg, #2d2d2d);
  color: var(--app-text, #cccccc);
  cursor: default;
  font: inherit;
}
.editor-tab--active {
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #ffffff);
}
.editor-tab__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.editor-tab__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  color: var(--app-text-muted, #858585);
}
.editor-tab__close:hover {
  background: var(--app-hover-bg, #3c3c3c);
  color: var(--app-text, #ffffff);
}
.editor-group__content {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}
.editor-group__panel {
  display: flex;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
</style>
