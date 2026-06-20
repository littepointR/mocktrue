import { describe, expect, it } from 'vitest'
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
    expect(snapshot?.serial.graph.nodes.length).toBeGreaterThan(0)
    expect(snapshot?.serial.graph.edges.length).toBeGreaterThan(0)
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    expect(snapshot?.serial.workspace.selectedOperation).toBe('fecbus')
  })

  it('opens the serial graph example with normal graph workspace state', () => {
    const snapshot = createDemoWorkspaceSnapshot('serial-graph-demo')

    expect(snapshot?.serial.graph.nodes.map(node => node.type)).toEqual(expect.arrayContaining([
      'serial.sender',
      'serial.virtual',
      'serial.tap',
      'serial.receiver',
      'serial.modbus.master',
    ]))
    const sender = snapshot?.serial.graph.nodes.find(node => node.type === 'serial.sender')
    const virtualPort = snapshot?.serial.graph.nodes.find(node => node.type === 'serial.virtual')
    const receiver = snapshot?.serial.graph.nodes.find(node => node.type === 'serial.receiver')
    const tap = snapshot?.serial.graph.nodes.find(node => node.type === 'serial.tap')
    expect(sender?.config).toEqual(expect.objectContaining({
      autoSend: true,
      intervalMs: 1000,
      mode: 'ascii',
    }))
    expect(String(sender?.config.payload ?? '').length).toBeGreaterThan(0)
    expect(snapshot?.serial.graph.edges).toEqual(expect.arrayContaining([
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
    expect(snapshot?.serial.graph.selectedNodeId).toBe(receiver?.id)
    expect(snapshot?.serial.graph.edges.length).toBeGreaterThanOrEqual(3)
    expect(snapshot?.serial.workspace.selectedOperation).toBe('graph')
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    if (snapshot?.serial.workspace.editorLayout.type === 'group') {
      expect(snapshot.serial.workspace.editorLayout.tabs).toContain('serial.graph')
      expect(snapshot.serial.workspace.activeByGroup[snapshot.serial.workspace.editorLayout.id]).toBe('serial.graph')
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
        expect(layout.tabs).toContain(`modbus:${session?.ID}`)
        if (id === 'modbus-demo') {
          expect(snapshot?.serial.workspace.activeByGroup[layout.id]).toBe(`modbus:${session?.ID}`)
        }
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
        expect(layout.tabs).toContain(`fecbus:${session?.ID}`)
        expect(snapshot?.serial.workspace.activeByGroup[layout.id]).toBe(`fecbus:${session?.ID}`)
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
})
