import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SerialView from './SerialView.vue'
import { useSerialStore } from '../stores/serialStore'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'

vi.mock('../services/serialService', () => ({
  serialService: {
    listPorts: vi.fn(async () => []),
  },
}))

vi.mock('../services/serialEvents', () => ({
  serialEvents: {
    onData: vi.fn(() => vi.fn()),
  },
}))

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => ({
  ListMonitors: vi.fn(async () => []),
  QueryMonitorFrames: vi.fn(async () => ({ Frames: [], Total: 0, NextOffset: 0 })),
  StartMonitor: vi.fn(async () => null),
  StopMonitor: vi.fn(async () => undefined),
  DeleteMonitor: vi.fn(async () => undefined),
  ExportMonitor: vi.fn(async () => ''),
  SetMonitorAutoSave: vi.fn(async () => null),
  ClearMonitorFrames: vi.fn(async () => undefined),
}))

describe('SerialView workspace layout', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('uses persisted editor layout after remounting', async () => {
    const serial = useSerialStore()
    serial.handles.set('port-1', {
      ID: 'port-1',
      Config: {
        PortName: '/tmp/ttyA',
        BaudRate: 115200,
        DataBits: 8,
        StopBits: '1',
        Parity: 'none',
        FlowMode: 'none',
        ReadBufKB: 32,
      },
      IsOpen: true,
      RxBytes: 0,
      TxBytes: 0,
    })
    serial.handles.set('port-2', {
      ID: 'port-2',
      Config: {
        PortName: '/tmp/ttyB',
        BaudRate: 115200,
        DataBits: 8,
        StopBits: '1',
        Parity: 'none',
        FlowMode: 'none',
        ReadBufKB: 32,
      },
      IsOpen: true,
      RxBytes: 0,
      TxBytes: 0,
    })
    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['port-1'] },
        { type: 'group', id: 'group-right', tabs: ['port-2'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'port-1', 'group-right': 'port-2' })

    const first = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    first.unmount()
    const second = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })

    expect(second.find('.layout-json').text()).toContain('split-1')
    expect(second.find('.layout-json').text()).toContain('group-left')
  })
})

const stubs = {
  EditorLayoutNode: {
    props: ['node', 'activeByGroup'],
    template: '<pre class="layout-json">{{ JSON.stringify(node) }} {{ JSON.stringify(activeByGroup) }}</pre>',
  },
  PortConfigPanel: true,
  VirtualPairPanel: true,
  BridgePanel: true,
  MonitorPanel: true,
}
