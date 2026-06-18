import { defineStore } from 'pinia'
import { ref } from 'vue'
import { serialEvents, type DataEvent } from '../services/serialEvents'

export const useBufferStore = defineStore('buffer', () => {
  const buffers = ref<Map<string, Uint8Array>>(new Map())
  const maxBufferSize = 100 * 1024 * 1024 // 100MB per port

  function appendData(portId: string, data: number[]) {
    const existing = buffers.value.get(portId) ?? new Uint8Array(0)
    const newData = new Uint8Array(data)
    const combined = new Uint8Array(existing.length + newData.length)
    combined.set(existing)
    combined.set(newData, existing.length)

    // LRU eviction - keep only the most recent maxBufferSize bytes
    if (combined.length > maxBufferSize) {
      const keep = combined.slice(combined.length - maxBufferSize)
      buffers.value.set(portId, keep)
    } else {
      buffers.value.set(portId, combined)
    }
  }

  function getBuffer(portId: string): Uint8Array {
    return buffers.value.get(portId) ?? new Uint8Array(0)
  }

  function clearBuffer(portId: string) {
    buffers.value.delete(portId)
  }

  let unsubscribe: (() => void) | null = null

  function initEventListeners() {
    if (unsubscribe) return // Prevent duplicate initialization
    unsubscribe = serialEvents.onData((event: DataEvent) => {
      appendData(event.PortID, event.Data)
    })
  }

  function cleanup() {
    unsubscribe?.()
    unsubscribe = null
  }

  return {
    buffers,
    appendData,
    getBuffer,
    clearBuffer,
    initEventListeners,
    cleanup,
  }
})
