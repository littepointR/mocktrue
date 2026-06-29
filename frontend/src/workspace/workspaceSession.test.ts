import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { buildGraphTabSnapshot, buildWorkspaceSnapshot, graphTabSnapshotsFromUnknown, restoreGraphTabSnapshot, restoreWorkspaceSnapshot } from './workspaceSession'
import { base64ToBytes, graphTabKind } from './workspaceSnapshot'
import { useSettingsStore } from '../settings/stores/settingsStore'
import { useSerialStore } from '../serial/stores/serialStore'
import { useBufferStore } from '../serial/stores/bufferStore'
import { useVirtualStore } from '../serial/stores/virtualStore'
import { useSerialWorkspaceStore } from '../serial/stores/workspaceStore'
import { useMonitorStore } from '../serial/stores/monitorStore'
import { defaultModbusWorkspaceState, useModbusStore } from '../serial/stores/modbusStore'
import { defaultFecbusWorkspaceState, useFecbusStore } from '../serial/stores/fecbusStore'
import { useSerialGraphStore } from '../serial/stores/graphStore'
import { defaultSerialGraphState, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { createDemoWorkspaceSnapshot } from './demoWorkspaces'

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
  listFecbusSessions: vi.fn(async () => []),
  queryFecbusFrames: vi.fn(async () => ({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })),
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

vi.mock('../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => serialBindingsMock)

describe('workspace session snapshot', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('builds a snapshot containing module settings, handles, layout, virtual resources, and received bytes without theme', () => {
    useSettingsStore().updateGlobal({ Theme: 'light' })
    useSettingsStore().updateSerial({ TerminalFontSize: 17 })
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
    const sender = useSerialGraphStore().addNode('serial.virtual', { x: 20, y: 24 })
    const receiver = useSerialGraphStore().addNode('serial.virtual', { x: 320, y: 24 })
    useSerialGraphStore().connect(sender.id, 'rx', receiver.id, 'tx')
    useMonitorStore().restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: 'aa', displayMode: 'hex' } },
      sessions: [],
      frames: {},
    })

    const snapshot = buildWorkspaceSnapshot()
    const graph = activeGraph(snapshot.serial.graph)

    expect(snapshot.kind).toBe('portweave.workspace.v1')
    expect((snapshot.settings as any).global).toBeUndefined()
    expect(snapshot.settings.serial.TerminalFontSize).toBe(17)
    expect(snapshot.serial.handles[0].id).toBe('port-1')
    expect(snapshot.serial.virtualPorts[0].ID).toBe('vp-1')
    expect(Array.from(base64ToBytes(snapshot.serial.buffers['port-1'][0].data))).toEqual([1, 2, 3])
    expect(snapshot.serial.monitors.activeMonitorId).toBe('mon-1')
    expect(snapshot.serial.modbus.masterForm.functionCode).toBe(3)
    expect(snapshot.serial.fecbus.sendForm.functionCode).toBe(34)
    expect(graph.nodes.map(node => node.type)).toEqual(['serial.virtual', 'serial.virtual'])
    expect(graph.edges).toHaveLength(1)
    expect(snapshot.serial.workspace.tabStates['port-1'].sendHeight).toBe(240)
  })

  it('builds graph tab snapshots from the fallback active graph and raw runtime buffers', () => {
    const graphStore = useSerialGraphStore()
    const sender = graphStore.addNode('serial.virtual')
    graphStore.restoreRuntimeSnapshot('graph-1', {
      nodeBuffers: { [sender.id]: new Uint8Array([90]) },
      nodeFrames: {},
    } as any)
    ;(graphStore as any).activeGraphId = null

    const snapshot = buildGraphTabSnapshot(null)

    expect(snapshot.graph.id).toBe('graph-1')
    expect(Array.from(base64ToBytes(snapshot.runtime.nodeBuffers[sender.id][0].data))).toEqual([90])
  })

  it('throws when building graph tab snapshots without any graph', () => {
    const graphStore = useSerialGraphStore() as any
    graphStore.graphs = []
    graphStore.activeGraphId = null

    expect(() => buildGraphTabSnapshot()).toThrow('serial graph not found')
  })

  it('restores a snapshot with a legacy single graph and remaps old handle ids to newly opened handles', async () => {
    useSettingsStore().updateGlobal({ Theme: 'dark' })
    const result = await restoreWorkspaceSnapshot({
      kind: 'portweave.workspace.v1',
      settings: {
        ...useSettingsStore().snapshot(),
        global: { Theme: 'light' },
      } as any,
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
        modbus: {
          ...defaultModbusWorkspaceState(),
          activeSessionId: 'modbus-1',
          portForm: {
            ...defaultModbusWorkspaceState().portForm,
            port: '/tmp/ttyM0',
          },
        },
        fecbus: {
          ...defaultFecbusWorkspaceState(),
          activeSessionId: 'fec-1',
          portForm: {
            ...defaultFecbusWorkspaceState().portForm,
            port: '/tmp/ttyF0',
          },
        },
        graph: {
          nodes: [
            { id: 'graph-sender', type: 'serial.virtual', position: { x: 20, y: 20 }, config: {} },
            { id: 'graph-receiver', type: 'serial.virtual', position: { x: 300, y: 20 }, config: {} },
          ],
          edges: [{
            id: 'graph-edge',
            source: 'graph-sender',
            sourceHandle: 'rx',
            target: 'graph-receiver',
            targetHandle: 'tx',
          }],
          selectedNodeId: 'graph-receiver',
          selectedEdgeId: null,
        } as any,
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
    expect(useModbusStore().activeSessionId).toBe('modbus-1')
    expect(useModbusStore().portForm.port).toBe('/tmp/ttyM0')
    expect(useFecbusStore().activeSessionId).toBe('fec-1')
    expect(useFecbusStore().portForm.port).toBe('/tmp/ttyF0')
    expect(useSerialGraphStore().nodes.map(node => node.id)).toEqual(['graph-sender', 'graph-receiver'])
    expect(useSerialGraphStore().selectedNodeId).toBe('graph-receiver')
    expect(useSerialGraphStore().exportState()).toEqual({
      graphs: [
        expect.objectContaining({
          id: 'graph-1',
          nodes: [
            expect.objectContaining({ id: 'graph-sender' }),
            expect.objectContaining({ id: 'graph-receiver' }),
          ],
          edges: [expect.objectContaining({ id: 'graph-edge' })],
          selectedNodeId: 'graph-receiver',
          activeNodeTabId: 'graph-receiver',
        }),
      ],
      activeGraphId: 'graph-1',
    })
    expect(useSerialWorkspaceStore().editorLayout).toEqual({ type: 'group', id: 'group-1', tabs: ['new-port'] })
    expect(useSettingsStore().global.Theme).toBe('dark')
    expect(serialServiceMock.restoreCounters).toHaveBeenCalledWith('new-port', 3, 2)
    expect(serialBindingsMock.CreateVirtualPort).toHaveBeenCalledWith('vp-1', 'ttyV0')
    expect(serialBindingsMock.CreateBridge).toHaveBeenCalledWith('br-1', 'ttyV0', '/tmp/ttyA', 115200)
  })

  it('restores legacy snapshots without serial graph as an empty topology', async () => {
    const legacySnapshot = buildWorkspaceSnapshot() as any
    delete legacySnapshot.serial.graph
    useSerialGraphStore().addNode('serial.virtual')

    const result = await restoreWorkspaceSnapshot(legacySnapshot)

    expect(result.errors).toEqual([])
    expect(useSerialGraphStore().exportState()).toEqual(defaultSerialGraphState())
  })

  it('restores serial module settings from a graph tab snapshot without changing global theme', async () => {
    const settings = useSettingsStore()
    settings.updateGlobal({ Theme: 'light' })
    settings.updateSerial({ TerminalFontSize: 11, TextEncoding: 'utf-8' })

    await restoreGraphTabSnapshot({
      kind: 'portweave.graph.v1',
      settings: {
        serial: {
          ...settings.snapshot().serial,
          TerminalFontSize: 21,
          TextEncoding: 'gb18030',
        },
      },
      graph: {
        id: 'graph-settings',
        name: '带设置拓扑',
        nodes: [],
        edges: [],
        selectedNodeId: null,
        selectedEdgeId: null,
        nodeTabs: [],
        activeNodeTabId: null,
      },
      runtime: { nodeBuffers: {}, nodeFrames: {} },
    })

    expect(settings.global.Theme).toBe('light')
    expect(settings.serial.TerminalFontSize).toBe(21)
    expect(settings.serial.TextEncoding).toBe('gb18030')
    expect(useSerialGraphStore().activeGraph?.name).toBe('带设置拓扑')
  })

  it('replaces the active topology when loading different demo snapshots sequentially', async () => {
    const monitorSnapshot = createDemoWorkspaceSnapshot('monitor-demo')
    const bridgeSnapshot = createDemoWorkspaceSnapshot('bridge-demo')
    expect(monitorSnapshot).toBeDefined()
    expect(bridgeSnapshot).toBeDefined()

    await restoreWorkspaceSnapshot(monitorSnapshot!)
    const monitorGraph = useSerialGraphStore().activeGraph
    expect(monitorGraph?.name).toBe('串口监控演示')
    expect(monitorGraph?.nodes.map(node => node.type)).toEqual(expect.arrayContaining([
      'serial.monitor',
    ]))
    expect(monitorGraph?.nodes.map(node => node.type)).not.toContain('serial.tap')
    expect(monitorGraph?.nodes.map(node => node.type)).not.toContain('serial.bridge')

    await restoreWorkspaceSnapshot(bridgeSnapshot!)
    const bridgeGraph = useSerialGraphStore().activeGraph
    expect(bridgeGraph?.name).toBe('串口桥接演示')
    expect(bridgeGraph?.id).not.toBe(monitorGraph?.id)
    expect(bridgeGraph?.nodes.map(node => node.type)).toEqual(expect.arrayContaining([
      'serial.bridge',
    ]))
    expect(bridgeGraph?.nodes.map(node => node.type)).toContain('serial.monitor')

    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.editorLayout.tabs).toEqual([`graph:${bridgeGraph?.id}`])
      expect(workspace.activeByGroup[workspace.editorLayout.id]).toBe(`graph:${bridgeGraph?.id}`)
    }
  })

  it('exports and restores graph tab runtime buffers, frames, and fallback workspace graph snapshots', async () => {
    const graphStore = useSerialGraphStore()
    const sender = graphStore.addNode('serial.virtual')
    graphStore.restoreRuntimeSnapshot('graph-1', {
      nodeBuffers: { [sender.id]: new Uint8Array([65]) },
      nodeBufferChunks: {
        [sender.id]: [
          { offset: 0, timestamp: 123 as any, data: new Uint8Array([65]) },
          { offset: 1, timestamp: '2026-06-23T02:01:02.003', data: new Uint8Array([66, 67]) },
          { offset: 3, timestamp: 'invalid-date', data: new Uint8Array([68]) },
        ],
        'empty-node': [],
      },
      nodeFrames: { [sender.id]: [{ Seq: 1, Direction: 'tx', Length: 1, DisplayHex: '41' } as any] },
    })

    const snapshot = buildGraphTabSnapshot('graph-1')

    expect(snapshot.kind).toBe('portweave.graph.v1')
    expect(snapshot.runtime.nodeBuffers[sender.id].map(chunk => Array.from(base64ToBytes(chunk.data)))).toEqual([[65], [66, 67], [68]])
    expect(snapshot.runtime.nodeBuffers[sender.id][2].timestamp).toBe(0)
    expect(Array.from(base64ToBytes(snapshot.runtime.nodeBuffers['empty-node'][0].data))).toEqual([])
    expect(snapshot.runtime.nodeFrames[sender.id]).toEqual([expect.objectContaining({ Seq: 1, DisplayHex: '41' })])

    const restored = await restoreGraphTabSnapshot(snapshot, { activate: false })

    expect(restored.graphIds).toEqual(['graph-2'])
    expect(restored.activeGraphId).toBe('graph-2')
    expect(graphStore.activeGraphId).toBe('graph-1')
    const runtime = graphStore.exportRuntimeSnapshot('graph-2')
    expect(Array.from(runtime.nodeBuffers[sender.id])).toEqual([65, 66, 67, 68])
    expect(runtime.nodeFrames[sender.id]).toEqual([expect.objectContaining({ Seq: 1, DisplayHex: '41' })])

    const fallback = await restoreGraphTabSnapshot({
      kind: 'portweave.workspace.v1',
      settings: { serial: useSettingsStore().snapshot().serial },
      serial: {
        activePortId: null,
        handles: [],
        virtualPorts: [],
        bridges: [],
        buffers: {},
        monitors: { activeMonitorId: null, filters: {}, sessions: [], frames: {} },
        modbus: defaultModbusWorkspaceState(),
        fecbus: defaultFecbusWorkspaceState(),
        graph: graphStore.exportState(),
        workspace: useSerialWorkspaceStore().exportState(),
      },
    })

    expect(fallback.graphIds.length).toBeGreaterThan(0)
    expect(fallback.activeGraphId).toBe(fallback.graphIds[fallback.graphIds.length - 1])
  })

  it('preserves graph, node, edge, and runtime identities across graph tab export and restore', async () => {
    const graphStore = useSerialGraphStore()
    const graph = {
      id: 'identity-graph',
      name: 'Identity Graph',
      nodes: [
        {
          id: 'identity-generator',
          type: 'serial.script.generator',
          position: { x: 20, y: 40 },
          config: { script: 'output.text("id", "utf-8")', autoRun: true },
        },
        {
          id: 'identity-virtual',
          type: 'serial.virtual',
          position: { x: 260, y: 40 },
          config: { portName: 'identity-vport' },
        },
      ],
      edges: [{
        id: 'identity-edge-generator-virtual',
        source: 'identity-generator',
        sourceHandle: 'out',
        target: 'identity-virtual',
        targetHandle: 'tx',
      }],
      selectedNodeId: 'identity-virtual',
      selectedEdgeId: 'identity-edge-generator-virtual',
      nodeTabs: [
        { nodeId: 'identity-generator', title: 'Generator' },
        { nodeId: 'identity-virtual', title: 'Virtual' },
      ],
      activeNodeTabId: 'identity-virtual',
    }
    graphStore.restoreState({ graphs: [graph], activeGraphId: graph.id })
    graphStore.restoreRuntimeSnapshot(graph.id, {
      nodeBuffers: { 'identity-virtual': new Uint8Array([1, 2, 3]) },
      nodeFrames: { 'identity-virtual': [{ Seq: 9, Direction: 'rx', Length: 3, DisplayHex: '01 02 03' } as any] },
    })

    const snapshot = buildGraphTabSnapshot(graph.id)

    expect(snapshot.kind).toBe(graphTabKind)
    expect(snapshot.graph.id).toBe('identity-graph')
    expect(snapshot.graph.nodes.map(node => node.id)).toEqual(['identity-generator', 'identity-virtual'])
    expect(snapshot.graph.edges.map(edge => edge.id)).toEqual(['identity-edge-generator-virtual'])
    expect(snapshot.runtime.nodeBuffers['identity-virtual'].map(chunk => Array.from(base64ToBytes(chunk.data)))).toEqual([[1, 2, 3]])

    graphStore.resetWorkspace()
    const restored = await restoreGraphTabSnapshot(snapshot)
    const restoredGraph = graphStore.graphById('identity-graph')
    const restoredRuntime = graphStore.exportRuntimeSnapshot('identity-graph')

    expect(restored).toEqual({ graphIds: ['identity-graph'], activeGraphId: 'identity-graph' })
    expect(graphStore.activeGraphId).toBe('identity-graph')
    expect(restoredGraph?.nodes.map(node => node.id)).toEqual(['identity-generator', 'identity-virtual'])
    expect(restoredGraph?.edges).toEqual([expect.objectContaining({
      id: 'identity-edge-generator-virtual',
      source: 'identity-generator',
      sourceHandle: 'out',
      target: 'identity-virtual',
      targetHandle: 'tx',
    })])
    expect(restoredGraph?.selectedNodeId).toBe('identity-virtual')
    expect(restoredGraph?.selectedEdgeId).toBe('identity-edge-generator-virtual')
    expect(restoredGraph?.activeNodeTabId).toBe('identity-virtual')
    expect(Array.from(restoredRuntime.nodeBuffers['identity-virtual'])).toEqual([1, 2, 3])
    expect(restoredRuntime.nodeFrames['identity-virtual']).toEqual([expect.objectContaining({ Seq: 9, DisplayHex: '01 02 03' })])
  })

  it('normalizes sparse graph tab snapshots and workspace graph state fallbacks', async () => {
    const graphStore = useSerialGraphStore()
    const sparse = await restoreGraphTabSnapshot({
      kind: 'portweave.graph.v1',
      graph: {
        id: 'sparse-graph',
        name: '',
        nodes: [],
        edges: [],
        selectedNodeId: null,
        selectedEdgeId: null,
        nodeTabs: [],
        activeNodeTabId: null,
      },
    } as any)
    expect(sparse.graphIds).toEqual(['sparse-graph'])
    expect(graphStore.activeGraphId).toBe('sparse-graph')
    expect(graphStore.exportRuntimeSnapshot('sparse-graph').nodeBuffers).toEqual({})

    const graphA = defaultSerialGraphState().graphs[0]
    const graphB = {
      ...graphA,
      id: 'graph-b',
      name: 'Graph B',
    }
    const graphTabs = graphTabSnapshotsFromUnknown({
      serial: {
        graph: {
          graphs: [graphA, graphB],
          activeGraphId: null,
        },
      },
    } as any)

    expect(graphTabs.map(tab => tab.graph.id)).toEqual([graphA.id, 'graph-b'])
    expect(graphTabs.every(tab => tab.settings.serial.TerminalFontSize)).toBe(true)
  })

  it('rejects unsupported graph tab snapshots and preserves partial restore errors', async () => {
    await expect(restoreGraphTabSnapshot({ kind: 'unknown' } as any)).rejects.toThrow('unsupported PortWeave config file')

    serialBindingsMock.CreateVirtualPort.mockRejectedValueOnce(new Error('virtual denied'))
    serialServiceMock.openPort.mockRejectedValueOnce(new Error('port denied'))

    const result = await restoreWorkspaceSnapshot({
      kind: 'portweave.workspace.v1',
      settings: { serial: useSettingsStore().snapshot().serial },
      serial: {
        activePortId: 'bad-port',
        handles: [{ id: 'bad-port', config: serialConfig('/tmp/bad'), isOpen: true, rxBytes: 0, txBytes: 0 }],
        virtualPorts: [{ ID: 'bad-vp', Port: 'ttyBad' }],
        bridges: [],
        buffers: {},
        monitors: { activeMonitorId: null, filters: {}, sessions: [], frames: {} },
        modbus: defaultModbusWorkspaceState(),
        fecbus: defaultFecbusWorkspaceState(),
        graph: defaultSerialGraphState(),
        workspace: useSerialWorkspaceStore().exportState(),
      },
    })

    expect(result.errors).toEqual(expect.arrayContaining([
      { target: 'virtual:bad-vp', message: 'virtual denied' },
      { target: 'port:bad-port', message: 'port denied' },
    ]))
    expect(result.handleMap).toEqual({})
  })

  it('captures non-Error restore failures as readable messages', async () => {
    serialBindingsMock.CreateVirtualPort.mockRejectedValueOnce('virtual string denied')

    const result = await restoreWorkspaceSnapshot({
      kind: 'portweave.workspace.v1',
      settings: { serial: useSettingsStore().snapshot().serial },
      serial: {
        activePortId: null,
        handles: [],
        virtualPorts: [{ ID: 'bad-vp', Port: 'ttyBad' }],
        bridges: [],
        buffers: {},
        monitors: { activeMonitorId: null, filters: {}, sessions: [], frames: {} },
        modbus: defaultModbusWorkspaceState(),
        fecbus: defaultFecbusWorkspaceState(),
        graph: defaultSerialGraphState(),
        workspace: useSerialWorkspaceStore().exportState(),
      },
    })

    expect(result.errors).toContainEqual({ target: 'virtual:bad-vp', message: 'virtual string denied' })
  })

  it('handles empty graph-tab workspaces and closed workspace handles without reopening ports', async () => {
    const restoredGraphs = await restoreGraphTabSnapshot({
      kind: 'portweave.workspace.v1',
      settings: { serial: useSettingsStore().snapshot().serial },
      serial: {
        activePortId: null,
        handles: [],
        virtualPorts: [],
        bridges: [],
        buffers: {},
        monitors: { activeMonitorId: null, filters: {}, sessions: [], frames: {} },
        modbus: defaultModbusWorkspaceState(),
        fecbus: defaultFecbusWorkspaceState(),
        graph: { graphs: [], activeGraphId: null },
        workspace: useSerialWorkspaceStore().exportState(),
      },
    } as any)

    expect(restoredGraphs.graphIds).toEqual(['graph-2'])
    expect(restoredGraphs.activeGraphId).toBe(restoredGraphs.graphIds[0])

    const restoredWorkspace = await restoreWorkspaceSnapshot({
      kind: 'portweave.workspace.v1',
      settings: { serial: useSettingsStore().snapshot().serial },
      serial: {
        activePortId: 'closed-port',
        handles: [{ id: 'closed-port', config: serialConfig('/tmp/closed'), isOpen: false, rxBytes: 7, txBytes: 9 }],
        virtualPorts: [],
        bridges: [],
        buffers: { 'closed-port': [{ timestamp: 123, data: 'AQID' }] },
        monitors: { activeMonitorId: null, filters: {}, sessions: [], frames: {} },
        modbus: defaultModbusWorkspaceState(),
        fecbus: defaultFecbusWorkspaceState(),
        graph: defaultSerialGraphState(),
        workspace: useSerialWorkspaceStore().exportState(),
      },
    })

    expect(restoredWorkspace.handleMap).toEqual({})
    expect(serialServiceMock.openPort).not.toHaveBeenCalled()
    expect(serialServiceMock.restoreCounters).not.toHaveBeenCalled()
    expect(useSerialStore().activePortId).toBe('closed-port')
    expect(Array.from(useBufferStore().getBuffer('closed-port'))).toEqual([1, 2, 3])
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

function activeGraph(state: SerialGraphWorkspaceState) {
  const graph = state.graphs.find(item => item.id === state.activeGraphId)
  expect(graph).toBeDefined()
  return graph!
}
