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
      TerminalFontFamily: 'Monaco',
      TerminalFontSize: 16,
      TextEncoding: 'gbk',
      EnterString: '\r\n',
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
      TerminalFontFamily: 'Monaco',
      TerminalFontSize: 16,
      TextEncoding: 'gbk',
      EnterString: '\r\n',
    })
  })

  it('fills new serial display settings when restoring old settings', () => {
    localStorage.setItem('mocktrue.settings.v1', JSON.stringify({
      global: { Theme: 'dark' },
      serial: {
        BaudRate: 9600,
      },
    }))

    const store = useSettingsStore()

    expect(store.serial).toMatchObject({
      BaudRate: 9600,
      TerminalFontFamily: 'Consolas',
      TerminalFontSize: 14,
      TextEncoding: 'utf-8',
      EnterString: '\n',
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

  it('replaces all settings from a workspace snapshot', () => {
    const store = useSettingsStore()

    store.replaceSettings({
      global: { Theme: 'light' },
      serial: {
        BaudRate: 38400,
        DataBits: 7,
        StopBits: '2',
        Parity: 'odd',
        FlowMode: 'sw_xonxoff',
        ReadBufKB: 128,
        TerminalFontFamily: 'Monaco',
        TerminalFontSize: 18,
        TextEncoding: 'gbk',
        EnterString: '\r\n',
      },
    })

    expect(store.global.Theme).toBe('light')
    expect(store.serial).toMatchObject({
      BaudRate: 38400,
      TerminalFontFamily: 'Monaco',
      EnterString: '\r\n',
    })
  })
})
