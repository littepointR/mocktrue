<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { HTMLAttributes } from 'vue'

export interface ResizableTableColumn {
  key: string
  label: string
  width?: number
  minWidth?: number
  class?: string
}

const props = withDefaults(defineProps<{
  columns: ResizableTableColumn[]
  tableClass?: HTMLAttributes['class']
  minWidth?: number
}>(), {
  tableClass: '',
  minWidth: 0,
})

const tableRef = ref<HTMLTableElement | null>(null)
const widths = ref<Record<string, number>>({})
let dragState: { key: string, startX: number, startWidth: number, minWidth: number } | null = null

const tableClasses = computed(() => ['resizable-table', props.tableClass].filter(Boolean))
const tableStyle = computed(() => {
  const declaredWidth = props.minWidth ? `${props.minWidth}px` : undefined
  return declaredWidth ? { minWidth: declaredWidth } : undefined
})
let previousCursor = ''
let previousUserSelect = ''

watch(
  () => props.columns,
  columns => {
    const next: Record<string, number> = {}
    for (const column of columns) {
      next[column.key] = widths.value[column.key] ?? column.width ?? column.minWidth ?? 80
    }
    widths.value = next
  },
  { immediate: true }
)

function columnWidth(column: ResizableTableColumn): number {
  return widths.value[column.key] ?? column.width ?? column.minWidth ?? 80
}

function columnMinWidth(column: ResizableTableColumn): number {
  return column.minWidth ?? 48
}

function startResize(event: PointerEvent, column: ResizableTableColumn) {
  const target = event.currentTarget as HTMLElement
  const header = target.closest('th') as HTMLTableCellElement | null
  const currentWidth = header?.getBoundingClientRect().width || columnWidth(column)
  dragState = {
    key: column.key,
    startX: event.clientX,
    startWidth: currentWidth,
    minWidth: columnMinWidth(column),
  }
  previousCursor = document.body.style.cursor
  previousUserSelect = document.body.style.userSelect
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  target.setPointerCapture?.(event.pointerId)
  window.addEventListener('pointermove', moveResize)
  window.addEventListener('pointerup', stopResize, { once: true })
  event.preventDefault()
}

function moveResize(event: PointerEvent) {
  if (!dragState) return
  const nextWidth = Math.max(dragState.minWidth, Math.round(dragState.startWidth + event.clientX - dragState.startX))
  widths.value = { ...widths.value, [dragState.key]: nextWidth }
}

function stopResize() {
  dragState = null
  document.body.style.cursor = previousCursor
  document.body.style.userSelect = previousUserSelect
  window.removeEventListener('pointermove', moveResize)
}

function autoFitColumn(column: ResizableTableColumn, index: number) {
  const table = tableRef.value
  if (!table) return
  const cells = Array.from(table.querySelectorAll<HTMLTableCellElement>('th, td'))
    .filter(cell => cell.cellIndex === index)
  const measured = cells.reduce((max, cell) => Math.max(max, cell.scrollWidth), 0)
  const nextWidth = Math.max(columnMinWidth(column), measured + 16)
  widths.value = { ...widths.value, [column.key]: nextWidth }
}

onBeforeUnmount(() => {
  if (dragState) {
    stopResize()
  } else {
    window.removeEventListener('pointermove', moveResize)
  }
  window.removeEventListener('pointerup', stopResize)
})
</script>

<template>
  <table ref="tableRef" :class="tableClasses" :style="tableStyle">
    <colgroup>
      <col
        v-for="column in columns"
        :key="column.key"
        :class="column.class"
        :style="{ width: `${columnWidth(column)}px`, minWidth: `${columnMinWidth(column)}px` }"
      >
    </colgroup>
    <thead>
      <tr>
        <th v-for="(column, index) in columns" :key="column.key">
          <span class="resizable-table__header-label">{{ column.label }}</span>
          <button
            type="button"
            class="resizable-table__resize-handle"
            :data-testid="`resize-handle-${column.key}`"
            :aria-label="`调整${column.label || '列'}列宽`"
            @pointerdown="startResize($event, column)"
            @dblclick.stop.prevent="autoFitColumn(column, index)"
          />
        </th>
      </tr>
    </thead>
    <tbody>
      <slot />
    </tbody>
  </table>
</template>

<style scoped>
.resizable-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}
.resizable-table th {
  position: relative;
}
.resizable-table__header-label {
  display: block;
  overflow: hidden;
  padding-right: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.resizable-table__resize-handle {
  position: absolute;
  top: 0;
  right: -4px;
  z-index: 2;
  width: 8px;
  height: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: col-resize;
  touch-action: none;
}
.resizable-table__resize-handle::after {
  position: absolute;
  top: 20%;
  right: 3px;
  width: 1px;
  height: 60%;
  background: var(--app-border, #3c3c3c);
  content: '';
  opacity: 0.55;
}
.resizable-table__resize-handle:hover::after,
.resizable-table__resize-handle:focus-visible::after {
  background: var(--app-accent, #409eff);
  opacity: 1;
}
</style>
