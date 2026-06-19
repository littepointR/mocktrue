import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  ClearMonitorFrames,
  DeleteMonitor,
  ExportMonitor,
  ListMonitors,
  QueryMonitorFrames,
  SetMonitorAutoSave,
  StartMonitor,
  StopMonitor,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'
import type {
  AutoSaveOptions,
  ExportRequest,
  Frame,
  QueryRequest,
  SessionInfo,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/monitor/models.js'

export type MonitorDisplayMode = 'text' | 'hex' | 'dec' | 'oct' | 'bin'

export interface MonitorFilterState {
  direction: string
  search: string
  displayMode: MonitorDisplayMode
  modbusFunction: number
}

export interface StartBridgeMonitorInput {
  id: string
  name: string
  portA: string
  portB: string
  baudRate: number
  dataBits?: number
  stopBits?: string
  parity?: string
  flowMode?: string
  readBufKB?: number
  encoding?: string
  autoSave?: AutoSaveOptions | null
}

export interface MonitorWorkspaceState {
  activeMonitorId: string | null
  filters: Record<string, MonitorFilterState>
  sessions: SessionInfo[]
  frames: Record<string, Frame[]>
}

export function defaultMonitorAutoSave(): AutoSaveOptions {
  return {
    Enabled: false,
    Path: '',
    Directory: '',
    BaseName: 'serial-monitor',
    Format: 'csv',
    SplitMode: 'none',
    SplitSizeKB: 1024,
    SplitIntervalSeconds: 60,
    Encoding: 'utf-8',
  }
}

export function defaultMonitorFilter(): MonitorFilterState {
  return {
    direction: 'all',
    search: '',
    displayMode: 'hex',
    modbusFunction: 0,
  }
}

export const useMonitorStore = defineStore('serialMonitor', () => {
  const sessions = ref<Map<string, SessionInfo>>(new Map())
  const framesByMonitor = ref<Map<string, Frame[]>>(new Map())
  const frameTotals = ref<Map<string, number>>(new Map())
  const filters = ref<Record<string, MonitorFilterState>>({})
  const activeMonitorId = ref<string | null>(null)
  const error = ref<string | null>(null)
  let pollTimer: number | null = null

  const sessionList = computed(() => Array.from(sessions.value.values()))
  const runningSessions = computed(() => sessionList.value.filter(session => session.Status === 'running'))

  function filterFor(id: string): MonitorFilterState {
    if (!filters.value[id]) {
      filters.value[id] = defaultMonitorFilter()
    }
    return filters.value[id]
  }

  function setActiveMonitor(id: string | null) {
    activeMonitorId.value = id
  }

  function setFilter(id: string, patch: Partial<MonitorFilterState>) {
    filters.value[id] = { ...filterFor(id), ...patch }
  }

  async function startBridgeMonitor(input: StartBridgeMonitorInput): Promise<string> {
    try {
      const session = await StartMonitor({
        ID: input.id,
        Name: input.name,
        Provider: 'bridge',
        PortA: input.portA,
        PortB: input.portB,
        EndpointA: '',
        EndpointB: '',
        Config: {
          PortName: '',
          BaudRate: input.baudRate,
          DataBits: input.dataBits ?? 8,
          StopBits: input.stopBits ?? '1',
          Parity: input.parity ?? 'none',
          FlowMode: input.flowMode ?? 'none',
          ReadBufKB: input.readBufKB ?? 32,
        },
        Encoding: input.encoding ?? 'utf-8',
        AutoSave: input.autoSave ?? null,
      })
      if (!session) throw new Error('monitor did not return session')
      sessions.value.set(session.ID, session)
      framesByMonitor.value.set(session.ID, [])
      frameTotals.value.set(session.ID, 0)
      activeMonitorId.value = session.ID
      error.value = null
      startPolling()
      return session.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to start monitor'
      throw e
    }
  }

  async function refreshSessions() {
    try {
      const list = await ListMonitors()
      const next = new Map(sessions.value)
      for (const session of list ?? []) {
        next.set(session.ID, session)
      }
      sessions.value = next
      error.value = null
      if (runningSessions.value.length > 0) startPolling()
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to list monitors'
    }
  }

  async function refreshFrames(id: string, patch?: Partial<MonitorFilterState>) {
    if (patch) setFilter(id, patch)
    const filter = filterFor(id)
    const req: QueryRequest = {
      MonitorID: id,
      Offset: 0,
      Limit: 1000,
      Direction: filter.direction,
      Search: filter.search,
      ModbusFunction: filter.modbusFunction,
    }
    try {
      const page = await QueryMonitorFrames(req)
      framesByMonitor.value.set(id, page?.Frames ?? [])
      frameTotals.value.set(id, page?.Total ?? 0)
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to query monitor frames'
    }
  }

  async function stopMonitor(id: string) {
    try {
      await StopMonitor(id)
      await refreshSessions()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to stop monitor'
      throw e
    }
  }

  async function deleteMonitor(id: string) {
    try {
      await DeleteMonitor(id)
      sessions.value.delete(id)
      framesByMonitor.value.delete(id)
      frameTotals.value.delete(id)
      delete filters.value[id]
      if (activeMonitorId.value === id) {
        activeMonitorId.value = sessionList.value[0]?.ID ?? null
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to delete monitor'
      throw e
    }
  }

  async function clearFrames(id: string) {
    try {
      await ClearMonitorFrames(id)
      framesByMonitor.value.set(id, [])
      frameTotals.value.set(id, 0)
      const session = sessions.value.get(id)
      if (session) {
        sessions.value.set(id, { ...session, RxBytes: 0, TxBytes: 0, FrameCount: 0 })
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to clear monitor frames'
      throw e
    }
  }

  async function exportMonitor(request: ExportRequest): Promise<string> {
    try {
      const path = await ExportMonitor(request)
      error.value = null
      return path
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to export monitor'
      throw e
    }
  }

  async function setAutoSave(id: string, options: AutoSaveOptions) {
    try {
      const session = await SetMonitorAutoSave({ MonitorID: id, Options: options })
      if (session) sessions.value.set(session.ID, session)
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to set monitor auto save'
      throw e
    }
  }

  function startPolling() {
    if (pollTimer !== null || typeof window === 'undefined') return
    pollTimer = window.setInterval(async () => {
      await refreshSessions()
      for (const session of runningSessions.value) {
        await refreshFrames(session.ID)
      }
      if (runningSessions.value.length === 0) {
        stopPolling()
      }
    }, 500)
  }

  function stopPolling() {
    if (pollTimer === null || typeof window === 'undefined') return
    clearInterval(pollTimer)
    pollTimer = null
  }

  function exportState(): MonitorWorkspaceState {
    return {
      activeMonitorId: activeMonitorId.value,
      filters: filters.value,
      sessions: sessionList.value.map(session => ({ ...session, Status: 'stopped' })),
      frames: Object.fromEntries(Array.from(framesByMonitor.value.entries()).map(([id, frames]) => [
        id,
        frames.map(frame => ({ ...frame })),
      ])),
    }
  }

  function restoreState(snapshot?: MonitorWorkspaceState) {
    sessions.value = new Map()
    framesByMonitor.value = new Map()
    frameTotals.value = new Map()
    filters.value = {}
    activeMonitorId.value = null
    if (!snapshot) return
    for (const session of snapshot.sessions ?? []) {
      sessions.value.set(session.ID, { ...session, Status: 'stopped' })
    }
    for (const [id, frames] of Object.entries(snapshot.frames ?? {})) {
      framesByMonitor.value.set(id, frames.map(frame => ({ ...frame })))
      frameTotals.value.set(id, frames.length)
    }
    filters.value = { ...(snapshot.filters ?? {}) }
    activeMonitorId.value = snapshot.activeMonitorId
  }

  function cleanup() {
    stopPolling()
  }

  function clearError() {
    error.value = null
  }

  return {
    sessions,
    sessionList,
    runningSessions,
    framesByMonitor,
    frameTotals,
    filters,
    activeMonitorId,
    error,
    filterFor,
    setActiveMonitor,
    setFilter,
    startBridgeMonitor,
    refreshSessions,
    refreshFrames,
    stopMonitor,
    deleteMonitor,
    clearFrames,
    exportMonitor,
    setAutoSave,
    startPolling,
    stopPolling,
    exportState,
    restoreState,
    cleanup,
    clearError,
  }
})
