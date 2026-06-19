import { defineStore } from 'pinia'
import { ref } from 'vue'
import { serialEvents, type DataEvent } from '../services/serialEvents'
import { toByteArray, type BytePayload } from '../utils/bytes'

export interface BufferChunk {
  timestamp: number
  data: Uint8Array
}

export const useBufferStore = defineStore('buffer', () => {
  const buffers = ref<Map<string, Uint8Array>>(new Map())
  const chunks = ref<Map<string, BufferChunk[]>>(new Map())
  const maxBufferSize = 100 * 1024 * 1024 // 100MB per port

  function appendData(portId: string, data: BytePayload, timestamp = Date.now()) {
    const existing = buffers.value.get(portId) ?? new Uint8Array(0)
    const newData = new Uint8Array(toByteArray(data))
    if (newData.length === 0) return

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

    const existingChunks = chunks.value.get(portId) ?? []
    chunks.value.set(portId, [...existingChunks, { timestamp, data: newData }])
  }

  function getBuffer(portId: string): Uint8Array {
    return buffers.value.get(portId) ?? new Uint8Array(0)
  }

  function getChunks(portId: string): BufferChunk[] {
    return chunks.value.get(portId) ?? []
  }

  function clearBuffer(portId: string) {
    buffers.value.delete(portId)
    chunks.value.delete(portId)
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
    chunks,
    appendData,
    getBuffer,
    getChunks,
    clearBuffer,
    initEventListeners,
    cleanup,
  }
})
