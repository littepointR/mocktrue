<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import EditorLayoutNode from './EditorLayoutNode.vue'
import type {
  EditorGroupNode,
  EditorLayoutNode as EditorLayoutTreeNode,
  EditorSplitDirection,
  EditorSplitNode,
} from './editorLayout'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'
import { useSerialGraphStore } from '../stores/graphStore'
import { useWorkspaceFileStore } from '../../workspace/stores/workspaceFileStore'

const props = defineProps<{
  activeViewId: string | null
  activeViewVersion: number
}>()

const workspaceStore = useSerialWorkspaceStore()
const graphStore = useSerialGraphStore()
const workspaceFile = useWorkspaceFileStore()

const editorLayout = computed({
  get: () => workspaceStore.editorLayout,
  set: value => workspaceStore.setEditorLayout(value),
})
const activeByGroup = computed({
  get: () => workspaceStore.activeByGroup,
  set: value => workspaceStore.setActiveByGroup(value),
})
const dragState = ref<{
  handleId: string
  sourceGroupId: string
  pointerId: number
  startX: number
  startY: number
} | null>(null)
const dropPreview = ref<{
  edge: 'left' | 'right' | 'top' | 'bottom' | 'center'
  style: Record<string, string>
} | null>(null)

const tabs = computed(() => graphStore.graphList.map(graph => ({
  id: graphTabId(graph.id),
  name: workspaceFile.graphTitle(graph.id, graph.name),
  kind: 'graph' as const,
  sourceId: graph.id,
  tooltip: workspaceFile.graphTooltip(graph.id),
})))

onUnmounted(() => {
  stopTabDrag()
})

watch(
  () => props.activeViewVersion,
  () => {
    if (!props.activeViewId || props.activeViewId.startsWith('serial.')) {
      focusGraphTab()
    }
  }
)

watch(
  () => graphStore.activeGraphId,
  () => {
    focusGraphTab()
  }
)

watch(
  () => tabs.value.map(tab => tab.id),
  (ids) => {
    const idSet = new Set(ids)
    let nextLayout = filterLayoutTabs(editorLayout.value, idSet) ?? createGroup()
    const assigned = new Set(collectTabIds(nextLayout))
    const missing = ids.filter(id => !assigned.has(id))
    if (missing.length > 0) {
      nextLayout = addTabsToFirstGroup(nextLayout, missing)
    }

    const activeGraphTab = graphStore.activeGraphId ? graphTabId(graphStore.activeGraphId) : null
    const groups = collectGroups(nextLayout)
    const active: Record<string, string | null> = {}
    for (const group of groups) {
      const current = activeByGroup.value[group.id]
      active[group.id] = activeGraphTab && group.tabs.includes(activeGraphTab)
        ? activeGraphTab
        : current && group.tabs.includes(current) ? current : group.tabs[0] ?? null
    }

    editorLayout.value = nextLayout
    activeByGroup.value = active
  },
  { immediate: true }
)

function focusGraphTab() {
  const activeGraphId = graphStore.activeGraphId ?? graphStore.graphList[0]?.id
  if (!activeGraphId) return

  const tabId = graphTabId(activeGraphId)
  let nextLayout = editorLayout.value
  if (!collectTabIds(nextLayout).includes(tabId)) {
    nextLayout = addTabsToFirstGroup(nextLayout, [tabId])
    editorLayout.value = nextLayout
  }

  const group = collectGroups(nextLayout).find(item => item.tabs.includes(tabId))
  if (!group) return

  activeByGroup.value = {
    ...activeByGroup.value,
    [group.id]: tabId,
  }
}

async function handleCloseTab(id: string) {
  if (!isGraphTabId(id)) return
  const graphId = graphIdFromTabId(id)
  if (workspaceFile.isGraphDirty(graphId) && !confirm('当前标签页有未保存的配置，确定关闭吗？')) {
    return
  }
  await graphStore.removeGraph(graphId)
  if (!graphStore.graphById(graphId)) {
    workspaceFile.removeGraphState(graphId)
  }
}

function nextGroupId(): string {
  return `group-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
}

function nextSplitId(): string {
  return `split-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
}

function createGroup(tabs: string[] = []): EditorGroupNode {
  return { type: 'group', id: nextGroupId(), tabs }
}

function createSplit(direction: EditorSplitDirection, children: EditorLayoutTreeNode[]): EditorSplitNode {
  return { type: 'split', id: nextSplitId(), direction, children }
}

function setActiveTab(groupId: string, handleId: string) {
  activeByGroup.value = { ...activeByGroup.value, [groupId]: handleId }
  if (isGraphTabId(handleId)) {
    graphStore.setActiveGraph(graphIdFromTabId(handleId))
  }
}

function handleTabPointerDown(event: PointerEvent, groupId: string, handleId: string) {
  if (event.button !== 0) return

  dragState.value = {
    handleId,
    sourceGroupId: groupId,
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
  }
  window.addEventListener('pointerup', handleTabPointerUp)
  window.addEventListener('pointermove', handleTabPointerMove)
  window.addEventListener('pointercancel', stopTabDrag)
}

function handleTabPointerMove(event: PointerEvent) {
  const dragged = dragState.value
  if (!dragged || event.pointerId !== dragged.pointerId) return
  updateDropPreview(event.clientX, event.clientY)
}

function handleTabPointerUp(event: PointerEvent) {
  const dragged = dragState.value
  if (!dragged || event.pointerId !== dragged.pointerId) return

  const deltaX = Math.abs(event.clientX - dragged.startX)
  const deltaY = Math.abs(event.clientY - dragged.startY)
  const didDrag = deltaX > 4 || deltaY > 4
  const dropElement = document.elementFromPoint(event.clientX, event.clientY)
  const target = dropElement?.closest<HTMLElement>('[data-editor-group-id]')

  stopTabDrag()

  if (!target) return

  const targetGroupId = target.dataset.editorGroupId
  if (!targetGroupId) return

  if (!didDrag) {
    setActiveTab(dragged.sourceGroupId, dragged.handleId)
    return
  }

  const rect = target.getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top
  const edge = dropElement?.closest('.editor-tabs, .editor-tab')
    ? 'center'
    : closestDropEdge(x, y, rect.width, rect.height)

  if (edge === 'center') {
    if (dragged.sourceGroupId !== targetGroupId) {
      moveTabToGroup(dragged.handleId, targetGroupId)
    } else {
      setActiveTab(targetGroupId, dragged.handleId)
    }
  } else {
    splitTabToNewGroup(dragged.handleId, dragged.sourceGroupId, targetGroupId, edge)
  }
}

function stopTabDrag() {
  window.removeEventListener('pointerup', handleTabPointerUp)
  window.removeEventListener('pointermove', handleTabPointerMove)
  window.removeEventListener('pointercancel', stopTabDrag)
  dragState.value = null
  dropPreview.value = null
}

function updateDropPreview(clientX: number, clientY: number) {
  const dropElement = document.elementFromPoint(clientX, clientY)
  const target = dropElement?.closest<HTMLElement>('[data-editor-group-id]')
  const layout = document.querySelector<HTMLElement>('.editor-layout')
  if (!target || !layout) {
    dropPreview.value = null
    return
  }

  const rect = target.getBoundingClientRect()
  const layoutRect = layout.getBoundingClientRect()
  const x = clientX - rect.left
  const y = clientY - rect.top
  const edge = dropElement?.closest('.editor-tabs, .editor-tab')
    ? 'center'
    : closestDropEdge(x, y, rect.width, rect.height)
  const previewRect = getPreviewRect(rect, edge)

  dropPreview.value = {
    edge,
    style: {
      left: `${previewRect.left - layoutRect.left}px`,
      top: `${previewRect.top - layoutRect.top}px`,
      width: `${previewRect.width}px`,
      height: `${previewRect.height}px`,
    },
  }
}

function getPreviewRect(rect: DOMRect, edge: 'left' | 'right' | 'top' | 'bottom' | 'center') {
  const halfWidth = rect.width / 2
  const halfHeight = rect.height / 2
  switch (edge) {
    case 'left':
      return { left: rect.left, top: rect.top, width: halfWidth, height: rect.height }
    case 'right':
      return { left: rect.left + halfWidth, top: rect.top, width: halfWidth, height: rect.height }
    case 'top':
      return { left: rect.left, top: rect.top, width: rect.width, height: halfHeight }
    case 'bottom':
      return { left: rect.left, top: rect.top + halfHeight, width: rect.width, height: halfHeight }
    case 'center':
      return { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
  }
}

function closestDropEdge(x: number, y: number, width: number, height: number): 'left' | 'right' | 'top' | 'bottom' | 'center' {
  const marginX = width * 0.24
  const marginY = height * 0.24
  if (x < marginX) return 'left'
  if (x > width - marginX) return 'right'
  if (y < marginY) return 'top'
  if (y > height - marginY) return 'bottom'
  return 'center'
}

function collectGroups(node: EditorLayoutTreeNode): EditorGroupNode[] {
  if (node.type === 'group') return [node]
  return node.children.flatMap(collectGroups)
}

function collectTabIds(node: EditorLayoutTreeNode): string[] {
  if (node.type === 'group') return [...node.tabs]
  return node.children.flatMap(collectTabIds)
}

function filterLayoutTabs(node: EditorLayoutTreeNode, idSet: Set<string>): EditorLayoutTreeNode | null {
  if (node.type === 'group') {
    const filteredTabs = node.tabs.filter(id => idSet.has(id))
    return filteredTabs.length > 0 ? { ...node, tabs: filteredTabs } : null
  }

  const children = node.children
    .map(child => filterLayoutTabs(child, idSet))
    .filter((child): child is EditorLayoutTreeNode => child !== null)

  return normalizeSplit({ ...node, children })
}

function normalizeSplit(split: EditorSplitNode): EditorLayoutTreeNode | null {
  if (split.children.length === 0) return null
  if (split.children.length === 1) return split.children[0]

  const children = split.children.flatMap(child => (
    child.type === 'split' && child.direction === split.direction ? child.children : [child]
  ))
  return { ...split, children }
}

function addTabsToFirstGroup(node: EditorLayoutTreeNode, tabIds: string[]): EditorLayoutTreeNode {
  if (node.type === 'group') {
    return { ...node, tabs: [...node.tabs, ...tabIds] }
  }

  const [first, ...rest] = node.children
  return { ...node, children: [addTabsToFirstGroup(first, tabIds), ...rest] }
}

function removeTabFromLayout(node: EditorLayoutTreeNode, handleId: string): EditorLayoutTreeNode | null {
  if (node.type === 'group') {
    const tabs = node.tabs.filter(id => id !== handleId)
    return tabs.length > 0 ? { ...node, tabs } : null
  }

  const children = node.children
    .map(child => removeTabFromLayout(child, handleId))
    .filter((child): child is EditorLayoutTreeNode => child !== null)

  return normalizeSplit({ ...node, children })
}

function addTabToGroup(node: EditorLayoutTreeNode, groupId: string, handleId: string): { node: EditorLayoutTreeNode; added: boolean } {
  if (node.type === 'group') {
    if (node.id !== groupId) {
      return { node, added: false }
    }
    return {
      node: { ...node, tabs: node.tabs.includes(handleId) ? node.tabs : [...node.tabs, handleId] },
      added: true,
    }
  }

  let added = false
  const children = node.children.map(child => {
    if (added) return child
    const result = addTabToGroup(child, groupId, handleId)
    added = result.added
    return result.node
  })
  return { node: { ...node, children }, added }
}

function containsGroup(node: EditorLayoutTreeNode, groupId: string): boolean {
  if (node.type === 'group') return node.id === groupId
  return node.children.some(child => containsGroup(child, groupId))
}

function moveTabToGroup(handleId: string, targetGroupId: string) {
  const withoutTab = removeTabFromLayout(editorLayout.value, handleId)
  if (!withoutTab || !containsGroup(withoutTab, targetGroupId)) {
    return
  }

  const result = addTabToGroup(withoutTab, targetGroupId, handleId)
  commitLayout(result.node, targetGroupId, handleId)
}

function splitTabToNewGroup(
  handleId: string,
  sourceGroupId: string,
  targetGroupId: string,
  edge: 'left' | 'right' | 'top' | 'bottom'
) {
  const target = collectGroups(editorLayout.value).find(group => group.id === targetGroupId)
  if (!target || (target.id === sourceGroupId && target.tabs.length <= 1)) {
    return
  }

  const withoutTab = removeTabFromLayout(editorLayout.value, handleId)
  if (!withoutTab || !containsGroup(withoutTab, targetGroupId)) {
    return
  }

  const newGroup = createGroup([handleId])
  const result = insertSplitGroup(withoutTab, targetGroupId, newGroup, edge)
  if (result.inserted) {
    commitLayout(result.node, newGroup.id, handleId)
  }
}

function insertSplitGroup(
  node: EditorLayoutTreeNode,
  targetGroupId: string,
  newGroup: EditorGroupNode,
  edge: 'left' | 'right' | 'top' | 'bottom'
): { node: EditorLayoutTreeNode; inserted: boolean } {
  const direction: EditorSplitDirection = edge === 'top' || edge === 'bottom' ? 'vertical' : 'horizontal'
  const before = edge === 'left' || edge === 'top'

  if (node.type === 'group') {
    if (node.id !== targetGroupId) {
      return { node, inserted: false }
    }
    return {
      node: createSplit(direction, before ? [newGroup, node] : [node, newGroup]),
      inserted: true,
    }
  }

  const directIndex = node.children.findIndex(child => child.type === 'group' && child.id === targetGroupId)
  if (directIndex >= 0) {
    const children = [...node.children]
    if (node.direction === direction) {
      children.splice(before ? directIndex : directIndex + 1, 0, newGroup)
    } else {
      const target = children[directIndex]
      children[directIndex] = createSplit(direction, before ? [newGroup, target] : [target, newGroup])
    }
    return { node: { ...node, children }, inserted: true }
  }

  for (let i = 0; i < node.children.length; i += 1) {
    if (!containsGroup(node.children[i], targetGroupId)) continue
    const result = insertSplitGroup(node.children[i], targetGroupId, newGroup, edge)
    if (!result.inserted) return { node, inserted: false }
    const children = [...node.children]
    children[i] = result.node
    return { node: { ...node, children }, inserted: true }
  }

  return { node, inserted: false }
}

function commitLayout(nextLayout: EditorLayoutTreeNode, activeGroupId: string, activeHandleId: string) {
  const groups = collectGroups(nextLayout)
  const active: Record<string, string | null> = {}
  for (const group of groups) {
    const previous = activeByGroup.value[group.id]
    active[group.id] = group.id === activeGroupId
      ? activeHandleId
      : previous && group.tabs.includes(previous) ? previous : group.tabs[0] ?? null
  }
  editorLayout.value = nextLayout
  activeByGroup.value = active
  if (isGraphTabId(activeHandleId)) {
    graphStore.setActiveGraph(graphIdFromTabId(activeHandleId))
  }
}

function graphTabId(id: string): string {
  return `graph:${id}`
}

function isGraphTabId(id: string): boolean {
  return id.startsWith('graph:')
}

function graphIdFromTabId(id: string): string {
  return id.replace(/^graph:/, '')
}
</script>

<template>
  <div class="serial-view">
    <div class="serial-view__main">
      <div
        v-if="tabs.length > 0"
        class="editor-layout"
      >
        <EditorLayoutNode
          :node="editorLayout"
          :tabs="tabs"
          :active-by-group="activeByGroup"
          @set-active-tab="setActiveTab"
          @close-tab="handleCloseTab"
          @tab-pointer-down="handleTabPointerDown"
        />
        <div
          v-if="dropPreview"
          class="editor-drop-preview"
          :data-edge="dropPreview.edge"
          :style="dropPreview.style"
        />
      </div>
      <div v-else class="serial-view__empty">
        <p>新建拓扑图</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.serial-view {
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.serial-view__main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.editor-layout {
  flex: 1;
  display: flex;
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 0;
  background: var(--app-bg, #1e1e1e);
  overflow: hidden;
}
.editor-drop-preview {
  position: absolute;
  z-index: 20;
  pointer-events: none;
  border: 1px solid #007acc;
  background:
    linear-gradient(90deg, rgba(0, 122, 204, 0.16) 25%, rgba(0, 122, 204, 0.28) 50%, rgba(0, 122, 204, 0.16) 75%);
  background-size: 180px 100%;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
  opacity: 0.95;
  animation: drop-preview-shimmer 1.2s linear infinite;
  transition:
    left 0.14s ease,
    top 0.14s ease,
    width 0.14s ease,
    height 0.14s ease;
}
@keyframes drop-preview-shimmer {
  from {
    background-position: -180px 0;
  }
  to {
    background-position: 180px 0;
  }
}
.serial-view__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted, #858585);
  font-size: 14px;
}
</style>
