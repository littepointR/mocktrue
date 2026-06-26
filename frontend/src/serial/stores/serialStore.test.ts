import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useSerialStore } from './serialStore'
import type { HandleStatus } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/manager/models.js'
import type { PortInfo, SerialConfig } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/port/models.js'

const serialServiceMock = vi.hoisted(() => ({
  enumeratePorts: vi.fn(),
  openPort: vi.fn(),
  closePort: vi.fn(),
  listPorts: vi.fn(),
  resetCounters: vi.fn(),
  restoreCounters: vi.fn(),
}))

vi.mock('../services/serialService', () => ({
  serialService: serialServiceMock,
}))

describe('serial store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
    serialServiceMock.enumeratePorts.mockResolvedValue([])
    serialServiceMock.openPort.mockResolvedValue(sampleHandle('handle-1', sampleConfig('/dev/tty.usbserial')))
    serialServiceMock.closePort.mockResolvedValue(undefined)
    serialServiceMock.listPorts.mockResolvedValue([])
    serialServiceMock.resetCounters.mockResolvedValue(undefined)
    serialServiceMock.restoreCounters.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('refreshes ports and handles and surfaces fallback errors', async () => {
    const store = useSerialStore()

    serialServiceMock.enumeratePorts.mockResolvedValueOnce([
      {
        Name: '/dev/tty.usbserial',
        Vendor: 'ACME',
        Product: 'USB Serial',
        VID: '2341',
        PID: '0043',
        SerialNumber: 'ABC123',
        IsUSB: true,
        FriendlyName: 'ACME USB Serial',
      } satisfies PortInfo,
    ])
    serialServiceMock.listPorts.mockResolvedValueOnce([
      sampleHandle('handle-1', sampleConfig('/dev/tty.usbserial')),
    ])

    await store.refreshPorts()
    await store.refreshHandles()

    expect(store.ports).toHaveLength(1)
    expect(store.handles.get('handle-1')?.Config.PortName).toBe('/dev/tty.usbserial')
    expect(store.error).toBeNull()

    serialServiceMock.enumeratePorts.mockRejectedValueOnce(undefined)
    serialServiceMock.listPorts.mockRejectedValueOnce(undefined)

    await store.refreshPorts()
    expect(store.error).toBe('Failed to enumerate ports')

    await store.refreshHandles()
    expect(store.error).toBe('Failed to refresh handles')
  })

  it('opens ports with defaults, reuses an existing handle, and refreshes polling stats', async () => {
    const store = useSerialStore()
    serialServiceMock.openPort.mockImplementation(async request => sampleHandle('handle-1', request.Config))
    serialServiceMock.listPorts.mockResolvedValue([
      sampleHandle('handle-1', sampleConfig('/dev/tty.usbserial'), { RxBytes: 9, TxBytes: 12 }),
    ])

    const id = await store.openPort('/dev/tty.usbserial', 9600)

    expect(id).toBe('handle-1')
    expect(serialServiceMock.openPort).toHaveBeenCalledWith({
      Config: {
        PortName: '/dev/tty.usbserial',
        BaudRate: 9600,
        DataBits: 8,
        StopBits: '1',
        Parity: 'none',
        FlowMode: 'none',
        ReadBufKB: 32,
      },
    })
    expect(store.activeHandle?.ID).toBe('handle-1')
    expect(store.openHandles.map(handle => handle.ID)).toEqual(['handle-1'])

    const reusedId = await store.openPort('/dev/tty.usbserial', 115200, { dataBits: 7 })

    expect(reusedId).toBe('handle-1')
    expect(serialServiceMock.openPort).toHaveBeenCalledTimes(1)
    expect(store.activePortId).toBe('handle-1')

    await vi.advanceTimersByTimeAsync(5000)

    expect(store.handles.get('handle-1')).toEqual(expect.objectContaining({
      RxBytes: 9,
      TxBytes: 12,
    }))
  })

  it('opens saved configs verbatim and reuses matching open handles', async () => {
    const store = useSerialStore()
    const config = sampleConfig('/dev/profile', 38400, 7, '2', 'odd', 'xon_xoff', 16)
    serialServiceMock.openPort.mockResolvedValueOnce(sampleHandle('handle-config', config))

    const id = await store.openConfig(config)

    expect(id).toBe('handle-config')
    expect(serialServiceMock.openPort).toHaveBeenCalledWith({ Config: config })
    expect(store.activePortId).toBe('handle-config')
    expect(store.handles.get('handle-config')?.Config).toEqual(config)

    store.error = 'stale error'
    const reusedId = await store.openConfig(sampleConfig('/dev/profile', 115200))

    expect(reusedId).toBe('handle-config')
    expect(serialServiceMock.openPort).toHaveBeenCalledTimes(1)
    expect(store.error).toBeNull()
  })

  it('records backend failures for open close and reset operations', async () => {
    const store = useSerialStore()
    serialServiceMock.openPort.mockRejectedValueOnce(new Error('open denied'))

    await expect(store.openPort('/dev/tty.fail', 9600)).rejects.toThrow('open denied')
    expect(store.error).toBe('open denied')

    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a'), { RxBytes: 4, TxBytes: 5 }))
    serialServiceMock.closePort.mockRejectedValueOnce(undefined)

    await expect(store.closePort('handle-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to close port')
    expect(store.handles.has('handle-1')).toBe(true)

    serialServiceMock.resetCounters.mockRejectedValueOnce(new Error('reset denied'))

    await expect(store.resetCounters('handle-1')).rejects.toThrow('reset denied')
    expect(store.error).toBe('reset denied')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('logs stats polling failures without dropping local handle state', async () => {
    const pollError = new Error('poll failed')
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      const store = useSerialStore()
      serialServiceMock.listPorts.mockRejectedValueOnce(pollError)

      await store.openPort('/dev/tty.usbserial', 9600)
      await vi.advanceTimersByTimeAsync(5000)

      expect(consoleError).toHaveBeenCalledWith('Stats polling failed:', pollError)
      expect(store.handles.has('handle-1')).toBe(true)
      expect(store.activePortId).toBe('handle-1')
    } finally {
      consoleError.mockRestore()
    }
  })

  it('closes ports, reassigns the active port, and stops polling on the last close', async () => {
    const store = useSerialStore()
    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a')))
    store.handles.set('handle-2', sampleHandle('handle-2', sampleConfig('/dev/b')))
    store.setActivePort('handle-1')
    store.initEventListeners()

    expect(vi.getTimerCount()).toBe(1)

    await store.closePort('handle-1')

    expect(serialServiceMock.closePort).toHaveBeenCalledWith('handle-1')
    expect(store.activePortId).toBe('handle-2')
    expect(store.handles.has('handle-1')).toBe(false)

    await store.closePort('handle-2')

    expect(serialServiceMock.closePort).toHaveBeenCalledWith('handle-2')
    expect(store.activePortId).toBeNull()
    expect(store.handles.size).toBe(0)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('updates port configs and reports unknown or backend failures', async () => {
    const store = useSerialStore()

    await expect(store.updatePortConfig('missing', sampleConfig('/dev/new'))).rejects.toThrow('unknown port handle: missing')

    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a')))
    serialServiceMock.closePort.mockResolvedValueOnce(undefined)
    serialServiceMock.openPort.mockResolvedValueOnce(sampleHandle('handle-2', sampleConfig('/dev/new', 57600, 7, '2', 'even', 'hw_rtscts', 64)))

    const id = await store.updatePortConfig('handle-1', sampleConfig('/dev/new', 57600, 7, '2', 'even', 'hw_rtscts', 64))

    expect(id).toBe('handle-2')
    expect(serialServiceMock.closePort).toHaveBeenCalledWith('handle-1')
    expect(serialServiceMock.openPort).toHaveBeenCalledWith({
      Config: {
        PortName: '/dev/new',
        BaudRate: 57600,
        DataBits: 7,
        StopBits: '2',
        Parity: 'even',
        FlowMode: 'hw_rtscts',
        ReadBufKB: 64,
      },
    })
    expect(store.handles.has('handle-1')).toBe(false)
    expect(store.handles.get('handle-2')?.Config.PortName).toBe('/dev/new')
    expect(store.activePortId).toBe('handle-2')

    store.handles.set('handle-3', sampleHandle('handle-3', sampleConfig('/dev/other')))
    serialServiceMock.closePort.mockRejectedValueOnce(new Error('close failed'))

    await expect(store.updatePortConfig('handle-3', sampleConfig('/dev/other-2'))).rejects.toThrow('close failed')
    expect(store.error).toBe('close failed')
  })

  it('increments counters locally, resets and restores them, and ignores missing handles', async () => {
    const store = useSerialStore()
    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a'), { RxBytes: 1, TxBytes: 2 }))

    store.addRxBytes('missing', 99)
    store.addTxBytes('missing', 99)
    store.addRxBytes('handle-1', 3)
    store.addTxBytes('handle-1', 4)

    expect(store.handles.get('handle-1')).toEqual(expect.objectContaining({
      RxBytes: 4,
      TxBytes: 6,
    }))

    await store.resetCounters('handle-1')
    expect(serialServiceMock.resetCounters).toHaveBeenCalledWith('handle-1')
    expect(store.handles.get('handle-1')).toEqual(expect.objectContaining({
      RxBytes: 0,
      TxBytes: 0,
    }))

    await store.restoreCounters('handle-1', 9, 8)
    expect(serialServiceMock.restoreCounters).toHaveBeenCalledWith('handle-1', 9, 8)
    expect(store.handles.get('handle-1')).toEqual(expect.objectContaining({
      RxBytes: 9,
      TxBytes: 8,
    }))

    serialServiceMock.restoreCounters.mockRejectedValueOnce(new Error('restore failed'))
    await expect(store.restoreCounters('handle-1', 1, 2)).rejects.toThrow('restore failed')
    expect(store.error).toBe('restore failed')
  })

  it('closes all ports, clears local handles, and stops cleanup polling', async () => {
    const store = useSerialStore()
    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a')))
    store.handles.set('handle-2', sampleHandle('handle-2', sampleConfig('/dev/b')))
    store.setActivePort('handle-1')
    store.initEventListeners()

    expect(vi.getTimerCount()).toBe(1)

    await store.closeAllPorts()

    expect(serialServiceMock.closePort).toHaveBeenNthCalledWith(1, 'handle-1')
    expect(serialServiceMock.closePort).toHaveBeenNthCalledWith(2, 'handle-2')
    expect(store.handles.size).toBe(0)
    expect(store.activePortId).toBeNull()
    expect(vi.getTimerCount()).toBe(0)

    store.handles.set('handle-3', sampleHandle('handle-3', sampleConfig('/dev/c')))
    store.setActivePort('handle-3')
    store.clearLocalHandles()
    store.cleanup()

    expect(store.handles.size).toBe(0)
    expect(store.activePortId).toBeNull()
  })

  it('handles inactive branches, missing handles, and fallback backend errors', async () => {
    const store = useSerialStore()
    expect(store.activeHandle).toBeNull()

    await store.resetCounters('missing')
    await store.restoreCounters('missing', 1, 2)
    expect(serialServiceMock.resetCounters).not.toHaveBeenCalled()
    expect(serialServiceMock.restoreCounters).not.toHaveBeenCalled()

    serialServiceMock.openPort.mockRejectedValueOnce(undefined)
    await expect(store.openConfig(sampleConfig('/dev/fail'))).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to open port')

    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a'), { RxBytes: 1, TxBytes: 2 }))
    serialServiceMock.resetCounters.mockRejectedValueOnce(undefined)
    await expect(store.resetCounters('handle-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to reset counters')

    serialServiceMock.restoreCounters.mockRejectedValueOnce(undefined)
    await expect(store.restoreCounters('handle-1', 5, 6)).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to restore counters')

    serialServiceMock.closePort.mockRejectedValueOnce(undefined)
    await expect(store.updatePortConfig('handle-1', sampleConfig('/dev/b'))).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to update port config')
  })

  it('keeps active handle when closing another port and ignores unknown poll handles', async () => {
    const store = useSerialStore()
    store.handles.set('handle-1', sampleHandle('handle-1', sampleConfig('/dev/a')))
    store.handles.set('handle-2', sampleHandle('handle-2', sampleConfig('/dev/b')))
    store.setActivePort('handle-1')
    serialServiceMock.listPorts.mockResolvedValue([
      sampleHandle('unknown', sampleConfig('/dev/unknown'), { RxBytes: 7, TxBytes: 8 }),
    ])

    store.initEventListeners()
    store.initEventListeners()
    expect(vi.getTimerCount()).toBe(1)

    await vi.advanceTimersByTimeAsync(5000)
    expect(store.handles.has('unknown')).toBe(false)

    await store.closePort('handle-2')
    expect(store.activePortId).toBe('handle-1')

    store.clearLocalHandles()
    store.handles.set('closed', sampleHandle('closed', sampleConfig('/dev/closed'), { IsOpen: false }))
    store.initEventListeners()
    expect(vi.getTimerCount()).toBe(0)
  })
})

function sampleConfig(
  portName: string,
  baudRate = 115200,
  dataBits = 8,
  stopBits = '1',
  parity = 'none',
  flowMode = 'none',
  readBufKB = 32,
): SerialConfig {
  return {
    PortName: portName,
    BaudRate: baudRate,
    DataBits: dataBits,
    StopBits: stopBits,
    Parity: parity,
    FlowMode: flowMode,
    ReadBufKB: readBufKB,
  }
}

function sampleHandle(
  id: string,
  config: SerialConfig,
  overrides: Partial<Pick<HandleStatus, 'RxBytes' | 'TxBytes' | 'IsOpen'>> = {},
): HandleStatus {
  return {
    ID: id,
    Config: config,
    IsOpen: overrides.IsOpen ?? true,
    RxBytes: overrides.RxBytes ?? 0,
    TxBytes: overrides.TxBytes ?? 0,
  }
}
