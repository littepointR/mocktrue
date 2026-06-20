import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SerialView from './SerialView.vue'
import { useSerialStore } from '../stores/serialStore'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'
import { useMonitorStore } from '../stores/monitorStore'
import { useModbusStore } from '../stores/modbusStore'
import { useFecbusStore } from '../stores/fecbusStore'
import { FrameMode, SessionRole } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/modbus/models.js'
import { SessionRole as FecbusSessionRole } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/fecbus/models.js'
import { createDemoWorkspaceSnapshot } from '../../workspace/demoWorkspaces'
import { restoreWorkspaceSnapshot } from '../../workspace/workspaceSession'

vi.mock('../services/serialService', () => ({
  serialService: {
    listPorts: vi.fn(async () => []),
    listModbusSessions: vi.fn(async () => []),
    listFecbusSessions: vi.fn(async () => []),
    queryFecbusFrames: vi.fn(async () => ({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })),
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
  ClearMonitorFrames: vi.fn(async () => undefined),
  ListModbusSessions: vi.fn(async () => []),
  OpenModbusSession: vi.fn(async () => null),
  CloseModbusSession: vi.fn(async () => undefined),
  ModbusMasterRequest: vi.fn(async () => null),
  StartModbusSlave: vi.fn(async () => null),
  StopModbusSlave: vi.fn(async () => undefined),
  UpdateModbusSlaveData: vi.fn(async () => undefined),
  ListFecbusSessions: vi.fn(async () => []),
  OpenFecbusSession: vi.fn(async () => null),
  CloseFecbusSession: vi.fn(async () => undefined),
  FecbusSendRequest: vi.fn(async () => null),
  StartFecbusSlave: vi.fn(async () => null),
  StopFecbusSlave: vi.fn(async () => undefined),
  UpdateFecbusSlaveState: vi.fn(async () => undefined),
  QueryFecbusFrames: vi.fn(async () => ({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })),
  ClearFecbusFrames: vi.fn(async () => undefined),
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

  it('keeps the selected monitor tab active when it shares a group with the active serial port', () => {
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
    serial.setActivePort('port-1')
    const monitor = useMonitorStore()
    monitor.restoreState({
      activeMonitorId: 'mon-1',
      filters: {},
      sessions: [{
        ID: 'mon-1',
        Name: '串口监控演示',
        Provider: 'bridge',
        PortA: '/tmp/mon-a',
        PortB: '/tmp/mon-b',
        ExternalPort: '',
        AutoVirtualPortID: '',
        Config: {
          PortName: '',
          BaudRate: 115200,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Encoding: 'utf-8',
        Status: 'stopped',
        RxBytes: 0,
        TxBytes: 0,
        FrameCount: 0,
        StartedAt: '',
        StoppedAt: '',
        Error: '',
      }],
      frames: {},
    })
    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'group',
      id: 'group-main',
      tabs: ['port-1', 'monitor:mon-1'],
    })
    workspace.setActiveByGroup({ 'group-main': 'monitor:mon-1' })

    mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })

    expect(workspace.activeByGroup['group-main']).toBe('monitor:mon-1')
  })

  it('opens the Modbus operation panel from the serial module view id', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await wrapper.setProps({ activeViewId: 'serial.modbus', activeViewVersion: 1 })

    expect(wrapper.find('[data-testid="modbus-stub"]').exists()).toBe(true)
  })

  it('opens the FECbus operation panel from the serial module view id', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await wrapper.setProps({ activeViewId: 'serial.fecbus', activeViewVersion: 1 })

    expect(wrapper.find('[data-testid="fecbus-stub"]').exists()).toBe(true)
  })

  it('opens the serial graph workbench from the serial module view id', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await wrapper.setProps({ activeViewId: 'serial.graph', activeViewVersion: 1 })

    expect(wrapper.find('[data-testid="serial-graph-panel"]').exists()).toBe(true)
    expect(wrapper.find('.serial-view__operation-panel').classes()).not.toContain('is-open')
    expect(wrapper.find('.tabs-json').text()).toContain('"kind":"graph"')
    expect(wrapper.find('.tabs-json').text()).toContain('节点编辑')
  })

  it('keeps the serial graph tab in the content area when another operation panel is opened', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await wrapper.setProps({ activeViewId: 'serial.graph', activeViewVersion: 1 })
    await wrapper.setProps({ activeViewId: 'serial.open', activeViewVersion: 2 })

    expect(wrapper.find('[data-testid="open-port-stub"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-panel"]').exists()).toBe(true)
    expect(wrapper.find('.tabs-json').text()).toContain('"id":"serial.graph"')
  })

  it('focuses an existing serial graph tab when selecting the serial graph view again', async () => {
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
    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'group',
      id: 'group-main',
      tabs: ['port-1', 'serial.graph'],
    })
    workspace.setActiveByGroup({ 'group-main': 'port-1' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await wrapper.setProps({ activeViewId: 'serial.graph', activeViewVersion: 1 })

    expect(workspace.activeByGroup['group-main']).toBe('serial.graph')
    expect(wrapper.find('.serial-view__operation-panel').classes()).not.toContain('is-open')
  })

  it('shows the selected operation panel when the serial module opens on its default view', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: 'serial.open', activeViewVersion: 1 },
      global: { stubs },
    })

    expect(wrapper.find('[data-testid="open-port-stub"]').exists()).toBe(true)
    expect(wrapper.find('.serial-view__operation-panel').classes()).toContain('is-open')
  })

  it('adds Modbus sessions to the editor tab layout', () => {
    const modbus = useModbusStore()
    modbus.restoreState({
      ...modbus.exportState(),
      activeSessionId: 'modbus-1',
      sessions: [{
        ID: 'modbus-1',
        Name: 'Modbus 主站',
        Mode: FrameMode.FrameModeRTU,
        Role: SessionRole.SessionRoleMaster,
        Config: {
          PortName: '/tmp/ttyM0',
          BaudRate: 115200,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Status: 'open',
        RxBytes: 0,
        TxBytes: 0,
        SlaveRunning: false,
        UnitID: 1,
        UnitIDs: [],
        StartedAt: '',
        StoppedAt: '',
        LastError: '',
      }],
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })

    expect(wrapper.find('.tabs-json').text()).toContain('modbus:modbus-1')
    const layout = useSerialWorkspaceStore().editorLayout
    expect(layout.type).toBe('group')
    if (layout.type === 'group') {
      expect(layout.tabs).toEqual(['modbus:modbus-1'])
      expect(useSerialWorkspaceStore().activeByGroup[layout.id]).toBe('modbus:modbus-1')
    }
  })

  it('adds FECbus sessions to the editor tab layout', () => {
    const fecbus = useFecbusStore()
    fecbus.restoreState({
      ...fecbus.exportState(),
      activeSessionId: 'fec-1',
      sessions: [{
        ID: 'fec-1',
        Name: 'FECbus 主控',
        Role: FecbusSessionRole.SessionRoleMaster,
        Config: {
          PortName: '/tmp/ttyF0',
          BaudRate: 9600,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Status: 'open',
        RxBytes: 0,
        TxBytes: 0,
        SlaveRunning: false,
        SourceAddress: 1,
        TargetAddress: 2,
        SlaveUnits: [],
        StartedAt: '',
        StoppedAt: '',
        LastError: '',
      }],
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })

    expect(wrapper.find('.tabs-json').text()).toContain('fecbus:fec-1')
    const layout = useSerialWorkspaceStore().editorLayout
    expect(layout.type).toBe('group')
    if (layout.type === 'group') {
      expect(layout.tabs).toEqual(['fecbus:fec-1'])
      expect(useSerialWorkspaceStore().activeByGroup[layout.id]).toBe('fecbus:fec-1')
    }
  })

  it('renders Modbus session content in the editor area', async () => {
    const modbus = useModbusStore()
    modbus.restoreState({
      ...modbus.exportState(),
      activeSessionId: 'modbus-1',
      sessions: [{
        ID: 'modbus-1',
        Name: 'Modbus 主站',
        Mode: FrameMode.FrameModeRTU,
        Role: SessionRole.SessionRoleMaster,
        Config: {
          PortName: '/tmp/ttyM0',
          BaudRate: 115200,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Status: 'open',
        RxBytes: 0,
        TxBytes: 0,
        SlaveRunning: false,
        UnitID: 1,
        UnitIDs: [],
        StartedAt: '',
        StoppedAt: '',
        LastError: '',
      }],
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: {
        stubs: {
          PortConfigPanel: true,
          VirtualPairPanel: true,
          BridgePanel: true,
          MonitorPanel: true,
        },
      },
    })

    expect(wrapper.find('.serial-view__main').text()).toContain('Modbus 主站')
    expect(wrapper.find('.serial-view__main').text()).toContain('主站')
    expect(wrapper.find('.serial-view__main').text()).toContain('读')
    expect(wrapper.find('.serial-view__main').text()).toContain('配置')
    expect(wrapper.find('.serial-view__main').text()).toContain('Coils')
    expect(wrapper.find('.serial-view__main').text()).toContain('Holding Registers')
  })

  it('renders Modbus example content in the editor area after restoring its workspace snapshot', async () => {
    const snapshot = createDemoWorkspaceSnapshot('modbus-demo')
    expect(snapshot).not.toBeNull()
    await restoreWorkspaceSnapshot(snapshot!)

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: {
        stubs: {
          PortConfigPanel: true,
          VirtualPairPanel: true,
          BridgePanel: true,
          MonitorPanel: true,
        },
      },
    })

    expect(wrapper.find('.serial-view__main').text()).toContain('Modbus RTU 演示')
    expect(wrapper.find('.serial-view__main').text()).toContain('读')
    expect(wrapper.find('.serial-view__main').text()).toContain('Coils')
    expect(wrapper.find('.serial-view__main').text()).toContain('Holding Registers')
  })
})

const stubs = {
  EditorLayoutNode: {
    props: ['node', 'activeByGroup', 'tabs'],
    template: '<div><pre class="layout-json">{{ JSON.stringify(node) }} {{ JSON.stringify(activeByGroup) }}</pre><pre class="tabs-json">{{ JSON.stringify(tabs) }}</pre><div v-if="tabs.some(tab => tab.kind === \'graph\')" data-testid="serial-graph-panel" /></div>',
  },
  PortConfigPanel: {
    template: '<div data-testid="open-port-stub" />',
  },
  VirtualPairPanel: true,
  BridgePanel: true,
  MonitorPanel: true,
  ModbusPanel: {
    template: '<div data-testid="modbus-stub" class="modbus-panel" />',
  },
  FecbusPanel: {
    template: '<div data-testid="fecbus-stub" class="fecbus-panel" />',
  },
}
