import { defineStore } from 'pinia'
import { ref } from 'vue'

const storageKey = 'mocktrue.settings.v1'

export interface GlobalSettings {
  Theme: 'dark' | 'light' | 'system'
}

export interface SerialModuleSettings {
  BaudRate: number
  DataBits: number
  StopBits: string
  Parity: string
  FlowMode: string
  ReadBufKB: number
  TerminalFontFamily: string
  TerminalFontSize: number
  TextEncoding: string
  EnterString: string
}

export interface SettingsSnapshot {
  global: GlobalSettings
  serial: SerialModuleSettings
}

export const defaultGlobalSettings: GlobalSettings = {
  Theme: 'dark',
}

export const defaultSerialSettings: SerialModuleSettings = {
  BaudRate: 115200,
  DataBits: 8,
  StopBits: '1',
  Parity: 'none',
  FlowMode: 'none',
  ReadBufKB: 32,
  TerminalFontFamily: 'Consolas',
  TerminalFontSize: 14,
  TextEncoding: 'utf-8',
  EnterString: '\n',
}

export const useSettingsStore = defineStore('settings', () => {
  const initial = loadSettings()
  const global = ref<GlobalSettings>(initial.global)
  const serial = ref<SerialModuleSettings>(initial.serial)

  function updateGlobal(next: Partial<GlobalSettings>) {
    global.value = { ...global.value, ...next }
    persist()
  }

  function updateSerial(next: Partial<SerialModuleSettings>) {
    serial.value = { ...serial.value, ...next }
    persist()
  }

  function resetGlobal() {
    global.value = { ...defaultGlobalSettings }
    persist()
  }

  function resetSerial() {
    serial.value = { ...defaultSerialSettings }
    persist()
  }

  function snapshot(): SettingsSnapshot {
    return {
      global: { ...global.value },
      serial: { ...serial.value },
    }
  }

  function replaceSettings(next: Partial<SettingsSnapshot>) {
    global.value = normalizeGlobalSettings(next.global)
    serial.value = { ...defaultSerialSettings, ...(next.serial ?? {}) }
    persist()
  }

  function replaceSerialSettings(next: Partial<SerialModuleSettings> | undefined) {
    serial.value = { ...defaultSerialSettings, ...(next ?? {}) }
    persist()
  }

  function persist() {
    if (typeof localStorage === 'undefined') return
    const snapshot: SettingsSnapshot = {
      global: global.value,
      serial: serial.value,
    }
    localStorage.setItem(storageKey, JSON.stringify(snapshot))
  }

  return {
    global,
    serial,
    updateGlobal,
    updateSerial,
    resetGlobal,
    resetSerial,
    snapshot,
    replaceSettings,
    replaceSerialSettings,
  }
})

function loadSettings(): SettingsSnapshot {
  if (typeof localStorage === 'undefined') {
    return defaults()
  }
  const raw = localStorage.getItem(storageKey)
  if (!raw) {
    return defaults()
  }
  try {
    const parsed = JSON.parse(raw) as Partial<SettingsSnapshot>
    return {
      global: normalizeGlobalSettings(parsed.global),
      serial: { ...defaultSerialSettings, ...(parsed.serial ?? {}) },
    }
  } catch {
    return defaults()
  }
}

function defaults(): SettingsSnapshot {
  return {
    global: { ...defaultGlobalSettings },
    serial: { ...defaultSerialSettings },
  }
}

function normalizeGlobalSettings(value: Partial<GlobalSettings> | undefined): GlobalSettings {
  const theme = value?.Theme
  return {
    Theme: theme === 'light' || theme === 'system' ? theme : defaultGlobalSettings.Theme,
  }
}
