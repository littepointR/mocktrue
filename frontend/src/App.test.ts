import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'
import appSource from './App.vue?raw'
import { __resetRegistryForTest, useRegistry } from './core/registry'
import { serialModule } from './serial'
import { defaultSerialSettings, settingsModule, useSettingsStore } from './settings'
import { useWorkspaceFileStore } from './workspace/stores/workspaceFileStore'
import { useSerialGraphStore } from './serial/stores/graphStore'

const workspaceBindings = vi.hoisted(() => ({
  LoadLastWorkspace: vi.fn(async () => ({ Found: false, Path: '', Content: '' })),
  DefaultWorkspacePath: vi.fn(async () => '/tmp/default.portweave.json'),
  ExportWorkspace: vi.fn(async () => undefined),
  SaveWorkspace: vi.fn(async () => undefined),
  ReadWorkspace: vi.fn(async (path: string) => ({
    Path: path,
    Content: JSON.stringify({
      kind: 'mocktrue.graph.v1',
      settings: {},
      graph: {
        id: 'opened-graph',
        name: '打开拓扑',
        nodes: [],
        edges: [],
        selectedNodeId: null,
        selectedEdgeId: null,
        nodeTabs: [],
        activeNodeTabId: null,
      },
      runtime: { nodeBuffers: {}, nodeFrames: {} },
    }),
  })),
  RememberLastWorkspace: vi.fn(async () => undefined),
  SelectWorkspaceOpenPath: vi.fn(async () => '/tmp/opened.portweave.json'),
  SelectWorkspaceSavePath: vi.fn(async () => '/tmp/save-as.portweave.json'),
}))

vi.mock('../bindings/github.com/littepointR/mocktrue/internal/core/workspace/service.js', () => workspaceBindings)

describe('App settings effects', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
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
    media.setMatches(true)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--light')
    wrapper.unmount()
    expect(media.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
  })

  it('supports legacy system color scheme listeners and removes them on unmount', async () => {
    const media = installLegacySystemThemeMedia(true)
    const settings = useSettingsStore()
    settings.updateGlobal({ Theme: 'system' })

    const wrapper = mount(App, {
      global: { stubs },
    })

    expect(media.addListener).toHaveBeenCalledWith(expect.any(Function))
    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--light')

    media.setMatches(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.app-shell').classes()).toContain('app-shell--dark')

    wrapper.unmount()
    expect(media.removeListener).toHaveBeenCalledWith(expect.any(Function))
  })

  it('falls back to graph snapshot sync when loading the last workspace fails', async () => {
    workspaceBindings.LoadLastWorkspace.mockRejectedValueOnce(new Error('load failed'))
    const files = useWorkspaceFileStore()
    const syncSpy = vi.spyOn(files, 'syncAllGraphSnapshots')

    mount(App, {
      global: { stubs },
    })
    await flushPromises()

    expect(syncSpy).toHaveBeenCalled()
  })

  it('defines shared content theme variables used by Modbus panels', () => {
    expect(appSource).toContain('--app-hover-bg')
    expect(appSource).toContain('--app-accent')
    expect(appSource).toContain('--app-table-header')
  })

  it('does not pass a global dirty flag to the status bar', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    useSerialGraphStore().addNode('serial.sender')
    await wrapper.vm.$nextTick()

    const status = wrapper.findComponent({ name: 'StatusBar' })
    expect(status.props('dirty')).toBeUndefined()
    expect(status.props('configPath')).toBeUndefined()
  })

  it('marks only the active graph tab dirty when serial settings change', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    const settings = useSettingsStore()
    const second = graph.createGraph('第二拓扑')
    await wrapper.vm.$nextTick()
    graph.setActiveGraph('graph-1')
    await wrapper.vm.$nextTick()

    settings.updateSerial({ TerminalFontSize: 23 })
    await wrapper.vm.$nextTick()

    expect(files.isGraphDirty('graph-1')).toBe(true)
    expect(files.isGraphDirty(second.id)).toBe(false)
  })

  it('restores serial settings from the active graph tab when switching tabs', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    const settings = useSettingsStore()
    const second = graph.createGraph('第二拓扑')
    files.markClean('/tmp/first.portweave.json', graphSnapshot('graph-1', '第一拓扑', 15), 'graph-1')
    files.markClean('/tmp/second.portweave.json', graphSnapshot(second.id, '第二拓扑', 21), second.id)

    graph.setActiveGraph(second.id)
    await wrapper.vm.$nextTick()

    expect(settings.serial.TerminalFontSize).toBe(21)

    graph.setActiveGraph('graph-1')
    await wrapper.vm.$nextTick()

    expect(settings.serial.TerminalFontSize).toBe(15)
  })

  it('toggles the operation sidebar from the active activity icon', async () => {
    const wrapper = mount(App, {
      global: { stubs: interactiveShellStubs },
    })

    await wrapper.find('[data-testid="activity-serial"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(useRegistry().active.value).toBe('serial')
    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(true)

    const viewVersion = useRegistry().activeViewVersion.value
    await wrapper.find('[data-testid="sidebar-select-view"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(useRegistry().activeView.value).toBe('serial.graph')
    expect(useRegistry().activeViewVersion.value).toBe(viewVersion + 1)

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

  it('places app actions in the titlebar safe area and omits custom window controls', () => {
    const wrapper = mount(App, {
      global: { stubs },
    })

    const titlebar = wrapper.find('.app-shell__titlebar')

    expect(titlebar.exists()).toBe(true)
    expect(titlebar.find('[title="新建拓扑图"]').exists()).toBe(true)
    expect(titlebar.find('[title="打开拓扑图配置文件"]').exists()).toBe(true)
    expect(titlebar.find('[title="保存当前拓扑图"]').exists()).toBe(true)
    expect(titlebar.find('[title="将当前拓扑图另存为文件"]').exists()).toBe(true)
    expect(wrapper.find('.app-shell__window-controls').exists()).toBe(false)
    expect(wrapper.find('.app-shell__window-button').exists()).toBe(false)
  })

  it('runs graph file actions from the titlebar', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()

    await wrapper.find('[data-testid="app-titlebar-new-graph"]').trigger('click')
    expect(useSerialGraphStore().activeGraph?.name).toBe('拓扑图 2')

    await wrapper.find('[data-testid="app-titlebar-open-graph"]').trigger('click')
    await flushPromises()
    expect(workspaceBindings.SelectWorkspaceOpenPath).toHaveBeenCalled()
    expect(workspaceBindings.ReadWorkspace).toHaveBeenCalledWith('/tmp/opened.portweave.json')
    expect(useSerialGraphStore().activeGraph?.name).toBe('打开拓扑')

    useSerialGraphStore().renameGraph('opened-graph', '保存拓扑')
    await wrapper.find('[data-testid="app-titlebar-save-graph"]').trigger('click')
    await flushPromises()
    expect(workspaceBindings.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/opened.portweave.json',
      expect.stringContaining('保存拓扑')
    )

    await wrapper.find('[data-testid="app-titlebar-save-as-graph"]').trigger('click')
    await flushPromises()
    expect(workspaceBindings.SelectWorkspaceSavePath).toHaveBeenCalledWith('/tmp/opened.portweave.json')
    expect(workspaceBindings.SaveWorkspace).toHaveBeenLastCalledWith(
      '/tmp/save-as.portweave.json',
      expect.stringContaining('保存拓扑')
    )
  })

  it('skips canceled open and save actions when no graph is available', async () => {
    workspaceBindings.SelectWorkspaceOpenPath.mockResolvedValueOnce('')
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()
    const graph = useSerialGraphStore() as any
    graph.graphs = []
    graph.activeGraphId = null

    await wrapper.find('[data-testid="app-titlebar-open-graph"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="app-titlebar-save-graph"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="app-titlebar-save-as-graph"]').trigger('click')
    await flushPromises()

    expect(workspaceBindings.ReadWorkspace).not.toHaveBeenCalled()
    expect(workspaceBindings.SaveWorkspace).not.toHaveBeenCalled()
    expect(workspaceBindings.SelectWorkspaceSavePath).not.toHaveBeenCalled()
  })

  it('saves the first graph when no active graph id is set', async () => {
    const wrapper = mount(App, {
      global: { stubs },
    })
    await flushPromises()
    const graph = useSerialGraphStore() as any
    graph.activeGraphId = null

    await wrapper.find('[data-testid="app-titlebar-save-as-graph"]').trigger('click')
    await flushPromises()

    expect(workspaceBindings.SelectWorkspaceSavePath).toHaveBeenCalledWith('')
    expect(workspaceBindings.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/save-as.portweave.json',
      expect.stringContaining('graph-1')
    )
  })
})

const stubs = {
  NConfigProvider: { template: '<div><slot /></div>' },
  ActivityBar: { template: '<nav />' },
  Sidebar: { template: '<aside />' },
  EditorGroups: { template: '<main />' },
  Panel: { template: '<section />' },
  StatusBar: { name: 'StatusBar', template: '<footer />' },
}

function graphSnapshot(id: string, name: string, terminalFontSize: number) {
  return {
    kind: 'mocktrue.graph.v1',
    settings: {
      serial: {
        ...defaultSerialSettings,
        TerminalFontSize: terminalFontSize,
      },
    },
    graph: {
      id,
      name,
      nodes: [],
      edges: [],
      selectedNodeId: null,
      selectedEdgeId: null,
      nodeTabs: [],
      activeNodeTabId: null,
    },
    runtime: { nodeBuffers: {}, nodeFrames: {} },
  }
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
    template: `
      <aside data-testid="sidebar">
        {{ activeId }}
        <button type="button" data-testid="sidebar-select-view" @click="$emit('selectView', 'serial.graph')">
          select view
        </button>
      </aside>
    `,
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

function installLegacySystemThemeMedia(initialMatches: boolean) {
  let matches = initialMatches
  const listeners = new Set<(event: MediaQueryListEvent) => void>()
  const query = '(prefers-color-scheme: light)'
  const media = {
    get matches() {
      return matches
    },
    media: query,
    onchange: null,
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
