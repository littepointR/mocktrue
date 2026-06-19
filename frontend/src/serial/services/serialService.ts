import {
  EnumeratePorts,
  OpenPort,
  ClosePort,
  ListPorts,
  ResetCounters,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'

import type { PortInfo } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/port/models.js'
import type { HandleStatus, OpenRequest } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/manager/models.js'

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
}

export const serialService = new SerialService()
