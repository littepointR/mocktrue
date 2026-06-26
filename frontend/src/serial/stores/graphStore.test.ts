import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSerialGraphStore } from './graphStore'

const bindings = vi.hoisted(() => ({
  StartSerialGraph: vi.fn(),
  StopSerialGraph: vi.fn(),
  GetSerialGraphStatus: vi.fn(),
  SendSerialGraphNode: vi.fn(),
  QuerySerialGraphNodeBuffer: vi.fn(),
  QuerySerialGraphNodeFrames: vi.fn(),
  ClearSerialGraphNodeBuffer: vi.fn(),
  ResetSerialGraphNodeCounters: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => bindings)

describe('serial graph store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    bindings.StartSerialGraph.mockImplementation(async (req: { ID: string }) => ({
      ID: req.ID,
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    }))
    bindings.GetSerialGraphStatus.mockResolvedValue({
      ID: 'graph-1',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    bindings.SendSerialGraphNode.mockResolvedValue(5)
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa('hello'),
      Total: 5,
      EOF: false,
    })
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [{ Seq: 1, Direction: '接收', Length: 5, DisplayText: 'hello', DisplayHex: '68 65 6c 6c 6f' }],
      Total: 1,
      NextOffset: 1,
    })
  })

  it('adds, connects, selects, exports, and restores graph nodes', () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender', { x: 24, y: 32 })
    const receiver = store.addNode('serial.receiver', { x: 320, y: 48 })

    const edge = store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)

    expect(sender.id).toBe('node-1')
    expect(receiver.position).toEqual({ x: 320, y: 48 })
    expect(edge?.id).toBe('edge-1')
    expect(store.selectedNodeId).toBe(receiver.id)
    expect(store.nodeTabs.map(tab => tab.nodeId)).toEqual([sender.id, receiver.id])
    expect(store.activeNodeTabId).toBe(receiver.id)
    expect(store.validationErrors).toEqual([])

    const snapshot = store.exportState()
    setActivePinia(createPinia())
    const restored = useSerialGraphStore()
    restored.restoreState(snapshot)

    expect(restored.nodes.map(node => node.type)).toEqual(['serial.sender', 'serial.receiver'])
    expect(restored.edges).toEqual(snapshot.graphs[0].edges)
    expect(restored.selectedNodeId).toBe(receiver.id)
  })

  it('creates and switches multiple topology graphs', () => {
    const store = useSerialGraphStore()
    store.renameGraph(store.activeGraphId!, '主拓扑')
    const firstSender = store.addNode('serial.sender')

    const second = store.createGraph('备用拓扑')
    const secondReceiver = store.addNode('serial.receiver')

    expect(store.graphList.map(graph => graph.name)).toEqual(['主拓扑', '备用拓扑'])
    expect(store.activeGraphId).toBe(second.id)
    expect(store.nodes.map(node => node.id)).toEqual([secondReceiver.id])

    store.setActiveGraph('graph-1')

    expect(store.nodes.map(node => node.id)).toEqual([firstSender.id])
  })

  it('duplicates topology graphs without stale runtime node status', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    await store.startRuntime()

    const copy = store.duplicateGraph('graph-1')

    expect(store.graphById('graph-1')?.nodes.find(node => node.id === sender.id)?.status).toBe('running')
    expect(copy?.nodes.find(node => node.id === sender.id)?.status).toBeUndefined()
    expect(copy?.nodes.find(node => node.id === sender.id)?.error).toBeUndefined()
    expect(store.runtimeStateForGraph(copy?.id).runtimeStatus).toBe('idle')
  })

  it('restores legacy single-graph snapshots into the first topology graph', () => {
    const store = useSerialGraphStore()
    store.restoreState({
      id: 'legacy',
      name: '旧拓扑',
      nodes: [
        { id: 'node-1', type: 'serial.sender', position: { x: 0, y: 0 }, config: {} },
      ],
      edges: [],
      selectedNodeId: 'node-1',
      selectedEdgeId: null,
      nodeTabs: [],
      activeNodeTabId: null,
    })

    expect(store.graphList).toEqual([{ id: 'legacy', name: '旧拓扑' }])
    expect(store.activeGraphId).toBe('legacy')
    expect(store.nodes[0].type).toBe('serial.sender')
    expect(store.nodeTabs).toEqual([])
  })

  it('imports duplicate graph documents without activating them and recreates an empty workspace on demand', async () => {
    const store = useSerialGraphStore()
    const imported = store.importGraphDocument({
      id: 'graph-1',
      name: '',
      nodes: [
        { id: 'node-x', type: 'serial.sender', position: { x: 1, y: 2 }, config: {}, status: 'running', error: 'stale' },
      ],
      edges: [],
      selectedNodeId: 'node-x',
      selectedEdgeId: null,
      nodeTabs: [{ nodeId: 'node-x', title: 'stale' }],
      activeNodeTabId: 'node-x',
    } as any, false)

    expect(imported.id).toBe('graph-2')
    expect(imported.name).toBe('拓扑图 2')
    expect(imported.nodes[0].status).toBeUndefined()
    expect(imported.nodes[0].error).toBeUndefined()
    expect(store.activeGraphId).toBe('graph-1')

    await store.removeGraph('graph-1')
    await store.removeGraph('graph-2')
    expect(store.graphList).toEqual([])

    const recreated = store.addNode('serial.sender')
    expect(recreated.id).toBe('node-1')
    expect(store.graphList).toEqual([{ id: 'graph-1', name: '拓扑图 1' }])
  })

  it('reports validation and string backend errors while normalizing runtime statuses', async () => {
    const store = useSerialGraphStore()
    store.restoreState({
      graphs: [{
        id: 'graph-9',
        name: '坏拓扑',
        nodes: [{ id: 'missing-provider', type: 'serial.unknown', position: { x: 0, y: 0 }, config: {} }],
        edges: [{ id: 'edge-bad', source: 'missing-provider', sourceHandle: 'out', target: 'missing-provider', targetHandle: 'in' }],
        selectedNodeId: null,
        selectedEdgeId: null,
        nodeTabs: [],
        activeNodeTabId: null,
      }],
      activeGraphId: 'graph-9',
    } as any)

    await expect(store.startRuntime()).rejects.toThrow('provider not found')
    expect(bindings.StartSerialGraph).not.toHaveBeenCalled()
    expect(store.operationLogs.at(-1)).toEqual(expect.objectContaining({
      action: 'start-error',
      level: 'error',
    }))

    store.resetWorkspace()
    const sender = store.addNode('serial.sender')
    bindings.StartSerialGraph.mockResolvedValueOnce({
      ID: 'graph-1',
      Status: 'paused',
      Error: 'runtime paused',
      Nodes: [{ ID: sender.id, Type: 'serial.sender', Status: 'paused', RxBytes: 0, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' }],
    })
    await store.startRuntime()
    expect(store.runtimeStatus).toBe('idle')
    expect(store.runtimeError).toBe('runtime paused')
    expect(store.nodes[0].status).toBe('idle')
    expect(store.nodes[0].error).toBeUndefined()

    bindings.StopSerialGraph.mockRejectedValueOnce('stop denied')
    await expect(store.stopRuntime()).rejects.toBe('stop denied')
    expect(store.operationLogs.at(-1)).toEqual(expect.objectContaining({
      action: 'stop-error',
      details: 'stop denied',
    }))
  })

  it('adds filter nodes with matcher defaults and restores saved filter options', () => {
    const store = useSerialGraphStore()
    const filter = store.addNode('serial.filter')

    expect(filter.config).toEqual({
      mode: 'plain',
      expression: '',
      caseSensitive: false,
      wholeWord: false,
    })

    store.updateNodeConfig(filter.id, {
      mode: 'regex',
      expression: 'temp\\d+',
      caseSensitive: true,
      wholeWord: true,
    })
    const snapshot = store.exportState()

    setActivePinia(createPinia())
    const restored = useSerialGraphStore()
    restored.restoreState(snapshot)

    expect(restored.nodes[0].type).toBe('serial.filter')
    expect(restored.nodes[0].config).toEqual(expect.objectContaining({
      mode: 'regex',
      expression: 'temp\\d+',
      caseSensitive: true,
      wholeWord: true,
    }))
  })

  it('records runtime actions while keeping sender and receiver payload logging opt-in', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')

    expect(sender.config).toEqual(expect.objectContaining({
      enableLogging: false,
      logLevel: 'info',
      logFormat: '',
    }))
    expect(receiver.config).toEqual(expect.objectContaining({
      enableLogging: false,
      logLevel: 'info',
      logFormat: '',
    }))

    await store.startRuntime()
    await store.sendNode(sender.id, 'hello', 'ascii')
    await store.queryNodeBuffer(receiver.id)
    await store.clearNodeBuffer(receiver.id)
    await store.resetNodeCounters(receiver.id)
    await store.stopRuntime()

    expect(store.operationLogs.map(entry => ({
      level: entry.level,
      source: entry.source,
      action: entry.action,
      nodeId: entry.nodeId,
    }))).toEqual([
      { level: 'info', source: 'serial.graph', action: 'start', nodeId: undefined },
      { level: 'info', source: 'serial.sender', action: 'send-command', nodeId: sender.id },
      { level: 'info', source: 'serial.receiver', action: 'clear-buffer', nodeId: receiver.id },
      { level: 'info', source: 'serial.receiver', action: 'reset-counters', nodeId: receiver.id },
      { level: 'info', source: 'serial.graph', action: 'stop', nodeId: undefined },
    ])
    expect(store.operationLogs.some(entry => entry.action === 'send')).toBe(false)
    expect(store.operationLogs.some(entry => entry.action === 'receive')).toBe(false)
    expect(store.operationLogs.find(entry => entry.action === 'send-command')).toEqual(expect.objectContaining({
      message: '发送器发送 5 bytes',
      payloadText: 'hello',
      payloadHex: '68 65 6c 6c 6f',
      byteLength: 5,
    }))
  })

  it('records runtime start failures as operation log errors', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    bindings.StartSerialGraph.mockRejectedValueOnce(new Error('backend unavailable'))

    await expect(store.startRuntime()).rejects.toThrow('backend unavailable')

    expect(store.operationLogs).toEqual([
      expect.objectContaining({
        level: 'error',
        source: 'serial.graph',
        action: 'start-error',
        category: 'serial.graph',
        message: '拓扑启动失败',
        details: 'backend unavailable',
      }),
    ])
  })

  it('records sender and receiver operation logs when logging is enabled', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.updateNodeConfig(sender.id, { enableLogging: true })
    store.updateNodeConfig(receiver.id, { enableLogging: true })

    await store.startRuntime()
    await store.sendNode(sender.id, 'hello', 'ascii')
    await store.queryNodeBuffer(receiver.id)
    await store.queryNodeBuffer(receiver.id)

    const payloadLogs = store.operationLogs.filter(entry => entry.action === 'send' || entry.action === 'receive')
    expect(payloadLogs).toHaveLength(2)
    expect(payloadLogs[0]).toEqual(expect.objectContaining({
      level: 'info',
      source: 'serial.sender',
      action: 'send',
      category: 'serial.graph',
      nodeId: sender.id,
      direction: 'tx',
      payloadText: 'hello',
      payloadHex: '68 65 6c 6c 6f',
      byteLength: 5,
    }))
    expect(payloadLogs[0].message).toContain('发送器')
    expect(payloadLogs[0].message).toContain('tx')
    expect(payloadLogs[0].message).toContain('hello')
    expect(payloadLogs[1]).toEqual(expect.objectContaining({
      level: 'info',
      source: 'serial.receiver',
      action: 'receive',
      category: 'serial.graph',
      nodeId: receiver.id,
      direction: 'rx',
      payloadText: 'hello',
      payloadHex: '68 65 6c 6c 6f',
      byteLength: 5,
    }))
    expect(payloadLogs[1].message).toContain('接收器')
    expect(payloadLogs[1].message).toContain('rx')
    expect(payloadLogs[1].message).toContain('hello')
  })

  it('formats logging entries with node templates and falls back on invalid templates', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.updateNodeConfig(sender.id, {
      enableLogging: true,
      logLevel: 'debug',
      logFormat: '{direction}:{nodeId}:{length}:{hex}:{text}',
    })
    store.updateNodeConfig(receiver.id, {
      enableLogging: true,
      logFormat: 'received {missing}',
    })
    bindings.SendSerialGraphNode.mockResolvedValueOnce(2)
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValueOnce({
      Offset: 0,
      Data: btoa('hi'),
      Total: 2,
      EOF: false,
    })

    await store.startRuntime()
    await store.sendNode(sender.id, 'hi', 'ascii')
    await store.queryNodeBuffer(receiver.id)

    const payloadLogs = store.operationLogs.filter(entry => entry.action === 'send' || entry.action === 'receive')
    expect(payloadLogs[0]).toEqual(expect.objectContaining({
      level: 'debug',
      message: `tx:${sender.id}:2:68 69:hi`,
    }))
    expect(payloadLogs[1]).toEqual(expect.objectContaining({
      level: 'error',
      action: 'receive',
      payloadText: 'hi',
      byteLength: 2,
    }))
    expect(payloadLogs[1].message).toContain('接收器')
    expect(payloadLogs[1].message).toContain('rx')
    expect(payloadLogs[1].details).toContain('unknown field missing')
    expect(store.nodeBufferText.get(receiver.id)).toBe('hi')
  })

  it('normalizes hex sender payloads the same way as the backend before logging', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    store.updateNodeConfig(sender.id, { enableLogging: true })
    bindings.SendSerialGraphNode.mockResolvedValueOnce(2)

    await store.startRuntime()
    await store.sendNode(sender.id, '6 8 6 9', 'hex')

    const payloadLog = store.operationLogs.find(entry => entry.action === 'send')
    expect(payloadLog).toEqual(expect.objectContaining({
      action: 'send',
      direction: 'tx',
      payloadText: 'hi',
      payloadHex: '68 69',
      byteLength: 2,
    }))
  })

  it('records filter pass and drop operation logs for direct sends', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const filter = store.addNode('serial.filter')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', filter.id, 'in')
    store.connect(filter.id, 'out', receiver.id, 'in')
    store.updateNodeConfig(filter.id, { mode: 'plain', expression: 'ok' })

    await store.startRuntime()
    await store.sendNode(sender.id, 'ok telemetry', 'ascii')
    await store.sendNode(sender.id, 'drop telemetry', 'ascii')

    const filterLogs = store.operationLogs.filter(entry => entry.source === 'serial.filter' && ['pass', 'drop'].includes(entry.action))
    expect(filterLogs.map(entry => ({ source: entry.source, action: entry.action, nodeId: entry.nodeId }))).toEqual([
      { source: 'serial.filter', action: 'pass', nodeId: filter.id },
      { source: 'serial.filter', action: 'drop', nodeId: filter.id },
    ])
    expect(filterLogs[0]).toEqual(expect.objectContaining({
      level: 'debug',
      category: 'serial.graph',
      direction: 'rx',
      payloadText: 'ok telemetry',
      payloadHex: '6f 6b 20 74 65 6c 65 6d 65 74 72 79',
      byteLength: 12,
    }))
    expect(filterLogs[1].message).toContain('drop')
  })

  it('records invalid filter matcher errors without throwing direct sends', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const filter = store.addNode('serial.filter')
    store.connect(sender.id, 'out', filter.id, 'in')
    store.updateNodeConfig(filter.id, { mode: 'regex', expression: '[unterminated' })

    await store.startRuntime()
    await store.sendNode(sender.id, 'anything', 'ascii')

    const filterLogs = store.operationLogs.filter(entry => entry.source === 'serial.filter' && entry.action === 'error')
    expect(filterLogs).toHaveLength(1)
    expect(filterLogs[0]).toEqual(expect.objectContaining({
      level: 'error',
      source: 'serial.filter',
      action: 'error',
      category: 'serial.graph',
      nodeId: filter.id,
      direction: 'rx',
      payloadText: 'anything',
    }))
    expect(filterLogs[0].details).toContain('invalid regex')
  })

  it('removes connected edges when a node is removed', () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')

    store.removeNode(sender.id)

    expect(store.nodes).toHaveLength(1)
    expect(store.edges).toEqual([])
    expect(store.selectedNodeId).toBeNull()
    expect(store.nodeTabs.some(tab => tab.nodeId === sender.id)).toBe(false)
  })

  it('closes node tabs without deleting graph nodes', () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')

    store.closeNodeTab(sender.id)

    expect(store.nodes).toHaveLength(1)
    expect(store.nodeTabs).toEqual([])
    expect(store.activeNodeTabId).toBeNull()
  })

  it('rejects invalid graph connections', () => {
    const store = useSerialGraphStore()
    const master = store.addNode('serial.modbus.master')
    const receiver = store.addNode('serial.receiver')

    const edge = store.connect(master.id, 'registers', receiver.id, 'in')

    expect(edge).toBeNull()
    expect(store.edges).toEqual([])
  })

  it('starts the graph runtime with topology nodes and updates node statuses', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender', { x: 12, y: 18 })
    const receiver = store.addNode('serial.receiver', { x: 260, y: 18 })
    store.connect(sender.id, 'out', receiver.id, 'in')

    await store.startRuntime()

    expect(bindings.StartSerialGraph).toHaveBeenCalledWith(expect.objectContaining({
      ID: 'graph-1',
      Nodes: [
        expect.objectContaining({ ID: sender.id, Type: 'serial.sender', Position: { X: 12, Y: 18 } }),
        expect.objectContaining({ ID: receiver.id, Type: 'serial.receiver', Position: { X: 260, Y: 18 } }),
      ],
      Edges: [
        expect.objectContaining({ Source: sender.id, SourceHandle: 'out', Target: receiver.id, TargetHandle: 'in' }),
      ],
    }))
    expect(store.runtimeStatus).toBe('running')
    expect(store.nodeStatuses.get(receiver.id)?.RxBytes).toBe(5)
    expect(store.nodes.find(node => node.id === receiver.id)?.status).toBe('running')
  })

  it('keeps runtime state isolated per topology graph', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    await store.startRuntime()

    const second = store.createGraph('第二拓扑')
    const secondReceiver = store.addNode('serial.receiver')

    expect(store.runtimeStatus).toBe('idle')
    expect(store.nodeStatuses.get(secondReceiver.id)).toBeUndefined()

    store.setActiveGraph('graph-1')

    expect(store.runtimeStatus).toBe('running')
    expect(store.nodeStatuses.get(receiver.id)?.RxBytes).toBe(5)
    expect(second.id).toBe('graph-2')
  })

  it('runs and queries a requested graph without switching the active graph', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')

    const second = store.createGraph('第二拓扑')
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.setActiveGraph('graph-1')

    await store.startRuntimeForGraph(second.id)
    await store.queryNodeBufferForGraph(second.id, receiver.id)

    expect(store.activeGraphId).toBe('graph-1')
    expect(bindings.StartSerialGraph).toHaveBeenCalledWith(expect.objectContaining({
      ID: second.id,
      Nodes: [
        expect.objectContaining({ ID: sender.id, Type: 'serial.sender' }),
        expect.objectContaining({ ID: receiver.id, Type: 'serial.receiver' }),
      ],
    }))
    expect(bindings.QuerySerialGraphNodeBuffer).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: second.id,
      NodeID: receiver.id,
    }))
    expect(store.runtimeStateForGraph(second.id).runtimeStatus).toBe('running')
    expect(store.runtimeStateForGraph(second.id).nodeBufferText.get(receiver.id)).toBe('hello')
  })

  it('sends through a runtime node and queries receiver data', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    await store.startRuntime()

    await store.sendNode(sender.id, 'hello', 'ascii')
    await store.queryNodeBuffer(receiver.id)

    expect(bindings.SendSerialGraphNode).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'graph-1',
      NodeID: sender.id,
      Content: 'hello',
      Mode: 'ascii',
    }))
    expect(bindings.QuerySerialGraphNodeBuffer).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'graph-1',
      NodeID: receiver.id,
    }))
    expect(store.nodeBufferText.get(receiver.id)).toBe('hello')
  })

  it('clears buffers, resets counters, and stops runtime', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    await store.startRuntime()

    await store.queryNodeBuffer(receiver.id)
    await store.clearNodeBuffer(receiver.id)
    await store.resetNodeCounters(receiver.id)
    await store.stopRuntime()

    expect(bindings.ClearSerialGraphNodeBuffer).toHaveBeenCalledWith('graph-1', receiver.id)
    expect(bindings.ResetSerialGraphNodeCounters).toHaveBeenCalledWith('graph-1', receiver.id)
    expect(bindings.StopSerialGraph).toHaveBeenCalledWith('graph-1')
    expect(store.nodeBufferText.has(receiver.id)).toBe(false)
    expect(store.runtimeStatus).toBe('stopped')
  })

  it('stops a running runtime before removing its topology graph', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    await store.startRuntime('graph-1-runtime')
    store.createGraph('第二拓扑')

    await store.removeGraph('graph-1')

    expect(bindings.StopSerialGraph).toHaveBeenCalledWith('graph-1-runtime')
    expect(store.graphList).toEqual([{ id: 'graph-2', name: '第二拓扑' }])
    expect(store.runtimeStatus).toBe('idle')
  })

  it('allows removing the last topology graph after stopping its runtime', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    await store.startRuntime('graph-1-runtime')

    await store.removeGraph('graph-1')

    expect(bindings.StopSerialGraph).toHaveBeenCalledWith('graph-1-runtime')
    expect(store.graphList).toEqual([])
    expect(store.activeGraphId).toBeNull()
    expect(store.runtimeStatus).toBe('idle')
  })

  it('allows all graph tabs to be removed after an async remove race', async () => {
    const stopResolvers: Array<() => void> = []
    bindings.StopSerialGraph.mockImplementationOnce(() => new Promise<void>(resolve => {
      stopResolvers.push(resolve)
    }))
    const store = useSerialGraphStore()
    store.renameGraph('graph-1', '运行拓扑')
    store.addNode('serial.sender')
    await store.startRuntime('graph-1-runtime')
    const second = store.createGraph('保留拓扑')

    const removingFirst = store.removeGraph('graph-1')
    await Promise.resolve()
    await store.removeGraph(second.id)
    stopResolvers[0]?.()
    await removingFirst

    expect(store.graphList).toEqual([])
    expect(store.activeGraphId).toBeNull()
  })

  it('queries recent node frames with a larger default page size', async () => {
    const store = useSerialGraphStore()
    const monitor = store.addNode('serial.monitor')
    await store.startRuntime()

    await store.queryNodeFramesForGraph('graph-1', monitor.id)

    expect(bindings.QuerySerialGraphNodeFrames).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'graph-1',
      NodeID: monitor.id,
      Offset: -1000,
      Limit: 1000,
    }))
  })

  it('preserves fresh counters when a stale frame query completes later', async () => {
    const store = useSerialGraphStore()
    const monitor = store.addNode('serial.monitor')
    bindings.StartSerialGraph.mockResolvedValue({
      ID: 'graph-1',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: monitor.id, Type: 'serial.monitor', Status: 'running', RxBytes: 0, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    await store.startRuntime()

    let resolveFrames: (value: { Frames: any[]; Total: number; NextOffset: number }) => void = () => {}
    bindings.QuerySerialGraphNodeFrames.mockImplementationOnce(() => new Promise(resolve => {
      resolveFrames = resolve
    }))
    const query = store.queryNodeFramesForGraph('graph-1', monitor.id)

    bindings.GetSerialGraphStatus.mockResolvedValue({
      ID: 'graph-1',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: monitor.id, Type: 'serial.monitor', Status: 'running', RxBytes: 44, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    await store.refreshRuntimeForGraph('graph-1')
    expect(store.runtimeStateForGraph('graph-1').nodeStatuses.get(monitor.id)?.RxBytes).toBe(44)

    resolveFrames({ Frames: [], Total: 0, NextOffset: 0 })
    await query

    expect(store.runtimeStateForGraph('graph-1').nodeStatuses.get(monitor.id)?.RxBytes).toBe(44)
  })

  it('adds, caps, and clears operation log entries per graph', () => {
    const store = useSerialGraphStore()

    for (let index = 0; index < 2005; index += 1) {
      store.appendOperationLogForGraph('graph-1', {
        level: 'info',
        source: 'test',
        action: 'append',
        message: `event ${index}`,
      })
    }

    expect(store.operationLogs).toHaveLength(2000)
    expect(store.operationLogs[0].message).toBe('event 5')
    expect(store.operationLogs[store.operationLogs.length - 1]).toEqual(expect.objectContaining({
      graphId: 'graph-1',
      level: 'info',
      source: 'test',
      action: 'append',
      message: 'event 2004',
    }))

    store.clearOperationLogsForGraph('graph-1')

    expect(store.operationLogs).toEqual([])
  })

  it('filters operation logs by level and shared filter modes without throwing on invalid patterns', () => {
    const store = useSerialGraphStore()
    store.appendOperationLogForGraph('graph-1', {
      level: 'info',
      source: 'sender',
      action: 'send',
      category: 'serial',
      message: 'Sent TEMP OK payload',
      details: 'manual send',
      nodeId: 'tx-1',
      direction: 'tx',
      payloadText: 'TEMP OK',
      payloadHex: '54 45 4d 50 20 4f 4b',
      byteLength: 7,
    })
    store.appendOperationLogForGraph('graph-1', {
      level: 'error',
      source: 'receiver',
      action: 'template',
      category: 'serial',
      message: 'Log template failed',
      details: 'fallback used',
      nodeId: 'rx-1',
      direction: 'rx',
      byteLength: 7,
    })

    store.setOperationLogFilterForGraph('graph-1', { level: 'error' })
    expect(store.filteredOperationLogsForGraph('graph-1').map(entry => entry.message)).toEqual(['Log template failed'])
    expect(store.operationLogFilterErrorForGraph('graph-1')).toBeNull()

    store.setOperationLogFilterForGraph('graph-1', {
      level: 'all',
      mode: 'plain',
      expression: 'temp',
      caseSensitive: false,
      wholeWord: true,
    })
    expect(store.filteredOperationLogsForGraph('graph-1').map(entry => entry.message)).toEqual(['Sent TEMP OK payload'])

    store.setOperationLogFilterForGraph('graph-1', { mode: 'regex', expression: 'template\\s+failed' })
    expect(store.filteredOperationLogsForGraph('graph-1').map(entry => entry.message)).toEqual(['Log template failed'])

    store.setOperationLogFilterForGraph('graph-1', { mode: 'expression', expression: 'len >= 7 and direction == "tx"' })
    expect(store.filteredOperationLogsForGraph('graph-1').map(entry => entry.message)).toEqual(['Sent TEMP OK payload'])

    store.setOperationLogFilterForGraph('graph-1', { mode: 'regex', expression: '[broken' })
    expect(store.filteredOperationLogsForGraph('graph-1')).toEqual([])
    expect(store.operationLogFilterErrorForGraph('graph-1')).toContain('invalid regex')
  })

  it('mutates explicit graph nodes, tabs, and edges without switching context', () => {
    const store = useSerialGraphStore()
    expect(store.duplicateGraph('missing')).toBeNull()
    store.renameGraph('graph-1', '   ')
    expect(store.graphList[0].name).toBe('拓扑图 1')

    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const edge = store.connect(sender.id, 'out', receiver.id, 'in')
    const second = store.createGraph('备用拓扑', false)

    store.moveNodeInGraph('graph-1', sender.id, { x: 99, y: 101 })
    store.setActiveNodeTabInGraph('graph-1', sender.id)
    store.selectEdgeInGraph('graph-1', edge!.id)

    expect(store.graphById('graph-1')?.nodes.find(node => node.id === sender.id)?.position).toEqual({ x: 99, y: 101 })
    expect(store.graphById('graph-1')?.activeNodeTabId).toBe(sender.id)
    expect(store.graphById('graph-1')?.selectedEdgeId).toBe(edge!.id)
    expect(store.activeGraphId).toBe('graph-1')
    expect(second.id).toBe('graph-2')

    store.removeEdgeFromGraph('graph-1', edge!.id)
    expect(store.graphById('graph-1')?.edges).toEqual([])
    expect(store.graphById('graph-1')?.selectedEdgeId).toBeNull()

    store.setActiveGraph(null)
    store.setActiveGraph('missing')
    expect(store.activeGraphId).toBe('graph-1')
    expect(() => store.addNode('serial.unknown')).toThrow('unknown graph provider')
    expect(() => store.moveNodeInGraph('missing', sender.id, { x: 0, y: 0 })).toThrow('serial graph not found: missing')
  })

  it('guards stopped runtime commands and records backend failure logs', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')

    await expect(store.sendNode(sender.id, 'hello')).rejects.toThrow('serial graph is not running')
    await expect(store.queryNodeBuffer(sender.id)).rejects.toThrow('serial graph is not running')
    await expect(store.queryNodeFrames(sender.id)).rejects.toThrow('serial graph is not running')
    await expect(store.refreshRuntime()).resolves.toBeNull()
    await store.clearNodeBuffer(sender.id)
    await store.resetNodeCounters(sender.id)
    expect(bindings.ClearSerialGraphNodeBuffer).not.toHaveBeenCalled()
    expect(bindings.ResetSerialGraphNodeCounters).not.toHaveBeenCalled()

    bindings.StartSerialGraph.mockResolvedValueOnce(null)
    await expect(store.startRuntime()).rejects.toThrow('serial graph did not return runtime info')
    expect(store.operationLogs.at(-1)).toEqual(expect.objectContaining({
      action: 'start-error',
      details: 'serial graph did not return runtime info',
    }))

    await store.startRuntime()
    bindings.StopSerialGraph.mockRejectedValueOnce(new Error('stop denied'))
    await expect(store.stopRuntime()).rejects.toThrow('stop denied')
    expect(store.operationLogs.at(-1)).toEqual(expect.objectContaining({
      action: 'stop-error',
      details: 'stop denied',
    }))

    bindings.SendSerialGraphNode.mockRejectedValueOnce(new Error('send denied'))
    await expect(store.sendNode(sender.id, 'hello')).rejects.toThrow('send denied')
    expect(store.operationLogs.at(-1)).toEqual(expect.objectContaining({
      action: 'send-error',
      details: 'send denied',
      nodeId: sender.id,
    }))
  })

  it('decodes runtime snapshots, clone exports, and null query responses', async () => {
    const store = useSerialGraphStore()
    const receiver = store.addNode('serial.receiver')
    await store.startRuntime()
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValueOnce({
      Offset: 10,
      Data: 'not base64?',
      Total: 21,
      EOF: true,
      Chunks: [
        { Offset: 10, Timestamp: '2026-06-23T02:01:02.003Z', Data: btoa('ok') },
        { Offset: 12, Timestamp: '2026-06-23T02:01:03.004Z', Data: '' },
      ],
    })

    await store.queryNodeBuffer(receiver.id, 10, 11)

    expect(store.nodeBufferText.get(receiver.id)).toBe('not base64?')
    expect(store.nodeBufferChunks.get(receiver.id)).toEqual([
      expect.objectContaining({ offset: 10, timestamp: '2026-06-23T02:01:02.003Z' }),
    ])

    const exported = store.exportRuntimeSnapshot('graph-1')
    exported.nodeBuffers[receiver.id][0] = 88
    expect(store.nodeBufferText.get(receiver.id)).toBe('not base64?')

    store.restoreRuntimeSnapshot('graph-1', {
      nodeBuffers: { legacy: [104, 105] as any },
      nodeBufferChunks: { legacy: [{ offset: 0, timestamp: 't', data: [33] as any }] },
      nodeFrames: { legacy: [{ Seq: 1, Direction: '接收', Length: 2, DisplayText: 'hi', DisplayHex: '68 69' } as any] },
    })
    expect(store.nodeBufferText.get('legacy')).toBe('hi')
    expect(store.nodeBufferChunks.get('legacy')?.[0].data).toEqual(new Uint8Array([33]))
    expect(store.nodeFrames.get('legacy')?.[0]).toEqual(expect.objectContaining({ DisplayText: 'hi' }))

    bindings.QuerySerialGraphNodeBuffer.mockResolvedValueOnce(null)
    await expect(store.queryNodeBuffer(receiver.id)).rejects.toThrow('serial graph did not return buffer page')
    bindings.QuerySerialGraphNodeFrames.mockResolvedValueOnce(null)
    await expect(store.queryNodeFrames(receiver.id)).rejects.toThrow('serial graph did not return frame page')
  })

  it('logs filter traversal through intermediate nodes and log template edge cases', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const tap = store.addNode('serial.tap')
    const filter = store.addNode('serial.filter')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', tap.id, 'in')
    store.connect(tap.id, 'out', filter.id, 'in')
    store.connect(filter.id, 'out', receiver.id, 'in')
    store.updateNodeConfig(sender.id, { enableLogging: true, logFormat: 'bad }' })
    store.updateNodeConfig(filter.id, { mode: 'plain', expression: 'ok' })

    await store.startRuntime()
    await store.sendNode(sender.id, 'ok', 'ascii')
    await store.sendNode(sender.id, 'drop', 'ascii')

    const filterLogs = store.operationLogs.filter(entry => entry.source === 'serial.filter' && ['pass', 'drop'].includes(entry.action))
    expect(filterLogs.map(entry => entry.action)).toEqual(['pass', 'drop'])
    expect(store.operationLogs.find(entry => entry.action === 'send' && entry.details?.includes('unmatched }'))).toBeTruthy()

    store.updateNodeConfig(sender.id, { logFormat: 'missing {text' })
    await store.sendNode(sender.id, 'again', 'ascii')
    expect(store.operationLogs.find(entry => entry.action === 'send' && entry.details?.includes('missing }'))).toBeTruthy()

    store.updateNodeConfig(sender.id, { logFormat: '' })
    await store.sendNode(sender.id, 'zz', 'hex')
    expect(store.operationLogs.find(entry => entry.action === 'send' && entry.payloadText === '' && entry.payloadHex === '')).toBeTruthy()
  })

  it('keeps graph fallback views stable after no-op mutations and empty workspaces', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const edge = store.connect(sender.id, 'out', receiver.id, 'in')!

    store.selectNodeInGraph('graph-1', null)
    store.removeEdgeFromGraph('graph-1', 'missing-edge')
    store.removeNodeFromGraph('graph-1', 'missing-node')
    store.closeNodeTabInGraph('graph-1', 'missing-tab')

    expect(store.graphById(null)).toBeNull()
    expect(store.validationForGraph(null)).toEqual({ valid: true, errors: [] })
    expect(store.edges.map(item => item.id)).toEqual([edge.id])
    expect(store.nodeTabs.map(item => item.nodeId)).toEqual([sender.id, receiver.id])

    await store.removeGraph('graph-1')

    expect(store.graphList).toEqual([])
    expect(store.nodes).toEqual([])
    expect(store.edges).toEqual([])
    expect(store.selectedNodeId).toBeNull()
    expect(store.selectedEdgeId).toBeNull()
    expect(store.nodeTabs).toEqual([])
    expect(store.activeNodeTabId).toBeNull()
    expect(store.validation).toEqual({ valid: true, errors: [] })
    expect(store.filteredOperationLogs).toEqual([])
    expect(store.operationLogFilterError).toBeNull()

    const inactive = store.createGraph(undefined, false)
    expect(store.activeGraphId).toBeNull()
    expect(store.activeGraph?.id).toBe(inactive.id)
    expect(store.runtimeStateForGraph(null).runtimeStatus).toBe('idle')

    store.restoreRuntimeSnapshot(inactive.id, null)
    expect(store.runtimeStateForGraph(inactive.id).nodeBuffers).toEqual(new Map())
  })

  it('handles sparse runtime responses and commands for missing nodes', async () => {
    const store = useSerialGraphStore()
    const receiver = store.addNode('serial.receiver')
    bindings.StartSerialGraph.mockResolvedValueOnce({
      ID: 'graph-1-runtime',
      Status: 'running',
      Error: '',
      Nodes: undefined,
    })

    await store.startRuntime('graph-1-runtime')

    expect(store.nodeStatuses).toEqual(new Map())
    expect(store.nodes.find(node => node.id === receiver.id)?.status).toBe('idle')

    bindings.GetSerialGraphStatus.mockResolvedValueOnce(null)
    await expect(store.refreshRuntimeForGraph('graph-1')).resolves.toBeNull()

    bindings.SendSerialGraphNode.mockResolvedValueOnce(3)
    await expect(store.sendNodeForGraph('graph-1', 'missing-node', 'abc')).resolves.toBe(3)
    expect(store.operationLogs.some(entry => entry.action === 'send-error' || entry.action === 'send-command')).toBe(false)

    bindings.SendSerialGraphNode.mockRejectedValueOnce(new Error('missing send denied'))
    await expect(store.sendNodeForGraph('graph-1', 'missing-node', 'abc')).rejects.toThrow('missing send denied')
    expect(store.operationLogs.some(entry => entry.details === 'missing send denied')).toBe(false)

    bindings.QuerySerialGraphNodeBuffer.mockResolvedValueOnce({
      Data: btoa('loose'),
      EOF: true,
    })
    await store.queryNodeBufferForGraph('graph-1', 'missing-node')
    expect(store.nodeBufferText.get('missing-node')).toBe('loose')
    expect(store.operationLogs.some(entry => entry.action === 'receive')).toBe(false)

    bindings.QuerySerialGraphNodeFrames.mockResolvedValueOnce({ Total: 0, NextOffset: 0 })
    await store.queryNodeFramesForGraph('graph-1', receiver.id, 0, 5)
    expect(store.nodeFrames.get(receiver.id)).toEqual([])

    await store.clearNodeBufferForGraph('graph-1', 'missing-node')
    await store.resetNodeCountersForGraph('graph-1', 'missing-node')
    expect(store.operationLogs.some(entry => entry.action === 'clear-buffer' || entry.action === 'reset-counters')).toBe(false)
  })
})
