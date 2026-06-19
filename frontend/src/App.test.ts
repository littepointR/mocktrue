import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import App from './App.vue'
import { __resetRegistryForTest, useRegistry } from './core/registry'
import { serialModule } from './serial'
import { settingsModule, useSettingsStore } from './settings'

describe('App settings effects', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    __resetRegistryForTest()
    const registry = useRegistry()
    registry.register(serialModule)
    registry.register(settingsModule)
  })

  it('applies the configured light theme to the shell', () => {
    const settings = useSettingsStore()
    settings.updateGlobal({ Theme: 'light' })

    const wrapper = mount(App, {
      global: { stubs },
    })

    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--light')
  })
})

const stubs = {
  NConfigProvider: { template: '<div><slot /></div>' },
  ActivityBar: { template: '<nav />' },
  Sidebar: { template: '<aside />' },
  EditorGroups: { template: '<main />' },
  Panel: { template: '<section />' },
  StatusBar: { template: '<footer />' },
}
