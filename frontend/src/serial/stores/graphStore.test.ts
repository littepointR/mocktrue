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
    bindings.StartSerialGraph.mockResolvedValue({
      ID: 'serial.graph',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    bindings.GetSerialGraphStatus.mockResolvedValue({
      ID: 'serial.graph',
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
    expect(store.validationErrors).toEqual([])

    const snapshot = store.exportState()
    setActivePinia(createPinia())
    const restored = useSerialGraphStore()
    restored.restoreState(snapshot)

    expect(restored.nodes.map(node => node.type)).toEqual(['serial.sender', 'serial.receiver'])
    expect(restored.edges).toEqual(snapshot.edges)
    expect(restored.selectedNodeId).toBe(receiver.id)
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
      ID: 'serial.graph',
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

  it('sends through a runtime node and queries receiver data', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    await store.startRuntime()

    await store.sendNode(sender.id, 'hello', 'ascii')
    await store.queryNodeBuffer(receiver.id)

    expect(bindings.SendSerialGraphNode).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'serial.graph',
      NodeID: sender.id,
      Content: 'hello',
      Mode: 'ascii',
    }))
    expect(bindings.QuerySerialGraphNodeBuffer).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'serial.graph',
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

    expect(bindings.ClearSerialGraphNodeBuffer).toHaveBeenCalledWith('serial.graph', receiver.id)
    expect(bindings.ResetSerialGraphNodeCounters).toHaveBeenCalledWith('serial.graph', receiver.id)
    expect(bindings.StopSerialGraph).toHaveBeenCalledWith('serial.graph')
    expect(store.nodeBufferText.has(receiver.id)).toBe(false)
    expect(store.runtimeStatus).toBe('stopped')
  })
})
