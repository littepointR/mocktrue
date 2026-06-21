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

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => bindings)

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

  it('does not replace the last remaining graph after an async remove race', async () => {
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

    expect(store.graphList).toEqual([{ id: 'graph-1', name: '运行拓扑' }])
    expect(store.activeGraphId).toBe('graph-1')
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
})
