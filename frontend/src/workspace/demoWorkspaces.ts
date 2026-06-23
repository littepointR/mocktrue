import { defaultSerialSettings } from '../settings/stores/settingsStore'
import type { SerialWorkspaceState } from '../serial/stores/workspaceStore'
import type { MonitorWorkspaceState } from '../serial/stores/monitorStore'
import type { ModbusWorkspaceState } from '../serial/stores/modbusStore'
import type { FecbusWorkspaceState } from '../serial/stores/fecbusStore'
import type { EditorLayoutNode } from '../serial/views/editorLayout'
import { defaultSerialGraphState, nodeTabTitle, type SerialGraphEdge, type SerialGraphNode, type SerialGraphWorkspaceState } from '../serial/graph/serialGraph'
import { FrameType, FunctionCode, StatusCode } from '../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/fecbus/models.js'
import { workspaceKind, type WorkspaceSnapshot } from './workspaceSnapshot'

export interface DemoWorkspace {
  id: string
  title: string
  description: string
}

interface DemoWorkspaceDefinition extends DemoWorkspace {
  snapshotFactory: () => WorkspaceSnapshot
}

let demoSequence = 0

const demoDefinitions: DemoWorkspaceDefinition[] = [
  {
    id: 'serial-open-demo',
    title: '串口收发演示',
    description: '展示虚拟串口和打开串口操作的基础配置。',
    snapshotFactory: createSerialOpenDemo,
  },
  {
    id: 'virtual-port-demo',
    title: '虚拟串口演示',
    description: '展示多个自动创建的单端虚拟串口，适合验证虚拟串口资源管理。',
    snapshotFactory: createVirtualPortDemo,
  },
  {
    id: 'bridge-demo',
    title: '串口桥接演示',
    description: '展示拓扑图中两路虚拟串口字节流的桥接配置。',
    snapshotFactory: createBridgeDemo,
  },
  {
    id: 'monitor-demo',
    title: '串口监控演示',
    description: '展示串口监听操作和可监听的虚拟串口配置。',
    snapshotFactory: createMonitorDemo,
  },
  {
    id: 'script-transform-demo',
    title: '脚本转换演示',
    description: '展示脚本生成、脚本转换和接收显示组成的自动数据链路。',
    snapshotFactory: createScriptTransformDemo,
  },
  {
    id: 'script-analyzer-demo',
    title: '脚本分析演示',
    description: '展示脚本分析节点从真实数据链路中提取字段。',
    snapshotFactory: createScriptAnalyzerDemo,
  },
  {
    id: 'modbus-demo',
    title: 'Modbus 调试演示',
    description: '展示拓扑图中的 Modbus RTU/ASCII 主站和多 Unit ID 从站。',
    snapshotFactory: createModbusDemo,
  },
  {
    id: 'fecbus-demo',
    title: 'FECbus 调试演示',
    description: '展示拓扑图中的 FECbus 主控发送和从机应答配置。',
    snapshotFactory: createFecbusDemo,
  },
  {
    id: 'serial-graph-demo',
    title: '串口拓扑演示',
    description: '展示串口、分流器、接收器和协议节点的图形化连接。',
    snapshotFactory: createSerialGraphDemo,
  },
  {
    id: 'serial-observability-demo',
    title: '串口过滤与日志演示',
    description: '展示串口过滤器、收发日志模板、接收时间戳和运行后操作日志。',
    snapshotFactory: createSerialObservabilityDemo,
  },
  {
    id: 'full-workspace-demo',
    title: '完整工作区演示',
    description: '展示设置以及拓扑图中的虚拟串口、桥接、监控、Modbus 和 FECbus 配置。',
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
  const portName = `portweave-demo-open-${suffix}`
  const graph = serialOpenGraphState(suffix, `${portName}-graph`)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createVirtualPortDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portNames = ['sensor', 'gateway', 'logger'].map(name => `portweave-demo-${name}-${suffix}`)
  const graph = virtualPortGraphState(suffix, portNames)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createBridgeDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portA = `portweave-demo-bridge-a-${suffix}`
  const portB = `portweave-demo-bridge-b-${suffix}`
  const graph = bridgeGraphState(suffix, portA, portB)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createMonitorDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const sourcePort = `portweave-demo-monitor-${suffix}`
  const graph = monitorGraphState(suffix, `${sourcePort}-graph`)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createScriptTransformDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `portweave-demo-script-transform-${suffix}`
  const graph = scriptTransformGraphState(suffix, portName)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createScriptAnalyzerDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `portweave-demo-script-analyzer-${suffix}`
  const graph = scriptAnalyzerGraphState(suffix, portName)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createModbusDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const requestPort = `portweave-demo-modbus-${suffix}`
  const graph = modbusGraphState(suffix, requestPort)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createFecbusDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const requestPort = `portweave-demo-fecbus-${suffix}`
  const graph = fecbusGraphState(suffix, requestPort)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createSerialGraphDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `portweave-demo-graph-${suffix}`
  const graph = serialGraphState(suffix, portName)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createSerialObservabilityDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `portweave-demo-observability-${suffix}`
  const graph = serialObservabilityGraphState(suffix, portName)

  return snapshot({
    graph,
    workspace: graphWorkspace(graph),
  })
}

function createFullWorkspaceDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const terminalPort = `portweave-demo-terminal-${suffix}`
  const bridgePortA = `portweave-demo-full-a-${suffix}`
  const bridgePortB = `portweave-demo-full-b-${suffix}`
  const scriptPort = `portweave-demo-full-script-${suffix}`
  const modbusPort = `portweave-demo-full-modbus-${suffix}`
  const fecbusPort = `portweave-demo-full-fecbus-${suffix}`
  const graph = fullWorkspaceGraphState(suffix, terminalPort, bridgePortA, bridgePortB, scriptPort, modbusPort, fecbusPort)

  return snapshot({
    settings: {
      serial: {
        ...defaultSerialSettings,
        BaudRate: 115200,
        TerminalFontFamily: 'Menlo',
        TerminalFontSize: 15,
        TextEncoding: 'utf-8',
        EnterString: '\r\n',
      },
    },
    graph,
    workspace: graphWorkspace(graph),
  })
}

function snapshot(input: {
  settings?: WorkspaceSnapshot['settings']
  activePortId?: string | null
  handles?: WorkspaceSnapshot['serial']['handles']
  virtualPorts?: WorkspaceSnapshot['serial']['virtualPorts']
  bridges?: WorkspaceSnapshot['serial']['bridges']
  buffers?: WorkspaceSnapshot['serial']['buffers']
  monitors?: MonitorWorkspaceState
  modbus?: ModbusWorkspaceState
  fecbus?: FecbusWorkspaceState
  graph?: SerialGraphWorkspaceState
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
      modbus: input.modbus ?? emptyModbusState(),
      fecbus: input.fecbus ?? emptyFecbusState(),
      graph: input.graph ?? defaultSerialGraphState(),
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

function graphWorkspace(graph: SerialGraphWorkspaceState): SerialWorkspaceState {
  const tabId = graphTabId(graph.activeGraphId ?? graph.graphs[0]?.id ?? 'graph-1')
  return workspace({
    selectedOperation: 'graph',
    editorLayout: { type: 'group', id: 'group-1', tabs: [tabId] },
    activeByGroup: { 'group-1': tabId },
  })
}

function emptyLayout(): EditorLayoutNode {
  return { type: 'group', id: 'group-1', tabs: [] }
}

function defaultSettings(): WorkspaceSnapshot['settings'] {
  return {
    serial: { ...defaultSerialSettings },
  }
}

function emptyMonitorState(): MonitorWorkspaceState {
  return {
    activeMonitorId: null,
    filters: {},
    sessions: [],
    frames: {},
  }
}

function emptyModbusState(): ModbusWorkspaceState {
  const slaveForm = {
    unitId: 1,
    coils: '0=1\n1=0\n2=1',
    discreteInputs: '0=1\n1=1',
    inputRegisters: '0=24\n1=42',
    holdingRegisters: '0=24\n1=42',
  }
  return {
    activeSessionId: null,
    sessions: [],
    portForm: {
      sessionId: 'modbus-demo',
      name: '',
      port: '',
      mode: 'rtu',
      role: 'master',
      baudRate: 115200,
      dataBits: 8,
      stopBits: '1',
      parity: 'none',
      flowMode: 'none',
      timeoutMs: 800,
      retries: 0,
    },
    masterForm: {
      unitId: 1,
      functionCode: 3,
      addressMode: 'zero-based',
      address: 0,
      quantity: 2,
      value: 0,
      coilValues: '1 0 1 1',
      registerValues: '24 42',
      timeoutMs: 800,
      retries: 0,
    },
    slaveForm,
    activeSlaveUnitId: 1,
    slaveUnitForms: [{ ...slaveForm }],
    masterGrid: {
      unitId: 1,
      registerType: 'holding_registers',
      address: 0,
      length: 4,
      addressBase: 0,
      readConfigured: false,
      littleEndian: false,
      rawVisible: true,
      logVisible: true,
      pollRateMs: 1000,
      timeoutMs: 800,
      retries: 0,
    },
    masterMappings: [
      {
        id: 'demo-map-0',
        address: 0,
        dataType: 'int32',
        wordOrder: 'big',
        length: 0,
        scalingFactor: 0,
        comment: '累计计数',
        groupEnd: false,
      },
      {
        id: 'demo-map-2',
        address: 2,
        dataType: 'float',
        wordOrder: 'big',
        length: 0,
        scalingFactor: 1,
        comment: '温度',
        groupEnd: true,
      },
    ],
    slaveUnitGrids: [{
      unitId: 1,
      coils: [
        { id: 'demo-u1-coil-0', address: 0, value: true },
        { id: 'demo-u1-coil-1', address: 1, value: false },
        { id: 'demo-u1-coil-2', address: 2, value: true },
      ],
      discreteInputs: [
        { id: 'demo-u1-di-0', address: 0, value: true },
        { id: 'demo-u1-di-1', address: 1, value: true },
      ],
      inputRegisters: [
        { id: 'demo-u1-ir-0', address: 0, value: 24, dataType: 'uint16', comment: '输入温度' },
        { id: 'demo-u1-ir-1', address: 1, value: 42, dataType: 'uint16', comment: '输入湿度' },
      ],
      holdingRegisters: [
        { id: 'demo-u1-hr-0', address: 0, value: 24, dataType: 'uint16', comment: '设定值' },
        { id: 'demo-u1-hr-1', address: 1, value: 42, dataType: 'uint16', comment: '报警阈值' },
      ],
    }],
    registerReadForm: {
      unitId: 1,
      functionCode: 3,
      addressMode: 'zero-based',
      address: 0,
      quantity: 4,
      mappingText: '0:int32:big\n2:float:big',
      timeoutMs: 800,
      retries: 0,
      pollIntervalMs: 1000,
      polling: false,
    },
    unitScanForm: {
      unitIds: '1-10',
      functionCode: 3,
      addressMode: 'zero-based',
      address: 0,
      quantity: 1,
      timeoutMs: 120,
    },
    registerScanForm: {
      unitId: 1,
      functionCode: 3,
      addressMode: 'zero-based',
      startAddress: 0,
      endAddress: 16,
      chunkSize: 8,
      timeoutMs: 200,
    },
    registerReadResult: null,
    unitScanResult: null,
    registerScanResult: null,
    history: [],
  }
}

function emptyFecbusState(): FecbusWorkspaceState {
  return {
    activeSessionId: null,
    sessions: [],
    portForm: {
      sessionId: 'fecbus-demo',
      name: '',
      port: '',
      role: 'master',
      baudRate: 9600,
      dataBits: 8,
      stopBits: '1',
      parity: 'none',
      flowMode: 'none',
      timeoutMs: 1000,
      retries: 3,
    },
    sendForm: {
      frameType: FrameType.FrameTypeRequest,
      targetAddress: 2,
      priority: 2,
      sourceAddress: 1,
      messageNumber: 1,
      groupNumber: 0,
      functionCode: FunctionCode.FunctionQueryDeviceStatus,
      payloadHex: '',
      expectAnswer: true,
      timeoutMs: 1000,
      retries: 3,
      inputMode: 'hex',
      structuredFields: {},
    },
    slaveForm: {
      address: 2,
      statusCode: StatusCode.StatusReceivedOK,
      autoStatusAnswer: true,
      acceptBroadcast: true,
    },
    slaveUnits: [{
      address: 2,
      statusCode: StatusCode.StatusReceivedOK,
      autoStatusAnswer: true,
      acceptBroadcast: true,
    }],
    customFunctions: [],
    frameFilters: {},
    framePages: {},
    history: [],
  }
}

function graphTabId(id: string): string {
  return `graph:${id}`
}

function serialGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-receiver-${suffix}`
  const nodes: SerialGraphNode[] = [
    graphNode(suffix, 'sender', 'serial.sender', 32, 32, senderConfig(`PortWeave graph ${suffix}\r\n`)),
    graphNode(suffix, 'vport', 'serial.virtual', 264, 32, virtualConfig(portName)),
    graphNode(suffix, 'tap', 'serial.tap', 496, 32),
    graphNode(suffix, 'receiver', 'serial.receiver', 752, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'modbus', 'serial.modbus.master', 752, 168, modbusMasterConfig()),
    graphNode(suffix, 'monitor', 'serial.monitor', 752, 304, monitorConfig()),
  ]

  return graphState(suffix, '串口拓扑演示', nodes, [
    graphEdge(suffix, 'sender-vport', `graph-sender-${suffix}`, 'out', `graph-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'vport-tap', `graph-vport-${suffix}`, 'rx', `graph-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'tap-receiver', `graph-tap-${suffix}`, 'out', selectedNodeId, 'in'),
    graphEdge(suffix, 'tap-modbus', `graph-tap-${suffix}`, 'out', `graph-modbus-${suffix}`, 'rx'),
    graphEdge(suffix, 'tap-monitor', `graph-tap-${suffix}`, 'out', `graph-monitor-${suffix}`, 'in'),
  ], selectedNodeId)
}

function serialObservabilityGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-observability-receiver-expression-${suffix}`
  const nodes: SerialGraphNode[] = [
    graphNode(suffix, 'observability-sender', 'serial.sender', 32, 168, {
      ...senderConfig('TEMP=42 STATUS OK\r\n'),
      ...loggingConfig('info', '[{timestamp}] {nodeName} {direction} {length} bytes {text}'),
    }),
    graphNode(suffix, 'observability-vport', 'serial.virtual', 280, 168, virtualConfig(portName)),
    graphNode(suffix, 'observability-tap', 'serial.tap', 528, 168),
    graphNode(suffix, 'observability-filter-plain', 'serial.filter', 776, 32, filterConfig('plain', 'STATUS OK')),
    graphNode(suffix, 'observability-filter-regex', 'serial.filter', 776, 168, filterConfig('regex', /TEMP=\d+/.source)),
    graphNode(suffix, 'observability-filter-expression', 'serial.filter', 776, 304, filterConfig('expression', 'len >= 4 and text contains "OK"')),
    graphNode(suffix, 'observability-receiver-plain', 'serial.receiver', 1040, 32, timestampReceiverConfig('ascii', 'debug')),
    graphNode(suffix, 'observability-receiver-regex', 'serial.receiver', 1040, 168, timestampReceiverConfig('hexClassic', 'info')),
    graphNode(suffix, 'observability-receiver-expression', 'serial.receiver', 1040, 304, timestampReceiverConfig('ascii', 'info')),
  ]

  return graphState(suffix, '串口过滤与日志演示', nodes, [
    graphEdge(suffix, 'observability-sender-vport', `graph-observability-sender-${suffix}`, 'out', `graph-observability-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'observability-vport-tap', `graph-observability-vport-${suffix}`, 'rx', `graph-observability-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-tap-filter-plain', `graph-observability-tap-${suffix}`, 'out', `graph-observability-filter-plain-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-filter-plain-receiver', `graph-observability-filter-plain-${suffix}`, 'out', `graph-observability-receiver-plain-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-tap-filter-regex', `graph-observability-tap-${suffix}`, 'out', `graph-observability-filter-regex-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-filter-regex-receiver', `graph-observability-filter-regex-${suffix}`, 'out', `graph-observability-receiver-regex-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-tap-filter-expression', `graph-observability-tap-${suffix}`, 'out', `graph-observability-filter-expression-${suffix}`, 'in'),
    graphEdge(suffix, 'observability-filter-expression-receiver', `graph-observability-filter-expression-${suffix}`, 'out', selectedNodeId, 'in'),
  ], selectedNodeId)
}

function serialOpenGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-open-receiver-${suffix}`
  const nodes = [
    graphNode(suffix, 'open-sender', 'serial.sender', 32, 32, senderConfig(`open demo ${suffix}\r\n`)),
    graphNode(suffix, 'open-vport', 'serial.virtual', 280, 32, virtualConfig(portName)),
    graphNode(suffix, 'open-receiver', 'serial.receiver', 528, 32, receiverConfig()),
  ]

  return graphState(suffix, '串口收发演示', nodes, [
    graphEdge(suffix, 'open-sender-vport', `graph-open-sender-${suffix}`, 'out', `graph-open-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'open-vport-receiver', `graph-open-vport-${suffix}`, 'rx', selectedNodeId, 'in'),
  ], selectedNodeId)
}

function virtualPortGraphState(suffix: string, portNames: string[]): SerialGraphWorkspaceState {
  const nodes: SerialGraphNode[] = []
  const edges: SerialGraphEdge[] = []
  portNames.forEach((portName, index) => {
    const y = 32 + index * 136
    const key = `vport-${index + 1}`
    const senderId = `graph-${key}-sender-${suffix}`
    const vportId = `graph-${key}-port-${suffix}`
    const receiverId = `graph-${key}-receiver-${suffix}`
    nodes.push(
      graphNode(suffix, `${key}-sender`, 'serial.sender', 32, y, senderConfig(`virtual ${index + 1} ${suffix}\r\n`)),
      graphNode(suffix, `${key}-port`, 'serial.virtual', 280, y, virtualConfig(portName)),
      graphNode(suffix, `${key}-receiver`, 'serial.receiver', 528, y, receiverConfig())
    )
    edges.push(
      graphEdge(suffix, `${key}-sender-port`, senderId, 'out', vportId, 'tx'),
      graphEdge(suffix, `${key}-port-receiver`, vportId, 'rx', receiverId, 'in')
    )
  })

  return graphState(suffix, '虚拟串口演示', nodes, edges, `graph-vport-1-receiver-${suffix}`)
}

function bridgeGraphState(suffix: string, portA: string, portB: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-bridge-${suffix}`
  const nodes = [
    graphNode(suffix, 'bridge-sender-a', 'serial.sender', 32, 32, senderConfig(`bridge A ${suffix}\r\n`)),
    graphNode(suffix, 'bridge-vport-a', 'serial.virtual', 280, 32, virtualConfig(portA)),
    graphNode(suffix, 'bridge-sender-b', 'serial.sender', 32, 200, senderConfig(`bridge B ${suffix}\r\n`)),
    graphNode(suffix, 'bridge-vport-b', 'serial.virtual', 280, 200, virtualConfig(portB)),
    graphNode(suffix, 'bridge', 'serial.bridge', 528, 116),
    graphNode(suffix, 'bridge-receiver-a', 'serial.receiver', 776, 32, receiverConfig()),
    graphNode(suffix, 'bridge-receiver-b', 'serial.receiver', 776, 200, receiverConfig()),
  ]

  return graphState(suffix, '串口桥接演示', nodes, [
    graphEdge(suffix, 'bridge-sender-a-vport-a', `graph-bridge-sender-a-${suffix}`, 'out', `graph-bridge-vport-a-${suffix}`, 'tx'),
    graphEdge(suffix, 'bridge-vport-a-in', `graph-bridge-vport-a-${suffix}`, 'rx', selectedNodeId, 'a-in'),
    graphEdge(suffix, 'bridge-b-out-receiver', selectedNodeId, 'b-out', `graph-bridge-receiver-b-${suffix}`, 'in'),
    graphEdge(suffix, 'bridge-sender-b-vport-b', `graph-bridge-sender-b-${suffix}`, 'out', `graph-bridge-vport-b-${suffix}`, 'tx'),
    graphEdge(suffix, 'bridge-vport-b-in', `graph-bridge-vport-b-${suffix}`, 'rx', selectedNodeId, 'b-in'),
    graphEdge(suffix, 'bridge-a-out-receiver', selectedNodeId, 'a-out', `graph-bridge-receiver-a-${suffix}`, 'in'),
  ], selectedNodeId)
}

function monitorGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-monitor-${suffix}`
  const nodes = [
    graphNode(suffix, 'monitor-sender', 'serial.sender', 32, 32, senderConfig(`monitor demo ${suffix}\r\n`)),
    graphNode(suffix, 'monitor-vport', 'serial.virtual', 280, 32, virtualConfig(portName)),
    graphNode(suffix, 'monitor-tap', 'serial.tap', 528, 32),
    graphNode(suffix, 'monitor-receiver', 'serial.receiver', 776, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'monitor', 'serial.monitor', 776, 168, monitorConfig()),
  ]

  return graphState(suffix, '串口监控演示', nodes, [
    graphEdge(suffix, 'monitor-sender-vport', `graph-monitor-sender-${suffix}`, 'out', `graph-monitor-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'monitor-vport-tap', `graph-monitor-vport-${suffix}`, 'rx', `graph-monitor-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'monitor-tap-receiver', `graph-monitor-tap-${suffix}`, 'out', `graph-monitor-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'monitor-tap-monitor', `graph-monitor-tap-${suffix}`, 'out', selectedNodeId, 'in'),
  ], selectedNodeId)
}

function scriptTransformGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-script-transform-${suffix}`
  const tapId = `graph-script-transform-tap-${suffix}`
  const nodes = [
    graphNode(suffix, 'script-generator', 'serial.script.generator', 32, 32, scriptGeneratorConfig()),
    graphNode(suffix, 'script-transform-vport', 'serial.virtual', 280, 32, virtualConfig(portName)),
    graphNode(suffix, 'script-transform', 'serial.script.transform', 528, 32, scriptTransformConfig()),
    graphNode(suffix, 'script-transform-tap', 'serial.tap', 776, 32),
    graphNode(suffix, 'script-transform-receiver', 'serial.receiver', 1024, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'script-transform-monitor', 'serial.monitor', 1024, 168, monitorConfig()),
  ]

  return graphState(suffix, '脚本转换演示', nodes, [
    graphEdge(suffix, 'script-generator-vport', `graph-script-generator-${suffix}`, 'out', `graph-script-transform-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'script-vport-transform', `graph-script-transform-vport-${suffix}`, 'rx', selectedNodeId, 'in'),
    graphEdge(suffix, 'script-transform-tap', selectedNodeId, 'out', tapId, 'in'),
    graphEdge(suffix, 'script-transform-tap-receiver', tapId, 'out', `graph-script-transform-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'script-transform-tap-monitor', tapId, 'out', `graph-script-transform-monitor-${suffix}`, 'in'),
  ], selectedNodeId)
}

function scriptAnalyzerGraphState(suffix: string, portName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-script-analyzer-${suffix}`
  const tapId = `graph-script-analyzer-tap-${suffix}`
  const nodes = [
    graphNode(suffix, 'script-analyzer-generator', 'serial.script.generator', 32, 32, scriptGeneratorConfig()),
    graphNode(suffix, 'script-analyzer-vport', 'serial.virtual', 280, 32, virtualConfig(portName)),
    graphNode(suffix, 'script-analyzer-tap', 'serial.tap', 528, 32),
    graphNode(suffix, 'script-analyzer-receiver', 'serial.receiver', 776, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'script-analyzer', 'serial.script.analyzer', 776, 168, scriptAnalyzerConfig()),
  ]

  return graphState(suffix, '脚本分析演示', nodes, [
    graphEdge(suffix, 'script-analyzer-generator-vport', `graph-script-analyzer-generator-${suffix}`, 'out', `graph-script-analyzer-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'script-analyzer-vport-tap', `graph-script-analyzer-vport-${suffix}`, 'rx', tapId, 'in'),
    graphEdge(suffix, 'script-analyzer-tap-receiver', tapId, 'out', `graph-script-analyzer-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'script-analyzer-tap-analyzer', tapId, 'out', selectedNodeId, 'in'),
  ], selectedNodeId)
}

function modbusGraphState(suffix: string, requestPortName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-modbus-master-${suffix}`
  const requestPortId = `graph-modbus-vport-${suffix}`
  const slaveId = `graph-modbus-slave-${suffix}`
  const tapId = `graph-modbus-response-tap-${suffix}`
  const nodes = [
    graphNode(suffix, 'modbus-master', 'serial.modbus.master', 32, 32, modbusMasterConfig({ autoSend: true })),
    graphNode(suffix, 'modbus-vport', 'serial.virtual', 280, 32, virtualConfig(requestPortName)),
    graphNode(suffix, 'modbus-slave', 'serial.modbus.slave', 528, 32, modbusSlaveConfig()),
    graphNode(suffix, 'modbus-response-tap', 'serial.tap', 776, 32),
    graphNode(suffix, 'modbus-receiver', 'serial.receiver', 1024, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'modbus-monitor', 'serial.monitor', 1024, 168, monitorConfig()),
  ]

  return graphState(suffix, 'Modbus 调试演示', nodes, [
    graphEdge(suffix, 'modbus-master-vport', selectedNodeId, 'tx', requestPortId, 'tx'),
    graphEdge(suffix, 'modbus-vport-slave', requestPortId, 'rx', slaveId, 'rx'),
    graphEdge(suffix, 'modbus-slave-tap', slaveId, 'tx', tapId, 'in'),
    graphEdge(suffix, 'modbus-tap-master', tapId, 'out', selectedNodeId, 'rx'),
    graphEdge(suffix, 'modbus-tap-receiver', tapId, 'out', `graph-modbus-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'modbus-tap-monitor', tapId, 'out', `graph-modbus-monitor-${suffix}`, 'in'),
  ], selectedNodeId)
}

function fecbusGraphState(suffix: string, requestPortName: string): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-fecbus-master-${suffix}`
  const requestPortId = `graph-fecbus-vport-${suffix}`
  const slaveId = `graph-fecbus-slave-${suffix}`
  const tapId = `graph-fecbus-response-tap-${suffix}`
  const nodes = [
    graphNode(suffix, 'fecbus-master', 'serial.fecbus.master', 32, 32, fecbusMasterConfig({ autoSend: true })),
    graphNode(suffix, 'fecbus-vport', 'serial.virtual', 280, 32, virtualConfig(requestPortName)),
    graphNode(suffix, 'fecbus-slave', 'serial.fecbus.slave', 528, 32, fecbusSlaveConfig()),
    graphNode(suffix, 'fecbus-response-tap', 'serial.tap', 776, 32),
    graphNode(suffix, 'fecbus-receiver', 'serial.receiver', 1024, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'fecbus-monitor', 'serial.monitor', 1024, 168, monitorConfig()),
  ]

  return graphState(suffix, 'FECbus 调试演示', nodes, [
    graphEdge(suffix, 'fecbus-master-vport', selectedNodeId, 'tx', requestPortId, 'tx'),
    graphEdge(suffix, 'fecbus-vport-slave', requestPortId, 'rx', slaveId, 'rx'),
    graphEdge(suffix, 'fecbus-slave-tap', slaveId, 'tx', tapId, 'in'),
    graphEdge(suffix, 'fecbus-tap-receiver', tapId, 'out', `graph-fecbus-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'fecbus-tap-monitor', tapId, 'out', `graph-fecbus-monitor-${suffix}`, 'in'),
  ], selectedNodeId)
}

function fullWorkspaceGraphState(
  suffix: string,
  terminalPort: string,
  bridgePortA: string,
  bridgePortB: string,
  scriptPort: string,
  modbusPort: string,
  fecbusPort: string
): SerialGraphWorkspaceState {
  const selectedNodeId = `graph-full-receiver-${suffix}`
  const scriptGeneratorId = `graph-full-script-generator-${suffix}`
  const scriptPortId = `graph-full-script-vport-${suffix}`
  const scriptTransformId = `graph-full-script-transform-${suffix}`
  const scriptTapId = `graph-full-script-tap-${suffix}`
  const scriptAnalyzerId = `graph-full-script-analyzer-${suffix}`
  const modbusMasterId = `graph-full-modbus-master-${suffix}`
  const modbusPortId = `graph-full-modbus-vport-${suffix}`
  const modbusSlaveId = `graph-full-modbus-slave-${suffix}`
  const fecbusMasterId = `graph-full-fecbus-master-${suffix}`
  const fecbusPortId = `graph-full-fecbus-vport-${suffix}`
  const fecbusSlaveId = `graph-full-fecbus-slave-${suffix}`
  const nodes = [
    graphNode(suffix, 'full-sender', 'serial.sender', 32, 32, senderConfig(`full workspace ${suffix}\r\n`)),
    graphNode(suffix, 'full-vport', 'serial.virtual', 280, 32, virtualConfig(terminalPort)),
    graphNode(suffix, 'full-tap', 'serial.tap', 528, 32),
    graphNode(suffix, 'full-filter', 'serial.filter', 652, 32, filterConfig('plain', 'full workspace')),
    graphNode(suffix, 'full-receiver', 'serial.receiver', 776, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'full-monitor', 'serial.monitor', 776, 168, monitorConfig()),

    graphNode(suffix, 'full-bridge-sender-a', 'serial.sender', 32, 344, senderConfig(`full bridge A ${suffix}\r\n`)),
    graphNode(suffix, 'full-bridge-vport-a', 'serial.virtual', 280, 344, virtualConfig(bridgePortA)),
    graphNode(suffix, 'full-bridge-sender-b', 'serial.sender', 32, 512, senderConfig(`full bridge B ${suffix}\r\n`)),
    graphNode(suffix, 'full-bridge-vport-b', 'serial.virtual', 280, 512, virtualConfig(bridgePortB)),
    graphNode(suffix, 'full-bridge', 'serial.bridge', 528, 428),
    graphNode(suffix, 'full-bridge-receiver-a', 'serial.receiver', 776, 344, receiverConfig()),
    graphNode(suffix, 'full-bridge-receiver-b', 'serial.receiver', 776, 512, receiverConfig()),

    graphNode(suffix, 'full-script-generator', 'serial.script.generator', 1040, 656, scriptGeneratorConfig()),
    graphNode(suffix, 'full-script-vport', 'serial.virtual', 1288, 656, virtualConfig(scriptPort)),
    graphNode(suffix, 'full-script-transform', 'serial.script.transform', 1536, 656, scriptTransformConfig()),
    graphNode(suffix, 'full-script-tap', 'serial.tap', 1784, 656),
    graphNode(suffix, 'full-script-receiver', 'serial.receiver', 2032, 656, receiverConfig('hexClassic')),
    graphNode(suffix, 'full-script-analyzer', 'serial.script.analyzer', 2032, 792, scriptAnalyzerConfig()),

    graphNode(suffix, 'full-modbus-master', 'serial.modbus.master', 1040, 32, modbusMasterConfig({ autoSend: true })),
    graphNode(suffix, 'full-modbus-vport', 'serial.virtual', 1288, 32, virtualConfig(modbusPort)),
    graphNode(suffix, 'full-modbus-slave', 'serial.modbus.slave', 1536, 32, modbusSlaveConfig()),
    graphNode(suffix, 'full-modbus-tap', 'serial.tap', 1784, 32),
    graphNode(suffix, 'full-modbus-receiver', 'serial.receiver', 2032, 32, receiverConfig('hexClassic')),
    graphNode(suffix, 'full-modbus-monitor', 'serial.monitor', 2032, 168, monitorConfig()),

    graphNode(suffix, 'full-fecbus-master', 'serial.fecbus.master', 1040, 344, fecbusMasterConfig({ autoSend: true })),
    graphNode(suffix, 'full-fecbus-vport', 'serial.virtual', 1288, 344, virtualConfig(fecbusPort)),
    graphNode(suffix, 'full-fecbus-slave', 'serial.fecbus.slave', 1536, 344, fecbusSlaveConfig()),
    graphNode(suffix, 'full-fecbus-tap', 'serial.tap', 1784, 344),
    graphNode(suffix, 'full-fecbus-receiver', 'serial.receiver', 2032, 344, receiverConfig('hexClassic')),
  ]

  return graphState(suffix, '完整工作区演示', nodes, [
    graphEdge(suffix, 'full-sender-vport', `graph-full-sender-${suffix}`, 'out', `graph-full-vport-${suffix}`, 'tx'),
    graphEdge(suffix, 'full-vport-tap', `graph-full-vport-${suffix}`, 'rx', `graph-full-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'full-tap-filter', `graph-full-tap-${suffix}`, 'out', `graph-full-filter-${suffix}`, 'in'),
    graphEdge(suffix, 'full-filter-receiver', `graph-full-filter-${suffix}`, 'out', selectedNodeId, 'in'),
    graphEdge(suffix, 'full-tap-monitor', `graph-full-tap-${suffix}`, 'out', `graph-full-monitor-${suffix}`, 'in'),
    graphEdge(suffix, 'full-bridge-sender-a-vport-a', `graph-full-bridge-sender-a-${suffix}`, 'out', `graph-full-bridge-vport-a-${suffix}`, 'tx'),
    graphEdge(suffix, 'full-bridge-vport-a-in', `graph-full-bridge-vport-a-${suffix}`, 'rx', `graph-full-bridge-${suffix}`, 'a-in'),
    graphEdge(suffix, 'full-bridge-b-out-receiver', `graph-full-bridge-${suffix}`, 'b-out', `graph-full-bridge-receiver-b-${suffix}`, 'in'),
    graphEdge(suffix, 'full-bridge-sender-b-vport-b', `graph-full-bridge-sender-b-${suffix}`, 'out', `graph-full-bridge-vport-b-${suffix}`, 'tx'),
    graphEdge(suffix, 'full-bridge-vport-b-in', `graph-full-bridge-vport-b-${suffix}`, 'rx', `graph-full-bridge-${suffix}`, 'b-in'),
    graphEdge(suffix, 'full-bridge-a-out-receiver', `graph-full-bridge-${suffix}`, 'a-out', `graph-full-bridge-receiver-a-${suffix}`, 'in'),
    graphEdge(suffix, 'full-script-generator-vport', scriptGeneratorId, 'out', scriptPortId, 'tx'),
    graphEdge(suffix, 'full-script-vport-transform', scriptPortId, 'rx', scriptTransformId, 'in'),
    graphEdge(suffix, 'full-script-transform-tap', scriptTransformId, 'out', scriptTapId, 'in'),
    graphEdge(suffix, 'full-script-tap-receiver', scriptTapId, 'out', `graph-full-script-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'full-script-tap-analyzer', scriptTapId, 'out', scriptAnalyzerId, 'in'),
    graphEdge(suffix, 'full-modbus-master-vport', modbusMasterId, 'tx', modbusPortId, 'tx'),
    graphEdge(suffix, 'full-modbus-vport-slave', modbusPortId, 'rx', modbusSlaveId, 'rx'),
    graphEdge(suffix, 'full-modbus-slave-tap', modbusSlaveId, 'tx', `graph-full-modbus-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'full-modbus-tap-master', `graph-full-modbus-tap-${suffix}`, 'out', modbusMasterId, 'rx'),
    graphEdge(suffix, 'full-modbus-tap-receiver', `graph-full-modbus-tap-${suffix}`, 'out', `graph-full-modbus-receiver-${suffix}`, 'in'),
    graphEdge(suffix, 'full-modbus-tap-monitor', `graph-full-modbus-tap-${suffix}`, 'out', `graph-full-modbus-monitor-${suffix}`, 'in'),
    graphEdge(suffix, 'full-fecbus-master-vport', fecbusMasterId, 'tx', fecbusPortId, 'tx'),
    graphEdge(suffix, 'full-fecbus-vport-slave', fecbusPortId, 'rx', fecbusSlaveId, 'rx'),
    graphEdge(suffix, 'full-fecbus-slave-tap', fecbusSlaveId, 'tx', `graph-full-fecbus-tap-${suffix}`, 'in'),
    graphEdge(suffix, 'full-fecbus-tap-receiver', `graph-full-fecbus-tap-${suffix}`, 'out', `graph-full-fecbus-receiver-${suffix}`, 'in'),
  ], selectedNodeId)
}

function graphState(
  suffix: string,
  name: string,
  nodes: SerialGraphNode[],
  edges: SerialGraphEdge[],
  selectedNodeId: string
): SerialGraphWorkspaceState {
  const graphId = `graph-demo-${suffix}`
  return {
    graphs: [{
      id: graphId,
      name,
      nodes,
      edges,
      selectedNodeId,
      selectedEdgeId: null,
      nodeTabs: nodes.map(node => ({ nodeId: node.id, title: nodeTabTitle(node) })),
      activeNodeTabId: selectedNodeId,
    }],
    activeGraphId: graphId,
  }
}

function graphNode(
  suffix: string,
  key: string,
  type: string,
  x: number,
  y: number,
  config: Record<string, unknown> = {}
): SerialGraphNode {
  return {
    id: `graph-${key}-${suffix}`,
    type,
    position: { x, y },
    config,
  }
}

function graphEdge(
  suffix: string,
  key: string,
  source: string,
  sourceHandle: string,
  target: string,
  targetHandle: string
): SerialGraphEdge {
  return {
    id: `graph-edge-${key}-${suffix}`,
    source,
    sourceHandle,
    target,
    targetHandle,
  }
}

function senderConfig(payload: string, intervalMs = 1000): Record<string, unknown> {
  return {
    mode: 'ascii',
    encoding: 'utf-8',
    payload,
    autoSend: true,
    intervalMs,
  }
}

function virtualConfig(portName: string): Record<string, unknown> {
  return {
    portName,
    baudRate: 115200,
    dataBits: 8,
    stopBits: '1',
    parity: 'none',
    flowMode: 'none',
    readBufKB: 32,
  }
}

function receiverConfig(viewMode = 'ascii'): Record<string, unknown> {
  return { viewMode, autoScroll: true }
}

function timestampReceiverConfig(viewMode = 'ascii', logLevel = 'info'): Record<string, unknown> {
  return {
    ...receiverConfig(viewMode),
    showTimestamp: true,
    ...loggingConfig(logLevel, '{nodeName} {direction} {length} bytes {hex}'),
  }
}

function filterConfig(
  mode: 'plain' | 'regex' | 'expression',
  expression: string,
  options: { caseSensitive?: boolean; wholeWord?: boolean } = {}
): Record<string, unknown> {
  return {
    mode,
    expression,
    caseSensitive: options.caseSensitive ?? false,
    wholeWord: options.wholeWord ?? false,
  }
}

function loggingConfig(logLevel: string, logFormat: string): Record<string, unknown> {
  return {
    enableLogging: true,
    logLevel,
    logFormat,
  }
}

function monitorConfig(): Record<string, unknown> {
  return { displayMode: 'hex' }
}

function scriptGeneratorConfig(): Record<string, unknown> {
  return {
    script: 'state.set("n", (state.get("n") || 0) + 1); output.text("script " + state.get("n") + "\\r\\n", "utf-8")',
    timeoutMs: 50,
    maxOutputBytes: 65536,
    maxStateBytes: 262144,
    onError: 'mark-error-and-drop',
    encoding: 'utf-8',
    autoRun: true,
    intervalMs: 1000,
    displayMode: 'hex',
  }
}

function scriptTransformConfig(): Record<string, unknown> {
  return {
    script: 'output.text(input.text("utf-8").toUpperCase(), "utf-8")',
    timeoutMs: 50,
    maxOutputBytes: 65536,
    maxStateBytes: 262144,
    onError: 'mark-error-and-drop',
    encoding: 'utf-8',
    autoRun: false,
    intervalMs: 1000,
    displayMode: 'hex',
  }
}

function scriptAnalyzerConfig(): Record<string, unknown> {
  return {
    script: 'field("length", input.bytes().length); field("hex", input.hex())',
    timeoutMs: 50,
    maxOutputBytes: 65536,
    maxStateBytes: 262144,
    onError: 'mark-error-and-drop',
    encoding: 'utf-8',
    autoRun: false,
    intervalMs: 1000,
    displayMode: 'hex',
  }
}

function modbusMasterConfig(options: { autoSend?: boolean } = {}): Record<string, unknown> {
  const config: Record<string, unknown> = {
    mode: 'rtu',
    unitIds: '1,2',
    addressMode: 'zero-based',
    functionCode: 3,
    address: 0,
    quantity: 2,
    value: 0,
  }
  if (options.autoSend) {
    config.autoSend = true
    config.intervalMs = 1000
  }
  return config
}

function modbusSlaveConfig(): Record<string, unknown> {
  return {
    mode: 'rtu',
    unitIds: '1,2',
  }
}

function fecbusMasterConfig(options: { autoSend?: boolean } = {}): Record<string, unknown> {
  const config: Record<string, unknown> = {
    frameType: FrameType.FrameTypeRequest,
    sourceAddress: 1,
    targetAddress: 2,
    priority: 2,
    messageNumber: 7,
    groupNumber: 0,
    functionCode: FunctionCode.FunctionQueryProtocolVersion,
    dataHex: '',
  }
  if (options.autoSend) {
    config.autoSend = true
    config.intervalMs = 1000
  }
  return config
}

function fecbusSlaveConfig(): Record<string, unknown> {
  return {
    address: 2,
    defaultStatus: StatusCode.StatusReceivedOK,
    autoStatusAnswer: true,
  }
}

function nextDemoSuffix(): string {
  demoSequence += 1
  return `${Date.now().toString(36)}-${demoSequence}`
}
