import { computed, type Ref } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'

export interface UseVirtualWindowOptions {
  parentRef: Ref<HTMLElement | null>
  rowCount: Ref<number>
  rowHeight?: number
  overscan?: number
}

/**
 * useVirtualWindow wraps @tanstack/vue-virtual for fixed-height rows
 * (e.g. HEX view: 16 bytes per row). Returns virtual items and the
 * visible byte range for data fetching.
 */
export function useVirtualWindow(options: UseVirtualWindowOptions) {
  const { parentRef, rowCount, rowHeight = 20, overscan = 10 } = options

  const virtualizer = useVirtualizer({
    count: rowCount.value,
    getScrollElement: () => parentRef.value,
    estimateSize: () => rowHeight,
    overscan,
  })

  const virtualItems = computed(() => virtualizer.value.getVirtualItems())
  const totalSize = computed(() => virtualizer.value.getTotalSize())

  return {
    virtualItems,
    totalSize,
    scrollToIndex: (index: number) => virtualizer.value.scrollToIndex(index),
  }
}
