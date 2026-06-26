import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { serialService } from '../services/serialService'
import type {
  FramePage,
  OpenSessionRequest,
  QueryRequest,
  SendRequest,
  SessionInfo,
  CustomFunctionDefinition,
  CustomDataFieldDefinition,
  SlaveState,
  SlaveUnitState,
  StartSlaveRequest,
  Transaction,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'
import {
  FrameType,
  FunctionCode,
  SessionRole,
  StatusCode,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'

export type FecbusRole = 'master' | 'slave'

export interface FecbusFunctionOption {
  value: number
  label: string
  name: string
  direction: string
  custom?: boolean
}

export interface FecbusPortFormState {
  sessionId: string
  name: string
  port: string
  role: FecbusRole
  baudRate: number
  dataBits: number
  stopBits: string
  parity: string
  flowMode: string
  timeoutMs: number
  retries: number
}

export interface FecbusSendFormState {
  frameType: FrameType
  targetAddress: number
  priority: number
  sourceAddress: number
  messageNumber: number
  groupNumber: number
  functionCode: number
  payloadHex: string
  expectAnswer: boolean
  timeoutMs: number
  retries: number
  inputMode: 'hex' | 'structured'
  structuredFields: Record<string, string>
}

export interface FecbusSlaveFormState {
  address: number
  statusCode: StatusCode
  autoStatusAnswer: boolean
  acceptBroadcast: boolean
}

export interface FecbusFrameFilterState {
  direction: string
  search: string
}

export interface FecbusCustomFunctionFormState {
  code: number
  name: string
  description: string
  direction: string
  answer: boolean
  fields: CustomDataFieldDefinition[]
}

export interface FecbusWorkspaceState {
  activeSessionId: string | null
  sessions: SessionInfo[]
  portForm: FecbusPortFormState
  sendForm: FecbusSendFormState
  slaveForm: FecbusSlaveFormState
  slaveUnits: FecbusSlaveFormState[]
  customFunctions: CustomFunctionDefinition[]
  frameFilters: Record<string, FecbusFrameFilterState>
  framePages: Record<string, FramePage>
  history: Transaction[]
}

export const fecbusFunctionOptions: FecbusFunctionOption[] = buildFunctionOptions()

export function defaultFecbusPortForm(): FecbusPortFormState {
  return {
    sessionId: defaultSessionId(),
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
  }
}

export function defaultFecbusSendForm(): FecbusSendFormState {
  return {
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
  }
}

export function defaultFecbusSlaveForm(): FecbusSlaveFormState {
  return {
    address: 2,
    statusCode: StatusCode.StatusReceivedOK,
    autoStatusAnswer: true,
    acceptBroadcast: true,
  }
}

export function defaultFecbusWorkspaceState(): FecbusWorkspaceState {
  return {
    activeSessionId: null,
    sessions: [],
    portForm: defaultFecbusPortForm(),
    sendForm: defaultFecbusSendForm(),
    slaveForm: defaultFecbusSlaveForm(),
    slaveUnits: [defaultFecbusSlaveForm()],
    customFunctions: [],
    frameFilters: {},
    framePages: {},
    history: [],
  }
}

export const useFecbusStore = defineStore('serialFecbus', () => {
  const activeSessionId = ref<string | null>(null)
  const sessions = ref<Map<string, SessionInfo>>(new Map())
  const localOnlySessionIds = ref<Set<string>>(new Set())
  const portForm = ref<FecbusPortFormState>(defaultFecbusPortForm())
  const sendForm = ref<FecbusSendFormState>(defaultFecbusSendForm())
  const slaveForm = ref<FecbusSlaveFormState>(defaultFecbusSlaveForm())
  const slaveUnits = ref<FecbusSlaveFormState[]>([defaultFecbusSlaveForm()])
  const customFunctions = ref<CustomFunctionDefinition[]>([])
  const frameFilters = ref<Record<string, FecbusFrameFilterState>>({})
  const framePages = ref<Record<string, FramePage>>({})
  const history = ref<Transaction[]>([])
  const error = ref<string | null>(null)

  const sessionList = computed(() => Array.from(sessions.value.values()))
  const activeSession = computed(() =>
    activeSessionId.value ? sessions.value.get(activeSessionId.value) ?? null : null
  )

  async function refreshSessions() {
    try {
      const list = await serialService.listFecbusSessions()
      const next = new Map(sessions.value)
      for (const session of list) {
        next.set(session.ID, normalizeSessionInfo(session))
        localOnlySessionIds.value.delete(session.ID)
      }
      sessions.value = next
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to refresh FECbus sessions'
    }
  }

  async function openSession() {
    if (!portForm.value.port) {
      error.value = '请选择串口'
      return null
    }
    const request: OpenSessionRequest = {
      ID: portForm.value.sessionId || defaultSessionId(),
      Name: portForm.value.name,
      Role: roleToBinding(portForm.value.role),
      Endpoint: '',
      Config: {
        PortName: portForm.value.port,
        BaudRate: Math.max(1, Math.trunc(portForm.value.baudRate)),
        DataBits: Math.max(5, Math.min(8, Math.trunc(portForm.value.dataBits || 8))),
        StopBits: portForm.value.stopBits || '1',
        Parity: portForm.value.parity || 'none',
        FlowMode: portForm.value.flowMode || 'none',
        ReadBufKB: 32,
      },
      TimeoutMs: Math.max(10, Math.trunc(portForm.value.timeoutMs)),
      Retries: Math.max(0, Math.trunc(portForm.value.retries)),
    }
    try {
      const session = await serialService.openFecbusSession(request)
      sessions.value.set(session.ID, normalizeSessionInfo(session))
      activeSessionId.value = session.ID
      localOnlySessionIds.value.delete(session.ID)
      portForm.value.sessionId = defaultSessionId()
      portForm.value.name = ''
      error.value = null
      return session.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to open FECbus session'
      throw e
    }
  }

  async function closeSession(id = activeSessionId.value) {
    if (!id) return
    try {
      if (!localOnlySessionIds.value.has(id)) {
        await serialService.closeFecbusSession(id)
      }
      sessions.value.delete(id)
      localOnlySessionIds.value.delete(id)
      delete framePages.value[id]
      delete frameFilters.value[id]
      if (activeSessionId.value === id) {
        activeSessionId.value = sessionList.value[0]?.ID ?? null
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to close FECbus session'
      throw e
    }
  }

  async function sendRequest() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 FECbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能直接发送，请先打开真实串口会话'
      return null
    }
    const request = buildSendRequest(session.ID)
    try {
      const tx = await serialService.fecbusSendRequest(request)
      history.value = [tx, ...history.value].slice(0, 200)
      error.value = null
      await Promise.all([refreshSessions(), queryFrames(session.ID)])
      return tx
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to send FECbus request'
      throw e
    }
  }

  async function startSlave() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 FECbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能启动从站，请先打开真实串口会话'
      return null
    }
    const request: StartSlaveRequest = {
      SessionID: session.ID,
      State: slaveState(),
      Units: slaveUnits.value.map(slaveUnitState),
    }
    try {
      const next = await serialService.startFecbusSlave(request)
      sessions.value.set(next.ID, normalizeSessionInfo(next))
      error.value = null
      return next
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to start FECbus slave'
      throw e
    }
  }

  async function stopSlave() {
    const session = activeSession.value
    if (!session) return
    try {
      if (!localOnlySessionIds.value.has(session.ID)) {
        await serialService.stopFecbusSlave(session.ID)
        await refreshSessions()
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to stop FECbus slave'
      throw e
    }
  }

  async function updateSlaveState() {
    const session = activeSession.value
    if (!session) return
    try {
      if (!localOnlySessionIds.value.has(session.ID)) {
        await serialService.updateFecbusSlaveState(session.ID, slaveState())
        await refreshSessions()
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to update FECbus slave state'
      throw e
    }
  }

  async function queryFrames(id = activeSessionId.value) {
    if (!id) return null
    if (localOnlySessionIds.value.has(id)) {
      const page = framePages.value[id] ?? { Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true }
      framePages.value = { ...framePages.value, [id]: page }
      error.value = null
      return page
    }
    const filter = frameFilters.value[id] ?? { direction: '', search: '' }
    const request: QueryRequest = {
      SessionID: id,
      Offset: 0,
      Limit: 200,
      Direction: filter.direction,
      Search: filter.search,
      Custom: customFunctions.value,
    }
    try {
      const page = await serialService.queryFecbusFrames(request)
      framePages.value = { ...framePages.value, [id]: page }
      error.value = null
      return page
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to query FECbus frames'
      throw e
    }
  }

  async function clearFrames(id = activeSessionId.value) {
    if (!id) return
    try {
      if (!localOnlySessionIds.value.has(id)) {
        await serialService.clearFecbusFrames(id)
      }
      framePages.value = {
        ...framePages.value,
        [id]: { Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true },
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to clear FECbus frames'
      throw e
    }
  }

  function setActiveSession(id: string | null) {
    activeSessionId.value = id
  }

  function setFrameFilter(id: string, filter: Partial<FecbusFrameFilterState>) {
    const current = frameFilters.value[id] ?? { direction: '', search: '' }
    frameFilters.value = { ...frameFilters.value, [id]: { ...current, ...filter } }
  }

  function buildSendRequest(sessionId: string): SendRequest {
    const functionCode = clampByte(sendForm.value.functionCode)
    const payload = sendForm.value.inputMode === 'structured'
      ? formatHexBytes(structuredPayload(functionCode, sendForm.value.structuredFields, customFunctions.value))
      : formatHexBytes(parseHexBytes(sendForm.value.payloadHex))
    const dataHex = payload ? `${formatHexBytes([functionCode])} ${payload}` : formatHexBytes([functionCode])
    return {
      SessionID: sessionId,
      FrameType: sendForm.value.frameType,
      TargetAddress: clamp(sendForm.value.targetAddress, 0, 63),
      Priority: clamp(sendForm.value.priority, 0, 3),
      SourceAddress: clamp(sendForm.value.sourceAddress, 1, 63),
      MessageNumber: clamp(sendForm.value.messageNumber, 1, 63),
      GroupNumber: clamp(sendForm.value.groupNumber, 0, 127),
      Function: functionCode as FunctionCode,
      DataHex: dataHex,
      Data: null,
      ExpectAnswer: sendForm.value.expectAnswer,
      TimeoutMs: Math.max(10, Math.trunc(sendForm.value.timeoutMs)),
      Retries: Math.max(0, Math.trunc(sendForm.value.retries)),
    }
  }

  function slaveState(): SlaveState {
    return {
      Address: clamp(slaveForm.value.address, 1, 63),
      DefaultStatus: slaveForm.value.statusCode,
      AutoStatusAnswer: slaveForm.value.autoStatusAnswer,
      AcceptBroadcast: slaveForm.value.acceptBroadcast,
      AnswerPayloadByFunction: null,
    }
  }

  function slaveUnitState(unit: FecbusSlaveFormState): SlaveUnitState {
    return {
      Address: clamp(unit.address, 1, 63),
      DefaultStatus: unit.statusCode,
      AutoStatusAnswer: unit.autoStatusAnswer,
      AcceptBroadcast: unit.acceptBroadcast,
      AnswerPayloadByFunction: null,
    }
  }

  async function addSlaveUnit(unit: Partial<FecbusSlaveFormState> = {}) {
    const next = { ...defaultFecbusSlaveForm(), ...unit }
    next.address = clamp(next.address, 1, 63)
    const existing = slaveUnits.value.findIndex(item => item.address === next.address)
    if (existing >= 0) {
      slaveUnits.value.splice(existing, 1, next)
    } else {
      slaveUnits.value.push(next)
      slaveUnits.value.sort((a, b) => a.address - b.address)
    }
    slaveForm.value = { ...next }
    const session = activeSession.value
    if (session && !localOnlySessionIds.value.has(session.ID)) {
      await serialService.addFecbusSlaveUnit(session.ID, slaveUnitState(next))
      await refreshSessions()
    }
  }

  async function removeSlaveUnit(address: number) {
    const target = clamp(address, 1, 63)
    slaveUnits.value = slaveUnits.value.filter(unit => unit.address !== target)
    if (slaveUnits.value.length === 0) {
      slaveUnits.value = [defaultFecbusSlaveForm()]
    }
    slaveForm.value = { ...slaveUnits.value[0] }
    const session = activeSession.value
    if (session && !localOnlySessionIds.value.has(session.ID)) {
      await serialService.removeFecbusSlaveUnit(session.ID, target)
      await refreshSessions()
    }
  }

  function upsertCustomFunction(definition: CustomFunctionDefinition) {
    const next = normalizeCustomFunction(definition)
    const index = customFunctions.value.findIndex(item => item.Code === next.Code)
    if (index >= 0) {
      customFunctions.value.splice(index, 1, next)
    } else {
      customFunctions.value.push(next)
      customFunctions.value.sort((a, b) => Number(a.Code) - Number(b.Code))
    }
  }

  function removeCustomFunction(code: number) {
    customFunctions.value = customFunctions.value.filter(item => Number(item.Code) !== clampByte(code))
  }

  function resetWorkspace() {
    const next = defaultFecbusWorkspaceState()
    activeSessionId.value = next.activeSessionId
    sessions.value = new Map()
    localOnlySessionIds.value = new Set()
    portForm.value = next.portForm
    sendForm.value = next.sendForm
    slaveForm.value = next.slaveForm
    slaveUnits.value = next.slaveUnits
    customFunctions.value = next.customFunctions
    frameFilters.value = {}
    framePages.value = {}
    history.value = []
    error.value = null
  }

  function exportState(): FecbusWorkspaceState {
    return {
      activeSessionId: activeSessionId.value,
      sessions: sessionList.value,
      portForm: { ...portForm.value },
      sendForm: { ...sendForm.value },
      slaveForm: { ...slaveForm.value },
      slaveUnits: cloneArray(slaveUnits.value),
      customFunctions: cloneArray(customFunctions.value),
      frameFilters: cloneRecord(frameFilters.value),
      framePages: cloneRecord(framePages.value),
      history: history.value,
    }
  }

  function restoreState(snapshot?: FecbusWorkspaceState) {
    const source = snapshot ?? defaultFecbusWorkspaceState()
    activeSessionId.value = source.activeSessionId
    sessions.value = new Map((source.sessions ?? []).map(session => [session.ID, normalizeSessionInfo(session)]))
    localOnlySessionIds.value = new Set((source.sessions ?? []).map(session => session.ID))
    portForm.value = { ...defaultFecbusPortForm(), ...source.portForm }
    sendForm.value = { ...defaultFecbusSendForm(), ...source.sendForm }
    slaveForm.value = { ...defaultFecbusSlaveForm(), ...source.slaveForm }
    slaveUnits.value = normalizeSlaveUnits(source.slaveUnits, slaveForm.value)
    customFunctions.value = cloneArray(source.customFunctions ?? []).map(normalizeCustomFunction)
    frameFilters.value = cloneRecord(source.frameFilters ?? {})
    framePages.value = cloneRecord(source.framePages ?? {})
    history.value = source.history ?? []
    error.value = null
  }

  function cleanup() {
    // Backend sessions are owned by the serial service and released on app shutdown.
  }

  return {
    activeSessionId,
    sessions,
    portForm,
    sendForm,
    slaveForm,
    slaveUnits,
    customFunctions,
    frameFilters,
    framePages,
    history,
    error,
    sessionList,
    activeSession,
    refreshSessions,
    openSession,
    closeSession,
    sendRequest,
    startSlave,
    stopSlave,
    updateSlaveState,
    addSlaveUnit,
    removeSlaveUnit,
    upsertCustomFunction,
    removeCustomFunction,
    queryFrames,
    clearFrames,
    setActiveSession,
    setFrameFilter,
    buildSendRequest,
    resetWorkspace,
    exportState,
    restoreState,
    cleanup,
  }
})

export function parseHexBytes(value: string): number[] {
  const compact = value.replace(/\s+/g, '')
  if (!compact) return []
  if (compact.length % 2 !== 0) {
    throw new Error('HEX 长度必须为偶数')
  }
  if (!/^[0-9a-fA-F]+$/.test(compact)) {
    throw new Error('HEX 包含无效字符')
  }
  const out: number[] = []
  for (let i = 0; i < compact.length; i += 2) {
    const byte = Number.parseInt(compact.slice(i, i + 2), 16)
    out.push(byte)
  }
  return out
}

export function formatHexBytes(bytes: number[]): string {
  return bytes.map(value => clampByte(value).toString(16).padStart(2, '0')).join(' ')
}

function normalizeSessionInfo(session: SessionInfo): SessionInfo {
  return {
    ...session,
    Role: session.Role || SessionRole.SessionRoleMaster,
    SlaveUnits: session.SlaveUnits ?? [],
  }
}

function roleToBinding(role: FecbusRole): SessionRole {
  return role === 'slave' ? SessionRole.SessionRoleSlave : SessionRole.SessionRoleMaster
}

function buildFunctionOptions(): FecbusFunctionOption[] {
  const named: Array<[number, string, string, string]> = [
    [0, '00 同步系统节拍', '同步系统节拍', 'controller_to_device'],
    [1, '01 系统复位', '系统复位', 'controller_to_device'],
    [2, '02 系统消音', '系统消音', 'controller_to_device'],
    [3, '03 系统自检', '系统自检', 'controller_to_device'],
    [4, '04 广播时钟', '广播时钟', 'controller_to_device'],
    [5, '05 控制器紧急事件', '控制器紧急事件', 'controller_to_device'],
    [6, '06 控制器一般事件', '控制器一般事件', 'controller_to_device'],
    [7, '07 控制器调试事件', '控制器调试事件', 'controller_to_device'],
    [15, '0F 状态应答', '状态应答', 'bidirectional'],
    [17, '11 设备紧急事件', '设备紧急事件', 'device_to_controller'],
    [18, '12 设备一般事件', '设备一般事件', 'device_to_controller'],
    [19, '13 设备调试事件', '设备调试事件', 'device_to_controller'],
    [20, '14 通告心跳', '通告心跳', 'device_to_controller'],
    [33, '21 设备巡检', '设备巡检', 'controller_to_device'],
    [34, '22 查设备状态', '查设备状态', 'controller_to_device'],
    [35, '23 查设备配置', '查设备配置', 'controller_to_device'],
    [36, '24 查设备标识', '查设备标识', 'controller_to_device'],
    [37, '25 查设备参量', '查设备参量', 'controller_to_device'],
    [38, '26 查设备注释', '查设备注释', 'controller_to_device'],
    [39, '27 查设备编程', '查设备编程', 'controller_to_device'],
    [40, '28 查注册登记信息', '查注册登记信息', 'controller_to_device'],
    [41, '29 查设备当前事件', '查设备当前事件', 'controller_to_device'],
    [42, '2A 查设备历史事件', '查设备历史事件', 'controller_to_device'],
    [43, '2B 停止查询设备事件', '停止查询设备事件', 'controller_to_device'],
    [44, '2C 查协议版本', '查协议版本', 'controller_to_device'],
    [45, '2D 查设备列表', '查设备列表', 'controller_to_device'],
  ]
  const options: FecbusFunctionOption[] = named.map(([value, label, name, direction]) => ({ value, label, name, direction }))
  for (const [start, end] of [[8, 14], [21, 31], [46, 127]]) {
    for (let value = start; value <= end; value += 1) {
      options.push({
        value,
        label: `${value.toString(16).toUpperCase().padStart(2, '0')} 用户自定义`,
        name: '用户自定义',
        direction: 'custom',
        custom: true,
      })
    }
  }
  return options.sort((a, b) => a.value - b.value)
}

function clamp(value: number, min: number, max: number): number {
  const next = Math.trunc(Number.isFinite(value) ? value : min)
  return Math.min(max, Math.max(min, next))
}

function clampByte(value: number): number {
  return clamp(value, 0, 255)
}

function structuredPayload(functionCode: number, values: Record<string, string>, customFunctions: CustomFunctionDefinition[]): number[] {
  const definition = builtinStructuredDefinitions[functionCode] ?? []
  const custom = customFunctions.find(item => Number(item.Code) === functionCode)?.Fields ?? []
  const fields = custom.length > 0 ? custom : definition
  let length = 0
  for (const field of fields) {
    if (values[field.Key] === undefined || values[field.Key] === '') continue
    length = Math.max(length, field.Offset + field.Length - 1)
  }
  const data = new Array(Math.max(0, length)).fill(0)
  for (const field of fields) {
    const value = values[field.Key]
    if (value === undefined || value === '') continue
    writeField(data, field.Offset - 1, field.Length, field.Type, field.Endian, value)
  }
  return data
}

const builtinStructuredDefinitions: Record<number, CustomDataFieldDefinition[]> = {
  [FunctionCode.FunctionQueryDeviceStatus]: [
    fieldDefinition('controller_id', '控制器编号', 1, 1, 'uint8'),
    fieldDefinition('unit_id', '单元编号', 2, 1, 'uint8'),
    fieldDefinition('device_id', '设备编号', 3, 1, 'uint8'),
    fieldDefinition('channel_id', '通道编号', 4, 1, 'uint8'),
  ],
  [FunctionCode.FunctionQueryConfig]: [
    fieldDefinition('controller_id', '控制器编号', 1, 1, 'uint8'),
    fieldDefinition('unit_id', '单元编号', 2, 1, 'uint8'),
    fieldDefinition('device_id', '设备编号', 3, 1, 'uint8'),
    fieldDefinition('channel_id', '通道编号', 4, 1, 'uint8'),
  ],
  [FunctionCode.FunctionQueryProtocolVersion]: [
    fieldDefinition('controller_id', '控制器编号', 1, 1, 'uint8'),
  ],
}

function fieldDefinition(key: string, label: string, offset: number, length: number, type: string): CustomDataFieldDefinition {
  return { Key: key, Label: label, Offset: offset, Length: length, Type: type, Endian: 'little', Enum: null, Meaning: '' }
}

function writeField(data: number[], offset: number, length: number, type: string, endian: string, value: string) {
  if (offset < 0 || length <= 0) return
  if (type === 'string' || type === 'utf8') {
    const bytes = Array.from(String(value)).map(char => char.charCodeAt(0) & 0xff)
    for (let i = 0; i < length && i < bytes.length; i += 1) data[offset + i] = bytes[i]
    return
  }
  if (type === 'hex') {
    const bytes = parseHexBytes(String(value))
    for (let i = 0; i < length && i < bytes.length; i += 1) data[offset + i] = bytes[i]
    return
  }
  let numeric = Number(value)
  if (!Number.isFinite(numeric)) numeric = 0
  if (length === 1) {
    data[offset] = clampByte(numeric)
    return
  }
  for (let i = 0; i < length; i += 1) {
    const shift = endian === 'big' ? 8 * (length - i - 1) : 8 * i
    data[offset + i] = clampByte(Math.trunc(numeric / (2 ** shift)))
  }
}

function normalizeSlaveUnits(units: FecbusSlaveFormState[] | undefined, fallback: FecbusSlaveFormState): FecbusSlaveFormState[] {
  const source = units?.length ? units : [fallback]
  return source.map(unit => ({
    address: clamp(unit.address, 1, 63),
    statusCode: unit.statusCode ?? StatusCode.StatusReceivedOK,
    autoStatusAnswer: unit.autoStatusAnswer ?? true,
    acceptBroadcast: unit.acceptBroadcast ?? true,
  })).sort((a, b) => a.address - b.address)
}

function normalizeCustomFunction(definition: CustomFunctionDefinition): CustomFunctionDefinition {
  return {
    Code: clampByte(Number(definition.Code)) as FunctionCode,
    Name: definition.Name || '用户自定义',
    Description: definition.Description || '',
    Direction: definition.Direction || 'custom',
    Answer: Boolean(definition.Answer),
    Fields: (definition.Fields ?? []).map(field => ({
      Key: field.Key,
      Label: field.Label || field.Key,
      Offset: Math.max(1, Math.trunc(field.Offset || 1)),
      Length: Math.max(1, Math.trunc(field.Length || 1)),
      Type: field.Type || 'uint8',
      Endian: field.Endian || 'little',
      Enum: field.Enum ?? null,
      Meaning: field.Meaning || '',
    })),
  }
}

function defaultSessionId(): string {
  return `fecbus-${Date.now().toString(36)}`
}

function cloneRecord<T>(record: Record<string, T>): Record<string, T> {
  return Object.fromEntries(Object.entries(record).map(([key, value]) => [key, JSON.parse(JSON.stringify(value)) as T]))
}

function cloneArray<T>(value: T[]): T[] {
  return JSON.parse(JSON.stringify(value)) as T[]
}
