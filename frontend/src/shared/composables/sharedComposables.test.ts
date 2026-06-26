import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useBlobChunk } from './useBlobChunk'

const virtualizer = vi.hoisted(() => ({
  getVirtualItems: vi.fn(() => [{ index: 2, start: 40, size: 20 }]),
  getTotalSize: vi.fn(() => 200),
  scrollToIndex: vi.fn(),
}))
const useVirtualizerMock = vi.hoisted(() => vi.fn(() => ref(virtualizer)))

vi.mock('@tanstack/vue-virtual', () => ({
  useVirtualizer: useVirtualizerMock,
}))

import { useVirtualWindow } from './useVirtualWindow'

describe('shared composables', () => {
  it('fetches base64 blob chunks once and serves repeated ranges from cache', async () => {
    const queryPage = vi.fn(async () => ({ Data: btoa('abcdef') }))
    const chunks = useBlobChunk(queryPage)

    await expect(chunks.fetch(1, 3)).resolves.toEqual(new Uint8Array([98, 99, 100]))
    await expect(chunks.fetch(2, 2)).resolves.toEqual(new Uint8Array([99, 100]))
    expect(queryPage).toHaveBeenCalledTimes(1)

    chunks.clear()
    await expect(chunks.fetch(0, 1)).resolves.toEqual(new Uint8Array([97]))
    expect(queryPage).toHaveBeenCalledTimes(2)
  })

  it('copies empty and Uint8Array backend pages across chunk boundaries', async () => {
    const first = new Uint8Array(64 * 1024)
    first[64 * 1024 - 1] = 7
    const second = new Uint8Array([8, 9])
    const queryPage = vi
      .fn()
      .mockResolvedValueOnce({ Data: first })
      .mockResolvedValueOnce({ Data: second })
      .mockResolvedValueOnce({ Data: null })

    const chunks = useBlobChunk(queryPage)

    await expect(chunks.fetch(64 * 1024 - 1, 3)).resolves.toEqual(new Uint8Array([7, 8, 9]))
    chunks.clear()
    await expect(chunks.fetch(0, 2)).resolves.toEqual(new Uint8Array([0, 0]))
  })

  it('wraps the virtualizer and exposes computed items, size, and scrolling', () => {
    const parentRef = ref<HTMLElement | null>(document.createElement('div'))
    const rowCount = ref(10)

    const window = useVirtualWindow({ parentRef, rowCount, rowHeight: 24, overscan: 3 })

    expect(useVirtualizerMock).toHaveBeenCalledWith(expect.objectContaining({
      count: 10,
      overscan: 3,
    }))
    expect(window.virtualItems.value).toEqual([{ index: 2, start: 40, size: 20 }])
    expect(window.totalSize.value).toBe(200)

    window.scrollToIndex(5)
    expect(virtualizer.scrollToIndex).toHaveBeenCalledWith(5)
  })
})
