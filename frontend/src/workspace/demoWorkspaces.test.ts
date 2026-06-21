import { describe, expect, it } from 'vitest'
import { serialGraphProviders, validateGraph, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from './demoWorkspaces'
import { workspaceKind } from './workspaceSnapshot'

const expectedDemoNodeTypes: Record<string, string[]> = {
  'serial-open-demo': ['serial.sender', 'serial.virtual', 'serial.receiver'],
  'virtual-port-demo': ['serial.sender', 'serial.virtual', 'serial.receiver'],
  'bridge-demo': ['serial.sender', 'serial.virtual', 'serial.bridge', 'serial.receiver'],
  'monitor-demo': ['serial.sender', 'serial.virtual', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'modbus-demo': ['serial.modbus.master', 'serial.modbus.slave', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'fecbus-demo': ['serial.fecbus.master', 'serial.fecbus.slave', 'serial.tap', 'serial.receiver', 'serial.monitor'],
  'serial-graph-demo': ['serial.sender', 'serial.virtual', 'serial.tap', 'serial.receiver', 'serial.modbus.master', 'serial.monitor'],
  'full-workspace-demo': [
    'serial.sender',
    'serial.virtual',
    'serial.bridge',
    'serial.monitor',
    'serial.tap',
    'serial.receiver',
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
      expect(master?.config).toEqual(expect.objectContaining({ mode: 'rtu', unitIds: '1,2', functionCode: 3 }))
      expect(slave?.config).toEqual(expect.objectContaining({ mode: 'rtu', unitIds: '1,2' }))
      expect(graph.edges.some(edge => edge.source === master?.id && edge.target === slave?.id)).toBe(true)
    }

    for (const id of ['fecbus-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const graph = activeGraph(snapshot?.serial.graph)
      const master = graph.nodes.find(node => node.type === 'serial.fecbus.master')
      const slave = graph.nodes.find(node => node.type === 'serial.fecbus.slave')

      expect(snapshot?.serial.fecbus.sessions).toHaveLength(0)
      expect(master?.config).toEqual(expect.objectContaining({ sourceAddress: 1, targetAddress: 2, functionCode: 44 }))
      expect(slave?.config).toEqual(expect.objectContaining({ address: 2, autoStatusAnswer: true }))
      expect(graph.edges.some(edge => edge.source === master?.id && edge.target === slave?.id)).toBe(true)
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
