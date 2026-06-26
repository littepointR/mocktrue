import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  ClearMonitorFrames,
  DeleteMonitor,
  ListMonitors,
  QueryMonitorFrames,
  StartAutoVirtualMonitor,
  StartMonitor,
  StopMonitor,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'
import type {
  Frame,
  QueryRequest,
  SessionInfo,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/monitor/models.js'

export type MonitorDisplayMode = 'text' | 'hex' | 'dec' | 'oct' | 'bin'

export interface MonitorFilterState {
  direction: string
  search: string
  displayMode: MonitorDisplayMode
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
}

export interface StartAutoVirtualMonitorInput {
  id: string
  name: string
  sourcePort: string
  baudRate: number
  dataBits?: number
  stopBits?: string
  parity?: string
  flowMode?: string
  readBufKB?: number
  encoding?: string
}

export interface MonitorWorkspaceState {
  activeMonitorId: string | null
  filters: Record<string, MonitorFilterState>
  sessions: SessionInfo[]
  frames: Record<string, Frame[]>
}

export function defaultMonitorFilter(): MonitorFilterState {
  return {
    direction: 'all',
    search: '',
    displayMode: 'hex',
  }
}

export const useMonitorStore = defineStore('serialMonitor', () => {
  const sessions = ref<Map<string, SessionInfo>>(new Map())
  const rawFramesByMonitor = ref<Map<string, Frame[]>>(new Map())
  const framesByMonitor = ref<Map<string, Frame[]>>(new Map())
  const frameTotals = ref<Map<string, number>>(new Map())
  const localOnlyMonitorIds = ref<Set<string>>(new Set())
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

  function cloneFrame(frame: Frame): Frame {
    return { ...frame }
  }

  function frameMatchesLocalFilter(frame: Frame, filter: MonitorFilterState): boolean {
    if (filter.direction !== 'all' && frame.Direction !== filter.direction) return false
    const query = filter.search.trim().toLowerCase()
    if (!query) return true
    return [
      frame.DisplayText,
      frame.DisplayHex,
      frame.DisplayDec,
      frame.DisplayOct,
      frame.DisplayBin,
      frame.Port,
    ].some(value => String(value ?? '').toLowerCase().includes(query))
  }

  function applyLocalFrameFilter(id: string) {
    const source = rawFramesByMonitor.value.get(id) ?? framesByMonitor.value.get(id) ?? []
    const filter = filterFor(id)
    const next = source.filter(frame => frameMatchesLocalFilter(frame, filter)).map(cloneFrame)
    framesByMonitor.value.set(id, next)
    frameTotals.value.set(id, next.length)
  }

  function isUnfiltered(filter: MonitorFilterState): boolean {
    return filter.direction === 'all' && filter.search.trim() === ''
  }

  function isMonitorNotFound(error: unknown): boolean {
    const message = String((error as any)?.message ?? error ?? '')
    return message.includes('not_found') && message.includes('monitor not found')
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
        ExternalPort: '',
        AutoVirtualPortID: '',
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
      })
      if (!session) throw new Error('monitor did not return session')
      sessions.value.set(session.ID, session)
      localOnlyMonitorIds.value.delete(session.ID)
      rawFramesByMonitor.value.set(session.ID, [])
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

  async function startAutoVirtualMonitor(input: StartAutoVirtualMonitorInput): Promise<string> {
    try {
      const session = await StartAutoVirtualMonitor({
        ID: input.id,
        Name: input.name,
        Port: input.sourcePort,
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
      })
      if (!session) throw new Error('monitor did not return session')
      sessions.value.set(session.ID, session)
      localOnlyMonitorIds.value.delete(session.ID)
      rawFramesByMonitor.value.set(session.ID, [])
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
        localOnlyMonitorIds.value.delete(session.ID)
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
    const isLocalOnly = localOnlyMonitorIds.value.has(id)
    const hasLocalFrames = rawFramesByMonitor.value.has(id) || framesByMonitor.value.has(id)
    if (isLocalOnly || (!sessions.value.has(id) && hasLocalFrames)) {
      applyLocalFrameFilter(id)
      error.value = null
      return
    }
    const req: QueryRequest = {
      MonitorID: id,
      Offset: 0,
      Limit: 1000,
      Direction: filter.direction,
      Search: filter.search,
    }
    try {
      const page = await QueryMonitorFrames(req)
      const frames = (page?.Frames ?? []).map(cloneFrame)
      framesByMonitor.value.set(id, frames)
      frameTotals.value.set(id, page?.Total ?? 0)
      if (isUnfiltered(filter)) {
        rawFramesByMonitor.value.set(id, frames.map(cloneFrame))
      }
      error.value = null
    } catch (e: any) {
      if (isMonitorNotFound(e) && hasLocalFrames) {
        localOnlyMonitorIds.value.add(id)
        applyLocalFrameFilter(id)
        error.value = null
        return
      }
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
      localOnlyMonitorIds.value.delete(id)
      rawFramesByMonitor.value.delete(id)
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
      rawFramesByMonitor.value.set(id, [])
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
      frames: Object.fromEntries(sessionList.value.map(session => {
        const frames = rawFramesByMonitor.value.get(session.ID) ?? framesByMonitor.value.get(session.ID) ?? []
        return [session.ID, frames.map(cloneFrame)]
      })),
    }
  }

  function restoreState(snapshot?: MonitorWorkspaceState) {
    sessions.value = new Map()
    rawFramesByMonitor.value = new Map()
    framesByMonitor.value = new Map()
    frameTotals.value = new Map()
    localOnlyMonitorIds.value = new Set()
    filters.value = {}
    activeMonitorId.value = null
    if (!snapshot) return
    for (const session of snapshot.sessions ?? []) {
      sessions.value.set(session.ID, { ...session, Status: 'stopped' })
      localOnlyMonitorIds.value.add(session.ID)
    }
    for (const [id, frames] of Object.entries(snapshot.frames ?? {})) {
      localOnlyMonitorIds.value.add(id)
      rawFramesByMonitor.value.set(id, frames.map(cloneFrame))
      framesByMonitor.value.set(id, frames.map(cloneFrame))
      frameTotals.value.set(id, frames.length)
    }
    filters.value = { ...(snapshot.filters ?? {}) }
    activeMonitorId.value = snapshot.activeMonitorId
    for (const id of rawFramesByMonitor.value.keys()) {
      applyLocalFrameFilter(id)
    }
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
    startAutoVirtualMonitor,
    refreshSessions,
    refreshFrames,
    stopMonitor,
    deleteMonitor,
    clearFrames,
    startPolling,
    stopPolling,
    exportState,
    restoreState,
    cleanup,
    clearError,
  }
})
