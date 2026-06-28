import { describe, expect, it } from 'vitest'
import { serialGraphProviders, validateGraph, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { matchSerialFilter } from '../serial/utils/serialFilter'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from './demoWorkspaces'
import { workspaceKind } from './workspaceSnapshot'

const expectedDemoNodeTypes: Record<string, string[]> = {
  'serial-open-demo': ['serial.script.generator', 'serial.virtual'],
  'virtual-port-demo': ['serial.script.generator', 'serial.virtual'],
  'bridge-demo': ['serial.script.generator', 'serial.virtual', 'serial.bridge', 'serial.monitor'],
  'monitor-demo': ['serial.script.generator', 'serial.virtual', 'serial.tap', 'serial.monitor'],
  'script-transform-demo': [
    'serial.script.generator',
    'serial.virtual',
    'serial.script.transform',
    'serial.tap',
    'serial.monitor',
  ],
  'script-analyzer-demo': [
    'serial.script.generator',
    'serial.virtual',
    'serial.tap',
    'serial.monitor',
    'serial.script.analyzer',
  ],
  'modbus-demo': ['serial.virtual', 'serial.modbus.master', 'serial.modbus.slave', 'serial.tap', 'serial.monitor'],
  'fecbus-demo': ['serial.virtual', 'serial.fecbus.master', 'serial.fecbus.slave', 'serial.tap', 'serial.monitor'],
  'serial-graph-demo': ['serial.script.generator', 'serial.virtual', 'serial.tap', 'serial.modbus.master', 'serial.monitor'],
  'serial-observability-demo': ['serial.script.generator', 'serial.virtual', 'serial.tap', 'serial.filter', 'serial.monitor'],
  'remote-serial-demo': ['serial.script.generator', 'serial.remote', 'serial.tap', 'serial.monitor'],
  'full-workspace-demo': [
    'serial.script.generator',
    'serial.virtual',
    'serial.bridge',
    'serial.monitor',
    'serial.tap',
    'serial.filter',
    'serial.script.transform',
    'serial.script.analyzer',
    'serial.modbus.master',
    'serial.modbus.slave',
    'serial.fecbus.master',
    'serial.fecbus.slave',
  ],
}

describe('demoWorkspaces', () => {
  it('lists one example workspace for each feature group', () => {
    const demos = listDemoWorkspaces()
    const ids = demos.map(demo => demo.id)

    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toEqual([
      'serial-open-demo',
      'virtual-port-demo',
      'bridge-demo',
      'monitor-demo',
      'script-transform-demo',
      'script-analyzer-demo',
      'modbus-demo',
      'fecbus-demo',
      'serial-graph-demo',
      'serial-observability-demo',
      'remote-serial-demo',
      'full-workspace-demo',
    ])
    expect(demos.every(demo => !('readonly' in demo))).toBe(true)
  })

  it('returns null for an unknown demo id', () => {
    expect(getDemoWorkspace('missing-demo')).toBeNull()
    expect(createDemoWorkspaceSnapshot('missing-demo')).toBeNull()
  })

  it('creates fresh snapshots with valid workspace structure', () => {
    const first = createDemoWorkspaceSnapshot('serial-open-demo')
    const second = createDemoWorkspaceSnapshot('serial-open-demo')

    expect(first?.kind).toBe(workspaceKind)
    expect(first?.serial.virtualPorts).toHaveLength(0)
    expect(first?.serial.handles).toHaveLength(0)
    expect(first?.serial.buffers).toEqual({})
    expect(first?.serial.workspace.editorLayout.type).toBe('group')
    const firstGraph = activeGraph(first?.serial.graph)
    const secondGraph = activeGraph(second?.serial.graph)
    const firstVirtual = firstGraph.nodes.find(node => node.type === 'serial.virtual')
    const secondVirtual = secondGraph.nodes.find(node => node.type === 'serial.virtual')
    expect(firstVirtual?.config.portName).toBeTruthy()
    expect(firstVirtual?.config.portName).not.toBe(secondVirtual?.config.portName)
  })

  it('builds a full example from normal topology resources only', () => {
    const snapshot = createDemoWorkspaceSnapshot('full-workspace-demo')
    const graph = activeGraph(snapshot?.serial.graph)

    expect(snapshot?.settings.serial.TerminalFontFamily).toBe('Menlo')
    expect(snapshot?.serial.virtualPorts).toHaveLength(0)
    expect(snapshot?.serial.bridges).toHaveLength(0)
    expect(snapshot?.serial.handles).toHaveLength(0)
    expect(snapshot?.serial.buffers).toEqual({})
    expect(snapshot?.serial.monitors.sessions).toHaveLength(0)
    expect(Object.values(snapshot?.serial.monitors.frames ?? {}).flat()).toHaveLength(0)
    expect(snapshot?.serial.modbus.sessions).toHaveLength(0)
    expect(snapshot?.serial.modbus.history).toHaveLength(0)
    expect(snapshot?.serial.fecbus.sessions).toHaveLength(0)
    expect(snapshot?.serial.fecbus.framePages).toEqual({})
    expect(graph.nodes.map(node => node.type)).toEqual(expect.arrayContaining(expectedDemoNodeTypes['full-workspace-demo']))
    expect(validateGraph(graph).errors).toEqual([])
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    expect(snapshot?.serial.workspace.selectedOperation).toBe('graph')
  })

  it('opens the serial graph example with normal graph workspace state', () => {
    const snapshot = createDemoWorkspaceSnapshot('serial-graph-demo')
    const graph = activeGraph(snapshot?.serial.graph)

    expect(snapshot?.serial.graph.graphs).toHaveLength(1)
    expect(snapshot?.serial.graph.activeGraphId).toBe(graph.id)
    expect(graph.nodes.map(node => node.type)).toEqual(expect.arrayContaining([
      'serial.script.generator',
      'serial.virtual',
      'serial.tap',
      'serial.modbus.master',
      'serial.monitor',
    ]))
    const providerTypes = new Set(serialGraphProviders.map(provider => provider.type))
    expect(graph.nodes.every(node => providerTypes.has(node.type))).toBe(true)
    const generator = graph.nodes.find(node => node.type === 'serial.script.generator')
    const virtualPort = graph.nodes.find(node => node.type === 'serial.virtual')
    const tap = graph.nodes.find(node => node.type === 'serial.tap')
    const monitor = graph.nodes.find(node => node.type === 'serial.monitor')
    expect(generator?.config).toEqual(expect.objectContaining({
      autoRun: true,
      intervalMs: 1000,
      encoding: 'utf-8',
    }))
    expect(String(generator?.config.script ?? '')).toContain('PortWeave graph')
    expect(graph.edges).toEqual(expect.arrayContaining([
      expect.objectContaining({
        source: generator?.id,
        sourceHandle: 'out',
        target: virtualPort?.id,
        targetHandle: 'tx',
      }),
      expect.objectContaining({
        source: virtualPort?.id,
        sourceHandle: 'rx',
        target: tap?.id,
        targetHandle: 'in',
      }),
      expect.objectContaining({
        source: tap?.id,
        sourceHandle: 'out',
        target: monitor?.id,
        targetHandle: 'in',
      }),
    ]))
    expect(graph.selectedNodeId).toBe(virtualPort?.id)
    expect(graph.nodeTabs.map(tab => tab.nodeId)).toEqual(graph.nodes.map(node => node.id))
    expect(graph.activeNodeTabId).toBe(virtualPort?.id)
    expect(graph.edges.length).toBeGreaterThanOrEqual(3)
    expect(snapshot?.serial.workspace.selectedOperation).toBe('graph')
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    if (snapshot?.serial.workspace.editorLayout.type === 'group') {
      expect(snapshot.serial.workspace.editorLayout.tabs).toContain(`graph:${graph.id}`)
      expect(snapshot.serial.workspace.activeByGroup[snapshot.serial.workspace.editorLayout.id]).toBe(`graph:${graph.id}`)
    }
  })

  it('opens the serial observability example with filters and monitor branches', () => {
    const demo = getDemoWorkspace('serial-observability-demo')
    const snapshot = createDemoWorkspaceSnapshot('serial-observability-demo')
    const graph = activeGraph(snapshot?.serial.graph)

    expect(demo?.title).toBe('串口过滤与日志演示')
    expect(snapshot?.serial.graph.graphs).toHaveLength(1)
    expect(snapshot?.serial.graph.activeGraphId).toBe(graph.id)
    expect(snapshot?.serial.workspace.selectedOperation).toBe('graph')
    expect(validateGraph(graph).errors).toEqual([])

    const providerTypes = new Set(serialGraphProviders.map(provider => provider.type))
    expect(graph.nodes.every(node => providerTypes.has(node.type))).toBe(true)

    const generator = graph.nodes.find(node => node.type === 'serial.script.generator')
    const virtualPort = graph.nodes.find(node => node.type === 'serial.virtual')
    const tap = graph.nodes.find(node => node.type === 'serial.tap')
    const filters = graph.nodes.filter(node => node.type === 'serial.filter')
    const monitors = graph.nodes.filter(node => node.type === 'serial.monitor')

    expect(generator?.config).toEqual(expect.objectContaining({
      autoRun: true,
      encoding: 'utf-8',
      intervalMs: 1000,
    }))
    expect(String(generator?.config.script ?? '')).toContain('TEMP=42 STATUS OK')
    expect(filters).toHaveLength(3)
    expect(filters.map(node => node.config.mode).sort()).toEqual(['expression', 'plain', 'regex'])
    expect(filters.find(node => node.config.mode === 'plain')?.config).toEqual(expect.objectContaining({
      expression: 'STATUS OK',
      caseSensitive: false,
      wholeWord: false,
    }))
    const regexFilter = filters.find(node => node.config.mode === 'regex')
    const regexExpression = String(regexFilter?.config.expression ?? '')
    expect(regexFilter?.config).toEqual(expect.objectContaining({
      expression: /TEMP=\d+/.source,
      caseSensitive: false,
      wholeWord: false,
    }))
    expect(matchSerialFilter({ text: 'TEMP=42 STATUS OK\r\n' }, {
      mode: 'regex',
      expression: regexExpression,
    })).toEqual({ matched: true })
    expect(filters.find(node => node.config.mode === 'expression')?.config).toEqual(expect.objectContaining({
      expression: 'len >= 4 and text contains "OK"',
      caseSensitive: false,
      wholeWord: false,
    }))
    expect(monitors).toHaveLength(3)
    expect(monitors.map(node => node.config.displayMode).sort()).toEqual(['ascii', 'ascii', 'hex'])
    expect(graph.edges).toEqual(expect.arrayContaining([
      expect.objectContaining({
        source: generator?.id,
        sourceHandle: 'out',
        target: virtualPort?.id,
        targetHandle: 'tx',
      }),
      expect.objectContaining({
        source: virtualPort?.id,
        sourceHandle: 'rx',
        target: tap?.id,
        targetHandle: 'in',
      }),
    ]))
    for (const filter of filters) {
      const receiverEdge = graph.edges.find(edge => (
        edge.source === filter.id
        && edge.sourceHandle === 'out'
        && edge.targetHandle === 'in'
        && monitors.some(monitor => monitor.id === edge.target)
      ))
      expect(graph.edges, `${filter.id} input`).toEqual(expect.arrayContaining([
        expect.objectContaining({
          source: tap?.id,
          sourceHandle: 'out',
          target: filter.id,
          targetHandle: 'in',
        }),
      ]))
      expect(receiverEdge, `${filter.id} monitor edge`).toBeTruthy()
    }
    expect(graph.selectedNodeId).toBe(monitors.at(-1)?.id)
    expect(graph.nodeTabs.map(tab => tab.nodeId)).toEqual(graph.nodes.map(node => node.id))
    expect(graph.activeNodeTabId).toBe(graph.selectedNodeId)
  })

  it('opens the remote serial example with an in-graph raw TCP server/client loopback', () => {
    const demo = getDemoWorkspace('remote-serial-demo')
    const snapshot = createDemoWorkspaceSnapshot('remote-serial-demo')
    const graph = activeGraph(snapshot?.serial.graph)
    const server = graph.nodes.find(node => node.type === 'serial.remote' && node.config.role === 'server')
    const client = graph.nodes.find(node => node.type === 'serial.remote' && node.config.role === 'client')
    const clientGenerator = graph.nodes.find(node => node.type === 'serial.script.generator' && String(node.config.script).includes('client -> server'))
    const serverMonitor = graph.nodes.find(node => node.id.includes('remote-server-monitor'))
    const clientMonitor = graph.nodes.find(node => node.id.includes('remote-client-monitor'))

    expect(demo?.title).toBe('远端串口演示')
    expect(demo?.description).toContain('raw TCP')
    expect(server?.config).toEqual(expect.objectContaining({
      protocol: 'raw-tcp',
      role: 'server',
      host: '127.0.0.1',
      port: 3001,
      allowStartDisconnected: false,
      reconnect: false,
      readBufKB: 32,
    }))
    expect(client?.config).toEqual(expect.objectContaining({
      protocol: 'raw-tcp',
      role: 'client',
      host: '127.0.0.1',
      port: 3001,
      allowStartDisconnected: false,
      reconnect: true,
      readBufKB: 32,
      viewMode: 'ascii',
      autoScroll: true,
      showTimestamp: true,
    }))
    expect(server?.config).toEqual(expect.not.objectContaining({ mode: expect.anything() }))
    expect(client?.config).toEqual(expect.not.objectContaining({ mode: expect.anything() }))
    expect(clientGenerator?.config).toEqual(expect.objectContaining({ autoRun: true, intervalMs: 1500 }))
    expect(String(clientGenerator?.config.script ?? '')).toContain('client -> server')
    expect(validateGraph(graph).errors).toEqual([])

    expect(graph.edges).toEqual(expect.arrayContaining([
      expect.objectContaining({ source: clientGenerator?.id, sourceHandle: 'out', target: client?.id, targetHandle: 'tx' }),
      expect.objectContaining({ source: server?.id, sourceHandle: 'rx', target: expect.stringContaining('remote-server-tap'), targetHandle: 'in' }),
      expect.objectContaining({ source: expect.stringContaining('remote-server-tap'), sourceHandle: 'out', target: serverMonitor?.id, targetHandle: 'in' }),
      expect.objectContaining({ source: client?.id, sourceHandle: 'rx', target: expect.stringContaining('remote-client-tap'), targetHandle: 'in' }),
      expect.objectContaining({ source: expect.stringContaining('remote-client-tap'), sourceHandle: 'out', target: clientMonitor?.id, targetHandle: 'in' }),
    ]))
    expect(graph.selectedNodeId).toBe(server?.id)
  })

  it('opens protocol examples as normal topology nodes', () => {
    for (const id of ['modbus-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const graph = activeGraph(snapshot?.serial.graph)
      const master = graph.nodes.find(node => node.type === 'serial.modbus.master')
      const slave = graph.nodes.find(node => node.type === 'serial.modbus.slave')

      expect(snapshot?.serial.modbus.sessions).toHaveLength(0)
      expect(master?.config).toEqual(expect.objectContaining({ mode: 'rtu', unitIds: '1,2', functionCode: 3, autoSend: true }))
      expect(slave?.config).toEqual(expect.objectContaining({ mode: 'rtu', unitIds: '1,2' }))
      expect(protocolRequestVirtualPort(graph, master?.id, slave?.id)).toBeTruthy()
      expect(protocolResponseReturnsToMaster(graph, master?.id, slave?.id)).toBe(true)
      expect(graph.edges.some(edge => edge.source === master?.id && edge.target === slave?.id)).toBe(false)
    }

    for (const id of ['fecbus-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const graph = activeGraph(snapshot?.serial.graph)
      const master = graph.nodes.find(node => node.type === 'serial.fecbus.master')
      const slave = graph.nodes.find(node => node.type === 'serial.fecbus.slave')

      expect(snapshot?.serial.fecbus.sessions).toHaveLength(0)
      expect(master?.config).toEqual(expect.objectContaining({ sourceAddress: 1, targetAddress: 2, functionCode: 44, autoSend: true }))
      expect(slave?.config).toEqual(expect.objectContaining({ address: 2, autoStatusAnswer: true }))
      expect(protocolRequestVirtualPort(graph, master?.id, slave?.id)).toBeTruthy()
      expect(graph.edges.some(edge => edge.source === master?.id && edge.target === slave?.id)).toBe(false)
    }
  })

  it('opens script examples as normal topology nodes', () => {
    for (const id of ['script-transform-demo', 'script-analyzer-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const graph = activeGraph(snapshot?.serial.graph)
      const scriptNodes = graph.nodes.filter(node => node.type.startsWith('serial.script.'))

      expect(scriptNodes.length, id).toBeGreaterThan(0)
      expect(graph.nodeTabs.some(tab => scriptNodes.some(node => node.id === tab.nodeId)), id).toBe(true)
      expect(graph.selectedNodeId, id).toBeTruthy()
      expect(graph.activeNodeTabId, id).toBe(graph.selectedNodeId)
      expect(scriptNodes.some(node => node.type === 'serial.script.generator'), id).toBe(true)
    }
  })

  it('uses demo-specific topology graphs for every example', () => {
    const providerTypes = new Set(serialGraphProviders.map(provider => provider.type))

    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)
      const nodeTypes = graph.nodes.map(node => node.type)

      expect(nodeTypes, demo.id).toEqual(expect.arrayContaining(expectedDemoNodeTypes[demo.id]))
      expect(graph.nodes.every(node => providerTypes.has(node.type)), demo.id).toBe(true)
      expect(validateGraph(graph).errors, demo.id).toEqual([])
      expect(graph.nodeTabs.map(tab => tab.nodeId), demo.id).toEqual(graph.nodes.map(node => node.id))
      expect(graph.activeNodeTabId, demo.id).toBeTruthy()
      expect(graph.selectedNodeId, demo.id).toBe(graph.activeNodeTabId)
      expect(graph.edges.length, demo.id).toBeGreaterThan(0)
    }
  })

  it('does not include legacy serial monitor mode in demo graph configs', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)
      const monitors = graph.nodes.filter(node => node.type === 'serial.monitor')

      for (const monitor of monitors) {
        expect(monitor.config, `${demo.id} ${monitor.id}`).toEqual(expect.not.objectContaining({ mode: expect.anything() }))
        expect(['ascii', 'hex'], `${demo.id} ${monitor.id}`).toContain(monitor.config.displayMode)
      }
    }
  })

  it('does not include topology config fields that runtime nodes ignore', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)

      for (const bridge of graph.nodes.filter(node => node.type === 'serial.bridge')) {
        expect(bridge.config, `${demo.id} ${bridge.id}`).toEqual(expect.not.objectContaining({ baudRate: expect.anything() }))
      }
      for (const slave of graph.nodes.filter(node => node.type === 'serial.modbus.slave')) {
        expect(slave.config, `${demo.id} ${slave.id}`).toEqual(expect.not.objectContaining({ addressMode: expect.anything() }))
      }
    }
  })

  it('gives every demo an observable serial data path on startup', () => {
    for (const demo of listDemoWorkspaces().filter(item => item.id !== 'remote-serial-demo')) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)
      const autoNodes = graph.nodes.filter(node => (
        (
          (node.type === 'serial.script.generator' && node.config.autoRun === true)
          || (
            (node.type === 'serial.modbus.master' || node.type === 'serial.fecbus.master')
            && node.config.autoSend === true
          )
        )
        && Number(node.config.intervalMs) >= 10
      ))
      const virtualPorts = graph.nodes.filter(node => (
        node.type === 'serial.virtual'
        && String(node.config.portName ?? '').trim().length > 0
      ))
      const observableNodes = graph.nodes.filter(node => node.type === 'serial.virtual' || node.type === 'serial.monitor')
      const hasLoopingVirtualPath = autoNodes.some(sender => (
        virtualPorts.some(virtualPort => {
          const senderToVirtual = graph.edges.some(edge => (
            edge.source === sender.id
            && edge.sourceHandle === outputHandleForNode(sender.type)
            && edge.target === virtualPort.id
            && edge.targetHandle === 'tx'
          ))
          return senderToVirtual && (
            isObservableInput(virtualPort.type, 'tx')
            || reachesObservableSinkFromOutput(graph, virtualPort.id, 'rx')
          )
        })
      ))

      expect(autoNodes.length, `${demo.id} auto nodes`).toBeGreaterThan(0)
      expect(virtualPorts.length, `${demo.id} virtual ports`).toBeGreaterThan(0)
      expect(observableNodes.length, `${demo.id} observable nodes`).toBeGreaterThan(0)
      expect(hasLoopingVirtualPath, `${demo.id} auto node -> virtual -> observable path`).toBe(true)
    }
  })

  it('keeps the remote serial demo self-contained so startup does not need an external endpoint', () => {
    const snapshot = createDemoWorkspaceSnapshot('remote-serial-demo')
    const graph = activeGraph(snapshot?.serial.graph)
    const server = graph.nodes.find(node => node.type === 'serial.remote' && node.config.role === 'server')
    const client = graph.nodes.find(node => node.type === 'serial.remote' && node.config.role === 'client')
    const clientGenerator = graph.nodes.find(node => node.type === 'serial.script.generator' && node.config.autoRun === true)

    expect(server).toBeTruthy()
    expect(client).toBeTruthy()
    expect(String(clientGenerator?.config.script ?? '')).toContain('client -> server')
    expect(graph.edges.some(edge => (
      edge.source === clientGenerator?.id
      && edge.sourceHandle === 'out'
      && edge.target === client?.id
      && edge.targetHandle === 'tx'
    ))).toBe(true)
    expect(reachesObservableSinkFromOutput(graph, server!.id, 'rx')).toBe(true)
    expect(reachesObservableSinkFromOutput(graph, client!.id, 'rx')).toBe(true)
  })

  it('does not add isolated protocol loop branches to make demos look active', () => {
    for (const id of ['modbus-demo', 'fecbus-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const graph = activeGraph(snapshot?.serial.graph)
      const loopNodes = graph.nodes.filter(node => node.id.includes('-loop-'))

      expect(loopNodes, id).toEqual([])
    }
  })

  it('does not include runtime-only serial or monitor content in any example', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)

      expect(snapshot?.serial.activePortId).toBeNull()
      expect(snapshot?.serial.handles).toHaveLength(0)
      expect(snapshot?.serial.buffers).toEqual({})
      expect(snapshot?.serial.virtualPorts, demo.id).toHaveLength(0)
      expect(snapshot?.serial.bridges, demo.id).toHaveLength(0)
      expect(snapshot?.serial.monitors.activeMonitorId).toBeNull()
      expect(snapshot?.serial.monitors.sessions).toHaveLength(0)
      expect(snapshot?.serial.monitors.frames).toEqual({})
      expect(snapshot?.serial.modbus.sessions).toHaveLength(0)
      expect(snapshot?.serial.modbus.registerReadResult).toBeNull()
      expect(snapshot?.serial.modbus.unitScanResult).toBeNull()
      expect(snapshot?.serial.modbus.registerScanResult).toBeNull()
      expect(snapshot?.serial.modbus.history).toHaveLength(0)
      expect(snapshot?.serial.fecbus.sessions).toHaveLength(0)
      expect(snapshot?.serial.fecbus.history).toHaveLength(0)
    }
  })

  it('does not preload legacy resources for graph-owned demo ports', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)
      const graphPortNames = new Set(graph.nodes
        .filter(node => node.type === 'serial.virtual')
        .map(node => String(node.config.portName ?? '').trim())
        .filter(Boolean))
      const graphPortPaths = new Set([...graphPortNames].map(portName => `/tmp/${portName}`))

      const duplicatedVirtualPorts = snapshot?.serial.virtualPorts
        .filter(port => graphPortNames.has(port.Port))
        .map(port => port.Port)
      const duplicatedBridgePorts = snapshot?.serial.bridges
        .flatMap(bridge => [bridge.Port1, bridge.Port2])
        .filter(port => graphPortNames.has(port) || graphPortPaths.has(port))

      expect(snapshot?.serial.virtualPorts, demo.id).toEqual([])
      expect(snapshot?.serial.bridges, demo.id).toEqual([])
      expect(duplicatedVirtualPorts, `${demo.id} virtual ports`).toEqual([])
      expect(duplicatedBridgePorts, `${demo.id} bridge ports`).toEqual([])
    }
  })
})

function activeGraph(state: SerialGraphWorkspaceState | undefined) {
  const graph = state?.graphs.find(item => item.id === state.activeGraphId)
  expect(graph).toBeDefined()
  return graph!
}

function protocolRequestVirtualPort(
  graph: SerialGraphWorkspaceState['graphs'][number],
  masterId: string | undefined,
  slaveId: string | undefined
) {
  const masterToVirtual = graph.edges.find(edge => (
    edge.source === masterId
    && edge.sourceHandle === 'tx'
    && edge.targetHandle === 'tx'
    && graph.nodes.some(node => node.id === edge.target && node.type === 'serial.virtual')
  ))
  if (!masterToVirtual) return null
  const virtualToSlave = graph.edges.find(edge => (
    edge.source === masterToVirtual.target
    && edge.sourceHandle === 'rx'
    && edge.target === slaveId
    && edge.targetHandle === 'rx'
  ))
  return virtualToSlave ? masterToVirtual.target : null
}

function protocolResponseReturnsToMaster(
  graph: SerialGraphWorkspaceState['graphs'][number],
  masterId: string | undefined,
  slaveId: string | undefined
): boolean {
  if (!masterId || !slaveId) return false
  const visited = new Set<string>()
  const queue = [{ nodeId: slaveId, outputHandle: 'tx' }]

  while (queue.length > 0) {
    const current = queue.shift()!
    const visitKey = `${current.nodeId}:${current.outputHandle}`
    if (visited.has(visitKey)) continue
    visited.add(visitKey)

    for (const edge of graph.edges.filter(item => item.source === current.nodeId && item.sourceHandle === current.outputHandle)) {
      if (edge.target === masterId && edge.targetHandle === 'rx') {
        return true
      }
      const target = graph.nodes.find(node => node.id === edge.target)
      if (!target) continue
      for (const nextOutput of outputsAfterInput(target.type, edge.targetHandle)) {
        queue.push({ nodeId: target.id, outputHandle: nextOutput })
      }
    }
  }

  return false
}

function reachesObservableSinkFromOutput(
  graph: SerialGraphWorkspaceState['graphs'][number],
  nodeId: string,
  outputHandle: string
): boolean {
  const queue = [{ nodeId, outputHandle }]
  const visited = new Set<string>()

  while (queue.length > 0) {
    const current = queue.shift()!
    const visitKey = `${current.nodeId}:${current.outputHandle}`
    if (visited.has(visitKey)) continue
    visited.add(visitKey)

    for (const edge of graph.edges.filter(item => item.source === current.nodeId && item.sourceHandle === current.outputHandle)) {
      const target = graph.nodes.find(node => node.id === edge.target)
      if (!target) continue
      if (isObservableInput(target.type, edge.targetHandle)) {
        return true
      }
      for (const nextOutput of outputsAfterInput(target.type, edge.targetHandle)) {
        queue.push({ nodeId: target.id, outputHandle: nextOutput })
      }
    }
  }

  return false
}

function isObservableInput(nodeType: string, inputHandle: string): boolean {
  if (nodeType === 'serial.monitor' && inputHandle === 'in') {
    return true
  }
  if (nodeType === 'serial.virtual' && inputHandle === 'tx') {
    return true
  }
  if ((nodeType === 'serial.modbus.master' || nodeType === 'serial.fecbus.master') && inputHandle === 'rx') {
    return true
  }
  return false
}

function outputsAfterInput(nodeType: string, inputHandle: string): string[] {
  if (nodeType === 'serial.virtual' && inputHandle === 'tx') {
    return ['rx']
  }
  if ((nodeType === 'serial.tap' || nodeType === 'serial.tee') && inputHandle === 'in') {
    return ['out']
  }
  if (nodeType === 'serial.remote' && inputHandle === 'tx') {
    return ['rx']
  }
  if (nodeType === 'serial.script.transform' && inputHandle === 'in') {
    return ['out']
  }
  if (nodeType === 'serial.filter' && inputHandle === 'in') {
    return ['out']
  }
  if (nodeType === 'serial.bridge') {
    if (inputHandle === 'a-in') return ['b-out']
    if (inputHandle === 'b-in') return ['a-out']
  }
  if ((nodeType === 'serial.modbus.slave' || nodeType === 'serial.fecbus.slave') && inputHandle === 'rx') {
    return ['tx']
  }
  return []
}

function outputHandleForNode(nodeType: string): string {
  if (nodeType === 'serial.modbus.master' || nodeType === 'serial.fecbus.master') {
    return 'tx'
  }
  return 'out'
}
