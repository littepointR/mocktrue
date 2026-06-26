import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useMonitorStore } from './monitorStore'
import type { Frame, SessionInfo } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/monitor/models.js'

const bindings = vi.hoisted(() => ({
  StartMonitor: vi.fn(),
  StartAutoVirtualMonitor: vi.fn(),
  StopMonitor: vi.fn(),
  DeleteMonitor: vi.fn(),
  ListMonitors: vi.fn(),
  QueryMonitorFrames: vi.fn(),
  ClearMonitorFrames: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => bindings)

describe('monitor store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.resetAllMocks()
    bindings.ListMonitors.mockResolvedValue([])
    bindings.QueryMonitorFrames.mockResolvedValue({ Frames: [], Total: 0, NextOffset: 0 })
  })

  afterEach(() => {
    useMonitorStore().cleanup()
    try {
      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('starts a bridge monitor and stores returned session', async () => {
    const session = sampleSession('mon-1')
    bindings.StartMonitor.mockResolvedValue(session)
    const store = useMonitorStore()

    const id = await store.startBridgeMonitor({
      id: 'mon-1',
      name: '测试监控',
      portA: '/tmp/a',
      portB: '/tmp/b',
      baudRate: 115200,
      encoding: 'utf-8',
    })

    expect(id).toBe('mon-1')
    expect(store.sessions.get('mon-1')?.Name).toBe('测试监控')
    expect(bindings.StartMonitor).toHaveBeenCalledWith(expect.objectContaining({
      ID: 'mon-1',
      Provider: 'bridge',
      PortA: '/tmp/a',
      PortB: '/tmp/b',
      Encoding: 'utf-8',
      Config: expect.objectContaining({ BaudRate: 115200, DataBits: 8 }),
    }))
  })

  it('starts an auto virtual monitor from a single source port', async () => {
    const session = {
      ...sampleSession('mon-auto'),
      PortA: '/dev/tty.usbserial',
      PortB: '/tmp/portweave-tty-usbserial',
      ExternalPort: '/tmp/portweave-tty-usbserial',
      AutoVirtualPortID: 'mon-auto-virtual',
    }
    bindings.StartAutoVirtualMonitor.mockResolvedValue(session)
    const store = useMonitorStore()

    const id = await store.startAutoVirtualMonitor({
      id: 'mon-auto',
      name: 'USB 串口监听',
      sourcePort: '/dev/tty.usbserial',
      baudRate: 115200,
      encoding: 'utf-8',
    })

    expect(id).toBe('mon-auto')
    expect(store.sessions.get('mon-auto')?.ExternalPort).toBe('/tmp/portweave-tty-usbserial')
    expect(bindings.StartAutoVirtualMonitor).toHaveBeenCalledWith(expect.objectContaining({
      ID: 'mon-auto',
      Name: 'USB 串口监听',
      Port: '/dev/tty.usbserial',
      Encoding: 'utf-8',
      Config: expect.objectContaining({ BaudRate: 115200, DataBits: 8 }),
    }))
  })

  it('records monitor start failures when the backend omits or rejects sessions', async () => {
    bindings.StartMonitor.mockResolvedValueOnce(undefined)
    bindings.StartAutoVirtualMonitor.mockRejectedValueOnce(undefined)
    const store = useMonitorStore()

    await expect(store.startBridgeMonitor({
      id: 'mon-missing',
      name: 'missing session',
      portA: '/tmp/a',
      portB: '/tmp/b',
      baudRate: 9600,
    })).rejects.toThrow('monitor did not return session')
    expect(store.error).toBe('monitor did not return session')

    await expect(store.startAutoVirtualMonitor({
      id: 'mon-auto-fail',
      name: 'auto fail',
      sourcePort: '/dev/tty.usbserial',
      baudRate: 9600,
    })).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to start monitor')
  })

  it('queries frames with filters and replaces local frame page', async () => {
    const frame = sampleFrame(1, 'a_to_b', 'aa bb')
    bindings.QueryMonitorFrames.mockResolvedValue({ Frames: [frame], Total: 1, NextOffset: 1 })
    const store = useMonitorStore()

    await store.refreshFrames('mon-1', { direction: 'a_to_b', search: 'aa' })

    expect(store.framesByMonitor.get('mon-1')).toEqual([frame])
    expect(store.frameTotals.get('mon-1')).toBe(1)
    expect(bindings.QueryMonitorFrames).toHaveBeenCalledWith(expect.objectContaining({
      MonitorID: 'mon-1',
      Direction: 'a_to_b',
      Search: 'aa',
    }))
  })

  it('refreshes sessions and surfaces fallback list errors', async () => {
    bindings.ListMonitors.mockResolvedValueOnce([{ ...sampleSession('mon-1'), Status: 'stopped' }])
    const store = useMonitorStore()

    await store.refreshSessions()

    expect(store.sessionList.map(session => session.ID)).toEqual(['mon-1'])
    expect(store.runningSessions).toEqual([])
    expect(store.error).toBeNull()

    bindings.ListMonitors.mockRejectedValueOnce(undefined)

    await store.refreshSessions()

    expect(store.error).toBe('Failed to list monitors')
    store.clearError()
    expect(store.error).toBeNull()
  })

  it('falls back to local frames when restored backend sessions disappear', async () => {
    const matching = { ...sampleFrame(1, 'a_to_b', 'aa bb'), DisplayText: 'alpha' }
    const other = { ...sampleFrame(2, 'b_to_a', 'cc dd'), DisplayText: 'beta' }
    bindings.QueryMonitorFrames.mockRejectedValueOnce(new Error('not_found: monitor not found'))
    const store = useMonitorStore()
    store.sessions.set('mon-1', sampleSession('mon-1'))
    store.framesByMonitor.set('mon-1', [matching, other])

    await store.refreshFrames('mon-1', { direction: 'a_to_b', search: 'alpha' })

    expect(bindings.QueryMonitorFrames).toHaveBeenCalledWith(expect.objectContaining({
      MonitorID: 'mon-1',
      Direction: 'a_to_b',
      Search: 'alpha',
    }))
    expect(store.error).toBeNull()
    expect(store.frameTotals.get('mon-1')).toBe(1)
    expect(store.framesByMonitor.get('mon-1')?.map(frame => frame.Seq)).toEqual([1])
  })

  it('stops monitors through refresh and stores stop failures', async () => {
    bindings.ListMonitors.mockResolvedValueOnce([{ ...sampleSession('mon-1'), Status: 'stopped' }])
    const store = useMonitorStore()

    await store.stopMonitor('mon-1')

    expect(bindings.StopMonitor).toHaveBeenCalledWith('mon-1')
    expect(store.sessions.get('mon-1')?.Status).toBe('stopped')
    expect(store.error).toBeNull()

    bindings.StopMonitor.mockRejectedValueOnce(undefined)

    await expect(store.stopMonitor('mon-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to stop monitor')
  })

  it('clears frames and deletes monitors while preserving remaining session selection', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'a_to_b', search: 'aa', displayMode: 'hex' } },
      sessions: [sampleSession('mon-1'), sampleSession('mon-2')],
      frames: { 'mon-1': [sampleFrame(1, 'a_to_b', 'aa bb')] },
    })

    await store.clearFrames('mon-1')

    expect(bindings.ClearMonitorFrames).toHaveBeenCalledWith('mon-1')
    expect(store.framesByMonitor.get('mon-1')).toEqual([])
    expect(store.frameTotals.get('mon-1')).toBe(0)
    expect(store.sessions.get('mon-1')).toEqual(expect.objectContaining({
      RxBytes: 0,
      TxBytes: 0,
      FrameCount: 0,
    }))

    await store.deleteMonitor('mon-1')

    expect(bindings.DeleteMonitor).toHaveBeenCalledWith('mon-1')
    expect(store.sessions.has('mon-1')).toBe(false)
    expect(store.framesByMonitor.has('mon-1')).toBe(false)
    expect(store.filters['mon-1']).toBeUndefined()
    expect(store.activeMonitorId).toBe('mon-2')

    bindings.ClearMonitorFrames.mockRejectedValueOnce(undefined)
    await expect(store.clearFrames('mon-2')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to clear monitor frames')

    bindings.DeleteMonitor.mockRejectedValueOnce(new Error('delete denied'))
    await expect(store.deleteMonitor('mon-2')).rejects.toThrow('delete denied')
    expect(store.error).toBe('delete denied')
  })

  it('filters restored monitor frames locally without querying missing backend sessions', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: 'aa', displayMode: 'hex' } },
      sessions: [{ ...sampleSession('mon-1'), Status: 'stopped' }],
      frames: {
        'mon-1': [
          { ...sampleFrame(1, 'a_to_b', 'aa bb'), DisplayText: 'alpha' },
          { ...sampleFrame(2, 'b_to_a', 'cc dd'), DisplayText: 'beta' },
        ],
      },
    })
    bindings.QueryMonitorFrames.mockRejectedValue(new Error('not_found: monitor not found'))

    await store.refreshFrames('mon-1', { search: '' })

    expect(bindings.QueryMonitorFrames).not.toHaveBeenCalled()
    expect(store.error).toBeNull()
    expect(store.frameTotals.get('mon-1')).toBe(2)
    expect(store.framesByMonitor.get('mon-1')?.map(frame => frame.Seq)).toEqual([1, 2])
  })

  it('exports and restores monitor workspace state', () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'b_to_a', search: 'beta', displayMode: 'hex' } },
      sessions: [sampleSession('mon-1')],
      frames: { 'mon-1': [sampleFrame(1, 'b_to_a', '62 65 74 61')] },
    })

    const snapshot = store.exportState()

    expect(snapshot.activeMonitorId).toBe('mon-1')
    expect(snapshot.filters['mon-1'].search).toBe('beta')
    expect(snapshot.frames['mon-1'][0].DisplayHex).toBe('62 65 74 61')
  })

  it('handles empty backend pages and local fallback searches across sparse frame fields', async () => {
    bindings.QueryMonitorFrames.mockResolvedValueOnce(undefined)
    const store = useMonitorStore()

    await store.refreshFrames('backend-empty')

    expect(store.framesByMonitor.get('backend-empty')).toEqual([])
    expect(store.frameTotals.get('backend-empty')).toBe(0)
    expect(store.error).toBeNull()

    store.restoreState({
      activeMonitorId: 'local-only',
      filters: { 'local-only': { direction: 'all', search: '/tmp/a', displayMode: 'hex' } },
      sessions: [],
      frames: {
        'local-only': [
          { ...sampleFrame(1, 'a_to_b', 'aa'), DisplayText: undefined as any, DisplayHex: undefined as any },
          { ...sampleFrame(2, 'b_to_a', 'bb'), Port: '/tmp/b' },
        ],
      },
    })

    await store.refreshFrames('local-only')

    expect(bindings.QueryMonitorFrames).toHaveBeenCalledTimes(1)
    expect(store.framesByMonitor.get('local-only')?.map(frame => frame.Seq)).toEqual([1])
  })

  it('clears missing monitors and resets active selection when the last monitor is deleted', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: '', displayMode: 'hex' } },
      sessions: [sampleSession('mon-1')],
      frames: {},
    })

    await store.clearFrames('missing-monitor')
    expect(bindings.ClearMonitorFrames).toHaveBeenCalledWith('missing-monitor')
    expect(store.frameTotals.get('missing-monitor')).toBe(0)

    await store.deleteMonitor('mon-1')

    expect(store.activeMonitorId).toBeNull()
    expect(store.error).toBeNull()
  })

  it('passes explicit serial options for bridge and auto virtual monitor starts', async () => {
    bindings.StartMonitor.mockResolvedValueOnce(sampleSession('bridge-explicit'))
    bindings.StartAutoVirtualMonitor.mockResolvedValueOnce(sampleSession('auto-explicit'))
    const store = useMonitorStore()

    await store.startBridgeMonitor({
      id: 'bridge-explicit',
      name: 'explicit bridge',
      portA: '/tmp/a',
      portB: '/tmp/b',
      baudRate: 57600,
      dataBits: 7,
      stopBits: '2',
      parity: 'even',
      flowMode: 'hw_rtscts',
      readBufKB: 64,
      encoding: 'gb18030',
    })
    await store.startAutoVirtualMonitor({
      id: 'auto-explicit',
      name: 'explicit auto',
      sourcePort: '/dev/tty.usbserial',
      baudRate: 38400,
      dataBits: 6,
      stopBits: '1.5',
      parity: 'odd',
      flowMode: 'xon_xoff',
      readBufKB: 16,
      encoding: 'ascii',
    })

    expect(bindings.StartMonitor).toHaveBeenCalledWith(expect.objectContaining({
      Config: expect.objectContaining({ DataBits: 7, StopBits: '2', Parity: 'even', FlowMode: 'hw_rtscts', ReadBufKB: 64 }),
      Encoding: 'gb18030',
    }))
    expect(bindings.StartAutoVirtualMonitor).toHaveBeenCalledWith(expect.objectContaining({
      Config: expect.objectContaining({ DataBits: 6, StopBits: '1.5', Parity: 'odd', FlowMode: 'xon_xoff', ReadBufKB: 16 }),
      Encoding: 'ascii',
    }))
  })

  it('polls running sessions and stops polling once refresh removes them', async () => {
    bindings.ListMonitors
      .mockResolvedValueOnce([{ ...sampleSession('mon-1'), Status: 'running' }])
      .mockResolvedValueOnce([{ ...sampleSession('mon-1'), Status: 'running' }])
      .mockResolvedValueOnce([{ ...sampleSession('mon-1'), Status: 'stopped' }])
    bindings.QueryMonitorFrames.mockResolvedValue({ Frames: [sampleFrame(1, 'a_to_b', 'aa')], Total: 1, NextOffset: 1 })
    const store = useMonitorStore()

    await store.refreshSessions()
    expect(vi.getTimerCount()).toBe(1)

    await vi.advanceTimersByTimeAsync(500)

    expect(bindings.QueryMonitorFrames).toHaveBeenCalledWith(expect.objectContaining({ MonitorID: 'mon-1' }))

    await vi.advanceTimersByTimeAsync(500)

    expect(vi.getTimerCount()).toBe(0)
  })

  it('records generic frame query errors without converting to local-only fallback', async () => {
    bindings.QueryMonitorFrames.mockRejectedValueOnce(undefined)
    const store = useMonitorStore()
    store.setActiveMonitor('mon-1')
    store.sessions.set('mon-1', sampleSession('mon-1'))
    store.framesByMonitor.set('mon-1', [sampleFrame(1, 'a_to_b', 'aa')])
    store.frameTotals.set('mon-1', 1)

    await store.refreshFrames('mon-1')

    expect(store.error).toBe('Failed to query monitor frames')
    expect(store.frameTotals.get('mon-1')).toBe(1)
  })
})

function sampleSession(id: string): SessionInfo {
  return {
    ID: id,
    Name: '测试监控',
    Provider: 'bridge',
    PortA: '/tmp/a',
    PortB: '/tmp/b',
    ExternalPort: '',
    AutoVirtualPortID: '',
    Config: {
      PortName: '',
      BaudRate: 115200,
      DataBits: 8,
      StopBits: '1',
      Parity: 'none',
      FlowMode: 'none',
      ReadBufKB: 32,
    },
    Encoding: 'utf-8',
    Status: 'running',
    RxBytes: 0,
    TxBytes: 0,
    FrameCount: 0,
    StartedAt: '',
    StoppedAt: '',
    Error: '',
  }
}

function sampleFrame(seq: number, direction: string, hex: string): Frame {
  return {
    Seq: seq,
    Timestamp: '',
    Direction: direction,
    Port: direction === 'a_to_b' ? '/tmp/a' : '/tmp/b',
    Length: hex.split(' ').length,
    Data: '',
    DisplayText: direction,
    DisplayHex: hex,
    DisplayDec: '',
    DisplayOct: '',
    DisplayBin: '',
    Encoding: 'utf-8',
    Error: '',
  }
}
