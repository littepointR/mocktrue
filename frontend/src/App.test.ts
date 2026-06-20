import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('applies the configured light theme to the shell', () => {
    const settings = useSettingsStore()
    settings.updateGlobal({ Theme: 'light' })

    const wrapper = mount(App, {
      global: { stubs },
    })

    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--light')
  })

  it('updates the shell when the system color scheme changes', async () => {
    const media = installSystemThemeMedia(true)
    const settings = useSettingsStore()
    settings.updateGlobal({ Theme: 'system' })

    const wrapper = mount(App, {
      global: { stubs },
    })

    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--light')

    media.setMatches(false)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--dark')
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

  it('toggles the operation sidebar from the active activity icon', async () => {
    const wrapper = mount(App, {
      global: { stubs: interactiveShellStubs },
    })

    await wrapper.find('[data-testid="activity-serial"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(useRegistry().active.value).toBe('serial')
    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(true)

    await wrapper.find('[data-testid="activity-serial"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(useRegistry().active.value).toBe('serial')
    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(false)

    await wrapper.find('[data-testid="activity-serial"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(true)

    await wrapper.find('[data-testid="activity-settings"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(useRegistry().active.value).toBe('settings')
    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(true)
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

const interactiveShellStubs = {
  ...stubs,
  ActivityBar: {
    props: ['contributions', 'activeId'],
    emits: ['select'],
    template: `
      <nav>
        <button
          v-for="item in contributions"
          :key="item.moduleId"
          :data-testid="'activity-' + item.moduleId"
          type="button"
          @click="$emit('select', item.moduleId)"
        >
          {{ item.activity.title }}
        </button>
      </nav>
    `,
  },
  Sidebar: {
    props: ['contributions', 'activeId', 'activeViewId'],
    emits: ['selectView'],
    template: '<aside data-testid="sidebar">{{ activeId }}</aside>',
  },
}

function installSystemThemeMedia(initialMatches: boolean) {
  let matches = initialMatches
  const listeners = new Set<(event: MediaQueryListEvent) => void>()
  const query = '(prefers-color-scheme: light)'
  const media = {
    get matches() {
      return matches
    },
    media: query,
    onchange: null,
    addEventListener: vi.fn((event: string, listener: (event: MediaQueryListEvent) => void) => {
      if (event === 'change') listeners.add(listener)
    }),
    removeEventListener: vi.fn((event: string, listener: (event: MediaQueryListEvent) => void) => {
      if (event === 'change') listeners.delete(listener)
    }),
    addListener: vi.fn((listener: (event: MediaQueryListEvent) => void) => listeners.add(listener)),
    removeListener: vi.fn((listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener)),
    dispatchEvent: vi.fn(),
    setMatches(next: boolean) {
      matches = next
      const event = { matches, media: query } as MediaQueryListEvent
      for (const listener of listeners) listener(event)
    },
  }
  vi.stubGlobal('matchMedia', vi.fn(() => media))
  return media
}
