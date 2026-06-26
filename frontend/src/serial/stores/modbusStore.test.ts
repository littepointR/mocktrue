import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  defaultModbusMasterGrid,
  defaultModbusMasterUnitGrid,
  defaultModbusRegisterTables,
  defaultModbusSlaveUnitGrid,
  dataModelToSlaveGrid,
  parseBoolPoints,
  parseBoolValues,
  parseRegisterMappings,
  parseRegisterPoints,
  parseRegisterValues,
  parseUnitIds,
  registerMappingRowsToMappings,
  registerTypeToFunction,
  slaveGridToDataModel,
  useModbusStore,
} from './modbusStore'
import { DataType, WordOrder } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/modbus/models.js'
import type { SessionInfo, SlaveUnitInfo } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/modbus/models.js'
import { FrameMode } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/modbus/models.js'

const serialServiceMock = vi.hoisted(() => ({
  listModbusSessions: vi.fn(async (): Promise<SessionInfo[]> => []),
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
  listModbusSlaveUnits: vi.fn(async (): Promise<SlaveUnitInfo[]> => []),
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

  it('normalizes backend data models, function codes, and malformed helper input', () => {
    expect(parseUnitIds('3-1, 0, 248, bad')).toEqual([1, 3, 247])
    expect(parseBoolValues('yes off true 0')).toEqual([false, false, true, false])
    expect(parseBoolPoints('5')).toEqual([{ Address: 5, Value: false }])
    expect(parseRegisterMappings('5:not-a-type:sideways:-1:nope')).toEqual([
      {
        Address: 5,
        DataType: DataType.DataTypeUint16,
        WordOrder: WordOrder.WordOrderBig,
        Length: 0,
        ScalingFactor: 0,
        Comment: '',
        Interpolate: null,
        GroupEnd: false,
      },
    ])
    expect(dataModelToSlaveGrid({
      UnitID: 300,
      DataModel: {
        Coils: [{ Address: -1, Value: true }],
        DiscreteInputs: [{ Address: 2, Value: false }],
        InputRegisters: [{ Address: 3, Value: -10 }],
        HoldingRegisters: [{ Address: 4, Value: 42 }],
      },
    })).toMatchObject({
      unitId: 247,
      coils: [{ address: 0, value: true }],
      discreteInputs: [{ address: 2, value: false }],
      inputRegisters: [{ address: 3, value: 0, dataType: 'uint16' }],
      holdingRegisters: [{ address: 4, value: 42, dataType: 'uint16' }],
    })
    expect(dataModelToSlaveGrid({ UnitID: Number.NaN, DataModel: null as any }).unitId).toBe(1)
    expect(dataModelToSlaveGrid({ UnitID: 3, DataModel: {} as any })).toMatchObject({
      unitId: 3,
      coils: [],
      discreteInputs: [],
      inputRegisters: [],
      holdingRegisters: [],
    })
    expect(registerMappingRowsToMappings([{
      id: '',
      address: -1,
      dataType: 'bad' as any,
      wordOrder: 'bad' as any,
      length: -1,
      scalingFactor: Number.NaN,
      comment: undefined as any,
      groupEnd: 1 as any,
    }])).toEqual([{
      Address: 0,
      DataType: DataType.DataTypeUint16,
      WordOrder: WordOrder.WordOrderBig,
      Length: 0,
      ScalingFactor: 0,
      Comment: '',
      Interpolate: null,
      GroupEnd: true,
    }])
    expect(registerTypeToFunction('coils')).toBe(1)
    expect(registerTypeToFunction('discrete_inputs')).toBe(2)
    expect(registerTypeToFunction('input_registers')).toBe(4)
    expect(registerTypeToFunction('holding_registers')).toBe(3)
  })

  it('normalizes sparse mapping and slave grid helper rows', () => {
    expect(registerMappingRowsToMappings([{
      id: '',
      address: undefined as any,
      dataType: '' as any,
      wordOrder: '' as any,
      length: undefined as any,
      scalingFactor: undefined as any,
      comment: undefined as any,
      groupEnd: 0 as any,
    }])).toEqual([{
      Address: 0,
      DataType: '',
      WordOrder: WordOrder.WordOrderBig,
      Length: 0,
      ScalingFactor: 0,
      Comment: '',
      Interpolate: null,
      GroupEnd: false,
    }])

    expect(slaveGridToDataModel({
      unitId: Number.NaN,
      coils: [{ id: '', address: undefined as any, value: 1 as any }],
      discreteInputs: undefined as any,
      inputRegisters: [{ id: '', address: undefined as any, value: undefined as any, dataType: '' as any, comment: undefined as any }],
      holdingRegisters: undefined as any,
    })).toEqual({
      Coils: [{ Address: 0, Value: true }],
      DiscreteInputs: [],
      InputRegisters: [{ Address: 0, Value: 0 }],
      HoldingRegisters: [],
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

  it('rejects duplicate and out-of-range Unit IDs', async () => {
    const store = useModbusStore()

    expect(store.addMasterUnit(2)).toBe(true)
    expect(store.addMasterUnit(2)).toBe(false)
    expect(store.error).toContain('Unit 2 已存在')
    expect(store.addMasterUnit(0)).toBe(false)
    expect(store.error).toContain('Unit ID 必须在 1-247')
    await expect(store.addSlaveUnit(300)).resolves.toBe(false)
  })

  it('surfaces guards, restored-session skips, and backend errors for Modbus actions', async () => {
    const store = useModbusStore()

    await expect(store.openSession()).resolves.toBeNull()
    expect(store.error).toBe('请选择串口')

    store.portForm = { ...store.portForm, port: '/tmp/ttyM0' }
    serialServiceMock.openModbusSession.mockRejectedValueOnce(undefined)
    await expect(store.openSession()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to open Modbus session')

    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'restored-master',
      sessions: [sampleSession('restored-master', 'master')],
    })
    await expect(store.sendMasterRequest()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能直接发送')
    await expect(store.readRegisters()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能直接读取')
    await expect(store.scanUnitIDs()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能直接扫描')
    await expect(store.scanRegisters()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能直接扫描')

    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'restored-slave',
      sessions: [sampleSession('restored-slave', 'slave')],
    })
    await expect(store.startSlave()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能启动从站')
    await expect(store.applySlaveData()).resolves.toBeUndefined()
    expect(store.error).toBeNull()
    serialServiceMock.closeModbusSession.mockClear()
    await store.closeSession('restored-slave')
    expect(serialServiceMock.closeModbusSession).not.toHaveBeenCalled()

    openRealSession(store, 'modbus-master', 'master')
    serialServiceMock.modbusMasterRequest.mockRejectedValueOnce(new Error('send denied'))
    await expect(store.sendMasterRequest()).rejects.toThrow('send denied')
    expect(store.error).toBe('send denied')
    serialServiceMock.closeModbusSession.mockRejectedValueOnce(new Error('close denied'))
    await expect(store.closeSession('modbus-master')).rejects.toThrow('close denied')
    expect(store.error).toBe('close denied')
    serialServiceMock.modbusReadRegisters.mockRejectedValueOnce(undefined)
    await expect(store.readRegisters()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to read Modbus registers')
    serialServiceMock.modbusScanUnitIDs.mockRejectedValueOnce(new Error('unit scan denied'))
    await expect(store.scanUnitIDs()).rejects.toThrow('unit scan denied')
    expect(store.error).toBe('unit scan denied')
    serialServiceMock.modbusScanRegisters.mockRejectedValueOnce(undefined)
    await expect(store.scanRegisters()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to scan Modbus registers')

    openRealSession(store, 'modbus-slave', 'slave')
    serialServiceMock.startModbusSlave.mockRejectedValueOnce(undefined)
    await expect(store.startSlave()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to start Modbus slave')
    serialServiceMock.stopModbusSlave.mockRejectedValueOnce(new Error('stop denied'))
    await expect(store.stopSlave()).rejects.toThrow('stop denied')
    expect(store.error).toBe('stop denied')
    serialServiceMock.updateModbusSlaveUnitData.mockRejectedValueOnce(new Error('apply denied'))
    await expect(store.applySlaveData()).rejects.toThrow('apply denied')
    expect(store.error).toBe('apply denied')
  })

  it('manages polling and master unit removal fallback state', () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-master', 'master')

    expect(store.selectMasterUnit(99)).toBe(false)
    expect(store.removeMasterUnit()).toBe(false)
    expect(store.addMasterUnit(2)).toBe(true)
    expect(store.activeMasterUnitId).toBe(2)
    expect(store.removeMasterUnit(2)).toBe(true)
    expect(store.activeMasterUnitId).toBe(1)

    store.activeSlaveUnitId = 9
    expect(store.activeSlaveGrid.unitId).toBe(9)

    store.startRegisterPolling()
    expect(store.registerReadForm.polling).toBe(true)
    store.stopRegisterPolling()
    expect(store.registerReadForm.polling).toBe(false)
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

  it.each([
    {
      functionCode: 5,
      form: { value: 0xff00 },
      backend: { Value: 0xff00 },
    },
    {
      functionCode: 6,
      form: { value: 42 },
      backend: { Value: 42 },
    },
    {
      functionCode: 15,
      form: { quantity: 4, coilValues: '1 0 true off' },
      backend: { Quantity: 4, CoilValues: [true, false, true, false] },
    },
    {
      functionCode: 16,
      form: { quantity: 3, registerValues: '11 0x16 33' },
      backend: { Quantity: 3, RegisterValues: [11, 22, 33] },
    },
  ])('passes function $functionCode write request fields through to backend master request', async ({ functionCode, form, backend }) => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'master')
    store.masterForm = {
      ...store.masterForm,
      unitId: 7,
      functionCode,
      addressMode: 'plc',
      address: 40010,
      timeoutMs: 500,
      retries: 2,
      ...form,
    }

    await store.sendMasterRequest()

    expect(serialServiceMock.modbusMasterRequest).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'modbus-1',
      UnitID: 7,
      Function: functionCode,
      AddressMode: 'plc',
      Address: 40010,
      TimeoutMs: 500,
      Retries: 2,
      ...backend,
    }))
  })

  it('loads slave unit data models from backend when refreshing a slave session', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-1', sampleSession('modbus-1', 'slave'))
    store.setActiveSession('modbus-1')
    serialServiceMock.listModbusSessions.mockResolvedValueOnce([sampleSession('modbus-1', 'slave')])
    serialServiceMock.listModbusSlaveUnits.mockResolvedValueOnce([
      {
        UnitID: 2,
        DataModel: {
          Coils: [{ Address: 0, Value: true }],
          DiscreteInputs: [{ Address: 1, Value: false }],
          InputRegisters: [{ Address: 2, Value: 24 }],
          HoldingRegisters: [{ Address: 3, Value: 42 }],
        },
      },
    ])

    await store.refreshSessions()

    expect(serialServiceMock.listModbusSlaveUnits).toHaveBeenCalledWith('modbus-1')
    expect(store.slaveUnitGrids).toEqual([
      expect.objectContaining({
        unitId: 2,
        coils: [expect.objectContaining({ address: 0, value: true })],
        discreteInputs: [expect.objectContaining({ address: 1, value: false })],
        inputRegisters: [expect.objectContaining({ address: 2, value: 24, dataType: 'uint16', comment: '' })],
        holdingRegisters: [expect.objectContaining({ address: 3, value: 42, dataType: 'uint16', comment: '' })],
      }),
    ])
    expect(store.slaveUnitForms).toEqual([
      expect.objectContaining({
        unitId: 2,
        coils: '0=1',
        discreteInputs: '1=0',
        inputRegisters: '2=24',
        holdingRegisters: '3=42',
      }),
    ])
    expect(store.activeSlaveUnitId).toBe(2)
    expect(store.slaveForm).toMatchObject({ unitId: 2, holdingRegisters: '3=42' })
  })

  it('keeps local-only restored slave sessions from listing backend slave units', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-restored',
      sessions: [sampleSession('modbus-restored', 'slave')],
    })
    serialServiceMock.listModbusSessions.mockResolvedValueOnce([])

    await store.refreshSessions()
    await store.syncSlaveUnitsFromBackend('modbus-restored')

    expect(serialServiceMock.listModbusSlaveUnits).not.toHaveBeenCalled()
  })

  it('does not list backend slave units for master sessions', async () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-master', 'master')
    serialServiceMock.listModbusSessions.mockResolvedValueOnce([sampleSession('modbus-master', 'master')])

    await store.refreshSessions()
    await store.syncSlaveUnitsFromBackend('modbus-master')

    expect(serialServiceMock.listModbusSlaveUnits).not.toHaveBeenCalled()
  })

  it('refreshes slave unit data from backend after slave mutations', async () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'slave')
    serialServiceMock.listModbusSessions.mockResolvedValue([sampleSession('modbus-1', 'slave')])
    serialServiceMock.listModbusSlaveUnits
      .mockResolvedValueOnce([slaveUnitSnapshot(1, 10)])
      .mockResolvedValueOnce([slaveUnitSnapshot(1, 20)])
      .mockResolvedValueOnce([slaveUnitSnapshot(1, 20), slaveUnitSnapshot(2, 30)])
      .mockResolvedValueOnce([slaveUnitSnapshot(1, 40)])

    await store.startSlave()
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.holdingRegisters[0].value).toBe(10)

    store.slaveUnitGrids.find(unit => unit.unitId === 1)!.holdingRegisters[0].value = 22
    await store.applySlaveData()
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.holdingRegisters[0].value).toBe(20)

    await store.addSlaveUnit(2)
    expect(store.slaveUnitGrids.map(unit => unit.unitId)).toEqual([1, 2])
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 2)?.holdingRegisters[0].value).toBe(30)

    await store.removeSlaveUnit(2)
    expect(store.slaveUnitGrids.map(unit => unit.unitId)).toEqual([1])
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.holdingRegisters[0].value).toBe(40)
    expect(serialServiceMock.listModbusSlaveUnits).toHaveBeenCalledTimes(4)
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

  it('reads registers, applies raw mapped and bit results, and records scan outputs', async () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-1', 'master')
    const coils = store.masterRegisterTables.find(table => table.type === 'coils')!
    coils.address = 10
    coils.length = 3
    coils.rows = [
      { id: 'coil-10', address: 10, value: '0', raw: '', display: '', mappingId: '' },
      { id: 'coil-11', address: 11, value: '0', raw: '', display: '', mappingId: '' },
      { id: 'coil-12', address: 12, value: '0', raw: '', display: '', mappingId: '' },
    ]
    store.masterGrid.registerType = 'coils'
    serialServiceMock.modbusReadRegisters.mockResolvedValueOnce({
      Transaction: sampleTransaction('read-bits'),
      RawRegisters: [],
      Bits: [true, false, true],
      Values: [],
    } as any)

    await store.readRegisters()

    expect(serialServiceMock.modbusReadRegisters).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'modbus-1',
      Function: 1,
      Address: 10,
      Quantity: 3,
    }))
    expect(coils.rows.map(row => row.display)).toEqual(['ON', 'OFF', 'ON'])
    expect(store.history[0].ID).toBe('read-bits')

    const holding = store.masterRegisterTables.find(table => table.type === 'holding_registers')!
    holding.address = 20
    holding.length = 2
    holding.rows = [
      { id: 'hr-20', address: 20, value: '0', raw: '', display: '', mappingId: '' },
      { id: 'hr-21', address: 21, value: '0', raw: '', display: '', mappingId: '' },
    ]
    store.masterGrid.registerType = 'holding_registers'
    serialServiceMock.modbusReadRegisters.mockResolvedValueOnce({
      Transaction: null,
      RawRegisters: [24, 42],
      Bits: [],
      Values: [{
        Address: 20,
        Mapping: { Address: 20, DataType: DataType.DataTypeUint16, WordOrder: WordOrder.WordOrderBig, Length: 0, ScalingFactor: 0, Comment: 'temp', Interpolate: null, GroupEnd: false },
        Value: { DataType: DataType.DataTypeUint16, Raw: [24], Display: '24 °C', Numeric: 24 },
        Error: '',
      }, {
        Address: 21,
        Mapping: { Address: 21, DataType: DataType.DataTypeUint16, WordOrder: WordOrder.WordOrderBig, Length: 0, ScalingFactor: 0, Comment: 'pressure', Interpolate: null, GroupEnd: false },
        Value: { DataType: DataType.DataTypeUint16, Raw: [42], Display: 'bad', Numeric: 42 },
        Error: 'decode failed',
      }],
    } as any)

    await store.readRegisters()

    expect(holding.rows.map(row => row.raw)).toEqual(['24', '42'])
    expect(holding.rows.map(row => row.display)).toEqual(['24 °C', 'decode failed'])
    expect(holding.rows[0].mappingId).toBe('20-uint16')

    serialServiceMock.modbusScanUnitIDs.mockResolvedValueOnce({
      SessionID: 'modbus-1',
      ActiveUnitIDs: [1, 9],
      Results: [{ UnitID: 9, Active: true, Error: '' }],
    } as any)
    serialServiceMock.modbusScanRegisters.mockResolvedValueOnce({
      SessionID: 'modbus-1',
      UnitID: 9,
      Values: [{ Address: 20, Value: 42 }],
      Ranges: [{ Address: 20, Quantity: 2, Error: '' }],
    } as any)

    await store.scanUnitIDs()
    await store.scanRegisters()

    expect(store.unitScanResult?.ActiveUnitIDs).toEqual([1, 9])
    expect(store.registerScanResult?.Values).toEqual([{ Address: 20, Value: 42 }])
  })

  it('surfaces inactive session guards and falls back to grid-only slave units', async () => {
    const store = useModbusStore()

    await expect(store.sendMasterRequest()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 Modbus 会话')
    await expect(store.readRegisters()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 Modbus 会话')
    await expect(store.scanUnitIDs()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 Modbus 会话')
    await expect(store.scanRegisters()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 Modbus 会话')
    await expect(store.startSlave()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 Modbus 会话')
    store.clearError()
    expect(store.error).toBeNull()

    store.slaveUnitForms = []
    store.slaveUnitGrids = [defaultModbusSlaveUnitGrid(5)]
    store.selectSlaveUnit(5)

    expect(store.activeSlaveUnitId).toBe(5)
    expect(store.slaveForm.unitId).toBe(5)
    expect(store.activeSlaveForm.unitId).toBe(5)
  })

  it.each([
    [1, 'coils', 1],
    [2, 'discrete_inputs', 2],
    [4, 'input_registers', 4],
    [99, 'holding_registers', 3],
  ] as const)('restores legacy function %s register-read snapshots', (functionCode, registerType, backendFunction) => {
    const store = useModbusStore()
    const snapshot = store.exportState()
    delete (snapshot as any).masterGrid
    delete (snapshot as any).masterMappings
    delete (snapshot as any).masterRegisterTables
    delete (snapshot as any).masterUnitGrids
    snapshot.activeSessionId = 'modbus-legacy'
    snapshot.sessions = [sampleSession('modbus-legacy', 'master')]
    snapshot.registerReadForm = {
      ...snapshot.registerReadForm,
      unitId: 4,
      functionCode,
      addressMode: 'plc',
      address: 10001,
      quantity: 3,
      mappingText: '5:float:little:2:0.5',
      timeoutMs: 5,
      retries: -1,
      pollIntervalMs: 5,
    }

    store.restoreState(snapshot)

    expect(store.masterGrid).toMatchObject({
      unitId: 4,
      registerType,
      address: 10001,
      length: 3,
      addressBase: 1,
      pollRateMs: 10,
      timeoutMs: 10,
      retries: 0,
    })
    expect(store.masterMappings[0]).toMatchObject({
      address: 5,
      dataType: 'float',
      wordOrder: 'little',
      length: 2,
      scalingFactor: 0.5,
    })
    expect(store.buildRegisterReadRequest()).toMatchObject({
      SessionID: 'modbus-legacy',
      UnitID: 4,
      Function: backendFunction,
      AddressMode: 'plc',
      Address: 10001,
      Quantity: 3,
      TimeoutMs: 10,
      Retries: 0,
    })
  })

  it('exports a derived master Unit when legacy grid state targets a new Unit ID', () => {
    const store = useModbusStore()
    store.masterGrid = {
      ...defaultModbusMasterGrid(),
      unitId: 6,
      registerType: 'input_registers',
      address: 12,
      length: 2,
    }

    const snapshot = store.exportState()

    expect(snapshot.activeMasterUnitId).toBe(6)
    expect(snapshot.masterUnitGrids?.map(unit => unit.unitId)).toContain(6)
    expect(snapshot.masterUnitGrids?.find(unit => unit.unitId === 6)?.registerType).toBe('input_registers')
  })

  it('covers refresh open and close fallback branches', async () => {
    const store = useModbusStore()

    serialServiceMock.listModbusSessions.mockRejectedValueOnce(undefined)
    await store.refreshSessions()
    expect(store.error).toBe('Failed to refresh Modbus sessions')

    store.portForm = { ...store.portForm, port: '/tmp/ttyM0', sessionId: '' }
    await store.openSession()
    expect(serialServiceMock.openModbusSession).toHaveBeenCalledWith(expect.objectContaining({
      ID: expect.stringMatching(/^modbus-/),
      Config: expect.objectContaining({ PortName: '/tmp/ttyM0' }),
    }))

    openRealSession(store, 'modbus-active', 'master')
    store.sessions.set('modbus-other', sampleSession('modbus-other', 'master'))
    await store.closeSession('modbus-other')
    expect(store.activeSessionId).toBe('modbus-active')

    await expect(store.closeSession(null as any)).resolves.toBeUndefined()

    serialServiceMock.closeModbusSession.mockRejectedValueOnce(undefined)
    await expect(store.closeSession('modbus-active')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to close Modbus session')
  })

  it('covers slave action guards fallback data and duplicate Unit branches', async () => {
    const store = useModbusStore()

    await expect(store.stopSlave()).resolves.toBeUndefined()
    await expect(store.applySlaveData()).resolves.toBeUndefined()

    store.slaveForm = { ...store.slaveForm, unitId: 4, holdingRegisters: '0=77' }
    store.slaveUnitGrids = []
    expect(store.slaveDataModel()).toMatchObject({
      HoldingRegisters: [{ Address: 0, Value: 77 }],
    })

    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'restored-slave',
      sessions: [sampleSession('restored-slave', 'slave')],
    })
    serialServiceMock.stopModbusSlave.mockClear()
    await store.stopSlave()
    expect(serialServiceMock.stopModbusSlave).not.toHaveBeenCalled()

    openRealSession(store, 'real-slave', 'slave')
    await expect(store.addSlaveUnit(2)).resolves.toBe(true)
    await expect(store.addSlaveUnit(2)).resolves.toBe(false)
    expect(store.error).toContain('Unit 2 已存在')

    serialServiceMock.stopModbusSlave.mockRejectedValueOnce(undefined)
    await expect(store.stopSlave()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to stop Modbus slave')

    serialServiceMock.updateModbusSlaveUnitData.mockRejectedValueOnce(undefined)
    await expect(store.applySlaveData()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to update Modbus slave data')
  })

  it('covers master table creation orphan read results and polling failure cleanup', async () => {
    const store = useModbusStore()
    openRealSession(store, 'modbus-master', 'master')

    expect(store.addMasterUnit(2)).toBe(true)
    expect(store.removeMasterUnit(99)).toBe(false)

    store.masterGrid = {
      ...defaultModbusMasterGrid(),
      registerType: 'holding_registers',
      address: 20,
      length: 1,
    }
    store.masterRegisterTables = store.masterRegisterTables.filter(table => table.type !== 'holding_registers')
    expect(store.buildRegisterReadRequest()).toMatchObject({
      SessionID: 'modbus-master',
      Function: 3,
      Address: 20,
      Quantity: 1,
    })

    const holding = store.masterRegisterTables.find(table => table.type === 'holding_registers')!
    holding.address = 20
    holding.length = 1
    holding.rows = [{ id: 'hr-20', address: 20, value: '0', raw: '', display: '', mappingId: '' }]
    serialServiceMock.modbusReadRegisters.mockResolvedValueOnce({
      Transaction: null,
      RawRegisters: [11, 22],
      Bits: [true, false],
      Values: [{
        Address: 20,
        Mapping: { Address: 20, DataType: DataType.DataTypeUint16, WordOrder: WordOrder.WordOrderBig, Length: 0, ScalingFactor: 0, Comment: '', Interpolate: null, GroupEnd: false },
        Value: { DataType: DataType.DataTypeUint16, Raw: undefined, Display: 'decoded', Numeric: 11 },
        Error: '',
      }, {
        Address: 999,
        Mapping: { Address: 999, DataType: DataType.DataTypeUint16, WordOrder: WordOrder.WordOrderBig, Length: 0, ScalingFactor: 0, Comment: '', Interpolate: null, GroupEnd: false },
        Value: { DataType: DataType.DataTypeUint16, Raw: [99], Display: 'orphan', Numeric: 99 },
        Error: '',
      }],
    } as any)

    await store.readRegisters()
    expect(holding.rows).toEqual([{ id: 'hr-20', address: 20, value: 'decoded', raw: '', display: 'decoded', mappingId: '20-uint16' }])

    vi.useFakeTimers()
    try {
      const pollingTable = store.masterRegisterTables.find(table => table.type === 'holding_registers')!
      pollingTable.pollRateMs = 0
      store.masterGrid.pollRateMs = 0
      store.registerReadForm.pollIntervalMs = 25
      serialServiceMock.modbusReadRegisters.mockRejectedValueOnce(new Error('poll failed'))

      store.startRegisterPolling()
      await vi.advanceTimersByTimeAsync(25)

      expect(store.registerReadForm.polling).toBe(false)
    } finally {
      store.stopRegisterPolling()
      vi.useRealTimers()
    }
  })

  it('covers request guards and restore fallback branches', () => {
    const store = useModbusStore()

    expect(store.buildRegisterReadRequest()).toBeNull()
    expect(store.buildUnitScanRequest()).toBeNull()
    expect(store.buildRegisterScanRequest()).toBeNull()

    store.restoreState()
    expect(store.activeSessionId).toBeNull()

    const snapshot = store.exportState()
    delete (snapshot as any).activeMasterUnitId
    delete (snapshot as any).masterUnitGrids
    delete (snapshot as any).masterRegisterTables
    delete (snapshot as any).slaveUnitForms
    delete (snapshot as any).history
    delete (snapshot.registerReadForm as any).mappingText
    snapshot.activeSlaveUnitId = 7
    snapshot.slaveForm = { ...snapshot.slaveForm, unitId: 8, holdingRegisters: '0=88' }
    snapshot.masterMappings = []
    snapshot.slaveUnitGrids = [defaultModbusSlaveUnitGrid(9)]

    store.restoreState(snapshot as any)

    expect(store.activeMasterUnitId).toBe(1)
    expect(store.activeSlaveUnitId).toBe(8)
    expect(store.slaveUnitForms.map(unit => unit.unitId)).toEqual([8])
    expect(store.slaveUnitGrids.map(unit => unit.unitId)).toEqual([8, 9])
    expect(store.history).toEqual([])
  })

  it('covers missing active sessions and backend error fallbacks for master requests', async () => {
    const store = useModbusStore()

    store.setActiveSession('missing-session')
    expect(store.activeSession).toBeNull()

    openRealSession(store, 'real-master', 'master')
    serialServiceMock.modbusMasterRequest.mockRejectedValueOnce(undefined)

    await expect(store.sendMasterRequest()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to send Modbus request')
  })

  it('normalizes malformed restored master and slave unit snapshots', () => {
    const store = useModbusStore()
    const malformedTable = {
      type: 'bad',
      unitId: Number.NaN,
      address: -10,
      length: 0,
      addressBase: 2,
      pollRateMs: 0,
      timeoutMs: 0,
      retries: -1,
      rows: [{ id: '', address: undefined, value: '', raw: undefined, display: undefined, mappingId: undefined }],
      mappings: [{
        id: '',
        address: undefined,
        dataType: 'bad',
        wordOrder: 'sideways',
        length: undefined,
        scalingFactor: Number.NaN,
        comment: undefined,
        groupEnd: 0,
      }],
    }
    const snapshot = {
      ...store.exportState(),
      activeSessionId: null,
      sessions: [],
      activeMasterUnitId: undefined,
      masterGrid: {
        ...defaultModbusMasterGrid(),
        unitId: Number.NaN,
        registerType: 'bad',
        address: -12.8,
        length: 0,
        addressBase: 2,
        pollRateMs: 0,
        timeoutMs: 0,
        retries: -2,
      },
      masterMappings: [],
      masterRegisterTables: [malformedTable],
      masterUnitGrids: [{
        unitId: Number.NaN,
        registerType: 'bad',
        littleEndian: false,
        rawVisible: undefined,
        logVisible: undefined,
        registerTables: [malformedTable],
        registerReadResult: undefined,
        unitScanResult: undefined,
        registerScanResult: undefined,
      }],
      activeSlaveUnitId: undefined,
      slaveForm: { ...store.slaveForm, unitId: Number.NaN, coils: '', discreteInputs: '', inputRegisters: '', holdingRegisters: '' },
      slaveUnitForms: [],
      slaveUnitGrids: [],
      registerReadForm: {
        ...store.registerReadForm,
        unitId: undefined,
        functionCode: undefined,
        addressMode: undefined,
        address: undefined,
        quantity: undefined,
        mappingText: undefined,
        timeoutMs: undefined,
        retries: undefined,
        pollIntervalMs: undefined,
      },
    }

    store.restoreState(snapshot as any)

    expect(store.masterGrid).toMatchObject({
      unitId: 1,
      registerType: 'holding_registers',
      address: 0,
      length: 1,
      addressBase: 0,
      pollRateMs: 10,
      timeoutMs: 10,
      retries: 0,
    })
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0]).toEqual({
      id: 'master-row-undefined-0',
      address: 0,
      value: '0',
      raw: '',
      display: '',
      mappingId: '',
    })
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.mappings[0]).toMatchObject({
      id: 'map-undefined-0',
      address: 0,
      dataType: 'uint16',
      wordOrder: 'big',
      length: 0,
      scalingFactor: 0,
      comment: '',
      groupEnd: false,
    })
    expect(store.activeSlaveUnitId).toBe(1)
    expect(store.slaveUnitForms).toEqual([expect.objectContaining({ unitId: 1 })])
    expect(store.slaveUnitGrids).toEqual([expect.objectContaining({ unitId: 1 })])
    expect(store.registerReadForm.polling).toBe(false)
  })

  it('derives missing slave grids and skips backend removals for restored sessions', async () => {
    const store = useModbusStore()
    store.slaveForm = { ...store.slaveForm, unitId: 3, holdingRegisters: '0=33' }
    store.slaveUnitForms = [{ ...store.slaveForm }]
    store.slaveUnitGrids = []

    store.selectSlaveUnit(3)

    expect(store.slaveUnitGrids).toEqual([
      expect.objectContaining({
        unitId: 3,
        holdingRegisters: [expect.objectContaining({ address: 0, value: 33 })],
      }),
    ])

    openRealSession(store, 'real-slave', 'slave')
    store.slaveForm = { ...store.slaveForm, unitId: 4, holdingRegisters: '0=44' }
    store.slaveUnitForms = [{ ...store.slaveForm }]
    store.slaveUnitGrids = []

    await store.startSlave()

    expect(serialServiceMock.startModbusSlave).toHaveBeenCalledWith(expect.objectContaining({
      UnitID: 4,
      Units: [expect.objectContaining({ UnitID: 4 })],
    }))

    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'restored-slave',
      sessions: [sampleSession('restored-slave', 'slave')],
      slaveUnitForms: [
        { ...store.slaveForm, unitId: 1, holdingRegisters: '0=11' },
        { ...store.slaveForm, unitId: 2, holdingRegisters: '0=22' },
      ],
      slaveUnitGrids: [defaultModbusSlaveUnitGrid(1), defaultModbusSlaveUnitGrid(2)],
      activeSlaveUnitId: 2,
    })
    serialServiceMock.removeModbusSlaveUnit.mockClear()
    store.selectSlaveUnit(1)

    await store.removeSlaveUnit(2)
    await store.removeSlaveUnit()

    expect(serialServiceMock.removeModbusSlaveUnit).not.toHaveBeenCalled()
    expect(store.slaveUnitForms.map(unit => unit.unitId)).toEqual([1])
  })

  it('covers sparse restored master defaults and scan fallback errors', async () => {
    const store = useModbusStore()
    const snapshot = store.exportState()
    delete (snapshot as any).masterGrid
    delete (snapshot as any).masterMappings
    delete (snapshot as any).registerReadForm
    ;(snapshot as any).activeMasterUnitId = undefined
    ;(snapshot as any).masterRegisterTables = [{
      type: 'bad',
      unitId: undefined,
      address: undefined,
      length: undefined,
      addressBase: 2,
      pollRateMs: undefined,
      timeoutMs: undefined,
      retries: undefined,
      rows: undefined,
      mappings: undefined,
    }]
    ;(snapshot as any).masterUnitGrids = [{
      unitId: undefined,
      registerType: undefined,
      littleEndian: false,
      rawVisible: undefined,
      logVisible: undefined,
      registerTables: (snapshot as any).masterRegisterTables,
      registerReadResult: undefined,
      unitScanResult: undefined,
      registerScanResult: undefined,
    }]

    store.restoreState(snapshot as any)

    expect(store.activeMasterUnitId).toBe(1)
    expect(store.masterGrid).toMatchObject({
      unitId: 1,
      registerType: 'holding_registers',
      address: 0,
      length: 1,
      pollRateMs: 1000,
      timeoutMs: 800,
      retries: 0,
    })
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0]).toMatchObject({
      address: 0,
      value: '0',
      raw: '',
      display: '',
      mappingId: '',
    })

    openRealSession(store, 'modbus-master', 'master')
    const activeTable = store.masterRegisterTables.find(table => table.type === 'holding_registers')!
    activeTable.retries = 1
    activeTable.mappings = []

    expect(store.buildRegisterReadRequest()).toMatchObject({
      SessionID: 'modbus-master',
      UnitID: 1,
      Function: 3,
      Retries: 1,
    })

    const previousSlaveUnitId = store.activeSlaveUnitId
    store.selectSlaveUnit(99)
    expect(store.activeSlaveUnitId).toBe(previousSlaveUnitId)

    serialServiceMock.modbusScanUnitIDs.mockRejectedValueOnce(undefined)
    await expect(store.scanUnitIDs()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to scan Modbus unit IDs')
  })
})

function openRealSession(store: ReturnType<typeof useModbusStore>, id: string, role: 'master' | 'slave') {
  store.sessions.set(id, sampleSession(id, role))
  store.setActiveSession(id)
}

function slaveUnitSnapshot(unitId: number, holdingValue: number): SlaveUnitInfo {
  return {
    UnitID: unitId,
    DataModel: {
      Coils: [{ Address: 0, Value: true }],
      DiscreteInputs: [],
      InputRegisters: [],
      HoldingRegisters: [{ Address: 0, Value: holdingValue }],
    },
  }
}

function sampleTransaction(id: string) {
  return {
    ID: id,
    SessionID: 'modbus-1',
    StartedAt: '',
    CompletedAt: '',
    UnitID: 1,
    Mode: FrameMode.FrameModeRTU,
    RequestPDU: { Function: 1, Data: '' },
    ResponsePDU: { Function: 1, Data: '' },
    RequestFrameHex: '01 01 00 0A 00 03',
    ResponseFrameHex: '01 01 01 05',
    BytesWritten: 8,
    Response: {
      Function: 1,
      Exception: false,
      ExceptionCode: 0,
      Address: 10,
      Quantity: 3,
      Value: 0,
      Values: [],
      Bits: [true, false, true],
      Raw: '',
    },
    Error: '',
  }
}

function sampleSession(id: string, role: 'master' | 'slave' = 'slave'): SessionInfo {
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
