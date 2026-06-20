import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { serialService } from '../services/serialService'
import type {
  DataModelSnapshot,
  MasterRequest,
  OpenSessionRequest,
  RegisterMapping,
  RegisterReadRequest,
  RegisterReadResult,
  RegisterScanRequest,
  RegisterScanResult,
  SessionInfo,
  StartSlaveRequest,
  Transaction,
  UnitScanRequest,
  UnitScanResult,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/modbus/models.js'
import { AddressMode, DataType, FrameMode, SessionRole, WordOrder } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/modbus/models.js'

export type ModbusFrameMode = 'rtu' | 'ascii'
export type ModbusAddressMode = 'zero-based' | 'plc'
export type ModbusDataType = 'none' | 'int16' | 'uint16' | 'int32' | 'uint32' | 'int64' | 'uint64' | 'float' | 'double' | 'unix' | 'datetime' | 'utf8'
export type ModbusWordOrder = 'big' | 'little'
export type ModbusRole = 'master' | 'slave'
export type ModbusRegisterType = 'coils' | 'discrete_inputs' | 'input_registers' | 'holding_registers'

export const registerTypes: ModbusRegisterType[] = ['coils', 'discrete_inputs', 'input_registers', 'holding_registers']

export interface ModbusMasterGridState {
  unitId: number
  registerType: ModbusRegisterType
  address: number
  length: number
  addressBase: 0 | 1
  readConfigured: boolean
  littleEndian: boolean
  rawVisible: boolean
  logVisible: boolean
  pollRateMs: number
  timeoutMs: number
  retries: number
}

export interface ModbusRegisterMappingRow {
  id: string
  address: number
  dataType: ModbusDataType
  wordOrder: ModbusWordOrder
  length: number
  scalingFactor: number
  comment: string
  groupEnd: boolean
}

export interface ModbusMasterTableRow {
  id: string
  address: number
  value: string
  raw: string
  display: string
  mappingId: string
}

export interface ModbusMasterRegisterTableState {
  type: ModbusRegisterType
  unitId: number
  address: number
  length: number
  addressBase: 0 | 1
  pollRateMs: number
  timeoutMs: number
  retries: number
  rows: ModbusMasterTableRow[]
  mappings: ModbusRegisterMappingRow[]
}

export interface ModbusSlaveBoolRow {
  id: string
  address: number
  value: boolean
}

export interface ModbusSlaveRegisterRow {
  id: string
  address: number
  value: number
  dataType: ModbusDataType
  comment: string
}

export interface ModbusSlaveUnitGridState {
  unitId: number
  coils: ModbusSlaveBoolRow[]
  discreteInputs: ModbusSlaveBoolRow[]
  inputRegisters: ModbusSlaveRegisterRow[]
  holdingRegisters: ModbusSlaveRegisterRow[]
}

export interface ModbusPortFormState {
  sessionId: string
  name: string
  port: string
  mode: ModbusFrameMode
  role: ModbusRole
  baudRate: number
  dataBits: number
  stopBits: string
  parity: string
  flowMode: string
  timeoutMs: number
  retries: number
}

export interface ModbusMasterFormState {
  unitId: number
  functionCode: number
  addressMode: ModbusAddressMode
  address: number
  quantity: number
  value: number
  coilValues: string
  registerValues: string
  timeoutMs: number
  retries: number
}

export interface ModbusSlaveFormState {
  unitId: number
  coils: string
  discreteInputs: string
  inputRegisters: string
  holdingRegisters: string
}

export interface ModbusRegisterReadFormState {
  unitId: number
  functionCode: number
  addressMode: ModbusAddressMode
  address: number
  quantity: number
  mappingText: string
  timeoutMs: number
  retries: number
  pollIntervalMs: number
  polling: boolean
}

export interface ModbusUnitScanFormState {
  unitIds: string
  functionCode: number
  addressMode: ModbusAddressMode
  address: number
  quantity: number
  timeoutMs: number
}

export interface ModbusRegisterScanFormState {
  unitId: number
  functionCode: number
  addressMode: ModbusAddressMode
  startAddress: number
  endAddress: number
  chunkSize: number
  timeoutMs: number
}

export interface ModbusWorkspaceState {
  activeSessionId: string | null
  sessions: SessionInfo[]
  portForm: ModbusPortFormState
  masterForm: ModbusMasterFormState
  slaveForm: ModbusSlaveFormState
  activeSlaveUnitId: number
  slaveUnitForms: ModbusSlaveFormState[]
  masterGrid: ModbusMasterGridState
  masterMappings: ModbusRegisterMappingRow[]
  masterRegisterTables?: ModbusMasterRegisterTableState[]
  slaveUnitGrids: ModbusSlaveUnitGridState[]
  registerReadForm: ModbusRegisterReadFormState
  unitScanForm: ModbusUnitScanFormState
  registerScanForm: ModbusRegisterScanFormState
  registerReadResult: RegisterReadResult | null
  unitScanResult: UnitScanResult | null
  registerScanResult: RegisterScanResult | null
  history: Transaction[]
}

export function defaultModbusPortForm(): ModbusPortFormState {
  return {
    sessionId: defaultSessionId(),
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
  }
}

export function defaultModbusMasterForm(): ModbusMasterFormState {
  return {
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
  }
}

export function defaultModbusSlaveForm(): ModbusSlaveFormState {
  return {
    unitId: 1,
    coils: '0=1\n1=0\n2=1',
    discreteInputs: '0=1\n1=1',
    inputRegisters: '0=24\n1=42',
    holdingRegisters: '0=24\n1=42',
  }
}

export function defaultModbusRegisterReadForm(): ModbusRegisterReadFormState {
  return {
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
  }
}

export function defaultModbusUnitScanForm(): ModbusUnitScanFormState {
  return {
    unitIds: '1-10',
    functionCode: 3,
    addressMode: 'zero-based',
    address: 0,
    quantity: 1,
    timeoutMs: 120,
  }
}

export function defaultModbusRegisterScanForm(): ModbusRegisterScanFormState {
  return {
    unitId: 1,
    functionCode: 3,
    addressMode: 'zero-based',
    startAddress: 0,
    endAddress: 64,
    chunkSize: 16,
    timeoutMs: 200,
  }
}

export function defaultModbusMasterGrid(): ModbusMasterGridState {
  return {
    unitId: 1,
    registerType: 'holding_registers',
    address: 0,
    length: 4,
    addressBase: 0,
    readConfigured: false,
    littleEndian: false,
    rawVisible: true,
    logVisible: false,
    pollRateMs: 1000,
    timeoutMs: 800,
    retries: 0,
  }
}

export function defaultModbusMasterMappings(): ModbusRegisterMappingRow[] {
  return [
    {
      id: 'map-0',
      address: 0,
      dataType: 'int32',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: '寄存器 0',
      groupEnd: false,
    },
    {
      id: 'map-2',
      address: 2,
      dataType: 'float',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: '寄存器 2',
      groupEnd: false,
    },
  ]
}

export function defaultModbusRegisterTables(): ModbusMasterRegisterTableState[] {
  const base = defaultModbusMasterGrid()
  return registerTypes.map(type => ({
    type,
    unitId: base.unitId,
    address: 0,
    length: type === 'coils' || type === 'discrete_inputs' ? 8 : base.length,
    addressBase: base.addressBase,
    pollRateMs: base.pollRateMs,
    timeoutMs: base.timeoutMs,
    retries: base.retries,
    rows: defaultMasterRows(type, type === 'coils' || type === 'discrete_inputs' ? 8 : base.length),
    mappings: type === 'input_registers' || type === 'holding_registers' ? defaultModbusMasterMappings() : [],
  }))
}

export function defaultModbusSlaveUnitGrid(unitId = 1): ModbusSlaveUnitGridState {
  return {
    unitId: normalizeUnitId(unitId),
    coils: [
      { id: `u${unitId}-coil-0`, address: 0, value: true },
      { id: `u${unitId}-coil-1`, address: 1, value: false },
      { id: `u${unitId}-coil-2`, address: 2, value: true },
    ],
    discreteInputs: [
      { id: `u${unitId}-di-0`, address: 0, value: true },
      { id: `u${unitId}-di-1`, address: 1, value: true },
    ],
    inputRegisters: [
      { id: `u${unitId}-ir-0`, address: 0, value: 24, dataType: 'uint16', comment: '输入 0' },
      { id: `u${unitId}-ir-1`, address: 1, value: 42, dataType: 'uint16', comment: '输入 1' },
    ],
    holdingRegisters: [
      { id: `u${unitId}-hr-0`, address: 0, value: 24, dataType: 'uint16', comment: '保持 0' },
      { id: `u${unitId}-hr-1`, address: 1, value: 42, dataType: 'uint16', comment: '保持 1' },
    ],
  }
}

export function defaultModbusWorkspaceState(): ModbusWorkspaceState {
  const slaveForm = defaultModbusSlaveForm()
  return {
    activeSessionId: null,
    sessions: [],
    portForm: defaultModbusPortForm(),
    masterForm: defaultModbusMasterForm(),
    slaveForm,
    activeSlaveUnitId: slaveForm.unitId,
    slaveUnitForms: [{ ...slaveForm }],
    masterGrid: defaultModbusMasterGrid(),
    masterMappings: defaultModbusMasterMappings(),
    masterRegisterTables: defaultModbusRegisterTables(),
    slaveUnitGrids: [defaultModbusSlaveUnitGrid(slaveForm.unitId)],
    registerReadForm: defaultModbusRegisterReadForm(),
    unitScanForm: defaultModbusUnitScanForm(),
    registerScanForm: defaultModbusRegisterScanForm(),
    registerReadResult: null,
    unitScanResult: null,
    registerScanResult: null,
    history: [],
  }
}

export const useModbusStore = defineStore('serialModbus', () => {
  const activeSessionId = ref<string | null>(null)
  const sessions = ref<Map<string, SessionInfo>>(new Map())
  const localOnlySessionIds = ref<Set<string>>(new Set())
  const portForm = ref<ModbusPortFormState>(defaultModbusPortForm())
  const masterForm = ref<ModbusMasterFormState>(defaultModbusMasterForm())
  const slaveForm = ref<ModbusSlaveFormState>(defaultModbusSlaveForm())
  const activeSlaveUnitId = ref<number>(slaveForm.value.unitId)
  const slaveUnitForms = ref<ModbusSlaveFormState[]>([{ ...slaveForm.value }])
  const masterGrid = ref<ModbusMasterGridState>(defaultModbusMasterGrid())
  const masterMappings = ref<ModbusRegisterMappingRow[]>(defaultModbusMasterMappings())
  const masterRegisterTables = ref<ModbusMasterRegisterTableState[]>(defaultModbusRegisterTables())
  const slaveUnitGrids = ref<ModbusSlaveUnitGridState[]>([defaultModbusSlaveUnitGrid(slaveForm.value.unitId)])
  const registerReadForm = ref<ModbusRegisterReadFormState>(defaultModbusRegisterReadForm())
  const unitScanForm = ref<ModbusUnitScanFormState>(defaultModbusUnitScanForm())
  const registerScanForm = ref<ModbusRegisterScanFormState>(defaultModbusRegisterScanForm())
  const registerReadResult = ref<RegisterReadResult | null>(null)
  const unitScanResult = ref<UnitScanResult | null>(null)
  const registerScanResult = ref<RegisterScanResult | null>(null)
  const history = ref<Transaction[]>([])
  const error = ref<string | null>(null)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const sessionList = computed(() => Array.from(sessions.value.values()))
  const activeSession = computed(() =>
    activeSessionId.value ? sessions.value.get(activeSessionId.value) ?? null : null
  )
  const activeSlaveForm = computed(() =>
    slaveUnitForms.value.find(unit => unit.unitId === activeSlaveUnitId.value) ?? slaveForm.value
  )
  const activeSlaveGrid = computed(() =>
    slaveUnitGrids.value.find(unit => unit.unitId === activeSlaveUnitId.value) ?? defaultModbusSlaveUnitGrid(activeSlaveUnitId.value)
  )

  async function refreshSessions() {
    try {
      const list = await serialService.listModbusSessions()
      const next = new Map(sessions.value)
      for (const session of list) {
        next.set(session.ID, normalizeSessionInfo(session))
        localOnlySessionIds.value.delete(session.ID)
      }
      sessions.value = next
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to refresh Modbus sessions'
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
      Mode: portForm.value.mode as FrameMode,
      Role: portForm.value.role as SessionRole,
      Endpoint: '',
      Config: {
        PortName: portForm.value.port,
        BaudRate: portForm.value.baudRate,
        DataBits: portForm.value.dataBits,
        StopBits: portForm.value.stopBits,
        Parity: portForm.value.parity,
        FlowMode: portForm.value.flowMode,
        ReadBufKB: 32,
      },
      TimeoutMs: portForm.value.timeoutMs,
      Retries: portForm.value.retries,
    }
    try {
      const session = await serialService.openModbusSession(request)
      sessions.value.set(session.ID, session)
      activeSessionId.value = session.ID
      localOnlySessionIds.value.delete(session.ID)
      portForm.value.sessionId = defaultSessionId()
      portForm.value.name = ''
      error.value = null
      return session.ID
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to open Modbus session'
      throw e
    }
  }

  async function closeSession(id = activeSessionId.value) {
    if (!id) return
    try {
      if (!localOnlySessionIds.value.has(id)) {
        await serialService.closeModbusSession(id)
      }
      sessions.value.delete(id)
      localOnlySessionIds.value.delete(id)
      if (activeSessionId.value === id) {
        activeSessionId.value = sessionList.value[0]?.ID ?? null
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to close Modbus session'
      throw e
    }
  }

  async function sendMasterRequest() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 Modbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能直接发送，请先打开真实串口会话'
      return null
    }
    const request: MasterRequest = {
      SessionID: session.ID,
      UnitID: masterForm.value.unitId,
      Function: masterForm.value.functionCode,
      AddressMode: masterForm.value.addressMode as AddressMode,
      Address: masterForm.value.address,
      Quantity: masterForm.value.quantity,
      Value: masterForm.value.value,
      CoilValues: parseBoolValues(masterForm.value.coilValues),
      RegisterValues: parseRegisterValues(masterForm.value.registerValues),
      TimeoutMs: masterForm.value.timeoutMs,
      Retries: masterForm.value.retries,
    }
    try {
      const tx = await serialService.modbusMasterRequest(request)
      history.value = [tx, ...history.value].slice(0, 200)
      error.value = null
      await refreshSessions()
      return tx
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to send Modbus request'
      throw e
    }
  }

  async function startSlave() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 Modbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能启动从站，请先打开真实串口会话'
      return null
    }
    syncActiveSlaveState()
    const units = normalizedSlaveUnitGrids()
    const request: StartSlaveRequest = {
      SessionID: session.ID,
      UnitID: activeSlaveUnitId.value,
      DataModel: slaveDataModel(),
      Units: units.map(unit => ({
        UnitID: unit.unitId,
        DataModel: slaveGridToDataModel(unit),
      })),
    }
    try {
      const next = await serialService.startModbusSlave(request)
      sessions.value.set(next.ID, next)
      error.value = null
      return next
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to start Modbus slave'
      throw e
    }
  }

  async function stopSlave() {
    const session = activeSession.value
    if (!session) return
    try {
      if (!localOnlySessionIds.value.has(session.ID)) {
        await serialService.stopModbusSlave(session.ID)
        await refreshSessions()
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to stop Modbus slave'
      throw e
    }
  }

  async function applySlaveData() {
    const session = activeSession.value
    if (!session) return
    syncActiveSlaveState()
    try {
      if (!localOnlySessionIds.value.has(session.ID)) {
        await serialService.updateModbusSlaveUnitData(session.ID, activeSlaveUnitId.value, slaveDataModel())
        await refreshSessions()
      }
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to update Modbus slave data'
      throw e
    }
  }

  function slaveDataModel(form = slaveForm.value): DataModelSnapshot {
    const grid = slaveUnitGrids.value.find(unit => unit.unitId === normalizeUnitId(form.unitId))
    return grid ? slaveGridToDataModel(grid) : slaveGridToDataModel(slaveFormToGrid(form))
  }

  function syncActiveSlaveForm() {
    const unitId = normalizeUnitId(slaveForm.value.unitId)
    slaveForm.value = { ...slaveForm.value, unitId }
    activeSlaveUnitId.value = unitId
    const index = slaveUnitForms.value.findIndex(unit => unit.unitId === unitId)
    if (index >= 0) {
      slaveUnitForms.value[index] = { ...slaveForm.value }
    } else {
      slaveUnitForms.value = [...slaveUnitForms.value, { ...slaveForm.value }]
    }
  }

  function syncActiveSlaveGrid() {
    const unitId = normalizeUnitId(activeSlaveUnitId.value || slaveForm.value.unitId)
    activeSlaveUnitId.value = unitId
    const existing = slaveUnitGrids.value.find(unit => unit.unitId === unitId)
    if (!existing) {
      const derived = slaveFormToGrid({ ...slaveForm.value, unitId })
      slaveUnitGrids.value = [...slaveUnitGrids.value, derived].sort((a, b) => a.unitId - b.unitId)
    }
  }

  function syncActiveSlaveState() {
    syncActiveSlaveForm()
    const gridIndex = slaveUnitGrids.value.findIndex(unit => unit.unitId === activeSlaveUnitId.value)
    if (gridIndex < 0) {
      slaveUnitGrids.value = [...slaveUnitGrids.value, slaveFormToGrid(activeSlaveForm.value)].sort((a, b) => a.unitId - b.unitId)
    }
  }

  function normalizedSlaveUnitForms(): ModbusSlaveFormState[] {
    syncActiveSlaveForm()
    const byUnit = new Map<number, ModbusSlaveFormState>()
    for (const unit of slaveUnitForms.value) {
      byUnit.set(normalizeUnitId(unit.unitId), { ...unit, unitId: normalizeUnitId(unit.unitId) })
    }
    return Array.from(byUnit.values()).sort((a, b) => a.unitId - b.unitId)
  }

  function normalizedSlaveUnitGrids(): ModbusSlaveUnitGridState[] {
    syncActiveSlaveGrid()
    const byUnit = new Map<number, ModbusSlaveUnitGridState>()
    for (const unit of slaveUnitGrids.value) {
      byUnit.set(normalizeUnitId(unit.unitId), normalizeSlaveGrid(unit))
    }
    return Array.from(byUnit.values()).sort((a, b) => a.unitId - b.unitId)
  }

  function exportSlaveUnitForms(): ModbusSlaveFormState[] {
    const byUnit = new Map<number, ModbusSlaveFormState>()
    for (const unit of slaveUnitForms.value) {
      const unitId = normalizeUnitId(unit.unitId)
      byUnit.set(unitId, { ...unit, unitId })
    }
    const activeUnitId = normalizeUnitId(slaveForm.value.unitId)
    byUnit.set(activeUnitId, { ...slaveForm.value, unitId: activeUnitId })
    return Array.from(byUnit.values()).sort((a, b) => a.unitId - b.unitId)
  }

  function exportSlaveUnitGrids(): ModbusSlaveUnitGridState[] {
    const byUnit = new Map<number, ModbusSlaveUnitGridState>()
    for (const unit of slaveUnitGrids.value) {
      byUnit.set(normalizeUnitId(unit.unitId), normalizeSlaveGrid(unit))
    }
    for (const unit of slaveUnitForms.value) {
      const unitId = normalizeUnitId(unit.unitId)
      if (!byUnit.has(unitId)) {
        byUnit.set(unitId, slaveFormToGrid(unit))
      }
    }
    return Array.from(byUnit.values()).sort((a, b) => a.unitId - b.unitId)
  }

  function exportMasterRegisterTables(): ModbusMasterRegisterTableState[] {
    return masterRegisterTables.value.map(table => ({
      ...table,
      rows: table.rows.map(row => ({ ...row })),
      mappings: table.mappings.map(row => ({ ...row })),
    }))
  }

  function activeMasterTable(): ModbusMasterRegisterTableState {
    const type = normalizeRegisterType(masterGrid.value.registerType)
    let table = masterRegisterTables.value.find(item => item.type === type)
    if (!table) {
      table = masterTableFromGrid(type, masterGrid.value, masterMappings.value)
      masterRegisterTables.value = normalizeMasterRegisterTables([...masterRegisterTables.value, table], masterGrid.value, masterMappings.value)
      table = masterRegisterTables.value.find(item => item.type === type)!
    }
    if (tableDiffersFromGrid(table, masterGrid.value)) {
      table.unitId = masterGrid.value.unitId
      table.address = masterGrid.value.address
      table.length = masterGrid.value.length
      table.addressBase = masterGrid.value.addressBase
      table.pollRateMs = masterGrid.value.pollRateMs
      table.timeoutMs = masterGrid.value.timeoutMs
      table.retries = masterGrid.value.retries
      table.rows = defaultMasterRows(table.type, table.length, table.address)
      if (masterMappings.value.length) {
        table.mappings = masterMappings.value.map(row => ({ ...row }))
      }
    }
    return table
  }

  function syncMasterGridFromTable(table: ModbusMasterRegisterTableState) {
    masterGrid.value = {
      ...masterGrid.value,
      unitId: table.unitId,
      registerType: table.type,
      address: table.address,
      length: table.length,
      addressBase: table.addressBase,
      pollRateMs: table.pollRateMs,
      timeoutMs: table.timeoutMs,
      retries: table.retries,
    }
    masterMappings.value = table.mappings.map(row => ({ ...row }))
  }

  function syncActiveMasterTableToLegacyState() {
    syncMasterGridFromTable(activeMasterTable())
  }

  function tableDiffersFromGrid(table: ModbusMasterRegisterTableState, grid: ModbusMasterGridState): boolean {
    return table.unitId !== grid.unitId ||
      table.address !== grid.address ||
      table.length !== grid.length ||
      table.addressBase !== grid.addressBase ||
      table.pollRateMs !== grid.pollRateMs ||
      table.timeoutMs !== grid.timeoutMs ||
      table.retries !== grid.retries
  }

  function applyRegisterReadResult(table: ModbusMasterRegisterTableState, result: RegisterReadResult) {
    const rowsByAddress = new Map(table.rows.map(row => [row.address, row]))
    result.RawRegisters?.forEach((value, index) => {
      const row = rowsByAddress.get(table.address + index)
      if (row) {
        row.raw = String(value)
        row.value = String(value)
        row.display = String(value)
      }
    })
    result.Bits?.forEach((value, index) => {
      const row = rowsByAddress.get(table.address + index)
      if (row) {
        row.raw = value ? '1' : '0'
        row.value = value ? 'ON' : 'OFF'
        row.display = row.value
      }
    })
    result.Values?.forEach(value => {
      const row = rowsByAddress.get(value.Address)
      if (row) {
        row.raw = (value.Value.Raw ?? []).join(', ')
        row.value = value.Error || value.Value.Display
        row.display = value.Error || value.Value.Display
        row.mappingId = `${value.Mapping.Address}-${value.Mapping.DataType}`
      }
    })
  }

  function selectSlaveUnit(unitId: number) {
    syncActiveSlaveForm()
    const normalized = normalizeUnitId(unitId)
    const next = slaveUnitForms.value.find(unit => unit.unitId === normalized)
    if (!next && !slaveUnitGrids.value.some(unit => unit.unitId === normalized)) return
    activeSlaveUnitId.value = normalized
    if (next) {
      slaveForm.value = { ...next }
    } else {
      slaveForm.value = gridToSlaveForm(slaveUnitGrids.value.find(unit => unit.unitId === normalized) ?? defaultModbusSlaveUnitGrid(normalized))
    }
    syncActiveSlaveGrid()
  }

  async function addSlaveUnit(unitId?: number) {
    syncActiveSlaveState()
    const used = new Set([
      ...slaveUnitForms.value.map(unit => unit.unitId),
      ...slaveUnitGrids.value.map(unit => unit.unitId),
    ])
    let nextUnitId = normalizeUnitId(unitId ?? 1)
    while (used.has(nextUnitId) && nextUnitId < 247) nextUnitId += 1
    const next = { ...defaultModbusSlaveForm(), unitId: nextUnitId }
    const nextGrid = defaultModbusSlaveUnitGrid(nextUnitId)
    slaveUnitForms.value = [...slaveUnitForms.value, next].sort((a, b) => a.unitId - b.unitId)
    slaveUnitGrids.value = [...slaveUnitGrids.value, nextGrid].sort((a, b) => a.unitId - b.unitId)
    selectSlaveUnit(nextUnitId)
    const session = activeSession.value
    if (session && !localOnlySessionIds.value.has(session.ID)) {
      await serialService.addModbusSlaveUnit(session.ID, {
        UnitID: nextUnitId,
        DataModel: slaveGridToDataModel(nextGrid),
      })
      await refreshSessions()
    }
  }

  async function removeSlaveUnit(unitId = activeSlaveUnitId.value) {
    if (slaveUnitForms.value.length <= 1) return
    const normalized = normalizeUnitId(unitId)
    slaveUnitForms.value = slaveUnitForms.value.filter(unit => unit.unitId !== normalized)
    slaveUnitGrids.value = slaveUnitGrids.value.filter(unit => unit.unitId !== normalized)
    const session = activeSession.value
    if (session && !localOnlySessionIds.value.has(session.ID)) {
      await serialService.removeModbusSlaveUnit(session.ID, normalized)
      await refreshSessions()
    }
    selectSlaveUnit(slaveUnitForms.value[0]?.unitId ?? 1)
  }

  async function readRegisters() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 Modbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能直接读取，请先打开真实串口会话'
      return null
    }
    const request = buildRegisterReadRequest()
    if (!request) return null
    try {
      const result = await serialService.modbusReadRegisters(request)
      registerReadResult.value = result
      applyRegisterReadResult(activeMasterTable(), result)
      if (result.Transaction) {
        history.value = [result.Transaction, ...history.value].slice(0, 200)
      }
      error.value = null
      await refreshSessions()
      return result
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to read Modbus registers'
      throw e
    }
  }

  async function scanUnitIDs() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 Modbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能直接扫描，请先打开真实串口会话'
      return null
    }
    const request = buildUnitScanRequest()
    if (!request) return null
    try {
      const result = await serialService.modbusScanUnitIDs(request)
      unitScanResult.value = result
      error.value = null
      await refreshSessions()
      return result
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to scan Modbus unit IDs'
      throw e
    }
  }

  async function scanRegisters() {
    const session = activeSession.value
    if (!session) {
      error.value = '请先打开 Modbus 会话'
      return null
    }
    if (localOnlySessionIds.value.has(session.ID)) {
      error.value = '已恢复会话不能直接扫描，请先打开真实串口会话'
      return null
    }
    const request = buildRegisterScanRequest()
    if (!request) return null
    try {
      const result = await serialService.modbusScanRegisters(request)
      registerScanResult.value = result
      error.value = null
      await refreshSessions()
      return result
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to scan Modbus registers'
      throw e
    }
  }

  function startRegisterPolling() {
    stopRegisterPolling()
    registerReadForm.value.polling = true
    pollTimer = setInterval(() => {
      readRegisters().catch(() => {
        stopRegisterPolling()
      })
    }, Math.max(10, activeMasterTable().pollRateMs || masterGrid.value.pollRateMs || registerReadForm.value.pollIntervalMs))
  }

  function stopRegisterPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    registerReadForm.value.polling = false
  }

  function buildRegisterReadRequest(): RegisterReadRequest | null {
    const session = activeSession.value
    if (!session) return null
    const grid = activeMasterTable()
    syncMasterGridFromTable(grid)
    return {
      SessionID: session.ID,
      UnitID: normalizeUnitId(grid.unitId),
      Function: registerTypeToFunction(grid.type),
      AddressMode: grid.addressBase === 1 ? AddressMode.AddressModePLC : AddressMode.AddressModeZeroBased,
      Address: Math.max(0, Math.trunc(grid.address)),
      Quantity: Math.max(1, Math.trunc(grid.length)),
      Mappings: registerMappingRowsToMappings(grid.mappings),
      TimeoutMs: Math.max(10, Math.trunc(grid.timeoutMs)),
      Retries: Math.max(0, Math.trunc(grid.retries)),
    }
  }

  function buildUnitScanRequest(): UnitScanRequest | null {
    const session = activeSession.value
    if (!session) return null
    const grid = activeMasterTable()
    syncMasterGridFromTable(grid)
    return {
      SessionID: session.ID,
      UnitIDs: parseUnitIds(unitScanForm.value.unitIds),
      Function: registerTypeToFunction(grid.type),
      AddressMode: grid.addressBase === 1 ? AddressMode.AddressModePLC : AddressMode.AddressModeZeroBased,
      Address: Math.max(0, Math.trunc(grid.address)),
      Quantity: Math.max(1, Math.trunc(grid.length)),
      TimeoutMs: Math.max(10, Math.trunc(grid.timeoutMs)),
      Retries: Math.max(0, Math.trunc(grid.retries)),
    }
  }

  function buildRegisterScanRequest(): RegisterScanRequest | null {
    const session = activeSession.value
    if (!session) return null
    const grid = activeMasterTable()
    syncMasterGridFromTable(grid)
    return {
      SessionID: session.ID,
      UnitID: normalizeUnitId(grid.unitId),
      Function: registerTypeToFunction(grid.type),
      AddressMode: grid.addressBase === 1 ? AddressMode.AddressModePLC : AddressMode.AddressModeZeroBased,
      StartAddress: Math.max(0, Math.trunc(registerScanForm.value.startAddress)),
      EndAddress: Math.max(0, Math.trunc(registerScanForm.value.endAddress)),
      ChunkSize: Math.max(1, Math.trunc(registerScanForm.value.chunkSize)),
      TimeoutMs: Math.max(10, Math.trunc(grid.timeoutMs)),
      Retries: Math.max(0, Math.trunc(grid.retries)),
    }
  }

  function setActiveSession(id: string | null) {
    activeSessionId.value = id
  }

  function clearError() {
    error.value = null
  }

  function resetWorkspace() {
    stopRegisterPolling()
    const next = defaultModbusWorkspaceState()
    activeSessionId.value = next.activeSessionId
    sessions.value = new Map()
    localOnlySessionIds.value = new Set()
    portForm.value = next.portForm
    masterForm.value = next.masterForm
    slaveForm.value = next.slaveForm
    activeSlaveUnitId.value = next.activeSlaveUnitId
    slaveUnitForms.value = next.slaveUnitForms
    masterGrid.value = next.masterGrid
    masterMappings.value = next.masterMappings
    masterRegisterTables.value = next.masterRegisterTables ?? defaultModbusRegisterTables()
    slaveUnitGrids.value = next.slaveUnitGrids
    registerReadForm.value = next.registerReadForm
    unitScanForm.value = next.unitScanForm
    registerScanForm.value = next.registerScanForm
    registerReadResult.value = next.registerReadResult
    unitScanResult.value = next.unitScanResult
    registerScanResult.value = next.registerScanResult
    history.value = []
    error.value = null
  }

  function exportState(): ModbusWorkspaceState {
    return {
      activeSessionId: activeSessionId.value,
      sessions: sessionList.value,
      portForm: { ...portForm.value },
      masterForm: { ...masterForm.value },
      slaveForm: { ...slaveForm.value },
      activeSlaveUnitId: activeSlaveUnitId.value,
      slaveUnitForms: exportSlaveUnitForms().map(unit => ({ ...unit })),
      masterGrid: { ...masterGrid.value },
      masterMappings: masterMappings.value.map(row => ({ ...row })),
      masterRegisterTables: exportMasterRegisterTables(),
      slaveUnitGrids: exportSlaveUnitGrids(),
      registerReadForm: { ...registerReadForm.value },
      unitScanForm: { ...unitScanForm.value },
      registerScanForm: { ...registerScanForm.value },
      registerReadResult: registerReadResult.value,
      unitScanResult: unitScanResult.value,
      registerScanResult: registerScanResult.value,
      history: history.value,
    }
  }

  function restoreState(snapshot?: ModbusWorkspaceState) {
    stopRegisterPolling()
    const source = snapshot ?? defaultModbusWorkspaceState()
    activeSessionId.value = source.activeSessionId
    sessions.value = new Map(source.sessions.map(session => [session.ID, normalizeSessionInfo(session)]))
    localOnlySessionIds.value = new Set(source.sessions.map(session => session.ID))
    portForm.value = { ...defaultModbusPortForm(), ...source.portForm, role: source.portForm?.role ?? 'master' }
    masterForm.value = { ...defaultModbusMasterForm(), ...source.masterForm }
    slaveForm.value = { ...defaultModbusSlaveForm(), ...source.slaveForm }
    activeSlaveUnitId.value = normalizeUnitId(source.activeSlaveUnitId ?? slaveForm.value.unitId)
    slaveUnitForms.value = (source.slaveUnitForms?.length ? source.slaveUnitForms : [slaveForm.value])
      .map(unit => ({ ...defaultModbusSlaveForm(), ...unit, unitId: normalizeUnitId(unit.unitId) }))
    const selected = slaveUnitForms.value.find(unit => unit.unitId === activeSlaveUnitId.value) ?? slaveUnitForms.value[0]
    activeSlaveUnitId.value = selected.unitId
    slaveForm.value = { ...selected }
    masterGrid.value = normalizeMasterGrid(source.masterGrid ?? masterGridFromRegisterReadForm(source.registerReadForm))
    masterMappings.value = normalizeMappingRows(
      source.masterMappings?.length ? source.masterMappings : registerMappingTextToRows(source.registerReadForm?.mappingText ?? '')
    )
    masterRegisterTables.value = normalizeMasterRegisterTables(source.masterRegisterTables, masterGrid.value, masterMappings.value)
    slaveUnitGrids.value = (source.slaveUnitGrids?.length ? source.slaveUnitGrids : slaveUnitForms.value.map(slaveFormToGrid))
      .map(normalizeSlaveGrid)
      .sort((a, b) => a.unitId - b.unitId)
    if (!slaveUnitGrids.value.some(unit => unit.unitId === activeSlaveUnitId.value)) {
      slaveUnitGrids.value = [...slaveUnitGrids.value, slaveFormToGrid(slaveForm.value)].sort((a, b) => a.unitId - b.unitId)
    }
    registerReadForm.value = { ...defaultModbusRegisterReadForm(), ...source.registerReadForm, polling: false }
    unitScanForm.value = { ...defaultModbusUnitScanForm(), ...source.unitScanForm }
    registerScanForm.value = { ...defaultModbusRegisterScanForm(), ...source.registerScanForm }
    registerReadResult.value = source.registerReadResult ?? null
    unitScanResult.value = source.unitScanResult ?? null
    registerScanResult.value = source.registerScanResult ?? null
    history.value = source.history ?? []
    error.value = null
  }

  function cleanup() {
    stopRegisterPolling()
    // Backend sessions are owned by the serial service and released on app shutdown.
  }

  return {
    activeSessionId,
    sessions,
    portForm,
    masterForm,
    slaveForm,
    activeSlaveUnitId,
    slaveUnitForms,
    masterGrid,
    masterMappings,
    masterRegisterTables,
    slaveUnitGrids,
    registerReadForm,
    unitScanForm,
    registerScanForm,
    registerReadResult,
    unitScanResult,
    registerScanResult,
    history,
    error,
    sessionList,
    activeSession,
    activeSlaveForm,
    activeSlaveGrid,
    refreshSessions,
    openSession,
    closeSession,
    sendMasterRequest,
    startSlave,
    stopSlave,
    applySlaveData,
    selectSlaveUnit,
    addSlaveUnit,
    removeSlaveUnit,
    readRegisters,
    scanUnitIDs,
    scanRegisters,
    startRegisterPolling,
    stopRegisterPolling,
    buildRegisterReadRequest,
    buildUnitScanRequest,
    buildRegisterScanRequest,
    slaveDataModel,
    setActiveSession,
    clearError,
    resetWorkspace,
    exportState,
    restoreState,
    cleanup,
  }
})

export function defaultSessionId(): string {
  return `modbus-${Date.now().toString(36)}`
}

export function parseBoolValues(input: string): boolean[] {
  return input.split(/[\s,]+/)
    .map(part => part.trim())
    .filter(Boolean)
    .map(part => part === '1' || part.toLowerCase() === 'true' || part.toLowerCase() === 'on')
}

export function parseRegisterValues(input: string): number[] {
  return input.split(/[\s,]+/)
    .map(part => part.trim())
    .filter(Boolean)
    .map(parseNumber)
}

export function parseBoolPoints(input: string): { Address: number; Value: boolean }[] {
  return parsePointLines(input).map(({ address, value }) => ({
    Address: address,
    Value: value === '1' || value.toLowerCase() === 'true' || value.toLowerCase() === 'on',
  }))
}

export function parseRegisterPoints(input: string): { Address: number; Value: number }[] {
  return parsePointLines(input).map(({ address, value }) => ({
    Address: address,
    Value: parseNumber(value),
  }))
}

export function parseUnitIds(input: string): number[] {
  const ids = new Set<number>()
  input.split(/[\s,]+/)
    .map(part => part.trim())
    .filter(Boolean)
    .forEach(part => {
      const range = part.split('-').map(value => normalizeUnitId(parseNumber(value)))
      if (range.length === 2 && range[1] >= range[0]) {
        for (let id = range[0]; id <= range[1]; id += 1) {
          ids.add(id)
        }
        return
      }
      ids.add(range[0])
    })
  return Array.from(ids).sort((a, b) => a - b)
}

export function parseRegisterMappings(input: string): RegisterMapping[] {
  return input.split(/\n+/)
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [addressText, dataTypeText = 'uint16', wordOrderText = 'big', lengthText = '0', scalingText = '0'] = line.split(/[\s,:]+/)
      const dataType = normalizeDataType(dataTypeText)
      const wordOrder = normalizeWordOrder(wordOrderText)
      return {
        Address: parseNumber(addressText),
        DataType: dataType,
        WordOrder: wordOrder,
        Length: Math.max(0, parseNumber(lengthText) || 0),
        ScalingFactor: Number.isFinite(Number(scalingText)) ? Number(scalingText) : 0,
        Comment: '',
        Interpolate: null,
        GroupEnd: false,
      } as RegisterMapping
    })
}

export function registerMappingRowsToMappings(rows: ModbusRegisterMappingRow[]): RegisterMapping[] {
  return normalizeMappingRows(rows).map(row => ({
    Address: Math.max(0, Math.trunc(row.address)),
    DataType: normalizeDataType(row.dataType),
    WordOrder: normalizeWordOrder(row.wordOrder),
    Length: Math.max(0, Math.trunc(row.length || 0)),
    ScalingFactor: Number.isFinite(row.scalingFactor) ? row.scalingFactor : 0,
    Comment: row.comment ?? '',
    Interpolate: null,
    GroupEnd: Boolean(row.groupEnd),
  }))
}

export function slaveGridToDataModel(grid: ModbusSlaveUnitGridState): DataModelSnapshot {
  const normalized = normalizeSlaveGrid(grid)
  return {
    Coils: normalized.coils.map(row => ({ Address: row.address, Value: row.value })),
    DiscreteInputs: normalized.discreteInputs.map(row => ({ Address: row.address, Value: row.value })),
    InputRegisters: normalized.inputRegisters.map(row => ({ Address: row.address, Value: row.value })),
    HoldingRegisters: normalized.holdingRegisters.map(row => ({ Address: row.address, Value: row.value })),
  }
}

export function registerTypeToFunction(type: ModbusRegisterType): number {
  switch (type) {
    case 'coils':
      return 1
    case 'discrete_inputs':
      return 2
    case 'input_registers':
      return 4
    case 'holding_registers':
    default:
      return 3
  }
}

function parsePointLines(input: string): Array<{ address: number; value: string }> {
  return input.split(/\n+/)
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [address, value] = line.split(/[=:]/).map(part => part.trim())
      return { address: parseNumber(address), value: value ?? '0' }
    })
}

function masterGridFromRegisterReadForm(form?: Partial<ModbusRegisterReadFormState>): ModbusMasterGridState {
  const defaults = defaultModbusMasterGrid()
  const functionCode = form?.functionCode ?? 3
  return {
    ...defaults,
    unitId: normalizeUnitId(form?.unitId ?? defaults.unitId),
    registerType: functionToRegisterType(functionCode),
    address: Math.max(0, Math.trunc(form?.address ?? defaults.address)),
    length: Math.max(1, Math.trunc(form?.quantity ?? defaults.length)),
    addressBase: form?.addressMode === 'plc' ? 1 : 0,
    pollRateMs: Math.max(10, Math.trunc(form?.pollIntervalMs ?? defaults.pollRateMs)),
    timeoutMs: Math.max(10, Math.trunc(form?.timeoutMs ?? defaults.timeoutMs)),
    retries: Math.max(0, Math.trunc(form?.retries ?? defaults.retries)),
  }
}

function normalizeMasterGrid(grid?: Partial<ModbusMasterGridState>): ModbusMasterGridState {
  const defaults = defaultModbusMasterGrid()
  return {
    ...defaults,
    ...grid,
    unitId: normalizeUnitId(grid?.unitId ?? defaults.unitId),
    registerType: normalizeRegisterType(grid?.registerType ?? defaults.registerType),
    address: Math.max(0, Math.trunc(grid?.address ?? defaults.address)),
    length: Math.max(1, Math.trunc(grid?.length ?? defaults.length)),
    addressBase: grid?.addressBase === 1 ? 1 : 0,
    pollRateMs: Math.max(10, Math.trunc(grid?.pollRateMs ?? defaults.pollRateMs)),
    timeoutMs: Math.max(10, Math.trunc(grid?.timeoutMs ?? defaults.timeoutMs)),
    retries: Math.max(0, Math.trunc(grid?.retries ?? defaults.retries)),
  }
}

function masterTableFromGrid(
  type: ModbusRegisterType,
  grid: Partial<ModbusMasterGridState>,
  mappings: ModbusRegisterMappingRow[]
): ModbusMasterRegisterTableState {
  const normalized = normalizeMasterGrid({ ...grid, registerType: type })
  return {
    type,
    unitId: normalized.unitId,
    address: normalized.address,
    length: normalized.length,
    addressBase: normalized.addressBase,
    pollRateMs: normalized.pollRateMs,
    timeoutMs: normalized.timeoutMs,
    retries: normalized.retries,
    rows: defaultMasterRows(type, normalized.length, normalized.address),
    mappings: normalizeMappingRows(mappings),
  }
}

function normalizeMasterRegisterTables(
  tables: ModbusMasterRegisterTableState[] | undefined,
  legacyGrid: ModbusMasterGridState,
  legacyMappings: ModbusRegisterMappingRow[]
): ModbusMasterRegisterTableState[] {
  const source = tables?.length
    ? tables
    : [masterTableFromGrid(legacyGrid.registerType, legacyGrid, legacyMappings)]
  const byType = new Map<ModbusRegisterType, ModbusMasterRegisterTableState>()
  for (const table of source) {
    const type = normalizeRegisterType(table.type)
    byType.set(type, normalizeMasterRegisterTable({ ...table, type }))
  }
  return registerTypes.map(type => byType.get(type) ?? normalizeMasterRegisterTable(defaultModbusRegisterTables().find(table => table.type === type)!))
}

function normalizeMasterRegisterTable(table: ModbusMasterRegisterTableState): ModbusMasterRegisterTableState {
  const length = Math.max(1, Math.trunc(table.length ?? 1))
  const address = Math.max(0, Math.trunc(table.address ?? 0))
  return {
    type: normalizeRegisterType(table.type),
    unitId: normalizeUnitId(table.unitId ?? 1),
    address,
    length,
    addressBase: table.addressBase === 1 ? 1 : 0,
    pollRateMs: Math.max(10, Math.trunc(table.pollRateMs ?? 1000)),
    timeoutMs: Math.max(10, Math.trunc(table.timeoutMs ?? 800)),
    retries: Math.max(0, Math.trunc(table.retries ?? 0)),
    rows: normalizeMasterRows(table.rows?.length ? table.rows : defaultMasterRows(table.type, length, address)),
    mappings: normalizeMappingRows(table.mappings ?? [], true),
  }
}

function defaultMasterRows(type: ModbusRegisterType, length: number, startAddress = 0): ModbusMasterTableRow[] {
  const count = Math.max(1, Math.trunc(length || 1))
  return Array.from({ length: count }, (_, index) => ({
    id: `${type}-${startAddress + index}`,
    address: startAddress + index,
    value: '0',
    raw: '',
    display: '',
    mappingId: '',
  }))
}

function normalizeMasterRows(rows: ModbusMasterTableRow[]): ModbusMasterTableRow[] {
  return (rows ?? []).map((row, index) => ({
    id: row.id || `master-row-${row.address}-${index}`,
    address: Math.max(0, Math.trunc(row.address ?? index)),
    value: row.value || '0',
    raw: row.raw ?? '',
    display: row.display ?? '',
    mappingId: row.mappingId ?? '',
  })).sort((a, b) => a.address - b.address)
}

function functionToRegisterType(functionCode: number): ModbusRegisterType {
  switch (functionCode) {
    case 1:
      return 'coils'
    case 2:
      return 'discrete_inputs'
    case 4:
      return 'input_registers'
    case 3:
    default:
      return 'holding_registers'
  }
}

function normalizeRegisterType(value: string): ModbusRegisterType {
  if (value === 'coils' || value === 'discrete_inputs' || value === 'input_registers' || value === 'holding_registers') {
    return value
  }
  return 'holding_registers'
}

function registerMappingTextToRows(input: string): ModbusRegisterMappingRow[] {
  return parseRegisterMappings(input).map((mapping, index) => ({
    id: `map-${mapping.Address}-${index}`,
    address: mapping.Address,
    dataType: mapping.DataType as ModbusDataType,
    wordOrder: mapping.WordOrder as ModbusWordOrder,
    length: mapping.Length,
    scalingFactor: mapping.ScalingFactor,
    comment: mapping.Comment,
    groupEnd: mapping.GroupEnd,
  }))
}

function normalizeMappingRows(rows: ModbusRegisterMappingRow[], allowEmpty = false): ModbusRegisterMappingRow[] {
  const source = rows.length || allowEmpty ? rows : defaultModbusMasterMappings()
  return source.map((row, index) => ({
    id: row.id || `map-${row.address}-${index}`,
    address: Math.max(0, Math.trunc(row.address ?? 0)),
    dataType: (normalizeDataType(row.dataType) as ModbusDataType),
    wordOrder: normalizeWordOrder(row.wordOrder) as ModbusWordOrder,
    length: Math.max(0, Math.trunc(row.length ?? 0)),
    scalingFactor: Number.isFinite(row.scalingFactor) ? row.scalingFactor : 0,
    comment: row.comment ?? '',
    groupEnd: Boolean(row.groupEnd),
  }))
}

function slaveFormToGrid(form: ModbusSlaveFormState): ModbusSlaveUnitGridState {
  const unitId = normalizeUnitId(form.unitId)
  return {
    unitId,
    coils: parseBoolPoints(form.coils).map((point, index) => ({
      id: `u${unitId}-coil-${point.Address}-${index}`,
      address: point.Address,
      value: point.Value,
    })),
    discreteInputs: parseBoolPoints(form.discreteInputs).map((point, index) => ({
      id: `u${unitId}-di-${point.Address}-${index}`,
      address: point.Address,
      value: point.Value,
    })),
    inputRegisters: parseRegisterPoints(form.inputRegisters).map((point, index) => ({
      id: `u${unitId}-ir-${point.Address}-${index}`,
      address: point.Address,
      value: point.Value,
      dataType: 'uint16',
      comment: '',
    })),
    holdingRegisters: parseRegisterPoints(form.holdingRegisters).map((point, index) => ({
      id: `u${unitId}-hr-${point.Address}-${index}`,
      address: point.Address,
      value: point.Value,
      dataType: 'uint16',
      comment: '',
    })),
  }
}

function gridToSlaveForm(grid: ModbusSlaveUnitGridState): ModbusSlaveFormState {
  const normalized = normalizeSlaveGrid(grid)
  return {
    unitId: normalized.unitId,
    coils: normalized.coils.map(row => `${row.address}=${row.value ? 1 : 0}`).join('\n'),
    discreteInputs: normalized.discreteInputs.map(row => `${row.address}=${row.value ? 1 : 0}`).join('\n'),
    inputRegisters: normalized.inputRegisters.map(row => `${row.address}=${row.value}`).join('\n'),
    holdingRegisters: normalized.holdingRegisters.map(row => `${row.address}=${row.value}`).join('\n'),
  }
}

function normalizeSlaveGrid(grid: ModbusSlaveUnitGridState): ModbusSlaveUnitGridState {
  const unitId = normalizeUnitId(grid.unitId)
  return {
    unitId,
    coils: normalizeBoolRows(grid.coils, `u${unitId}-coil`),
    discreteInputs: normalizeBoolRows(grid.discreteInputs, `u${unitId}-di`),
    inputRegisters: normalizeRegisterRows(grid.inputRegisters, `u${unitId}-ir`),
    holdingRegisters: normalizeRegisterRows(grid.holdingRegisters, `u${unitId}-hr`),
  }
}

function normalizeBoolRows(rows: ModbusSlaveBoolRow[], prefix: string): ModbusSlaveBoolRow[] {
  return (rows ?? []).map((row, index) => ({
    id: row.id || `${prefix}-${row.address}-${index}`,
    address: Math.max(0, Math.trunc(row.address ?? 0)),
    value: Boolean(row.value),
  })).sort((a, b) => a.address - b.address)
}

function normalizeRegisterRows(rows: ModbusSlaveRegisterRow[], prefix: string): ModbusSlaveRegisterRow[] {
  return (rows ?? []).map((row, index) => ({
    id: row.id || `${prefix}-${row.address}-${index}`,
    address: Math.max(0, Math.trunc(row.address ?? 0)),
    value: Math.max(0, Math.trunc(row.value ?? 0)),
    dataType: normalizeDataType(row.dataType) as ModbusDataType,
    comment: row.comment ?? '',
  })).sort((a, b) => a.address - b.address)
}

function parseNumber(input: string): number {
  if (/^0x/i.test(input)) {
    return Number.parseInt(input, 16)
  }
  return Number.parseInt(input, 10)
}

function normalizeUnitId(value: number): number {
  if (!Number.isFinite(value)) return 1
  return Math.min(247, Math.max(1, Math.trunc(value)))
}

function normalizeDataType(value: string): DataType {
  const normalized = value.toLowerCase() as DataType
  if (Object.values(DataType).includes(normalized)) {
    return normalized
  }
  return DataType.DataTypeUint16
}

function normalizeWordOrder(value: string): WordOrder {
  const normalized = value.toLowerCase() as WordOrder
  if (normalized === WordOrder.WordOrderLittle) {
    return WordOrder.WordOrderLittle
  }
  return WordOrder.WordOrderBig
}

function normalizeSessionInfo(session: SessionInfo): SessionInfo {
  return {
    ...session,
    Role: session.Role === SessionRole.SessionRoleSlave ? SessionRole.SessionRoleSlave : SessionRole.SessionRoleMaster,
  }
}
