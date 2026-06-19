import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useSettingsStore } from './settingsStore'

describe('settingsStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('persists serial settings locally', () => {
    const store = useSettingsStore()
    store.updateSerial({
      BaudRate: 57600,
      DataBits: 7,
      StopBits: '2',
      Parity: 'even',
      FlowMode: 'hw_rtscts',
      ReadBufKB: 64,
    })

    setActivePinia(createPinia())
    const restored = useSettingsStore()

    expect(restored.serial).toMatchObject({
      BaudRate: 57600,
      DataBits: 7,
      StopBits: '2',
      Parity: 'even',
      FlowMode: 'hw_rtscts',
      ReadBufKB: 64,
    })
  })

  it('drops stale global keys that are not effective settings', () => {
    localStorage.setItem('mocktrue.settings.v1', JSON.stringify({
      global: {
        Theme: 'light',
        MCPEnabled: false,
        MCPEndpoint: 'http://127.0.0.1:1/mcp',
      },
      serial: {},
    }))

    const store = useSettingsStore()

    expect(store.global).toEqual({ Theme: 'light' })
    expect('MCPEnabled' in store.global).toBe(false)
    expect('MCPEndpoint' in store.global).toBe(false)
  })
})
