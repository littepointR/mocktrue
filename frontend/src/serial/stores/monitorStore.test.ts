import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMonitorStore } from './monitorStore'
import type { Frame, SessionInfo } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/monitor/models.js'

const bindings = vi.hoisted(() => ({
  StartMonitor: vi.fn(),
  StartAutoVirtualMonitor: vi.fn(),
  StopMonitor: vi.fn(),
  DeleteMonitor: vi.fn(),
  ListMonitors: vi.fn(),
  QueryMonitorFrames: vi.fn(),
  ClearMonitorFrames: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => bindings)

describe('monitor store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    bindings.ListMonitors.mockResolvedValue([])
    bindings.QueryMonitorFrames.mockResolvedValue({ Frames: [], Total: 0, NextOffset: 0 })
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
