import { describe, expect, it } from 'vitest'
import {
  canConnect,
  cloneSerialGraphState,
  createSerialGraphNode,
  defaultSerialGraphDocument,
  defaultSerialGraphState,
  serialGraphProviders,
  validateGraph,
  type SerialGraphEdge,
  type SerialGraphNode,
} from './serialGraph'

describe('serial graph model', () => {
  it('registers serial feature node providers', () => {
    const types = serialGraphProviders.map(provider => provider.type)

    expect(types).toEqual(expect.arrayContaining([
      'serial.physical',
      'serial.virtual',
      'serial.bridge',
      'serial.monitor',
      'serial.tap',
      'serial.sender',
      'serial.receiver',
      'serial.modbus.master',
      'serial.modbus.slave',
      'serial.fecbus.master',
      'serial.fecbus.slave',
    ]))
  })

  it('creates a workspace state with one active graph document', () => {
    const state = defaultSerialGraphState()

    expect(state.graphs).toHaveLength(1)
    expect(state.activeGraphId).toBe(state.graphs[0].id)
    expect(state.graphs[0].nodes).toEqual([])
    expect(state.graphs[0].edges).toEqual([])
  })

  it('does not expose legacy serial monitor mode configuration', () => {
    const provider = serialGraphProviders.find(item => item.type === 'serial.monitor')
    const monitor = createSerialGraphNode('monitor', 'serial.monitor', { x: 0, y: 0 })

    expect(provider?.defaultConfig).toEqual({ displayMode: 'hex' })
    expect(provider?.outputs).toEqual([])
    expect(monitor.config).toEqual({ displayMode: 'hex' })
  })

  it('does not expose topology config fields that are not used by runtime nodes', () => {
    const bridge = serialGraphProviders.find(item => item.type === 'serial.bridge')
    const modbusSlave = serialGraphProviders.find(item => item.type === 'serial.modbus.slave')

    expect(bridge?.defaultConfig).toEqual({})
    expect(modbusSlave?.defaultConfig).toEqual({ mode: 'rtu', unitIds: '1' })
    expect(createSerialGraphNode('bridge', 'serial.bridge', { x: 0, y: 0 }).config).toEqual({})
    expect(createSerialGraphNode('slave', 'serial.modbus.slave', { x: 0, y: 0 }).config).toEqual({
      mode: 'rtu',
      unitIds: '1',
    })
  })

  it('removes legacy and unused topology config fields when restoring graph state', () => {
    const state = cloneSerialGraphState({
      graphs: [{
        ...defaultSerialGraphDocument(),
        nodes: [
          node('monitor', 'serial.monitor', { mode: 'auto-virtual', displayMode: 'hex' }),
          node('bridge', 'serial.bridge', { baudRate: 115200 }),
          node('slave', 'serial.modbus.slave', { mode: 'rtu', unitIds: '1,2', addressMode: 'zero-based' }),
          node('sender', 'serial.sender', { mode: 'ascii', payload: 'ok' }),
        ],
        nodeTabs: [
          { nodeId: 'monitor', title: '串口监控' },
          { nodeId: 'bridge', title: '串口桥接' },
          { nodeId: 'slave', title: 'Modbus 从站' },
          { nodeId: 'sender', title: '发送器' },
        ],
      }],
      activeGraphId: 'graph-1',
    })

    expect(state.graphs[0].nodes.find(item => item.id === 'monitor')?.config).toEqual({ displayMode: 'hex' })
    expect(state.graphs[0].nodes.find(item => item.id === 'bridge')?.config).toEqual({})
    expect(state.graphs[0].nodes.find(item => item.id === 'slave')?.config).toEqual({ mode: 'rtu', unitIds: '1,2' })
    expect(state.graphs[0].nodes.find(item => item.id === 'sender')?.config).toEqual({ mode: 'ascii', payload: 'ok' })
  })

  it('allows compatible byte stream connections and rejects non-emitting monitor outputs', () => {
    const state = graph([
      node('source', 'serial.sender'),
      node('sink', 'serial.receiver'),
      node('monitor', 'serial.monitor'),
    ])

    expect(canConnect(state, {
      source: 'source',
      sourceHandle: 'out',
      target: 'sink',
      targetHandle: 'in',
    }).valid).toBe(true)

    expect(canConnect(state, {
      source: 'monitor',
      sourceHandle: 'frames',
      target: 'sink',
      targetHandle: 'in',
    }).errors).toEqual(expect.arrayContaining(['output port not found: serial.monitor.frames']))
  })

  it('rejects duplicate resource-owner ports', () => {
    const state = graph([
      node('port-a', 'serial.physical', { portName: '/tmp/ttyA' }),
      node('port-b', 'serial.virtual', { portName: '/tmp/ttyA' }),
    ])

    const result = validateGraph(state)

    expect(result.valid).toBe(false)
    expect(result.errors.some(error => error.includes('/tmp/ttyA'))).toBe(true)
  })

  it('only allows fan-out through tap style nodes', () => {
    const direct = graph([
      node('source', 'serial.sender'),
      node('sink-a', 'serial.receiver'),
      node('sink-b', 'serial.receiver'),
    ], [
      { id: 'edge-1', source: 'source', sourceHandle: 'out', target: 'sink-a', targetHandle: 'in' },
      { id: 'edge-2', source: 'source', sourceHandle: 'out', target: 'sink-b', targetHandle: 'in' },
    ])

    const throughTap = graph([
      node('tap', 'serial.tap'),
      node('sink-a', 'serial.receiver'),
      node('sink-b', 'serial.receiver'),
    ], [
      { id: 'edge-1', source: 'tap', sourceHandle: 'out', target: 'sink-a', targetHandle: 'in' },
      { id: 'edge-2', source: 'tap', sourceHandle: 'out', target: 'sink-b', targetHandle: 'in' },
    ])

    expect(validateGraph(direct).valid).toBe(false)
    expect(validateGraph(throughTap).valid).toBe(true)
  })

  it('rejects two-node directed cycles when connecting', () => {
    const state = graph([
      node('node-a', 'serial.bridge'),
      node('node-b', 'serial.bridge'),
    ], [
      { id: 'edge-1', source: 'node-a', sourceHandle: 'a-out', target: 'node-b', targetHandle: 'a-in' },
    ])

    const result = canConnect(state, {
      source: 'node-b',
      sourceHandle: 'a-out',
      target: 'node-a',
      targetHandle: 'a-in',
    })

    expect(result.valid).toBe(false)
    expect(result.errors.some(error => error.includes('cycle'))).toBe(true)
  })

  it('rejects three-node directed cycles during validation', () => {
    const state = graph([
      node('node-a', 'serial.bridge'),
      node('node-b', 'serial.bridge'),
      node('node-c', 'serial.bridge'),
    ], [
      { id: 'edge-1', source: 'node-a', sourceHandle: 'a-out', target: 'node-b', targetHandle: 'a-in' },
      { id: 'edge-2', source: 'node-b', sourceHandle: 'a-out', target: 'node-c', targetHandle: 'a-in' },
      { id: 'edge-3', source: 'node-c', sourceHandle: 'a-out', target: 'node-a', targetHandle: 'a-in' },
    ])

    const result = validateGraph(state)

    expect(result.valid).toBe(false)
    expect(result.errors.some(error => error.includes('cycle'))).toBe(true)
  })
})

function graph(nodes: SerialGraphNode[], edges: SerialGraphEdge[] = []) {
  return {
    ...defaultSerialGraphDocument(),
    nodes,
    edges,
  }
}

function node(id: string, type: string, config: Record<string, unknown> = {}): SerialGraphNode {
  return {
    id,
    type,
    position: { x: 0, y: 0 },
    config,
  }
}
