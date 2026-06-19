import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { serialService } from '../services/serialService'
import type { PortInfo } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/port/models.js'
import type { HandleStatus } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/manager/models.js'
import type { SerialConfig } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/port/models.js'

export const useSerialStore = defineStore('serial', () => {
  // State
  const ports = ref<PortInfo[]>([])
  const handles = ref<Map<string, HandleStatus>>(new Map())
  const activePortId = ref<string | null>(null)
  const error = ref<string | null>(null)

  // Stats polling
  let statsInterval: number | null = null

  // Getters
  const activeHandle = computed(() =>
    activePortId.value ? handles.value.get(activePortId.value) ?? null : null
  )
  const openHandles = computed(() =>
    Array.from(handles.value.values()).filter(h => h.IsOpen)
  )

  // Actions
  async function refreshPorts() {
    try {
      ports.value = await serialService.enumeratePorts()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to enumerate ports'
    }
  }

  async function openPort(
    portName: string,
    baudRate: number,
    options?: {
      dataBits?: number
      stopBits?: string
      parity?: string
      flowMode?: string
      readBufKB?: number
    }
  ): Promise<string> {
    const existing = Array.from(handles.value.values()).find(h =>
      h.IsOpen && h.Config.PortName === portName
    )
    if (existing) {
      activePortId.value = existing.ID
      error.value = null
      return existing.ID
    }

    try {
      const request = {
        Config: {
          PortName: portName,
          BaudRate: baudRate,
          DataBits: options?.dataBits ?? 8,
          StopBits: options?.stopBits ?? '1',
          Parity: options?.parity ?? 'none',
          FlowMode: options?.flowMode ?? 'none',
          ReadBufKB: options?.readBufKB ?? 32,
        },
      }
      const status = await serialService.openPort(request)
      handles.value.set(status.ID, status)
      activePortId.value = status.ID
      error.value = null
      startStatsPolling()
      return status.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to open port'
      throw e
    }
  }

  async function openConfig(config: SerialConfig): Promise<string> {
    const existing = Array.from(handles.value.values()).find(h =>
      h.IsOpen && h.Config.PortName === config.PortName
    )
    if (existing) {
      activePortId.value = existing.ID
      error.value = null
      return existing.ID
    }

    try {
      const status = await serialService.openPort({ Config: config })
      handles.value.set(status.ID, status)
      activePortId.value = status.ID
      error.value = null
      startStatsPolling()
      return status.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to open port'
      throw e
    }
  }

  async function closePort(id: string) {
    try {
      await serialService.closePort(id)
      handles.value.delete(id)
      if (activePortId.value === id) {
        activePortId.value = openHandles.value[0]?.ID ?? null
      }
      error.value = null

      // Stop polling if no open handles remain
      if (handles.value.size === 0) {
        stopStatsPolling()
      }
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to close port'
      throw e
    }
  }

  async function updatePortConfig(id: string, nextConfig: SerialConfig): Promise<string> {
    const current = handles.value.get(id)
    if (!current) {
      throw new Error(`unknown port handle: ${id}`)
    }

    try {
      await serialService.closePort(id)
      handles.value.delete(id)

      const status = await serialService.openPort({ Config: nextConfig })
      handles.value.set(status.ID, status)
      activePortId.value = status.ID
      error.value = null
      startStatsPolling()
      return status.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to update port config'
      throw e
    }
  }

  function setActivePort(id: string | null) {
    activePortId.value = id
  }

  function addTxBytes(id: string, byteCount: number) {
    const handle = handles.value.get(id)
    if (!handle) return
    handles.value.set(id, {
      ...handle,
      TxBytes: handle.TxBytes + byteCount,
    })
  }

  function addRxBytes(id: string, byteCount: number) {
    const handle = handles.value.get(id)
    if (!handle) return
    handles.value.set(id, {
      ...handle,
      RxBytes: handle.RxBytes + byteCount,
    })
  }

  async function resetCounters(id: string) {
    const handle = handles.value.get(id)
    if (!handle) return
    try {
      await serialService.resetCounters(id)
      handles.value.set(id, {
        ...handle,
        RxBytes: 0,
        TxBytes: 0,
      })
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to reset counters'
      throw e
    }
  }

  async function restoreCounters(id: string, rxBytes: number, txBytes: number) {
    const handle = handles.value.get(id)
    if (!handle) return
    try {
      await serialService.restoreCounters(id, rxBytes, txBytes)
      handles.value.set(id, {
        ...handle,
        RxBytes: rxBytes,
        TxBytes: txBytes,
      })
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to restore counters'
      throw e
    }
  }

  async function closeAllPorts() {
    const ids = Array.from(handles.value.keys())
    for (const id of ids) {
      await closePort(id)
    }
    handles.value.clear()
    activePortId.value = null
    stopStatsPolling()
  }

  function clearLocalHandles() {
    handles.value.clear()
    activePortId.value = null
    stopStatsPolling()
  }

  async function refreshHandles() {
    try {
      const list = await serialService.listPorts()
      list.forEach(h => {
        handles.value.set(h.ID, h)
      })
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to refresh handles'
    }
  }

  function clearError() {
    error.value = null
  }

  function startStatsPolling() {
    if (statsInterval) return
    statsInterval = window.setInterval(async () => {
      try {
        const list = await serialService.listPorts()
        list.forEach(h => {
          const handle = handles.value.get(h.ID)
          if (handle) {
            // Update stats in place (creating new object for reactivity)
            handles.value.set(h.ID, {
              ...handle,
              RxBytes: h.RxBytes,
              TxBytes: h.TxBytes,
            })
          }
        })
      } catch (e) {
        console.error('Stats polling failed:', e)
      }
    }, 5000) // Poll every 5 seconds
  }

  function stopStatsPolling() {
    if (statsInterval) {
      clearInterval(statsInterval)
      statsInterval = null
    }
  }

  function initEventListeners() {
    // Event listeners are handled by bufferStore
    // This function ensures stats polling is started if there are open ports
    if (handles.value.size > 0) {
      startStatsPolling()
    }
  }

  function cleanup() {
    stopStatsPolling()
  }

  return {
    ports,
    handles,
    activePortId,
    error,
    activeHandle,
    openHandles,
    refreshPorts,
    refreshHandles,
    openPort,
    openConfig,
    closePort,
    closeAllPorts,
    updatePortConfig,
    setActivePort,
    addRxBytes,
    addTxBytes,
    resetCounters,
    restoreCounters,
    clearLocalHandles,
    clearError,
    initEventListeners,
    stopStatsPolling,
    cleanup,
  }
})
