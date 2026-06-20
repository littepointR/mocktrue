import { describe, expect, it } from 'vitest'
import {
  canConnect,
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

  it('allows compatible byte stream connections and rejects incompatible ports', () => {
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
    }).valid).toBe(false)
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
