import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'
import appSource from './App.vue?raw'
import { __resetRegistryForTest, useRegistry } from './core/registry'
import { serialModule } from './serial'
import { settingsModule, useSettingsStore } from './settings'
import { useWorkspaceFileStore } from './workspace/stores/workspaceFileStore'

vi.mock('../bindings/github.com/suyue/mocktrue/internal/core/workspace/service.js', () => ({
  LoadLastWorkspace: vi.fn(async () => ({ Found: false, Path: '', Content: '' })),
  DefaultWorkspacePath: vi.fn(async () => '/tmp/default.mocktrue.json'),
  SaveWorkspace: vi.fn(async () => undefined),
  ReadWorkspace: vi.fn(async () => null),
}))

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

  it('defines shared content theme variables used by Modbus panels', () => {
    expect(appSource).toContain('--app-hover-bg')
    expect(appSource).toContain('--app-accent')
    expect(appSource).toContain('--app-table-header')
  })

  it('marks the status bar dirty after workspace changes', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    useSettingsStore().updateSerial({ BaudRate: 9600 })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('footer').text()).toBe('dirty')
  })

  it('passes the current workspace path to the status bar', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    useWorkspaceFileStore().setPath('/tmp/current-workspace.json')
    await wrapper.vm.$nextTick()

    const status = wrapper.findComponent({ name: 'StatusBar' })

    expect(status.props('configPath')).toBe('/tmp/current-workspace.json')
  })
})

const stubs = {
  NConfigProvider: { template: '<div><slot /></div>' },
  ActivityBar: { template: '<nav />' },
  Sidebar: { template: '<aside />' },
  EditorGroups: { template: '<main />' },
  Panel: { template: '<section />' },
  StatusBar: { name: 'StatusBar', props: ['dirty', 'configPath'], template: '<footer>{{ dirty ? "dirty" : "clean" }}</footer>' },
}
