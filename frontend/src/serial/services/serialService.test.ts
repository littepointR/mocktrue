import { beforeEach, describe, expect, it, vi } from 'vitest'

const bindings = vi.hoisted(() => ({
  EnumeratePorts: vi.fn(),
  OpenPort: vi.fn(),
  ClosePort: vi.fn(),
  ListPorts: vi.fn(),
  ResetCounters: vi.fn(),
  RestoreCounters: vi.fn(),
  OpenFecbusSession: vi.fn(),
  CloseFecbusSession: vi.fn(),
  ListFecbusSessions: vi.fn(),
  FecbusSendRequest: vi.fn(),
  StartFecbusSlave: vi.fn(),
  StopFecbusSlave: vi.fn(),
  UpdateFecbusSlaveState: vi.fn(),
  AddFecbusSlaveUnit: vi.fn(),
  RemoveFecbusSlaveUnit: vi.fn(),
  ListFecbusSlaveUnits: vi.fn(),
  QueryFecbusFrames: vi.fn(),
  ClearFecbusFrames: vi.fn(),
  OpenModbusSession: vi.fn(),
  CloseModbusSession: vi.fn(),
  ListModbusSessions: vi.fn(),
  ModbusMasterRequest: vi.fn(),
  ModbusReadRegisters: vi.fn(),
  ModbusScanRegisters: vi.fn(),
  ModbusScanUnitIDs: vi.fn(),
  StartModbusSlave: vi.fn(),
  StopModbusSlave: vi.fn(),
  UpdateModbusSlaveData: vi.fn(),
  AddModbusSlaveUnit: vi.fn(),
  RemoveModbusSlaveUnit: vi.fn(),
  ListModbusSlaveUnits: vi.fn(),
  UpdateModbusSlaveUnitData: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => bindings)

import { SerialService } from './serialService'

describe('SerialService', () => {
  const service = new SerialService()

  beforeEach(() => {
    for (const key in bindings) bindings[key as keyof typeof bindings].mockReset()
  })

  it('normalizes nullable list results to empty arrays', async () => {
    bindings.EnumeratePorts.mockResolvedValue(null)
    bindings.ListPorts.mockResolvedValue(null)
    bindings.ListFecbusSessions.mockResolvedValue(null)
    bindings.ListFecbusSlaveUnits.mockResolvedValue(null)
    bindings.ListModbusSessions.mockResolvedValue(null)
    bindings.ListModbusSlaveUnits.mockResolvedValue(null)

    await expect(service.enumeratePorts()).resolves.toEqual([])
    await expect(service.listPorts()).resolves.toEqual([])
    await expect(service.listFecbusSessions()).resolves.toEqual([])
    await expect(service.listFecbusSlaveUnits('fecbus-1')).resolves.toEqual([])
    await expect(service.listModbusSessions()).resolves.toEqual([])
    await expect(service.listModbusSlaveUnits('modbus-1')).resolves.toEqual([])
  })

  it('delegates void commands to their generated bindings', async () => {
    const request = { ID: 'session-1' }
    const data = { Coils: {} }

    await service.closePort('port-1')
    await service.resetCounters('port-1')
    await service.restoreCounters('port-1', 3, 4)
    await service.closeFecbusSession('fecbus-1')
    await service.stopFecbusSlave('fecbus-1')
    await service.updateFecbusSlaveState('fecbus-1', data as any)
    await service.addFecbusSlaveUnit('fecbus-1', request as any)
    await service.removeFecbusSlaveUnit('fecbus-1', 7)
    await service.clearFecbusFrames('fecbus-1')
    await service.closeModbusSession('modbus-1')
    await service.stopModbusSlave('modbus-1')
    await service.updateModbusSlaveData('modbus-1', data as any)
    await service.addModbusSlaveUnit('modbus-1', request as any)
    await service.removeModbusSlaveUnit('modbus-1', 2)
    await service.updateModbusSlaveUnitData('modbus-1', 2, data as any)

    expect(bindings.ClosePort).toHaveBeenCalledWith('port-1')
    expect(bindings.ResetCounters).toHaveBeenCalledWith('port-1')
    expect(bindings.RestoreCounters).toHaveBeenCalledWith('port-1', 3, 4)
    expect(bindings.CloseFecbusSession).toHaveBeenCalledWith('fecbus-1')
    expect(bindings.StopFecbusSlave).toHaveBeenCalledWith('fecbus-1')
    expect(bindings.UpdateFecbusSlaveState).toHaveBeenCalledWith('fecbus-1', data)
    expect(bindings.AddFecbusSlaveUnit).toHaveBeenCalledWith('fecbus-1', request)
    expect(bindings.RemoveFecbusSlaveUnit).toHaveBeenCalledWith('fecbus-1', 7)
    expect(bindings.ClearFecbusFrames).toHaveBeenCalledWith('fecbus-1')
    expect(bindings.CloseModbusSession).toHaveBeenCalledWith('modbus-1')
    expect(bindings.StopModbusSlave).toHaveBeenCalledWith('modbus-1')
    expect(bindings.UpdateModbusSlaveData).toHaveBeenCalledWith('modbus-1', data)
    expect(bindings.AddModbusSlaveUnit).toHaveBeenCalledWith('modbus-1', request)
    expect(bindings.RemoveModbusSlaveUnit).toHaveBeenCalledWith('modbus-1', 2)
    expect(bindings.UpdateModbusSlaveUnitData).toHaveBeenCalledWith('modbus-1', 2, data)
  })

  it.each([
    ['openPort', 'OpenPort', [{ Config: { PortName: 'COM1' } }], 'Failed to open port: no status returned'],
    ['openFecbusSession', 'OpenFecbusSession', [{ PortHandleID: 'port-1' }], 'Failed to open FECbus session: no status returned'],
    ['fecbusSendRequest', 'FecbusSendRequest', [{ SessionID: 'fecbus-1' }], 'Failed to run FECbus request: no transaction returned'],
    ['startFecbusSlave', 'StartFecbusSlave', [{ PortHandleID: 'port-1' }], 'Failed to start FECbus slave: no session returned'],
    ['queryFecbusFrames', 'QueryFecbusFrames', [{ SessionID: 'fecbus-1' }], 'Failed to query FECbus frames: no page returned'],
    ['openModbusSession', 'OpenModbusSession', [{ PortHandleID: 'port-1' }], 'Failed to open Modbus session: no status returned'],
    ['modbusMasterRequest', 'ModbusMasterRequest', [{ SessionID: 'modbus-1' }], 'Failed to run Modbus request: no transaction returned'],
    ['modbusReadRegisters', 'ModbusReadRegisters', [{ SessionID: 'modbus-1' }], 'Failed to read Modbus registers: no result returned'],
    ['modbusScanUnitIDs', 'ModbusScanUnitIDs', [{ SessionID: 'modbus-1' }], 'Failed to scan Modbus unit IDs: no result returned'],
    ['modbusScanRegisters', 'ModbusScanRegisters', [{ SessionID: 'modbus-1' }], 'Failed to scan Modbus registers: no result returned'],
    ['startModbusSlave', 'StartModbusSlave', [{ PortHandleID: 'port-1' }], 'Failed to start Modbus slave: no session returned'],
  ])('throws when %s receives no generated binding result', async (method, binding, args, message) => {
    bindings[binding as keyof typeof bindings].mockResolvedValue(null)

    await expect((service as any)[method](...args)).rejects.toThrow(message)
  })

  it.each([
    ['enumeratePorts', 'EnumeratePorts', [], [{ Name: 'COM1' }]],
    ['openPort', 'OpenPort', [{ Config: { PortName: 'COM1' } }], { ID: 'port-1' }],
    ['listPorts', 'ListPorts', [], [{ ID: 'port-1' }]],
    ['openFecbusSession', 'OpenFecbusSession', [{ PortHandleID: 'port-1' }], { ID: 'fecbus-1' }],
    ['listFecbusSessions', 'ListFecbusSessions', [], [{ ID: 'fecbus-1' }]],
    ['fecbusSendRequest', 'FecbusSendRequest', [{ SessionID: 'fecbus-1' }], { ID: 'tx-1' }],
    ['startFecbusSlave', 'StartFecbusSlave', [{ PortHandleID: 'port-1' }], { ID: 'fecbus-slave' }],
    ['listFecbusSlaveUnits', 'ListFecbusSlaveUnits', ['fecbus-1'], [{ Address: 1 }]],
    ['queryFecbusFrames', 'QueryFecbusFrames', [{ SessionID: 'fecbus-1' }], { Frames: [] }],
    ['openModbusSession', 'OpenModbusSession', [{ PortHandleID: 'port-1' }], { ID: 'modbus-1' }],
    ['listModbusSessions', 'ListModbusSessions', [], [{ ID: 'modbus-1' }]],
    ['modbusMasterRequest', 'ModbusMasterRequest', [{ SessionID: 'modbus-1' }], { ID: 'tx-1' }],
    ['modbusReadRegisters', 'ModbusReadRegisters', [{ SessionID: 'modbus-1' }], { Values: [1] }],
    ['modbusScanUnitIDs', 'ModbusScanUnitIDs', [{ SessionID: 'modbus-1' }], { Units: [1] }],
    ['modbusScanRegisters', 'ModbusScanRegisters', [{ SessionID: 'modbus-1' }], { Registers: [1] }],
    ['startModbusSlave', 'StartModbusSlave', [{ PortHandleID: 'port-1' }], { ID: 'modbus-slave' }],
    ['listModbusSlaveUnits', 'ListModbusSlaveUnits', ['modbus-1'], [{ UnitID: 1 }]],
  ])('returns %s generated binding results', async (method, binding, args, result) => {
    bindings[binding as keyof typeof bindings].mockResolvedValue(result)

    await expect((service as any)[method](...args)).resolves.toBe(result)
    expect(bindings[binding as keyof typeof bindings]).toHaveBeenCalledWith(...args)
  })
})
