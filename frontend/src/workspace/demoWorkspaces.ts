import { defaultSerialSettings } from '../settings/stores/settingsStore'
import type { Bridge, VirtualPort } from '../serial/stores/virtualStore'
import type { SerialWorkspaceState } from '../serial/stores/workspaceStore'
import type { MonitorWorkspaceState } from '../serial/stores/monitorStore'
import type { ModbusWorkspaceState } from '../serial/stores/modbusStore'
import type { FecbusWorkspaceState } from '../serial/stores/fecbusStore'
import type { EditorLayoutNode } from '../serial/views/editorLayout'
import { FrameMode, SessionRole } from '../../bindings/github.com/suyue/mocktrue/internal/modules/serial/modbus/models.js'
import { FrameType, FunctionCode, SessionRole as FecbusSessionRole, StatusCode } from '../../bindings/github.com/suyue/mocktrue/internal/modules/serial/fecbus/models.js'
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
    description: '展示两个自动虚拟串口之间的桥接配置。',
    snapshotFactory: createBridgeDemo,
  },
  {
    id: 'monitor-demo',
    title: '串口监控演示',
    description: '展示串口监听操作和可监听的虚拟串口配置。',
    snapshotFactory: createMonitorDemo,
  },
  {
    id: 'modbus-demo',
    title: 'Modbus 调试演示',
    description: '展示 Modbus RTU/ASCII 主站配置和从站数据表。',
    snapshotFactory: createModbusDemo,
  },
  {
    id: 'fecbus-demo',
    title: 'FECbus 调试演示',
    description: '展示 FECbus 主控节点发送、设备状态应答和帧历史配置。',
    snapshotFactory: createFecbusDemo,
  },
  {
    id: 'full-workspace-demo',
    title: '完整工作区演示',
    description: '展示设置、虚拟串口、桥接和 Modbus 配置。',
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

  return snapshot({
    virtualPorts: [virtualPort(`demo-open-port-${suffix}`, portName)],
    workspace: workspace({
      selectedOperation: 'open',
      editorLayout: { type: 'group', id: 'group-1', tabs: [] },
      activeByGroup: { 'group-1': null },
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
  const sourcePort = `mocktrue-demo-monitor-${suffix}`

  return snapshot({
    virtualPorts: [virtualPort(`demo-monitor-port-${suffix}`, sourcePort)],
    workspace: workspace({
      selectedOperation: 'monitor',
      editorLayout: { type: 'group', id: 'group-1', tabs: [] },
      activeByGroup: { 'group-1': null },
    }),
  })
}

function createModbusDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `mocktrue-demo-modbus-${suffix}`
  const sessionId = `demo-modbus-${suffix}`

  return snapshot({
    virtualPorts: [virtualPort(`demo-modbus-port-${suffix}`, portName)],
    modbus: modbusState(sessionId, toPortPath(portName)),
    workspace: workspace({
      selectedOperation: 'modbus',
      editorLayout: { type: 'group', id: 'group-1', tabs: [modbusTabId(sessionId)] },
      activeByGroup: { 'group-1': modbusTabId(sessionId) },
    }),
  })
}

function createFecbusDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const portName = `mocktrue-demo-fecbus-${suffix}`
  const sessionId = `demo-fecbus-${suffix}`

  return snapshot({
    virtualPorts: [virtualPort(`demo-fecbus-port-${suffix}`, portName)],
    fecbus: fecbusState(sessionId, toPortPath(portName)),
    workspace: workspace({
      selectedOperation: 'fecbus',
      editorLayout: { type: 'group', id: 'group-1', tabs: [fecbusTabId(sessionId)] },
      activeByGroup: { 'group-1': fecbusTabId(sessionId) },
    }),
  })
}

function createFullWorkspaceDemo(): WorkspaceSnapshot {
  const suffix = nextDemoSuffix()
  const terminalPort = `mocktrue-demo-terminal-${suffix}`
  const bridgePortA = `mocktrue-demo-full-a-${suffix}`
  const bridgePortB = `mocktrue-demo-full-b-${suffix}`
  const fecbusPort = `mocktrue-demo-full-fecbus-${suffix}`

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
    virtualPorts: [
      virtualPort(`demo-full-terminal-${suffix}`, terminalPort),
      virtualPort(`demo-full-a-${suffix}`, bridgePortA),
      virtualPort(`demo-full-b-${suffix}`, bridgePortB),
      virtualPort(`demo-full-fecbus-${suffix}`, fecbusPort),
    ],
    bridges: [{
      ID: `demo-full-bridge-${suffix}`,
      Port1: toPortPath(bridgePortA),
      Port2: toPortPath(bridgePortB),
      BaudRate: 115200,
    }],
    modbus: modbusState(`demo-full-modbus-${suffix}`, toPortPath(terminalPort)),
    fecbus: fecbusState(`demo-full-fecbus-${suffix}`, toPortPath(fecbusPort)),
    workspace: workspace({
      selectedOperation: 'fecbus',
      editorLayout: { type: 'group', id: 'group-1', tabs: [modbusTabId(`demo-full-modbus-${suffix}`), fecbusTabId(`demo-full-fecbus-${suffix}`)] },
      activeByGroup: { 'group-1': fecbusTabId(`demo-full-fecbus-${suffix}`) },
    }),
  })
}

function snapshot(input: {
  settings?: WorkspaceSnapshot['settings']
  activePortId?: string | null
  handles?: WorkspaceSnapshot['serial']['handles']
  virtualPorts?: VirtualPort[]
  bridges?: Bridge[]
  buffers?: WorkspaceSnapshot['serial']['buffers']
  monitors?: MonitorWorkspaceState
  modbus?: ModbusWorkspaceState
  fecbus?: FecbusWorkspaceState
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

function defaultSettings(): WorkspaceSnapshot['settings'] {
  return {
    serial: { ...defaultSerialSettings },
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

function modbusState(id: string, portName: string): ModbusWorkspaceState {
  const state = emptyModbusState()
  const config = {
    PortName: portName,
    BaudRate: 115200,
    DataBits: 8,
    StopBits: '1',
    Parity: 'none',
    FlowMode: 'none',
    ReadBufKB: 32,
  }
  state.activeSessionId = id
  state.sessions = [{
    ID: id,
    Name: 'Modbus RTU 演示',
    Mode: FrameMode.FrameModeRTU,
    Role: SessionRole.SessionRoleMaster,
    Config: config,
    Status: 'stopped',
    RxBytes: 0,
    TxBytes: 0,
    SlaveRunning: false,
    UnitID: 1,
    UnitIDs: [1, 2],
    StartedAt: '',
    StoppedAt: '',
    LastError: '',
  }]
  state.portForm = {
    ...state.portForm,
    sessionId: `next-${id}`,
    port: portName,
    name: 'Modbus RTU 演示',
    mode: 'rtu',
  }
  state.slaveUnitForms = [
    {
      unitId: 1,
      coils: '0=1\n1=0\n2=1',
      discreteInputs: '0=1\n1=1',
      inputRegisters: '0=24\n1=42',
      holdingRegisters: '0=24\n1=42',
    },
    {
      unitId: 2,
      coils: '0=0\n1=1\n2=1',
      discreteInputs: '0=0\n1=1',
      inputRegisters: '0=120\n1=230',
      holdingRegisters: '0=100\n1=200',
    },
  ]
  state.slaveUnitGrids = [
    {
      unitId: 1,
      coils: [
        { id: `${id}-u1-coil-0`, address: 0, value: true },
        { id: `${id}-u1-coil-1`, address: 1, value: false },
        { id: `${id}-u1-coil-2`, address: 2, value: true },
      ],
      discreteInputs: [
        { id: `${id}-u1-di-0`, address: 0, value: true },
        { id: `${id}-u1-di-1`, address: 1, value: true },
      ],
      inputRegisters: [
        { id: `${id}-u1-ir-0`, address: 0, value: 24, dataType: 'uint16', comment: '输入温度' },
        { id: `${id}-u1-ir-1`, address: 1, value: 42, dataType: 'uint16', comment: '输入湿度' },
      ],
      holdingRegisters: [
        { id: `${id}-u1-hr-0`, address: 0, value: 24, dataType: 'uint16', comment: '保持寄存器 0' },
        { id: `${id}-u1-hr-1`, address: 1, value: 42, dataType: 'uint16', comment: '保持寄存器 1' },
      ],
    },
    {
      unitId: 2,
      coils: [
        { id: `${id}-u2-coil-0`, address: 0, value: false },
        { id: `${id}-u2-coil-1`, address: 1, value: true },
        { id: `${id}-u2-coil-2`, address: 2, value: true },
      ],
      discreteInputs: [
        { id: `${id}-u2-di-0`, address: 0, value: false },
        { id: `${id}-u2-di-1`, address: 1, value: true },
      ],
      inputRegisters: [
        { id: `${id}-u2-ir-0`, address: 0, value: 120, dataType: 'uint16', comment: '输入电压' },
        { id: `${id}-u2-ir-1`, address: 1, value: 230, dataType: 'uint16', comment: '输入电流' },
      ],
      holdingRegisters: [
        { id: `${id}-u2-hr-0`, address: 0, value: 100, dataType: 'uint16', comment: '目标值' },
        { id: `${id}-u2-hr-1`, address: 1, value: 200, dataType: 'uint16', comment: '限制值' },
      ],
    },
  ]
  state.activeSlaveUnitId = 1
  state.slaveForm = { ...state.slaveUnitForms[0] }
  state.masterGrid = {
    ...state.masterGrid,
    unitId: 1,
    registerType: 'holding_registers',
    address: 0,
    length: 4,
    addressBase: 0,
    rawVisible: true,
    logVisible: true,
  }
  state.masterMappings = [
    {
      id: `${id}-map-counter`,
      address: 0,
      dataType: 'int32',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: '累计计数',
      groupEnd: false,
    },
    {
      id: `${id}-map-temperature`,
      address: 2,
      dataType: 'float',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 1,
      comment: '温度',
      groupEnd: true,
    },
  ]
  return state
}

function modbusTabId(id: string): string {
  return `modbus:${id}`
}

function fecbusState(id: string, portName: string): FecbusWorkspaceState {
  const state = emptyFecbusState()
  const config = {
    PortName: portName,
    BaudRate: 9600,
    DataBits: 8,
    StopBits: '1',
    Parity: 'none',
    FlowMode: 'none',
    ReadBufKB: 32,
  }
  state.activeSessionId = id
  state.sessions = [{
    ID: id,
    Name: 'FECbus 主控演示',
    Role: FecbusSessionRole.SessionRoleMaster,
    Config: config,
    Status: 'stopped',
    RxBytes: 0,
    TxBytes: 0,
    SlaveRunning: false,
    SourceAddress: 1,
    TargetAddress: 2,
    SlaveUnits: [],
    StartedAt: '',
    StoppedAt: '',
    LastError: '',
  }]
  state.portForm = {
    ...state.portForm,
    sessionId: `next-${id}`,
    port: portName,
    name: 'FECbus 主控演示',
    role: 'master',
  }
  state.sendForm = {
    ...state.sendForm,
    targetAddress: 2,
    sourceAddress: 1,
    priority: 2,
    messageNumber: 7,
    functionCode: FunctionCode.FunctionQueryProtocolVersion,
    payloadHex: '',
    expectAnswer: true,
  }
  state.slaveForm = {
    address: 2,
    statusCode: StatusCode.StatusReceivedOK,
    autoStatusAnswer: true,
    acceptBroadcast: true,
  }
  state.slaveUnits = [state.slaveForm]
  state.customFunctions = [{
    Code: 46 as FunctionCode,
    Name: '用户自定义演示',
    Description: '演示用户自定义功能码和数据段字段。',
    Direction: 'custom',
    Answer: true,
    Fields: [{ Key: 'value', Label: '演示值', Offset: 1, Length: 2, Type: 'uint16', Endian: 'little', Enum: null, Meaning: '' }],
  }]
  state.frameFilters = {
    [id]: { direction: '', search: '' },
  }
  state.framePages = {
    [id]: {
      Frames: [{
        Seq: 1,
        SessionID: id,
        Direction: 'tx',
        Frame: {
          Type: FrameType.FrameTypeRequest,
          TargetAddress: 2,
          Priority: 2,
          SourceAddress: 1,
          MessageNumber: 7,
          GroupNumber: 0,
          Data: 'LA==',
          Raw: null,
          CRCOK: true,
          Timestamp: '',
        },
        Hex: '7e 00 02 02 01 07 00 01 2c 7d 9f 7e',
        Error: '',
        Timestamp: '',
        Annotated: fecbusAnnotation(),
      }],
      Offset: 0,
      Limit: 200,
      Total: 1,
      EOF: true,
    },
  }
  return state
}

function fecbusTabId(id: string): string {
  return `fecbus:${id}`
}

function fecbusAnnotation() {
  return {
    Segments: [
      { Key: 'frame_head', Label: '帧头', Start: 0, End: 1, Hex: '7e', Value: 0x7e, ValueText: '126', Meaning: '0x7E' },
      { Key: 'function', Label: '功能码', Start: 8, End: 9, Hex: '2c', Value: 44, ValueText: '44', Meaning: '查 FECbus 协议版本号' },
    ],
    DataFields: [{
      Key: 'function',
      Label: '功能码',
      Start: 8,
      End: 9,
      Hex: '2c',
      Value: 44,
      ValueText: '查 FECbus 协议版本号',
      Meaning: '查 FECbus 协议版本号',
    }],
    Function: {
      Code: FunctionCode.FunctionQueryProtocolVersion,
      Hex: '2CH',
      Name: '查 FECbus 协议版本号',
      Description: '',
      Direction: 'controller_to_device',
      Answer: true,
      Custom: false,
      Reserved: false,
    },
    GroupKey: '',
    GroupColorIndex: -1,
    Summary: '查 FECbus 协议版本号',
    Warnings: [],
  }
}

function toPortPath(portName: string): string {
  return `/tmp/${portName}`
}

function nextDemoSuffix(): string {
  demoSequence += 1
  return `${Date.now().toString(36)}-${demoSequence}`
}
