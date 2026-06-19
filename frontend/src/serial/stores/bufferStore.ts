import { defineStore } from 'pinia'
import { ref } from 'vue'
import { serialEvents, type DataEvent } from '../services/serialEvents'
import { toByteArray, type BytePayload } from '../utils/bytes'
import { useSerialStore } from './serialStore'

export interface BufferChunk {
  timestamp: number
  data: Uint8Array
}

export interface SerializableBufferChunk {
  timestamp: number
  data: number[]
}

export const useBufferStore = defineStore('buffer', () => {
  const buffers = ref<Map<string, Uint8Array>>(new Map())
  const chunks = ref<Map<string, BufferChunk[]>>(new Map())
  const maxBufferSize = 100 * 1024 * 1024 // 100MB per port

  function appendData(portId: string, data: BytePayload, timestamp = Date.now()) {
    const existing = buffers.value.get(portId) ?? new Uint8Array(0)
    const byteArray = toByteArray(data)
    const newData = new Uint8Array(byteArray)
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
    useSerialStore().addRxBytes(portId, byteArray.length)
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

  function clearAll() {
    buffers.value.clear()
    chunks.value.clear()
  }

  function exportChunks(): Record<string, SerializableBufferChunk[]> {
    return Object.fromEntries(Array.from(chunks.value.entries()).map(([portId, list]) => [
      portId,
      list.map(chunk => ({
        timestamp: chunk.timestamp,
        data: Array.from(chunk.data),
      })),
    ]))
  }

  function restoreChunks(next: Record<string, SerializableBufferChunk[]>) {
    clearAll()
    for (const [portId, list] of Object.entries(next)) {
      const restoredChunks = list.map(chunk => ({
        timestamp: chunk.timestamp,
        data: new Uint8Array(chunk.data),
      }))
      chunks.value.set(portId, restoredChunks)

      const total = restoredChunks.reduce((sum, chunk) => sum + chunk.data.length, 0)
      const combined = new Uint8Array(total)
      let offset = 0
      for (const chunk of restoredChunks) {
        combined.set(chunk.data, offset)
        offset += chunk.data.length
      }
      buffers.value.set(portId, combined)
    }
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
    clearAll,
    exportChunks,
    restoreChunks,
    initEventListeners,
    cleanup,
  }
})
