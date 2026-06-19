import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { buildWorkspaceSnapshot, restoreWorkspaceSnapshot } from './workspaceSession'
import { base64ToBytes } from './workspaceSnapshot'
import { useSettingsStore } from '../settings/stores/settingsStore'
import { useSerialStore } from '../serial/stores/serialStore'
import { useBufferStore } from '../serial/stores/bufferStore'
import { useVirtualStore } from '../serial/stores/virtualStore'
import { useSerialWorkspaceStore } from '../serial/stores/workspaceStore'
import { useMonitorStore } from '../serial/stores/monitorStore'

const serialServiceMock = vi.hoisted(() => ({
  openPort: vi.fn(async () => ({
    ID: 'new-port',
    Config: {
      PortName: '/tmp/ttyA',
      BaudRate: 115200,
      DataBits: 8,
      StopBits: '1',
      Parity: 'none',
      FlowMode: 'none',
      ReadBufKB: 32,
    },
    IsOpen: true,
    RxBytes: 0,
    TxBytes: 0,
  })),
  closePort: vi.fn(async () => undefined),
  restoreCounters: vi.fn(async () => undefined),
  listPorts: vi.fn(async () => []),
}))

const serialBindingsMock = vi.hoisted(() => ({
  CreateVirtualPort: vi.fn(async () => ({ ID: 'vp-1', Port: 'ttyV0' })),
  DeleteVirtualPort: vi.fn(async () => undefined),
  ListVirtualPorts: vi.fn(async () => []),
  CreateBridge: vi.fn(async () => ({ ID: 'br-1', Port1: 'ttyV0', Port2: '/tmp/ttyA', BaudRate: 115200 })),
  DeleteBridge: vi.fn(async () => undefined),
  ListBridges: vi.fn(async () => []),
  CleanupVirtual: vi.fn(async () => undefined),
  StartMonitor: vi.fn(async () => null),
  StopMonitor: vi.fn(async () => undefined),
  DeleteMonitor: vi.fn(async () => undefined),
  ListMonitors: vi.fn(async () => []),
  QueryMonitorFrames: vi.fn(async () => ({ Frames: [], Total: 0, NextOffset: 0 })),
  ClearMonitorFrames: vi.fn(async () => undefined),
}))

vi.mock('../serial/services/serialService', () => ({
  serialService: serialServiceMock,
}))

vi.mock('../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => serialBindingsMock)

describe('workspace session snapshot', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('builds a snapshot containing settings, handles, layout, virtual resources, and received bytes', () => {
    useSettingsStore().updateGlobal({ Theme: 'light' })
    const serial = useSerialStore()
    serial.handles.set('port-1', {
      ID: 'port-1',
      Config: serialConfig('/tmp/ttyA'),
      IsOpen: true,
      RxBytes: 3,
      TxBytes: 2,
    })
    serial.setActivePort('port-1')
    const virtual = useVirtualStore()
    virtual.virtualPorts = [{ ID: 'vp-1', Port: 'ttyV0' }]
    virtual.bridges = [{ ID: 'br-1', Port1: 'ttyV0', Port2: '/tmp/ttyA', BaudRate: 115200 }]
    useBufferStore().appendData('port-1', [1, 2, 3], 123)
    useSerialWorkspaceStore().updateTabState('port-1', { sendHeight: 240 })
    useMonitorStore().restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: 'aa', displayMode: 'hex' } },
      sessions: [],
      frames: {},
    })

    const snapshot = buildWorkspaceSnapshot()

    expect(snapshot.kind).toBe('mocktrue.workspace.v1')
    expect(snapshot.settings.global.Theme).toBe('light')
    expect(snapshot.serial.handles[0].id).toBe('port-1')
    expect(snapshot.serial.virtualPorts[0].ID).toBe('vp-1')
    expect(Array.from(base64ToBytes(snapshot.serial.buffers['port-1'][0].data))).toEqual([1, 2, 3])
    expect(snapshot.serial.monitors.activeMonitorId).toBe('mon-1')
    expect(snapshot.serial.workspace.tabStates['port-1'].sendHeight).toBe(240)
  })

  it('restores a snapshot and remaps old handle ids to newly opened handles', async () => {
    const result = await restoreWorkspaceSnapshot({
      kind: 'mocktrue.workspace.v1',
      settings: useSettingsStore().snapshot(),
      serial: {
        activePortId: 'old-port',
        handles: [{
          id: 'old-port',
          config: serialConfig('/tmp/ttyA'),
          isOpen: true,
          rxBytes: 3,
          txBytes: 2,
        }],
        virtualPorts: [{ ID: 'vp-1', Port: 'ttyV0' }],
        bridges: [{ ID: 'br-1', Port1: 'ttyV0', Port2: '/tmp/ttyA', BaudRate: 115200 }],
        buffers: {
          'old-port': [{ timestamp: 123, data: 'AQID' }],
        },
        monitors: {
          activeMonitorId: 'mon-1',
          filters: { 'mon-1': { direction: 'all', search: '01 03', displayMode: 'hex' } },
          sessions: [],
          frames: {},
        },
        workspace: {
          selectedOperation: null,
          editorLayout: { type: 'group', id: 'group-1', tabs: ['old-port'] },
          activeByGroup: { 'group-1': 'old-port' },
          tabStates: {
            'old-port': useSerialWorkspaceStore().tabState('old-port'),
          },
        },
      },
    })

    expect(result.errors).toEqual([])
    expect(result.handleMap['old-port']).toBe('new-port')
    expect(useSerialStore().handles.has('new-port')).toBe(true)
    expect(useSerialStore().activePortId).toBe('new-port')
    expect(Array.from(useBufferStore().getBuffer('new-port'))).toEqual([1, 2, 3])
    expect(useMonitorStore().activeMonitorId).toBe('mon-1')
    expect(useSerialWorkspaceStore().editorLayout).toEqual({ type: 'group', id: 'group-1', tabs: ['new-port'] })
    expect(serialServiceMock.restoreCounters).toHaveBeenCalledWith('new-port', 3, 2)
    expect(serialBindingsMock.CreateVirtualPort).toHaveBeenCalledWith('vp-1', 'ttyV0')
    expect(serialBindingsMock.CreateBridge).toHaveBeenCalledWith('br-1', 'ttyV0', '/tmp/ttyA', 115200)
  })
})

function serialConfig(portName: string) {
  return {
    PortName: portName,
    BaudRate: 115200,
    DataBits: 8,
    StopBits: '1',
    Parity: 'none',
    FlowMode: 'none',
    ReadBufKB: 32,
  }
}
