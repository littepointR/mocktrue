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
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('group')
    expect(snapshot?.serial.workspace.selectedOperation).toBe('modbus')
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
        expect(layout.tabs).toEqual([`modbus:${session?.ID}`])
        expect(snapshot?.serial.workspace.activeByGroup[layout.id]).toBe(`modbus:${session?.ID}`)
      }
    }
  })

  it('does not include runtime-only serial, monitor, or Modbus content in any example', () => {
    for (const demo of listDemoWorkspaces()) {
      const snapshot = createDemoWorkspaceSnapshot(demo.id)
      const expectedModbusSessions = ['modbus-demo', 'full-workspace-demo'].includes(demo.id) ? 1 : 0

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
    }
  })
})
