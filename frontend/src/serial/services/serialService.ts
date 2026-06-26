import {
  EnumeratePorts,
  OpenPort,
  ClosePort,
  ListPorts,
  ResetCounters,
  RestoreCounters,
  OpenFecbusSession,
  CloseFecbusSession,
  ListFecbusSessions,
  FecbusSendRequest,
  StartFecbusSlave,
  StopFecbusSlave,
  UpdateFecbusSlaveState,
  AddFecbusSlaveUnit,
  RemoveFecbusSlaveUnit,
  ListFecbusSlaveUnits,
  QueryFecbusFrames,
  ClearFecbusFrames,
  OpenModbusSession,
  CloseModbusSession,
  ListModbusSessions,
  ModbusMasterRequest,
  ModbusReadRegisters,
  ModbusScanRegisters,
  ModbusScanUnitIDs,
  StartModbusSlave,
  StopModbusSlave,
  UpdateModbusSlaveData,
  AddModbusSlaveUnit,
  RemoveModbusSlaveUnit,
  ListModbusSlaveUnits,
  UpdateModbusSlaveUnitData,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'

import type { PortInfo } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/port/models.js'
import type { HandleStatus, OpenRequest } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/manager/models.js'
import type {
  FramePage as FecbusFramePage,
  OpenSessionRequest as FecbusOpenSessionRequest,
  QueryRequest as FecbusQueryRequest,
  SendRequest as FecbusSendRequestType,
  SessionInfo as FecbusSessionInfo,
  SlaveState as FecbusSlaveState,
  SlaveUnitInfo as FecbusSlaveUnitInfo,
  SlaveUnitState as FecbusSlaveUnitState,
  StartSlaveRequest as FecbusStartSlaveRequest,
  Transaction as FecbusTransaction,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'
import type {
  DataModelSnapshot,
  MasterRequest,
  OpenSessionRequest,
  RegisterReadRequest,
  RegisterReadResult,
  RegisterScanRequest,
  RegisterScanResult,
  SessionInfo,
  SlaveUnitInfo,
  SlaveUnitSnapshot,
  StartSlaveRequest,
  Transaction,
  UnitScanRequest,
  UnitScanResult,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/modbus/models.js'

/**
 * SerialService encapsulates all serial port API calls.
 * Use this service instead of calling Wails bindings directly.
 */
export class SerialService {
  /**
   * Enumerate all available serial ports on the system.
   */
  async enumeratePorts(): Promise<PortInfo[]> {
    const result = await EnumeratePorts()
    return result ?? []
  }

  /**
   * Open a serial port with the given configuration.
   */
  async openPort(request: OpenRequest): Promise<HandleStatus> {
    const result = await OpenPort(request)
    if (!result) {
      throw new Error('Failed to open port: no status returned')
    }
    return result
  }

  /**
   * Close an open serial port by its handle ID.
   */
  async closePort(id: string): Promise<void> {
    await ClosePort(id)
  }

  /**
   * List all currently open serial port handles.
   */
  async listPorts(): Promise<HandleStatus[]> {
    const result = await ListPorts()
    return result ?? []
  }

  /**
   * Reset RX/TX counters for an open serial port handle.
   */
  async resetCounters(id: string): Promise<void> {
    await ResetCounters(id)
  }

  async restoreCounters(id: string, rxBytes: number, txBytes: number): Promise<void> {
    await RestoreCounters(id, rxBytes, txBytes)
  }

  async openFecbusSession(request: FecbusOpenSessionRequest): Promise<FecbusSessionInfo> {
    const result = await OpenFecbusSession(request)
    if (!result) {
      throw new Error('Failed to open FECbus session: no status returned')
    }
    return result
  }

  async closeFecbusSession(id: string): Promise<void> {
    await CloseFecbusSession(id)
  }

  async listFecbusSessions(): Promise<FecbusSessionInfo[]> {
    return await ListFecbusSessions() ?? []
  }

  async fecbusSendRequest(request: FecbusSendRequestType): Promise<FecbusTransaction> {
    const result = await FecbusSendRequest(request)
    if (!result) {
      throw new Error('Failed to run FECbus request: no transaction returned')
    }
    return result
  }

  async startFecbusSlave(request: FecbusStartSlaveRequest): Promise<FecbusSessionInfo> {
    const result = await StartFecbusSlave(request)
    if (!result) {
      throw new Error('Failed to start FECbus slave: no session returned')
    }
    return result
  }

  async stopFecbusSlave(id: string): Promise<void> {
    await StopFecbusSlave(id)
  }

  async updateFecbusSlaveState(sessionID: string, state: FecbusSlaveState): Promise<void> {
    await UpdateFecbusSlaveState(sessionID, state)
  }

  async addFecbusSlaveUnit(sessionID: string, unit: FecbusSlaveUnitState): Promise<void> {
    await AddFecbusSlaveUnit(sessionID, unit)
  }

  async removeFecbusSlaveUnit(sessionID: string, address: number): Promise<void> {
    await RemoveFecbusSlaveUnit(sessionID, address)
  }

  async listFecbusSlaveUnits(sessionID: string): Promise<FecbusSlaveUnitInfo[]> {
    return await ListFecbusSlaveUnits(sessionID) ?? []
  }

  async queryFecbusFrames(request: FecbusQueryRequest): Promise<FecbusFramePage> {
    const result = await QueryFecbusFrames(request)
    if (!result) {
      throw new Error('Failed to query FECbus frames: no page returned')
    }
    return result
  }

  async clearFecbusFrames(id: string): Promise<void> {
    await ClearFecbusFrames(id)
  }

  async openModbusSession(request: OpenSessionRequest): Promise<SessionInfo> {
    const result = await OpenModbusSession(request)
    if (!result) {
      throw new Error('Failed to open Modbus session: no status returned')
    }
    return result
  }

  async closeModbusSession(id: string): Promise<void> {
    await CloseModbusSession(id)
  }

  async listModbusSessions(): Promise<SessionInfo[]> {
    return await ListModbusSessions() ?? []
  }

  async modbusMasterRequest(request: MasterRequest): Promise<Transaction> {
    const result = await ModbusMasterRequest(request)
    if (!result) {
      throw new Error('Failed to run Modbus request: no transaction returned')
    }
    return result
  }

  async modbusReadRegisters(request: RegisterReadRequest): Promise<RegisterReadResult> {
    const result = await ModbusReadRegisters(request)
    if (!result) {
      throw new Error('Failed to read Modbus registers: no result returned')
    }
    return result
  }

  async modbusScanUnitIDs(request: UnitScanRequest): Promise<UnitScanResult> {
    const result = await ModbusScanUnitIDs(request)
    if (!result) {
      throw new Error('Failed to scan Modbus unit IDs: no result returned')
    }
    return result
  }

  async modbusScanRegisters(request: RegisterScanRequest): Promise<RegisterScanResult> {
    const result = await ModbusScanRegisters(request)
    if (!result) {
      throw new Error('Failed to scan Modbus registers: no result returned')
    }
    return result
  }

  async startModbusSlave(request: StartSlaveRequest): Promise<SessionInfo> {
    const result = await StartModbusSlave(request)
    if (!result) {
      throw new Error('Failed to start Modbus slave: no session returned')
    }
    return result
  }

  async stopModbusSlave(id: string): Promise<void> {
    await StopModbusSlave(id)
  }

  async updateModbusSlaveData(sessionID: string, data: DataModelSnapshot): Promise<void> {
    await UpdateModbusSlaveData(sessionID, data)
  }

  async addModbusSlaveUnit(sessionID: string, unit: SlaveUnitSnapshot): Promise<void> {
    await AddModbusSlaveUnit(sessionID, unit)
  }

  async removeModbusSlaveUnit(sessionID: string, unitID: number): Promise<void> {
    await RemoveModbusSlaveUnit(sessionID, unitID)
  }

  async listModbusSlaveUnits(sessionID: string): Promise<SlaveUnitInfo[]> {
    return await ListModbusSlaveUnits(sessionID) ?? []
  }

  async updateModbusSlaveUnitData(sessionID: string, unitID: number, data: DataModelSnapshot): Promise<void> {
    await UpdateModbusSlaveUnitData(sessionID, unitID, data)
  }
}

export const serialService = new SerialService()
