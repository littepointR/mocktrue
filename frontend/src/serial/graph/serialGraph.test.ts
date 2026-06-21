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
      'serial.script.transform',
      'serial.script.generator',
      'serial.script.analyzer',
    ]))
  })

  it('registers script node providers with stable ports and frozen config fields', () => {
    const transform = serialGraphProviders.find(item => item.type === 'serial.script.transform')
    const generator = serialGraphProviders.find(item => item.type === 'serial.script.generator')
    const analyzer = serialGraphProviders.find(item => item.type === 'serial.script.analyzer')

    expect(transform?.inputs).toEqual([{ id: 'in', label: '接收', kind: 'bytes', direction: 'input' }])
    expect(transform?.outputs).toEqual([{ id: 'out', label: '发送', kind: 'bytes', direction: 'output' }])
    expect(generator?.inputs).toEqual([])
    expect(generator?.outputs).toEqual([{ id: 'out', label: '发送', kind: 'bytes', direction: 'output' }])
    expect(analyzer?.inputs).toEqual([{ id: 'in', label: '接收', kind: 'bytes', direction: 'input' }])
    expect(analyzer?.outputs).toEqual([])

    for (const provider of [transform, generator, analyzer]) {
      expect(Object.keys(provider?.defaultConfig ?? {})).toEqual([
        'script',
        'timeoutMs',
        'maxOutputBytes',
        'maxStateBytes',
        'onError',
        'encoding',
        'autoRun',
        'intervalMs',
        'displayMode',
      ])
      expect(provider?.defaultConfig).toEqual(expect.objectContaining({
        timeoutMs: 50,
        maxOutputBytes: 65536,
        maxStateBytes: 262144,
        onError: 'mark-error-and-drop',
        encoding: 'utf-8',
      }))
    }
    expect(transform?.defaultConfig.script).toBe('output.bytes(input.bytes())')
    expect(generator?.defaultConfig.script).toBe('output.text("tick", "utf-8")')
    expect(generator?.defaultConfig.autoRun).toBe(true)
    expect(generator?.defaultConfig.intervalMs).toBe(1000)
    expect(analyzer?.defaultConfig.script).toBe('field("length", input.bytes().length)')
    expect(analyzer?.defaultConfig.displayMode).toBe('hex')
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

  it('allows protocol responses to return to a master receive input', () => {
    const state = graph([
      node('master', 'serial.modbus.master'),
      node('slave', 'serial.modbus.slave'),
      node('tap', 'serial.tap'),
      node('receiver', 'serial.receiver'),
    ], [
      { id: 'edge-request', source: 'master', sourceHandle: 'tx', target: 'slave', targetHandle: 'rx' },
      { id: 'edge-response-tap', source: 'slave', sourceHandle: 'tx', target: 'tap', targetHandle: 'in' },
      { id: 'edge-response-receiver', source: 'tap', sourceHandle: 'out', target: 'receiver', targetHandle: 'in' },
    ])

    const result = canConnect(state, {
      source: 'tap',
      sourceHandle: 'out',
      target: 'master',
      targetHandle: 'rx',
    })

    expect(result.errors).toEqual([])
    expect(validateGraph({
      ...state,
      edges: [
        ...state.edges,
        { id: 'edge-response-master', source: 'tap', sourceHandle: 'out', target: 'master', targetHandle: 'rx' },
      ],
    }).errors).toEqual([])
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
