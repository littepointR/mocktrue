import { Events } from '@wailsio/runtime'

/**
 * DataEvent is emitted when data is received from a serial port.
 */
export interface DataEvent {
  PortID: string
  Data: number[] | Uint8Array | string | null
}

/**
 * SerialEvents provides event subscription for serial port data.
 */
export class SerialEvents {
  /**
   * Subscribe to serial:data events.
   * Returns an unsubscribe function.
   */
  onData(callback: (event: DataEvent) => void): () => void {
    const handler = (e: { data: DataEvent }) => callback(e.data)
    Events.On('serial:data', handler)
    return () => Events.Off('serial:data')
  }
}

export const serialEvents = new SerialEvents()
