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
}

interface SettingsSnapshot {
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
