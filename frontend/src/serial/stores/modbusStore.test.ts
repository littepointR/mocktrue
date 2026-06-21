import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  defaultModbusMasterGrid,
  defaultModbusMasterUnitGrid,
  defaultModbusRegisterTables,
  parseBoolPoints,
  parseBoolValues,
  parseRegisterMappings,
  parseRegisterPoints,
  parseRegisterValues,
  parseUnitIds,
  registerMappingRowsToMappings,
  slaveGridToDataModel,
  useModbusStore,
} from './modbusStore'
import { DataType, WordOrder } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/modbus/models.js'
import { FrameMode } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/modbus/models.js'

const serialServiceMock = vi.hoisted(() => ({
  listModbusSessions: vi.fn(async () => []),
  openModbusSession: vi.fn(async (request: any) => sampleSession(request.ID ?? 'modbus-1', request.Role ?? 'master')),
  closeModbusSession: vi.fn(async () => undefined),
  modbusMasterRequest: vi.fn(async () => null),
  modbusReadRegisters: vi.fn(async () => ({
    Transaction: null,
    RawRegisters: [],
    Bits: [],
    Values: [],
  })),
  modbusScanUnitIDs: vi.fn(async () => ({
    SessionID: 'modbus-1',
    ActiveUnitIDs: [],
    Results: [],
  })),
  modbusScanRegisters: vi.fn(async () => ({
    SessionID: 'modbus-1',
    UnitID: 1,
    Values: [],
    Ranges: [],
  })),
  startModbusSlave: vi.fn(async (request: any) => sampleSession(request.SessionID ?? 'modbus-1')),
  stopModbusSlave: vi.fn(async () => undefined),
  updateModbusSlaveUnitData: vi.fn(async () => undefined),
  addModbusSlaveUnit: vi.fn(async () => undefined),
  removeModbusSlaveUnit: vi.fn(async () => undefined),
}))

vi.mock('../services/serialService', () => ({
  serialService: serialServiceMock,
}))

describe('modbusStore helpers', () => {
  it('parses master values', () => {
    expect(parseBoolValues('1 0 true off')).toEqual([true, false, true, false])
    expect(parseRegisterValues('24 0x2a')).toEqual([24, 42])
  })

  it('parses slave table text', () => {
    expect(parseBoolPoints('0=1\n1:0')).toEqual([
      { Address: 0, Value: true },
      { Address: 1, Value: false },
    ])
    expect(parseRegisterPoints('0=24\n1=0x2a')).toEqual([
      { Address: 0, Value: 24 },
      { Address: 1, Value: 42 },
    ])
  })

  it('parses unit scan ranges and register mappings', () => {
    expect(parseUnitIds('1-3, 7 9')).toEqual([1, 2, 3, 7, 9])
    expect(parseRegisterMappings('0:int32:little\n2 float big 0 0.1')).toMatchObject([
      { Address: 0, DataType: DataType.DataTypeInt32, WordOrder: WordOrder.WordOrderLittle },
      { Address: 2, DataType: DataType.DataTypeFloat, WordOrder: WordOrder.WordOrderBig, ScalingFactor: 0.1 },
    ])
  })

  it('converts master mapping rows to backend register mappings', () => {
    expect(registerMappingRowsToMappings([
      {
        id: 'map-1',
        address: 0,
        dataType: 'int32',
        wordOrder: 'little',
        length: 0,
        scalingFactor: 0.1,
        comment: 'temperature',
        groupEnd: true,
      },
      {
        id: 'map-2',
        address: 2,
        dataType: 'float',
        wordOrder: 'big',
        length: 0,
        scalingFactor: 0,
        comment: 'pressure',
        groupEnd: false,
      },
    ])).toEqual([
      {
        Address: 0,
        DataType: DataType.DataTypeInt32,
        WordOrder: WordOrder.WordOrderLittle,
        Length: 0,
        ScalingFactor: 0.1,
        Comment: 'temperature',
        Interpolate: null,
        GroupEnd: true,
      },
      {
        Address: 2,
        DataType: DataType.DataTypeFloat,
        WordOrder: WordOrder.WordOrderBig,
        Length: 0,
        ScalingFactor: 0,
        Comment: 'pressure',
        Interpolate: null,
        GroupEnd: false,
      },
    ])
  })

  it('creates a default table for every Modbus register type', () => {
    const tables = defaultModbusRegisterTables()

    expect(tables.map(table => table.type)).toEqual([
      'coils',
      'discrete_inputs',
      'input_registers',
      'holding_registers',
    ])
    expect(tables.every(table => table.rows.length > 0)).toBe(true)
    expect(tables.every(table => table.rows.every(row => row.value === '0'))).toBe(true)
    expect(tables.find(table => table.type === 'input_registers')?.mappings[0]).toMatchObject({
      address: 0,
      dataType: 'int32',
      wordOrder: 'big',
    })
    expect(tables.find(table => table.type === 'holding_registers')?.mappings[0]).toMatchObject({
      address: 0,
      dataType: 'int32',
      wordOrder: 'big',
    })
  })

  it('converts slave unit grid rows to backend data model snapshots', () => {
    expect(slaveGridToDataModel({
      unitId: 2,
      coils: [
        { id: 'coil-0', address: 0, value: true },
        { id: 'coil-1', address: 1, value: false },
      ],
      discreteInputs: [{ id: 'input-0', address: 0, value: true }],
      inputRegisters: [{ id: 'ir-0', address: 0, value: 24, dataType: 'uint16', comment: 'temperature' }],
      holdingRegisters: [{ id: 'hr-0', address: 10, value: 42, dataType: 'uint16', comment: 'setpoint' }],
    })).toEqual({
      Coils: [
        { Address: 0, Value: true },
        { Address: 1, Value: false },
      ],
      DiscreteInputs: [{ Address: 0, Value: true }],
      InputRegisters: [{ Address: 0, Value: 24 }],
      HoldingRegisters: [{ Address: 10, Value: 42 }],
    })
  })
})

describe('modbusStore workspace state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('normalizes old empty master row values to zero', () => {
    const store = useModbusStore()
    const snapshot = store.exportState()
    snapshot.masterRegisterTables = snapshot.masterRegisterTables?.map(table => ({
      ...table,
      rows: table.rows.map(row => ({ ...row, value: '', display: '' })),
    }))

    store.restoreState(snapshot)

    expect(store.masterRegisterTables.every(table => table.rows.every(row => row.value === '0'))).toBe(true)
  })

  it('exports and restores form and history state', () => {
    const store = useModbusStore()
    store.portForm.port = '/tmp/ttyM0'
    store.portForm.mode = 'ascii'
    store.portForm.role = 'slave'
    store.masterForm.addressMode = 'plc'
    store.masterForm.address = 40001
    store.slaveUnitForms = [
      { ...store.slaveForm, unitId: 1, holdingRegisters: '0=24' },
      { ...store.slaveForm, unitId: 2, holdingRegisters: '0=42' },
    ]
    store.activeSlaveUnitId = 2
    store.slaveForm = { ...store.slaveUnitForms[1] }
    store.registerReadForm.mappingText = '0:uint16'
    store.masterGrid = {
      ...defaultModbusMasterGrid(),
      unitId: 2,
      registerType: 'input_registers',
      address: 12,
      length: 8,
      addressBase: 1,
      readConfigured: true,
      littleEndian: true,
      rawVisible: true,
      logVisible: true,
    }
    store.masterMappings = [{
      id: 'map-1',
      address: 12,
      dataType: 'float',
      wordOrder: 'little',
      length: 0,
      scalingFactor: 1,
      comment: 'demo value',
      groupEnd: false,
    }]
    store.masterRegisterTables = defaultModbusRegisterTables().map(table => (
      table.type === 'input_registers'
        ? {
            ...table,
            unitId: 2,
            address: 12,
            length: 8,
            addressBase: 1,
            mappings: [{
              id: 'table-map-1',
              address: 12,
              dataType: 'float',
              wordOrder: 'little',
              length: 0,
              scalingFactor: 1,
              comment: 'table demo value',
              groupEnd: false,
            }],
          }
        : table
    ))
    store.masterUnitGrids = [{
      ...defaultModbusMasterUnitGrid(2),
      registerType: 'input_registers',
      littleEndian: true,
      rawVisible: true,
      logVisible: true,
      registerTables: store.masterRegisterTables,
      registerReadResult: {
        Transaction: null,
        RawRegisters: [24],
        Bits: [],
        Values: [],
      },
      unitScanResult: {
        SessionID: 'session-1',
        ActiveUnitIDs: [1, 2],
        Results: [],
      },
      registerScanResult: {
        SessionID: 'session-1',
        UnitID: 1,
        Values: [{ Address: 0, Value: 24 }],
        Ranges: [],
      },
    }]
    store.slaveUnitGrids = [{
      unitId: 2,
      coils: [{ id: 'coil-0', address: 0, value: true }],
      discreteInputs: [],
      inputRegisters: [{ id: 'ir-0', address: 0, value: 24, dataType: 'uint16', comment: '' }],
      holdingRegisters: [{ id: 'hr-0', address: 0, value: 42, dataType: 'uint16', comment: '' }],
    }]
    store.unitScanForm.unitIds = '1-4'
    store.registerScanForm.endAddress = 16
    store.registerReadResult = {
      Transaction: null,
      RawRegisters: [24],
      Bits: [],
      Values: [],
    }
    store.unitScanResult = {
      SessionID: 'session-1',
      ActiveUnitIDs: [1, 2],
      Results: [],
    }
    store.registerScanResult = {
      SessionID: 'session-1',
      UnitID: 1,
      Values: [{ Address: 0, Value: 24 }],
      Ranges: [],
    }
    store.history = [{
      ID: 'tx-1',
      SessionID: 'session-1',
      StartedAt: '',
      CompletedAt: '',
      UnitID: 1,
      Mode: FrameMode.FrameModeRTU,
      RequestPDU: { Function: 3, Data: '' },
      ResponsePDU: { Function: 3, Data: '' },
      RequestFrameHex: '01 03 00 00 00 02 c4 0b',
      ResponseFrameHex: '01 03 04 00 18 00 2a fb eb',
      BytesWritten: 8,
      Response: {
        Function: 3,
        Exception: false,
        ExceptionCode: 0,
        Address: 0,
        Quantity: 0,
        Value: 0,
        Values: [24, 42],
        Bits: [],
        Raw: '',
      },
      Error: '',
    }]

    const snapshot = store.exportState()
    store.resetWorkspace()
    store.restoreState(snapshot)

    expect(store.portForm.port).toBe('/tmp/ttyM0')
    expect(store.portForm.mode).toBe('ascii')
    expect(store.portForm.role).toBe('slave')
    expect(store.masterForm.addressMode).toBe('plc')
    expect(store.slaveUnitForms.map(unit => unit.unitId)).toEqual([1, 2])
    expect(store.activeSlaveUnitId).toBe(2)
    expect(store.registerReadForm.mappingText).toBe('0:uint16')
    expect(store.masterGrid.registerType).toBe('input_registers')
    expect(store.masterGrid.addressBase).toBe(1)
    expect(store.masterGrid.littleEndian).toBe(true)
    expect(store.masterMappings[0].comment).toBe('demo value')
    expect(store.masterRegisterTables.find(table => table.type === 'input_registers')?.mappings[0].comment).toBe('table demo value')
    expect(store.masterUnitGrids[0].unitId).toBe(2)
    expect(store.masterUnitGrids[0].registerTables.find(table => table.type === 'input_registers')?.mappings[0].comment).toBe('table demo value')
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 2)?.holdingRegisters[0].value).toBe(42)
    expect(store.unitScanForm.unitIds).toBe('1-4')
    expect(store.registerScanForm.endAddress).toBe(16)
    expect(store.registerReadResult?.RawRegisters).toEqual([24])
    expect(store.unitScanResult?.ActiveUnitIDs).toEqual([1, 2])
    expect(store.registerScanResult?.Values).toEqual([{ Address: 0, Value: 24 }])
    expect(store.history[0].Response.Values).toEqual([24, 42])
  })

  it('keeps master register tables isolated per Unit ID', () => {
    const store = useModbusStore()

    store.addMasterUnit(2)
    store.selectMasterUnit(1)
    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.rows[0].value = '11'
    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.mappings[0].comment = 'unit 1'

    store.selectMasterUnit(2)
    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.rows[0].value = '22'
    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.mappings[0].comment = 'unit 2'

    store.selectMasterUnit(1)
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0].value).toBe('11')
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.mappings[0].comment).toBe('unit 1')
    store.selectMasterUnit(2)
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0].value).toBe('22')
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.mappings[0].comment).toBe('unit 2')
  })

  it('rejects duplicate and out-of-range Unit IDs', () => {
    const store = useModbusStore()

    expect(store.addMasterUnit(2)).toBe(true)
    expect(store.addMasterUnit(2)).toBe(false)
    expect(store.error).toContain('Unit 2 已存在')
    expect(store.addMasterUnit(0)).toBe(false)
    expect(store.error).toContain('Unit ID 必须在 1-247')
    expect(store.addSlaveUnit(300)).resolves.toBe(false)
  })

  it('defaults old snapshots to master role', () => {
    const store = useModbusStore()
    const snapshot = store.exportState()
    delete (snapshot.portForm as any).role

    store.restoreState(snapshot)

    expect(store.portForm.role).toBe('master')
  })

  it('exports workspace state without mutating slave unit forms', () => {
    const store = useModbusStore()
    store.slaveForm = { ...store.slaveForm, unitId: 2, holdingRegisters: '0=42' }
    store.activeSlaveUnitId = 2
    store.slaveUnitForms = [{ ...store.slaveForm, unitId: 1, holdingRegisters: '0=24' }]

    const before = JSON.stringify({
      activeSlaveUnitId: store.activeSlaveUnitId,
      slaveForm: store.slaveForm,
      slaveUnitForms: store.slaveUnitForms,
    })

    const snapshot = store.exportState()

    expect(snapshot.slaveUnitForms.map(unit => unit.unitId)).toEqual([1, 2])
    expect(JSON.stringify({
      activeSlaveUnitId: store.activeSlaveUnitId,
      slaveForm: store.slaveForm,
      slaveUnitForms: store.slaveUnitForms,
    })).toBe(before)
  })

  it('builds register read requests from the master workbench state', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-1',
      sessions: [{
        ID: 'modbus-1',
        Name: 'Modbus 主站',
        Mode: FrameMode.FrameModeRTU,
        Role: 'master' as any,
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
    store.masterGrid = {
      ...defaultModbusMasterGrid(),
      unitId: 7,
      registerType: 'input_registers',
      address: 30001,
      length: 4,
      addressBase: 1,
      timeoutMs: 500,
      retries: 2,
    }
    store.masterMappings = [{
      id: 'map-1',
      address: 30001,
      dataType: 'float',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: 'flow',
      groupEnd: false,
    }]
    const inputTable = store.masterRegisterTables.find(table => table.type === 'input_registers')!
    inputTable.unitId = 7
    inputTable.address = 30001
    inputTable.length = 4
    inputTable.addressBase = 1
    inputTable.timeoutMs = 500
    inputTable.retries = 2
    inputTable.mappings = [{
      id: 'map-1',
      address: 30001,
      dataType: 'float',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: 'flow',
      groupEnd: false,
    }]

    expect(store.buildRegisterReadRequest()).toMatchObject({
      SessionID: 'modbus-1',
      UnitID: 7,
      Function: 4,
      AddressMode: 'plc',
      Address: 30001,
      Quantity: 4,
      TimeoutMs: 500,
      Retries: 2,
      Mappings: [{ Address: 30001, DataType: DataType.DataTypeFloat, Comment: 'flow' }],
    })
  })

  it('migrates old single-grid snapshots into the matching master register table', () => {
    const store = useModbusStore()
    const snapshot = store.exportState()
    snapshot.masterGrid = {
      ...defaultModbusMasterGrid(),
      registerType: 'input_registers',
      unitId: 3,
      address: 10,
      length: 6,
      addressBase: 1,
      timeoutMs: 450,
      retries: 2,
    }
    snapshot.masterMappings = [{
      id: 'legacy-map',
      address: 10,
      dataType: 'float',
      wordOrder: 'little',
      length: 0,
      scalingFactor: 0,
      comment: 'legacy',
      groupEnd: false,
    }]
    delete (snapshot as any).masterRegisterTables
    delete (snapshot as any).masterUnitGrids

    store.restoreState(snapshot)

    expect(store.masterUnitGrids[0].unitId).toBe(3)
    const table = store.masterRegisterTables.find(item => item.type === 'input_registers')
    expect(table).toMatchObject({
      unitId: 3,
      address: 10,
      length: 6,
      addressBase: 1,
      timeoutMs: 450,
      retries: 2,
    })
    expect(table?.mappings[0]).toMatchObject({ address: 10, comment: 'legacy' })
    expect(store.masterGrid.registerType).toBe('input_registers')
  })

  it('uses the active master Unit ID when building register requests', () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'master')

    store.addMasterUnit(9)
    store.selectMasterUnit(9)
    const holding = store.masterRegisterTables.find(table => table.type === 'holding_registers')!
    holding.address = 20
    holding.length = 2

    expect(store.buildRegisterReadRequest()).toMatchObject({
      SessionID: 'modbus-1',
      UnitID: 9,
      Function: 3,
      Address: 20,
      Quantity: 2,
    })
  })

  it('starts and applies only the active slave Unit ID', async () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'slave')
    await store.addSlaveUnit(2)
    store.selectSlaveUnit(2)
    store.slaveUnitGrids.find(unit => unit.unitId === 2)!.holdingRegisters[0].value = 77
    serialServiceMock.startModbusSlave.mockClear()
    serialServiceMock.updateModbusSlaveUnitData.mockClear()

    await store.startSlave()
    await store.applySlaveData()

    expect(serialServiceMock.startModbusSlave).toHaveBeenCalledWith(expect.objectContaining({
      UnitID: 2,
      Units: [{
        UnitID: 2,
        DataModel: expect.objectContaining({
          HoldingRegisters: expect.arrayContaining([expect.objectContaining({ Address: 0, Value: 77 })]),
        }),
      }],
    }))
    expect(serialServiceMock.updateModbusSlaveUnitData).toHaveBeenCalledWith(
      'modbus-1',
      2,
      expect.objectContaining({
        HoldingRegisters: expect.arrayContaining([expect.objectContaining({ Address: 0, Value: 77 })]),
      })
    )
  })

  it('builds scan requests from the current master workbench state', () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'master')
    store.masterGrid = {
      ...defaultModbusMasterGrid(),
      unitId: 9,
      registerType: 'discrete_inputs',
      address: 10001,
      length: 16,
      addressBase: 1,
      timeoutMs: 250,
      retries: 1,
    }
    store.unitScanForm.unitIds = '7-8'
    store.registerScanForm.startAddress = 10001
    store.registerScanForm.endAddress = 10032
    store.registerScanForm.chunkSize = 8

    expect(store.buildUnitScanRequest()).toMatchObject({
      SessionID: 'modbus-1',
      UnitIDs: [7, 8],
      Function: 2,
      AddressMode: 'plc',
      Address: 10001,
      Quantity: 16,
      TimeoutMs: 250,
      Retries: 1,
    })
    expect(store.buildRegisterScanRequest()).toMatchObject({
      SessionID: 'modbus-1',
      UnitID: 9,
      Function: 2,
      AddressMode: 'plc',
      StartAddress: 10001,
      EndAddress: 10032,
      ChunkSize: 8,
      TimeoutMs: 250,
      Retries: 1,
    })
  })
})

function openRealSession(store: ReturnType<typeof useModbusStore>, id: string, role: 'master' | 'slave') {
  store.sessions.set(id, sampleSession(id, role))
  store.setActiveSession(id)
}

function sampleSession(id: string, role: 'master' | 'slave' = 'slave') {
  return {
    ID: id,
    Name: 'Modbus',
    Mode: FrameMode.FrameModeRTU,
    Role: role as any,
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
  }
}
