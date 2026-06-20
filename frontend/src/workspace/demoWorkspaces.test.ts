import { describe, expect, it } from 'vitest'
import { serialGraphProviders, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from './demoWorkspaces'
import { workspaceKind } from './workspaceSnapshot'

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
    expect(first?.serial.virtualPorts).toHaveLength(1)
    expect(first?.serial.handles).toHaveLength(0)
    expect(first?.serial.buffers).toEqual({})
    expect(first?.serial.workspace.editorLayout.type).toBe('group')
    expect(first?.serial.virtualPorts[0].ID).not.toBe(second?.serial.virtualPorts[0].ID)
  })

  it('builds a full example from normal configurable resources only', () => {
    const snapshot = createDemoWorkspaceSnapshot('full-workspace-demo')
    const graph = activeGraph(snapshot?.serial.graph)

    expect(snapshot?.settings.serial.TerminalFontFamily).toBe('Menlo')
    expect(snapshot?.serial.virtualPorts.length).toBeGreaterThanOrEqual(3)
    expect(snapshot?.serial.bridges).toHaveLength(1)
    expect(snapshot?.serial.handles).toHaveLength(0)
    expect(snapshot?.serial.buffers).toEqual({})
    expect(snapshot?.serial.monitors.sessions).toHaveLength(0)
    expect(Object.values(snapshot?.serial.monitors.frames ?? {}).flat()).toHaveLength(0)
    expect(snapshot?.serial.modbus.sessions).toHaveLength(1)
    expect(snapshot?.serial.modbus.history).toHaveLength(0)
    expect(snapshot?.serial.fecbus.sessions).toHaveLength(1)
    expect(snapshot?.serial.fecbus.framePages[snapshot.serial.fecbus.activeSessionId ?? '']?.Total).toBe(1)
    expect(graph.nodes.length).toBeGreaterThan(0)
    expect(graph.edges.length).toBeGreaterThan(0)
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

  it('opens Modbus examples with a normal saved session tab', () => {
    for (const id of ['modbus-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const session = snapshot?.serial.modbus.sessions[0]
      const layout = snapshot?.serial.workspace.editorLayout

      expect(session?.ID).toBe(snapshot?.serial.modbus.activeSessionId)
      expect(session?.Status).toBe('stopped')
      expect(session?.RxBytes).toBe(0)
      expect(session?.TxBytes).toBe(0)
      expect(session?.Config.PortName).toBe(snapshot?.serial.modbus.portForm.port)
      expect(layout?.type).toBe('group')
      if (layout?.type === 'group') {
        const graph = activeGraph(snapshot?.serial.graph)
        expect(layout.tabs).toContain(`graph:${graph.id}`)
        expect(snapshot?.serial.workspace.activeByGroup[layout.id]).toBe(`graph:${graph.id}`)
      }
    }
  })

  it('opens FECbus examples with a normal saved session tab', () => {
    for (const id of ['fecbus-demo', 'full-workspace-demo']) {
      const snapshot = createDemoWorkspaceSnapshot(id)
      const session = snapshot?.serial.fecbus.sessions[0]
      const layout = snapshot?.serial.workspace.editorLayout

      expect(session?.ID).toBe(snapshot?.serial.fecbus.activeSessionId)
      expect(session?.Status).toBe('stopped')
      expect(session?.RxBytes).toBe(0)
      expect(session?.TxBytes).toBe(0)
      expect(session?.Config.PortName).toBe(snapshot?.serial.fecbus.portForm.port)
      expect(snapshot?.serial.fecbus.sendForm.functionCode).toBe(44)
      expect(layout?.type).toBe('group')
      if (layout?.type === 'group') {
        const graph = activeGraph(snapshot?.serial.graph)
        expect(layout.tabs).toContain(`graph:${graph.id}`)
        expect(snapshot?.serial.workspace.activeByGroup[layout.id]).toBe(`graph:${graph.id}`)
      }
    }
  })

  it('does not include runtime-only serial or monitor content in any example', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const expectedModbusSessions = ['modbus-demo', 'full-workspace-demo'].includes(demo.id) ? 1 : 0
      const expectedFecbusSessions = ['fecbus-demo', 'full-workspace-demo'].includes(demo.id) ? 1 : 0

      expect(snapshot?.serial.activePortId).toBeNull()
      expect(snapshot?.serial.handles).toHaveLength(0)
      expect(snapshot?.serial.buffers).toEqual({})
      expect(snapshot?.serial.monitors.activeMonitorId).toBeNull()
      expect(snapshot?.serial.monitors.sessions).toHaveLength(0)
      expect(snapshot?.serial.monitors.frames).toEqual({})
      expect(snapshot?.serial.modbus.sessions).toHaveLength(expectedModbusSessions)
      expect(snapshot?.serial.modbus.registerReadResult).toBeNull()
      expect(snapshot?.serial.modbus.unitScanResult).toBeNull()
      expect(snapshot?.serial.modbus.registerScanResult).toBeNull()
      expect(snapshot?.serial.modbus.history).toHaveLength(0)
      expect(snapshot?.serial.fecbus.sessions).toHaveLength(expectedFecbusSessions)
      expect(snapshot?.serial.fecbus.history).toHaveLength(0)
    }
  })

  it('does not preload legacy virtual resources with graph-owned port names', () => {
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
