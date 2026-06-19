import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useBufferStore } from './bufferStore'

describe('bufferStore workspace restore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('exports and restores chunks without changing byte content or timestamps', () => {
    const store = useBufferStore()
    store.appendData('port-1', [1, 2, 3], 100)
    store.appendData('port-1', [4, 5], 200)

    const exported = store.exportChunks()
    store.clearAll()
    store.restoreChunks(exported)

    expect(Array.from(store.getBuffer('port-1'))).toEqual([1, 2, 3, 4, 5])
    expect(store.getChunks('port-1').map(chunk => chunk.timestamp)).toEqual([100, 200])
    expect(store.getChunks('port-1').map(chunk => Array.from(chunk.data))).toEqual([[1, 2, 3], [4, 5]])
  })
})
