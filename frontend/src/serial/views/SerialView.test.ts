import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SerialView from './SerialView.vue'
import editorLayoutNodeSource from './EditorLayoutNode.vue?raw'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'
import { useSerialGraphStore } from '../stores/graphStore'
import { defaultSerialGraphDocument } from '../graph/serialGraph'
import { useWorkspaceFileStore } from '../../workspace/stores/workspaceFileStore'

describe('SerialView graph workspace layout', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('uses persisted graph editor layout after remounting', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
      ],
      activeGraphId: 'graph-2',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['graph:graph-1'] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-2'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'graph:graph-1', 'group-right': 'graph:graph-2' })

    const first = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()
    first.unmount()

    const second = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    expect(second.findAll('.editor-group')).toHaveLength(2)
    expect(second.find('[data-editor-group-id="group-left"]').exists()).toBe(true)
    expect(second.find('[data-editor-group-id="group-right"]').exists()).toBe(true)
    expect(second.findAll('.editor-tab__label').map(tab => tab.text())).toEqual(['拓扑 A', '拓扑 B'])
  })

  it('creates one editor graph tab for each graph document', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
      ],
      activeGraphId: 'graph-2',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    expect(wrapper.findAll('.editor-tab__label').map(tab => tab.text())).toEqual(expect.arrayContaining(['拓扑 A', '拓扑 B']))
    expect(wrapper.find('[data-testid="graph-panel-graph-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="graph-panel-graph-2"]').exists()).toBe(true)

    const layout = useSerialWorkspaceStore().editorLayout
    expect(layout.type).toBe('group')
    if (layout.type === 'group') {
      expect(layout.tabs).toEqual(['graph:graph-1', 'graph:graph-2'])
      expect(useSerialWorkspaceStore().activeByGroup[layout.id]).toBe('graph:graph-2')
    }
  })

  it('constrains graph panels to the editor content height so split view panes are visible', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    const panel = wrapper.find('[data-testid="graph-panel-graph-1"]')
    expect(panel.classes()).toContain('editor-group__panel')
    expect(editorLayoutNodeSource).toContain('.editor-group__panel')
    expect(editorLayoutNodeSource).toContain('height: 100%;')
    expect(editorLayoutNodeSource).toContain('overflow: hidden;')
  })

  it('shows unsaved state and path metadata on graph editor tabs', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '生产拓扑')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '生产拓扑')
    files.markClean('/tmp/production.portweave.json', { clean: true }, 'graph-1')
    graph.addNode('serial.sender')
    files.syncGraphSnapshot('graph-1')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    const tab = wrapper.find('[data-testid="editor-tab-graph-1"]')
    expect(tab.find('.editor-tab__label').text()).toBe('生产拓扑*')
    expect(tab.attributes('title')).toContain('/tmp/production.portweave.json')
  })

  it('filters legacy non-graph tabs from restored layout', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'group',
      id: 'group-main',
      tabs: ['port-1', 'monitor:mon-1', 'modbus:modbus-1', 'fecbus:fec-1', 'graph:graph-1'],
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    expect(wrapper.findAll('.editor-tab__label').map(tab => tab.text())).toEqual(['拓扑 A'])
    expect(wrapper.find('[data-testid="editor-tab-graph-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="editor-tab-port-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="editor-tab-monitor-mon-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="editor-tab-modbus-modbus-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="editor-tab-fecbus-fec-1"]').exists()).toBe(false)
  })

  it('focuses the graph tab when legacy serial module view ids are selected', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')

    const wrapper = mount(SerialView, {
      props: { activeViewId: 'serial.open', activeViewVersion: 1 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    expect(wrapper.find('.serial-view__operation-panel').exists()).toBe(false)
    expect(wrapper.find('[data-testid="open-port-stub"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="graph-panel-graph-1"]').exists()).toBe(true)

    await wrapper.setProps({ activeViewId: 'serial.modbus', activeViewVersion: 2 })
    await nextTick()

    expect(wrapper.find('.serial-view__operation-panel').exists()).toBe(false)
    expect(wrapper.find('[data-testid="modbus-stub"]').exists()).toBe(false)
    const layout = useSerialWorkspaceStore().editorLayout
    expect(layout.type).toBe('group')
    if (layout.type === 'group') {
      expect(useSerialWorkspaceStore().activeByGroup[layout.id]).toBe('graph:graph-1')
    }
  })

  it('syncs the active editor tab when the active graph changes', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    graph.setActiveGraph('graph-2')
    await nextTick()

    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.activeByGroup[workspace.editorLayout.id]).toBe('graph:graph-2')
    }

    await wrapper.find('[data-testid="close-graph-graph-2"]').trigger('click')
    await nextTick()

    expect(graph.graphList.map(item => item.id)).toEqual(['graph-1'])
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.activeByGroup[workspace.editorLayout.id]).toBe('graph:graph-1')
    }
  })

  it('closes the last graph editor tab and leaves an empty editor area', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '唯一拓扑')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '唯一拓扑')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    await wrapper.find('[data-testid="close-graph-graph-1"]').trigger('click')
    await nextTick()

    expect(graph.graphList).toEqual([])
    expect(graph.activeGraphId).toBeNull()
    expect(wrapper.find('.serial-view__empty').exists()).toBe(true)
    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.editorLayout.tabs).toEqual([])
      expect(workspace.activeByGroup[workspace.editorLayout.id]).toBeNull()
    }
  })

  it('asks before closing a graph editor tab with unsaved changes', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '未保存拓扑')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '未保存拓扑')
    files.markClean('/tmp/dirty.portweave.json', { clean: true }, 'graph-1')
    graph.addNode('serial.sender')
    files.syncGraphSnapshot('graph-1')
    const confirm = vi.fn()
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true)
    vi.stubGlobal('confirm', confirm)

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    await wrapper.find('[data-testid="close-graph-graph-1"]').trigger('click')
    await nextTick()

    expect(confirm).toHaveBeenCalledWith('当前标签页有未保存的配置，确定关闭吗？')
    expect(graph.graphList).toEqual([{ id: 'graph-1', name: '未保存拓扑' }])

    await wrapper.find('[data-testid="close-graph-graph-1"]').trigger('click')
    await nextTick()

    expect(graph.graphList).toEqual([])
    expect(files.graphState('graph-1')).toBeNull()
  })
})

const viewStubs = {
  SerialGraphPanel: {
    props: ['graphId'],
    template: '<div data-testid="serial-graph-panel-stub" />',
  },
  Splitpanes: {
    template: '<div class="splitpanes"><slot /></div>',
  },
  Pane: {
    template: '<div class="pane"><slot /></div>',
  },
}
