import { describe, expect, it } from 'vitest'
import { serialGraphProviders, validateGraph, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from './demoWorkspaces'
import { workspaceKind } from './workspaceSnapshot'

const expectedDemoNodeTypes: Record<string, string[]> = {
  'serial-open-demo': ['serial.sender', 'serial.virtual', 'serial.receiver'],
  'virtual-port-demo': ['serial.sender', 'serial.virtual', 'serial.receiver'],
  'bridge-demo': ['serial.sender', 'serial.virtual', 'serial.bridge', 'serial.receiver'],
  'monitor-demo': ['serial.sender', 'serial.virtual', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'script-transform-demo': [
    'serial.script.generator',
    'serial.virtual',
    'serial.script.transform',
    'serial.tap',
    'serial.receiver',
    'serial.monitor',
  ],
  'script-analyzer-demo': [
    'serial.script.generator',
    'serial.virtual',
    'serial.tap',
    'serial.receiver',
    'serial.script.analyzer',
  ],
  'modbus-demo': ['serial.virtual', 'serial.modbus.master', 'serial.modbus.slave', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'fecbus-demo': ['serial.virtual', 'serial.fecbus.master', 'serial.fecbus.slave', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'serial-graph-demo': ['serial.sender', 'serial.virtual', 'serial.tap', 'serial.receiver', 'serial.modbus.master', 'serial.monitor'],
  'full-workspace-demo': [
    'serial.sender',
    'serial.virtual',
    'serial.bridge',
    'serial.monitor',
    'serial.tap',
    'serial.receiver',
    'serial.script.generator',
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
      'serial.sender',
      'serial.virtual',
      'serial.tap',
      'serial.receiver',
      'serial.modbus.master',
    ]))
    const providerTypes = new Set(serialGraphProviders.map(provider => provider.type))
    expect(graph.nodes.every(node => providerTypes.has(node.type))).toBe(true)
    const sender = graph.nodes.find(node => node.type === 'serial.sender')
    const virtualPort = graph.nodes.find(node => node.type === 'serial.virtual')
    const receiver = graph.nodes.find(node => node.type === 'serial.receiver')
    const tap = graph.nodes.find(node => node.type === 'serial.tap')
    expect(sender?.config).toEqual(expect.objectContaining({
      autoSend: true,
      intervalMs: 1000,
      mode: 'ascii',
    }))
    expect(String(sender?.config.payload ?? '').length).toBeGreaterThan(0)
    expect(graph.edges).toEqual(expect.arrayContaining([
      expect.objectContaining({
        source: sender?.id,
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
        target: receiver?.id,
        targetHandle: 'in',
      }),
    ]))
    expect(graph.selectedNodeId).toBe(receiver?.id)
    expect(graph.nodeTabs.map(tab => tab.nodeId)).toEqual(graph.nodes.map(node => node.id))
    expect(graph.activeNodeTabId).toBe(receiver?.id)
    expect(graph.edges.length).toBeGreaterThanOrEqual(3)
    expect(snapshot?.serial.workspace.selectedOperation).toBe('graph')
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    if (snapshot?.serial.workspace.editorLayout.type === 'group') {
      expect(snapshot.serial.workspace.editorLayout.tabs).toContain(`graph:${graph.id}`)
      expect(snapshot.serial.workspace.activeByGroup[snapshot.serial.workspace.editorLayout.id]).toBe(`graph:${graph.id}`)
    }
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
        expect(monitor.config, `${demo.id} ${monitor.id}`).toEqual(expect.objectContaining({ displayMode: 'hex' }))
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

  it('gives every demo a looping virtual serial data path on startup', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const graph = activeGraph(snapshot?.serial.graph)
      const autoNodes = graph.nodes.filter(node => (
        (
          (node.type === 'serial.script.generator' && node.config.autoRun === true)
          || (
            (node.type === 'serial.sender' || node.type === 'serial.modbus.master' || node.type === 'serial.fecbus.master')
            && node.config.autoSend === true
          )
        )
        && Number(node.config.intervalMs) >= 10
        && (node.type !== 'serial.sender' || String(node.config.payload ?? '').length > 0)
      ))
      const virtualPorts = graph.nodes.filter(node => (
        node.type === 'serial.virtual'
        && String(node.config.portName ?? '').trim().length > 0
      ))
      const receivers = graph.nodes.filter(node => node.type === 'serial.receiver')
      const hasLoopingVirtualPath = autoNodes.some(sender => (
        virtualPorts.some(virtualPort => (
          graph.edges.some(edge => (
            edge.source === sender.id
            && edge.sourceHandle === outputHandleForNode(sender.type)
            && edge.target === virtualPort.id
            && edge.targetHandle === 'tx'
          ))
          && reachesReceiverFromOutput(graph, virtualPort.id, 'rx')
        ))
      ))

      expect(autoNodes.length, `${demo.id} auto nodes`).toBeGreaterThan(0)
      expect(virtualPorts.length, `${demo.id} virtual ports`).toBeGreaterThan(0)
      expect(receivers.length, `${demo.id} receivers`).toBeGreaterThan(0)
      expect(hasLoopingVirtualPath, `${demo.id} auto node -> virtual -> receiver path`).toBe(true)
    }
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

function reachesReceiverFromOutput(
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
      if (target.type === 'serial.receiver' && edge.targetHandle === 'in') {
        return true
      }
      for (const nextOutput of outputsAfterInput(target.type, edge.targetHandle)) {
        queue.push({ nodeId: target.id, outputHandle: nextOutput })
      }
    }
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
  if (nodeType === 'serial.script.transform' && inputHandle === 'in') {
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
