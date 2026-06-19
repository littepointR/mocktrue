import type { SerialConfig } from '../../bindings/github.com/suyue/mocktrue/internal/modules/serial/port/models.js'
import type { AutoSaveOptions, Frame, SessionInfo } from '../../bindings/github.com/suyue/mocktrue/internal/modules/serial/monitor/models.js'
import { defaultGlobalSettings, defaultSerialSettings, type SettingsSnapshot } from '../settings/stores/settingsStore'
import type { Bridge, VirtualPort } from '../serial/stores/virtualStore'
import type { SerialTabWorkspaceState, SerialWorkspaceState } from '../serial/stores/workspaceStore'
import type { MonitorFilterState, MonitorWorkspaceState } from '../serial/stores/monitorStore'
import type { EditorLayoutNode } from '../serial/views/editorLayout'
import { bytesToBase64, workspaceKind, type WorkspaceSnapshot } from './workspaceSnapshot'

export interface DemoWorkspace {
  id: string
  title: string
  description: string
  readonly: true
}

interface DemoWorkspaceDefinition extends DemoWorkspace {
  snapshotFactory: () => WorkspaceSnapshot
}

let demoSequence = 0

const demoDefinitions: DemoWorkspaceDefinition[] = [
  {
    id: 'serial-open-demo',
    title: '串口收发演示',
    description: '展示打开虚拟串口、接收缓冲、发送历史、自动发送和收发计数。',
    readonly: true,
    snapshotFactory: createSerialOpenDemo,
  },
  {
    id: 'virtual-port-demo',
    title: '虚拟串口演示',
    description: '展示多个自动创建的单端虚拟串口，适合验证虚拟串口资源管理。',
    readonly: true,
    snapshotFactory: createVirtualPortDemo,
  },
  {
    id: 'bridge-demo',
    title: '串口桥接演示',
    description: '展示两个自动虚拟串口之间的桥接配置。',
    readonly: true,
    snapshotFactory: createBridgeDemo,
  },
  {
    id: 'monitor-demo',
    title: '串口监控演示',
    description: '展示桥接式串口监听会话、过滤条件、帧列表和 Modbus 解析结果。',
    readonly: true,
    snapshotFactory: createMonitorDemo,
  },
  {
    id: 'full-workspace-demo',
    title: '完整工作区演示',
    description: '展示设置、虚拟串口、桥接、监控、收发数据、发送历史和拆分布局。',
    readonly: true,
    snapshotFactory: createFullWorkspaceDemo,
  },
]

export function listDemoWorkspaces(): DemoWorkspace[] {
  return demoDefinitions.map(({ snapshotFactory: _snapshotFactory, ...demo }) => ({ ...demo }))
}

export function getDemoWorkspace(id: string): DemoWorkspace | null {
  const demo = demoDefinitions.find(item => item.id === id)
  if (!demo) return null
  const { snapshotFactory: _snapshotFactory, ...metadata } = demo
  return { ...metadata }
}

export function createDemoWorkspaceSnapshot(id: string): WorkspaceSnapshot | null {
  return demoDefinitions.find(item => item.id === id)?.snapshotFactory() ?? null
}

function createSerialOpenDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `mocktrue-demo-open-${suffix}`
  const portPath = toPortPath(portName)
  const handleId = `demo-open-handle-${suffix}`

  return snapshot({
    activePortId: handleId,
    virtualPorts: [virtualPort(`demo-open-port-${suffix}`, portName)],
    handles: [{
      id: handleId,
      config: serialConfig(portPath, 115200),
      isOpen: true,
      rxBytes: 28,
      txBytes: 18,
    }],
    buffers: {
      [handleId]: [
        bufferChunk('MockTrue RX hello\n', 1718845201000),
        bufferChunk('status=ok voltage=3.30\n', 1718845202500),
      ],
    },
    workspace: workspace({
      selectedOperation: 'open',
      editorLayout: { type: 'group', id: 'group-1', tabs: [handleId] },
      activeByGroup: { 'group-1': handleId },
      tabStates: {
        [handleId]: tabState({
          sendData: 'AT+STATUS?',
          autoSend: true,
          sendIntervalMs: 500,
          history: [
            { id: 1, content: 'AT', mode: 'ascii' },
            { id: 2, content: 'AT+STATUS?', mode: 'ascii' },
          ],
        }),
      },
    }),
  })
}

function createVirtualPortDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portNames = ['sensor', 'gateway', 'logger'].map(name => `mocktrue-demo-${name}-${suffix}`)

  return snapshot({
    virtualPorts: portNames.map((portName, index) => virtualPort(`demo-vport-${index + 1}-${suffix}`, portName)),
    workspace: workspace({
      selectedOperation: 'virtual',
      editorLayout: { type: 'group', id: 'group-1', tabs: [] },
      activeByGroup: { 'group-1': null },
    }),
  })
}

function createBridgeDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portA = `mocktrue-demo-bridge-a-${suffix}`
  const portB = `mocktrue-demo-bridge-b-${suffix}`

  return snapshot({
    virtualPorts: [
      virtualPort(`demo-bridge-a-${suffix}`, portA),
      virtualPort(`demo-bridge-b-${suffix}`, portB),
    ],
    bridges: [{
      ID: `demo-bridge-${suffix}`,
      Port1: toPortPath(portA),
      Port2: toPortPath(portB),
      BaudRate: 115200,
    }],
    workspace: workspace({
      selectedOperation: 'bridge',
      editorLayout: { type: 'group', id: 'group-1', tabs: [] },
      activeByGroup: { 'group-1': null },
    }),
  })
}

function createMonitorDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portA = `mocktrue-demo-monitor-a-${suffix}`
  const portB = `mocktrue-demo-monitor-b-${suffix}`
  const monitorId = `demo-monitor-${suffix}`

  return snapshot({
    virtualPorts: [
      virtualPort(`demo-monitor-a-${suffix}`, portA),
      virtualPort(`demo-monitor-b-${suffix}`, portB),
    ],
    monitors: monitorState(monitorId, toPortPath(portA), toPortPath(portB)),
    workspace: workspace({
      selectedOperation: 'monitor',
      editorLayout: { type: 'group', id: 'group-1', tabs: [monitorTabId(monitorId)] },
      activeByGroup: { 'group-1': monitorTabId(monitorId) },
    }),
  })
}

function createFullWorkspaceDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const terminalPort = `mocktrue-demo-terminal-${suffix}`
  const monitorPortA = `mocktrue-demo-full-a-${suffix}`
  const monitorPortB = `mocktrue-demo-full-b-${suffix}`
  const handleId = `demo-full-handle-${suffix}`
  const monitorId = `demo-full-monitor-${suffix}`

  return snapshot({
    settings: {
      global: { Theme: 'dark' },
      serial: {
        ...defaultSerialSettings,
        BaudRate: 115200,
        TerminalFontFamily: 'Menlo',
        TerminalFontSize: 15,
        TextEncoding: 'utf-8',
        EnterString: '\r\n',
      },
    },
    activePortId: handleId,
    virtualPorts: [
      virtualPort(`demo-full-terminal-${suffix}`, terminalPort),
      virtualPort(`demo-full-a-${suffix}`, monitorPortA),
      virtualPort(`demo-full-b-${suffix}`, monitorPortB),
    ],
    bridges: [{
      ID: `demo-full-bridge-${suffix}`,
      Port1: toPortPath(monitorPortA),
      Port2: toPortPath(monitorPortB),
      BaudRate: 115200,
    }],
    handles: [{
      id: handleId,
      config: serialConfig(toPortPath(terminalPort), 115200),
      isOpen: true,
      rxBytes: 42,
      txBytes: 24,
    }],
    buffers: {
      [handleId]: [
        bufferChunk('boot: MockTrue demo board\n', 1718845301000),
        bufferChunk('temperature=24.6 humidity=42\n', 1718845303000),
      ],
    },
    monitors: monitorState(monitorId, toPortPath(monitorPortA), toPortPath(monitorPortB)),
    workspace: workspace({
      selectedOperation: 'monitor',
      editorLayout: {
        type: 'split',
        id: 'split-full-demo',
        direction: 'horizontal',
        children: [
          { type: 'group', id: 'group-serial-demo', tabs: [handleId] },
          { type: 'group', id: 'group-monitor-demo', tabs: [monitorTabId(monitorId)] },
        ],
      },
      activeByGroup: {
        'group-serial-demo': handleId,
        'group-monitor-demo': monitorTabId(monitorId),
      },
      tabStates: {
        [handleId]: tabState({
          sendData: '01 03 00 00 00 02 c4 0b',
          sendMode: 'hex',
          sendHeight: 240,
          showTimestamp: true,
          autoScroll: true,
          history: [
            { id: 1, content: 'AT+STATUS?', mode: 'ascii' },
            { id: 2, content: '01 03 00 00 00 02 c4 0b', mode: 'hex' },
          ],
        }),
      },
    }),
  })
}

function snapshot(input: {
  settings?: SettingsSnapshot
  activePortId?: string | null
  handles?: WorkspaceSnapshot['serial']['handles']
  virtualPorts?: VirtualPort[]
  bridges?: Bridge[]
  buffers?: WorkspaceSnapshot['serial']['buffers']
  monitors?: MonitorWorkspaceState
  workspace?: SerialWorkspaceState
}): WorkspaceSnapshot {
  return {
    kind: workspaceKind,
    settings: input.settings ?? defaultSettings(),
    serial: {
      activePortId: input.activePortId ?? null,
      handles: input.handles ?? [],
      virtualPorts: input.virtualPorts ?? [],
      bridges: input.bridges ?? [],
      buffers: input.buffers ?? {},
      monitors: input.monitors ?? emptyMonitorState(),
      workspace: input.workspace ?? workspace({}),
    },
  }
}

function workspace(input: Partial<SerialWorkspaceState>): SerialWorkspaceState {
  return {
    selectedOperation: input.selectedOperation ?? null,
    editorLayout: input.editorLayout ?? emptyLayout(),
    activeByGroup: input.activeByGroup ?? { 'group-1': null },
    tabStates: input.tabStates ?? {},
  }
}

function emptyLayout(): EditorLayoutNode {
  return { type: 'group', id: 'group-1', tabs: [] }
}

function tabState(input: {
  sendData?: string
  sendMode?: 'ascii' | 'hex'
  sendHeight?: number
  showTimestamp?: boolean
  autoScroll?: boolean
  autoSend?: boolean
  sendIntervalMs?: number
  history?: SerialTabWorkspaceState['sendPanel']['sendHistory']
}): SerialTabWorkspaceState {
  return {
    showConfig: true,
    sendHeight: input.sendHeight ?? 220,
    dataDisplay: {
      viewMode: 'ascii',
      layoutMode: 'combined',
      showTimestamp: input.showTimestamp ?? true,
      autoScroll: input.autoScroll ?? true,
    },
    sendPanel: {
      sendData: input.sendData ?? '',
      sendMode: input.sendMode ?? 'ascii',
      autoSend: input.autoSend ?? false,
      sendIntervalMs: input.sendIntervalMs ?? 1000,
      sendHistory: input.history ?? [],
    },
  }
}

function defaultSettings(): SettingsSnapshot {
  return {
    global: { ...defaultGlobalSettings },
    serial: { ...defaultSerialSettings },
  }
}

function serialConfig(portName: string, baudRate: number): SerialConfig {
  return {
    PortName: portName,
    BaudRate: baudRate,
    DataBits: 8,
    StopBits: '1',
    Parity: 'none',
    FlowMode: 'none',
    ReadBufKB: 32,
  }
}

function virtualPort(id: string, portName: string): VirtualPort {
  return { ID: id, Port: portName }
}

function emptyMonitorState(): MonitorWorkspaceState {
  return {
    activeMonitorId: null,
    filters: {},
    sessions: [],
    frames: {},
  }
}

function monitorState(id: string, portA: string, portB: string): MonitorWorkspaceState {
  const frames = [
    monitorFrame(1, 'a_to_b', portA, '01 03 00 00 00 02 c4 0b', '读保持寄存器 0x0000 x2', {
      Slave: 1,
      Function: 3,
      FunctionHex: '03',
      PayloadHex: '00 00 00 02',
      CRCOK: true,
      LRCOK: false,
      Summary: 'Read Holding Registers',
      Error: '',
      Protocol: 'modbus-rtu',
    }),
    monitorFrame(2, 'b_to_a', portB, '01 03 04 00 18 00 2a fa 33', '返回 24 和 42', {
      Slave: 1,
      Function: 3,
      FunctionHex: '03',
      PayloadHex: '04 00 18 00 2a',
      CRCOK: true,
      LRCOK: false,
      Summary: '2 registers: 24, 42',
      Error: '',
      Protocol: 'modbus-rtu',
    }),
    monitorFrame(3, 'a_to_b', portA, 'ping', 'ASCII 心跳'),
  ]

  return {
    activeMonitorId: id,
    filters: {
      [id]: monitorFilter(),
    },
    sessions: [{
      ID: id,
      Name: 'Modbus RTU 监听演示',
      Provider: 'bridge',
      PortA: portA,
      PortB: portB,
      Config: serialConfig('', 115200),
      Encoding: 'utf-8',
      Status: 'stopped',
      RxBytes: 9,
      TxBytes: 13,
      FrameCount: frames.length,
      StartedAt: '2026-06-20T01:00:00Z',
      StoppedAt: '2026-06-20T01:02:30Z',
      Error: '',
      AutoSave: autoSaveOptions(),
    } satisfies SessionInfo],
    frames: {
      [id]: frames,
    },
  }
}

function monitorFilter(): MonitorFilterState {
  return {
    direction: 'all',
    search: '03',
    displayMode: 'hex',
    modbusFunction: 3,
  }
}

function monitorFrame(
  seq: number,
  direction: string,
  portName: string,
  hexOrText: string,
  text: string,
  modbus: Frame['Modbus'] = null
): Frame {
  const hex = hexOrText.includes(' ') ? hexOrText : Array.from(new TextEncoder().encode(hexOrText))
    .map(byte => byte.toString(16).padStart(2, '0'))
    .join(' ')

  return {
    Seq: seq,
    Timestamp: `2026-06-20T01:00:0${seq}Z`,
    Direction: direction,
    Port: portName,
    Length: hex.split(' ').filter(Boolean).length,
    Data: null,
    DisplayText: text,
    DisplayHex: hex,
    DisplayDec: hexToBase(hex, 10),
    DisplayOct: hexToBase(hex, 8),
    DisplayBin: hexToBase(hex, 2),
    Encoding: 'utf-8',
    Modbus: modbus,
  }
}

function autoSaveOptions(): AutoSaveOptions {
  return {
    Enabled: false,
    Path: '',
    Directory: '',
    BaseName: 'mocktrue-monitor-demo',
    Format: 'csv',
    SplitMode: 'none',
    SplitSizeKB: 1024,
    SplitIntervalSeconds: 60,
    Encoding: 'utf-8',
  }
}

function bufferChunk(text: string, timestamp: number) {
  return {
    timestamp,
    data: bytesToBase64(new TextEncoder().encode(text)),
  }
}

function monitorTabId(id: string): string {
  return `monitor:${id}`
}

function toPortPath(portName: string): string {
  return `/tmp/${portName}`
}

function nextDemoSuffix(): string {
  demoSequence += 1
  return `${Date.now().toString(36)}-${demoSequence}`
}

function hexToBase(hex: string, base: number): string {
  return hex.split(' ')
    .filter(Boolean)
    .map(part => parseInt(part, 16).toString(base))
    .join(' ')
}
