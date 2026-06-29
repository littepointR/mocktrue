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
      'serial.modbus.master',
      'serial.modbus.slave',
      'serial.fecbus.master',
      'serial.fecbus.slave',
      'serial.script.transform',
      'serial.script.generator',
      'serial.script.analyzer',
    ]))
    expect(types).not.toContain('serial.sender')
    expect(types).not.toContain('serial.receiver')
    expect(types).not.toContain('serial.tap')
    expect(types).not.toContain('serial.tee')
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

  it('defaults virtual serial nodes to graph-only endpoints', () => {
    const provider = serialGraphProviders.find(item => item.type === 'serial.virtual')
    const virtual = createSerialGraphNode('virtual', 'serial.virtual', { x: 0, y: 0 })

    expect(provider?.defaultConfig.portName).toBe('')
    expect(virtual.config).toEqual(expect.objectContaining({ portName: '' }))
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

  it('exposes multi-write value config on Modbus master nodes', () => {
    const modbusMaster = serialGraphProviders.find(item => item.type === 'serial.modbus.master')
    const master = createSerialGraphNode('master', 'serial.modbus.master', { x: 0, y: 0 })

    expect(modbusMaster?.defaultConfig).toEqual(expect.objectContaining({
      coilValues: '1 0 1 1',
      registerValues: '24 42',
    }))
    expect(master.config).toEqual(expect.objectContaining({
      coilValues: '1 0 1 1',
      registerValues: '24 42',
    }))
  })

  it('documents that graph Modbus slave uses lightweight default responses', () => {
    const modbusSlave = serialGraphProviders.find(item => item.type === 'serial.modbus.slave')

    expect(modbusSlave?.description).toContain('协议合法的零值/默认响应')
    expect(modbusSlave?.description).toContain('完整可编辑的多 Unit 数据模型请使用 Modbus 会话面板')
  })

  it('keeps removed node types unsupported when restoring graph state', () => {
    const state = cloneSerialGraphState({
      graphs: [{
        ...defaultSerialGraphDocument(),
        nodes: [
          node('monitor', 'serial.monitor', { mode: 'auto-virtual', displayMode: 'hex' }),
          node('bridge', 'serial.bridge', { baudRate: 115200 }),
          node('slave', 'serial.modbus.slave', { mode: 'rtu', unitIds: '1,2', addressMode: 'zero-based' }),
          node('legacy-sender', 'serial.sender', { mode: 'ascii', payload: 'ok' }),
          node('legacy-tap', 'serial.tap'),
          node('legacy-tee', 'serial.tee'),
        ],
        nodeTabs: [
          { nodeId: 'monitor', title: '串口监控' },
          { nodeId: 'bridge', title: '串口桥接' },
          { nodeId: 'slave', title: 'Modbus 从站' },
          { nodeId: 'legacy-sender', title: '发送器' },
          { nodeId: 'legacy-tap', title: '分流器' },
          { nodeId: 'legacy-tee', title: 'T 型分支' },
        ],
      }],
      activeGraphId: 'graph-1',
    })

    expect(state.graphs[0].nodes.find(item => item.id === 'monitor')?.config).toEqual({ displayMode: 'hex' })
    expect(state.graphs[0].nodes.find(item => item.id === 'bridge')?.config).toEqual({})
    expect(state.graphs[0].nodes.find(item => item.id === 'slave')?.config).toEqual({ mode: 'rtu', unitIds: '1,2' })
    expect(state.graphs[0].nodes.find(item => item.id === 'legacy-sender')?.config).toEqual({ mode: 'ascii', payload: 'ok' })
    expect(validateGraph(state.graphs[0]).errors).toEqual(expect.arrayContaining(['provider not found: serial.sender']))
    expect(validateGraph(state.graphs[0]).errors).toEqual(expect.arrayContaining([
      'provider not found: serial.tap',
      'provider not found: serial.tee',
    ]))
  })

  it('allows compatible byte stream connections and rejects non-emitting monitor outputs', () => {
    const state = graph([
      node('source', 'serial.virtual'),
      node('sink', 'serial.virtual'),
      node('monitor', 'serial.monitor'),
    ])

    expect(canConnect(state, {
      source: 'source',
      sourceHandle: 'rx',
      target: 'sink',
      targetHandle: 'tx',
    }).valid).toBe(true)

    expect(canConnect(state, {
      source: 'monitor',
      sourceHandle: 'frames',
      target: 'sink',
      targetHandle: 'tx',
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

  it('allows output fan-out without dedicated branch nodes', () => {
    const direct = graph([
      node('source', 'serial.virtual'),
      node('sink-a', 'serial.virtual'),
      node('sink-b', 'serial.virtual'),
    ], [
      { id: 'edge-1', source: 'source', sourceHandle: 'rx', target: 'sink-a', targetHandle: 'tx' },
      { id: 'edge-2', source: 'source', sourceHandle: 'rx', target: 'sink-b', targetHandle: 'tx' },
    ])

    expect(validateGraph(direct).errors).toEqual([])
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
      node('sink', 'serial.virtual'),
    ], [
      { id: 'edge-request', source: 'master', sourceHandle: 'tx', target: 'slave', targetHandle: 'rx' },
      { id: 'edge-response-sink', source: 'slave', sourceHandle: 'tx', target: 'sink', targetHandle: 'tx' },
    ])

    const result = canConnect(state, {
      source: 'slave',
      sourceHandle: 'tx',
      target: 'master',
      targetHandle: 'rx',
    })

    expect(result.errors).toEqual([])
    expect(validateGraph({
      ...state,
      edges: [
        ...state.edges,
        { id: 'edge-response-master', source: 'slave', sourceHandle: 'tx', target: 'master', targetHandle: 'rx' },
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

  it('registers a remote serial provider with raw TCP client defaults and byte ports', () => {
    const provider = serialGraphProviders.find(item => item.type === 'serial.remote')

    expect(provider).toEqual(expect.objectContaining({
      title: '远端串口',
      category: '串口',
      resourceOwner: true,
    }))
    expect(provider?.inputs).toEqual([{ id: 'tx', label: '发送', kind: 'bytes', direction: 'input' }])
    expect(provider?.outputs).toEqual([{ id: 'rx', label: '接收', kind: 'bytes', direction: 'output' }])
    expect(provider?.defaultConfig).toEqual({
      protocol: 'raw-tcp',
      role: 'client',
      host: '',
      port: 3001,
      connectTimeoutMs: 3000,
      writeTimeoutMs: 3000,
      reconnect: true,
      reconnectIntervalMs: 1000,
      allowStartDisconnected: false,
      readBufKB: 32,
      baudRate: 115200,
      dataBits: 8,
      stopBits: '1',
      parity: 'none',
      flowMode: 'none',
      viewMode: 'ascii',
      autoScroll: true,
      showTimestamp: false,
    })
    expect(provider?.defaultConfig).not.toHaveProperty('mode')

    const remote = createSerialGraphNode('remote', 'serial.remote', { x: 120, y: 40 }, { host: '127.0.0.1' })
    const sink = node('sink', 'serial.virtual')
    const state = graph([remote, sink])

    expect(canConnect(state, {
      source: remote.id,
      sourceHandle: 'rx',
      target: sink.id,
      targetHandle: 'tx',
    }).valid).toBe(true)
  })

  it('validates remote serial config and duplicate raw TCP endpoints', () => {
    const validRemote = (id: string, config: Record<string, unknown> = {}) => node(id, 'serial.remote', {
      protocol: 'raw-tcp',
      role: 'client',
      host: '127.0.0.1',
      port: 3001,
      connectTimeoutMs: 3000,
      writeTimeoutMs: 3000,
      reconnectIntervalMs: 1000,
      readBufKB: 32,
      ...config,
    })

    expect(validateGraph(graph([validRemote('remote')])).errors).toEqual([])
    expect(validateGraph(graph([node('remote-with-missing-numeric-defaults', 'serial.remote', {
      protocol: 'raw-tcp',
      role: 'client',
      host: '127.0.0.1',
    })])).errors).toEqual([])
    expect(validateGraph(graph([validRemote('remote-with-undefined-numeric-defaults', {
      port: undefined,
      connectTimeoutMs: undefined,
      writeTimeoutMs: undefined,
      reconnectIntervalMs: undefined,
      readBufKB: undefined,
    })])).errors).toEqual([])
    expect(validateGraph(graph([validRemote('missing-host', { host: ' ' })])).errors.join('\n')).toContain('host required')
    expect(validateGraph(graph([validRemote('url-host', { host: 'tcp://127.0.0.1' })])).errors.join('\n')).toContain('host must not include URL scheme')
    expect(validateGraph(graph([validRemote('bad-port', { port: 70000 })])).errors.join('\n')).toContain('port out of range')
    expect(validateGraph(graph([validRemote('fractional-port', { port: 3.14 })])).errors.join('\n')).toContain('port must be an integer')
    expect(validateGraph(graph([validRemote('string-port', { port: '3001x' })])).errors.join('\n')).toContain('port must be an integer')
    for (const [key, message] of [
      ['port', 'remote port must be an integer'],
      ['connectTimeoutMs', 'connectTimeoutMs must be an integer'],
      ['writeTimeoutMs', 'writeTimeoutMs must be an integer'],
      ['reconnectIntervalMs', 'reconnectIntervalMs must be an integer'],
      ['readBufKB', 'readBufKB must be an integer'],
    ]) {
      expect(validateGraph(graph([validRemote(`empty-${key}`, { [key]: '' })])).errors.join('\n')).toContain(message)
    }
    expect(validateGraph(graph([validRemote('null-timeout', { connectTimeoutMs: null })])).errors.join('\n')).toContain('connectTimeoutMs must be an integer')
    expect(validateGraph(graph([validRemote('short-connect-timeout', { connectTimeoutMs: 99 })])).errors.join('\n')).toContain('connectTimeoutMs out of range')
    expect(validateGraph(graph([validRemote('long-connect-timeout', { connectTimeoutMs: 60001 })])).errors.join('\n')).toContain('connectTimeoutMs out of range')
    expect(validateGraph(graph([validRemote('short-write-timeout', { writeTimeoutMs: 99 })])).errors.join('\n')).toContain('writeTimeoutMs out of range')
    expect(validateGraph(graph([validRemote('long-write-timeout', { writeTimeoutMs: 60001 })])).errors.join('\n')).toContain('writeTimeoutMs out of range')
    expect(validateGraph(graph([validRemote('short-reconnect-interval', { reconnectIntervalMs: 99 })])).errors.join('\n')).toContain('reconnectIntervalMs out of range')
    expect(validateGraph(graph([validRemote('long-reconnect-interval', { reconnectIntervalMs: 60001 })])).errors.join('\n')).toContain('reconnectIntervalMs out of range')
    expect(validateGraph(graph([validRemote('large-read-buffer', { readBufKB: 2048 })])).errors.join('\n')).toContain('readBufKB out of range')
    expect(validateGraph(graph([validRemote('bad-protocol', { protocol: 'rfc2217' })])).errors.join('\n')).toContain('unsupported protocol')
    expect(validateGraph(graph([validRemote('server', { role: 'server' })])).errors).toEqual([])
    expect(validateGraph(graph([validRemote('bad-role', { role: 'peer' })])).errors.join('\n')).toContain('unsupported role')

    const duplicate = validateGraph(graph([
      validRemote('remote-a'),
      validRemote('remote-b'),
    ]))
    expect(duplicate.valid).toBe(false)
    expect(duplicate.errors.join('\n')).toContain('resource remote endpoint duplicated')
    expect(duplicate.errors.join('\n')).toContain('127.0.0.1:3001')
    expect(validateGraph(graph([
      validRemote('remote-server', { role: 'server' }),
      validRemote('remote-client', { role: 'client' }),
    ])).errors).toEqual([])

    expect(validateGraph(graph([
      validRemote('remote-a'),
      validRemote('remote-b', { port: 3002 }),
      validRemote('remote-c', { host: 'localhost' }),
    ])).errors).toEqual([])
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
