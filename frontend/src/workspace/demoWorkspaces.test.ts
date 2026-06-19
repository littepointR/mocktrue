import { describe, expect, it } from 'vitest'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from './demoWorkspaces'
import { workspaceKind } from './workspaceSnapshot'

describe('demoWorkspaces', () => {
  it('lists one readonly demo for each feature group', () => {
    const demos = listDemoWorkspaces()
    const ids = demos.map(demo => demo.id)

    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toEqual([
      'serial-open-demo',
      'virtual-port-demo',
      'bridge-demo',
      'monitor-demo',
      'full-workspace-demo',
    ])
    expect(demos.every(demo => demo.readonly)).toBe(true)
  })

  it('returns null for an unknown demo id', () => {
    expect(getDemoWorkspace('missing-demo')).toBeNull()
    expect(createDemoWorkspaceSnapshot('missing-demo')).toBeNull()
  })

  it('creates fresh readonly snapshots with valid workspace structure', () => {
    const first = createDemoWorkspaceSnapshot('serial-open-demo')
    const second = createDemoWorkspaceSnapshot('serial-open-demo')

    expect(first?.kind).toBe(workspaceKind)
    expect(first?.serial.virtualPorts).toHaveLength(1)
    expect(first?.serial.handles).toHaveLength(1)
    expect(first?.serial.buffers[first.serial.handles[0].id]).toHaveLength(2)
    expect(first?.serial.workspace.editorLayout.type).toBe('group')
    expect(first?.serial.workspace.tabStates[first.serial.handles[0].id].sendPanel.sendHistory).toHaveLength(2)
    expect(first?.serial.virtualPorts[0].ID).not.toBe(second?.serial.virtualPorts[0].ID)
  })

  it('builds a full demo that exercises virtual ports, bridges, monitor data, buffers, layout, and settings', () => {
    const snapshot = createDemoWorkspaceSnapshot('full-workspace-demo')

    expect(snapshot?.settings.serial.TerminalFontFamily).toBe('Menlo')
    expect(snapshot?.serial.virtualPorts.length).toBeGreaterThanOrEqual(3)
    expect(snapshot?.serial.bridges).toHaveLength(1)
    expect(snapshot?.serial.handles).toHaveLength(1)
    expect(Object.values(snapshot?.serial.buffers ?? {}).flat()).not.toHaveLength(0)
    expect(snapshot?.serial.monitors.sessions).toHaveLength(1)
    expect(Object.values(snapshot?.serial.monitors.frames ?? {}).flat()).toHaveLength(3)
    expect(snapshot?.serial.workspace.editorLayout.type).toBe('split')
    expect(snapshot?.serial.workspace.selectedOperation).toBe('monitor')
  })
})
