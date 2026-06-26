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
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    ;(document as { elementFromPoint?: Document['elementFromPoint'] }).elementFromPoint = undefined
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

  it('activates the dragged tab when pointer movement stays under the drag threshold', async () => {
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
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const target = wrapper.find('[data-editor-group-id]')
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => target.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 1,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', {
      pointerId: 1,
      clientX: 12,
      clientY: 13,
    }))
    await nextTick()

    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.activeByGroup[workspace.editorLayout.id]).toBe('graph:graph-2')
    }
    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('moves a dragged tab into another group when dropped on tab chrome', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
        defaultSerialGraphDocument('graph-3', '拓扑 C'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')
    graph.renameGraph('graph-3', '拓扑 C')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['graph:graph-1', 'graph:graph-2'] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-3'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'graph:graph-1', 'group-right': 'graph:graph-3' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetGroup = wrapper.find('[data-editor-group-id="group-right"]')
    const targetTabs = targetGroup.find('.editor-tabs')
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetTabs.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 1,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointermove', {
      pointerId: 1,
      clientX: 60,
      clientY: 12,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', {
      pointerId: 1,
      clientX: 60,
      clientY: 12,
    }))
    await nextTick()

    expect(workspace.editorLayout).toMatchObject({
      type: 'split',
      children: [
        { id: 'group-left', tabs: ['graph:graph-1'] },
        { id: 'group-right', tabs: ['graph:graph-3', 'graph:graph-2'] },
      ],
    })
    expect(workspace.activeByGroup).toMatchObject({
      'group-left': 'graph:graph-1',
      'group-right': 'graph:graph-2',
    })
    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('splits a dragged tab into a new group when dropped on an editor edge', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
        defaultSerialGraphDocument('graph-3', '拓扑 C'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')
    graph.renameGraph('graph-3', '拓扑 C')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['graph:graph-1', 'graph:graph-2'] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-3'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'graph:graph-1', 'group-right': 'graph:graph-3' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const layout = wrapper.find('.editor-layout')
    const targetGroup = wrapper.find('[data-editor-group-id="group-right"]')
    vi.spyOn(layout.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 0, top: 0, width: 300, height: 200 }))
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 120, top: 20, width: 160, height: 120 }))
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetGroup.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 1,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointermove', {
      pointerId: 1,
      clientX: 190,
      clientY: 24,
    }))
    await nextTick()

    expect(wrapper.find('.editor-drop-preview').attributes('data-edge')).toBe('top')

    window.dispatchEvent(pointerTestEvent('pointerup', {
      pointerId: 1,
      clientX: 190,
      clientY: 24,
    }))
    await nextTick()

    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      const nested = workspace.editorLayout.children[1]
      expect(nested.type).toBe('split')
      if (nested.type === 'split') {
        expect(nested.direction).toBe('vertical')
        expect(nested.children[0]).toMatchObject({ type: 'group', tabs: ['graph:graph-2'] })
        expect(nested.children[1]).toMatchObject({ type: 'group', id: 'group-right', tabs: ['graph:graph-3'] })
        const newGroup = nested.children[0]
        if (newGroup.type === 'group') {
          expect(workspace.activeByGroup[newGroup.id]).toBe('graph:graph-2')
        }
      }
    }
    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('focuses a missing active graph tab back into the first split group', async () => {
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

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-restored',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: [] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-2'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': null, 'group-right': 'graph:graph-2' })

    await wrapper.setProps({ activeViewId: null, activeViewVersion: 1 })
    await nextTick()

    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      expect(workspace.editorLayout.children[0]).toMatchObject({ id: 'group-left', tabs: ['graph:graph-1'] })
    }
    expect(workspace.activeByGroup['group-left']).toBe('graph:graph-1')
  })

  it('ignores non-primary and mismatched pointer tab drags', async () => {
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
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => null),
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 1,
      pointerId: 1,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1, clientX: 30, clientY: 30 }))
    await nextTick()
    expect(graph.activeGraphId).toBe('graph-1')

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 2,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 3, clientX: 80, clientY: 30 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 3, clientX: 80, clientY: 30 }))
    window.dispatchEvent(pointerTestEvent('pointercancel', { pointerId: 2, clientX: 10, clientY: 10 }))
    await nextTick()

    expect(graph.activeGraphId).toBe('graph-1')
  })

  it('updates drop previews for editor edges and clears invalid targets', async () => {
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
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const layout = wrapper.find('.editor-layout')
    const targetGroup = wrapper.find('[data-editor-group-id]')
    vi.spyOn(layout.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 0, top: 0, width: 300, height: 200 }))
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 100, top: 50, width: 100, height: 80 }))
    const elementFromPoint = vi.fn<() => Element | null>(() => targetGroup.element)
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: elementFromPoint,
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 4,
      clientX: 140,
      clientY: 80,
    }))

    for (const move of [
      { clientX: 101, clientY: 80, edge: 'left' },
      { clientX: 198, clientY: 80, edge: 'right' },
      { clientX: 150, clientY: 128, edge: 'bottom' },
      { clientX: 150, clientY: 80, edge: 'center' },
    ] as const) {
      window.dispatchEvent(pointerTestEvent('pointermove', {
        pointerId: 4,
        clientX: move.clientX,
        clientY: move.clientY,
      }))
      await nextTick()
      expect(wrapper.find('.editor-drop-preview').attributes('data-edge')).toBe(move.edge)
    }

    elementFromPoint.mockReturnValueOnce(null)
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 4, clientX: 20, clientY: 20 }))
    await nextTick()

    expect(wrapper.find('.editor-drop-preview').exists()).toBe(false)
    window.dispatchEvent(pointerTestEvent('pointercancel', { pointerId: 4, clientX: 20, clientY: 20 }))
  })

  it('splits a tab out of a single group on the left edge', async () => {
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
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetGroup = wrapper.find('[data-editor-group-id]')
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 0, top: 0, width: 200, height: 120 }))
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetGroup.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 5,
      clientX: 80,
      clientY: 60,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 5, clientX: 2, clientY: 60 }))
    await nextTick()

    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      expect(workspace.editorLayout.direction).toBe('horizontal')
      expect(workspace.editorLayout.children[0]).toMatchObject({ type: 'group', tabs: ['graph:graph-2'] })
      expect(workspace.editorLayout.children[1]).toMatchObject({ type: 'group', tabs: ['graph:graph-1'] })
    }
    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('inserts a split group next to a direct target when the split direction matches', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
        defaultSerialGraphDocument('graph-3', '拓扑 C'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')
    graph.renameGraph('graph-3', '拓扑 C')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['graph:graph-1', 'graph:graph-2'] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-3'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'graph:graph-1', 'group-right': 'graph:graph-3' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetGroup = wrapper.find('[data-editor-group-id="group-right"]')
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 100, top: 0, width: 200, height: 120 }))
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetGroup.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 6,
      clientX: 20,
      clientY: 40,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 6, clientX: 298, clientY: 40 }))
    await nextTick()

    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      expect(workspace.editorLayout.direction).toBe('horizontal')
      expect(workspace.editorLayout.children).toHaveLength(3)
      expect(workspace.editorLayout.children[0]).toMatchObject({ id: 'group-left', tabs: ['graph:graph-1'] })
      expect(workspace.editorLayout.children[1]).toMatchObject({ id: 'group-right', tabs: ['graph:graph-3'] })
      expect(workspace.editorLayout.children[2]).toMatchObject({ type: 'group', tabs: ['graph:graph-2'] })
    }
    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('ignores non-serial focus changes and non-graph layout events', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')

    const wrapper = mount(SerialView, {
      props: { activeViewId: 'workspace.files', activeViewVersion: 0 },
      global: {
        stubs: {
          ...viewStubs,
          EditorLayoutNode: {
            template: `
              <div data-editor-group-id="group-main">
                <button data-testid="activate-legacy" @click="$emit('set-active-tab', 'group-main', 'legacy-tab')" />
                <button data-testid="close-legacy" @click="$emit('close-tab', 'legacy-tab')" />
              </div>
            `,
          },
        },
      },
    })
    await nextTick()

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({ type: 'group', id: 'group-main', tabs: [] })
    workspace.setActiveByGroup({ 'group-main': null })

    await wrapper.setProps({ activeViewId: 'workspace.files', activeViewVersion: 1 })
    await nextTick()
    expect(workspace.editorLayout).toMatchObject({ id: 'group-main', tabs: [] })

    await wrapper.find('[data-testid="activate-legacy"]').trigger('click')
    expect(workspace.activeByGroup['group-main']).toBe('legacy-tab')
    expect(graph.activeGraphId).toBe('graph-1')

    await wrapper.find('[data-testid="close-legacy"]').trigger('click')
    expect(graph.graphList.map(item => item.id)).toEqual(['graph-1'])
  })

  it('handles invalid drop targets and same-group center drops', async () => {
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
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const tab = wrapper.find('[data-testid="editor-tab-graph-2"]')
    const elementFromPoint = vi.fn<() => Element | null>(() => null)
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: elementFromPoint,
    })

    tab.element.dispatchEvent(pointerTestEvent('pointerdown', { button: 0, pointerId: 7, clientX: 10, clientY: 10 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 7, clientX: 80, clientY: 80 }))
    await nextTick()
    expect(graph.activeGraphId).toBe('graph-1')

    const emptyGroupId = document.createElement('div')
    emptyGroupId.setAttribute('data-editor-group-id', '')
    elementFromPoint.mockReturnValue(emptyGroupId)
    tab.element.dispatchEvent(pointerTestEvent('pointerdown', { button: 0, pointerId: 8, clientX: 10, clientY: 10 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 8, clientX: 80, clientY: 80 }))
    await nextTick()
    expect(graph.activeGraphId).toBe('graph-1')

    const targetTabs = wrapper.find('.editor-tabs')
    elementFromPoint.mockReturnValue(targetTabs.element)
    tab.element.dispatchEvent(pointerTestEvent('pointerdown', { button: 0, pointerId: 9, clientX: 10, clientY: 10 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 9, clientX: 80, clientY: 80 }))
    await nextTick()

    expect(graph.activeGraphId).toBe('graph-2')
  })

  it('normalizes stale nested layouts while preserving valid graph groups', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
        defaultSerialGraphDocument('graph-3', '拓扑 C'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')
    graph.renameGraph('graph-3', '拓扑 C')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'outer',
      direction: 'horizontal',
      children: [
        {
          type: 'split',
          id: 'empty-after-filter',
          direction: 'vertical',
          children: [{ type: 'group', id: 'legacy-only', tabs: ['serial.open'] }],
        },
        {
          type: 'split',
          id: 'single-after-filter',
          direction: 'vertical',
          children: [
            { type: 'group', id: 'legacy-mixed', tabs: ['serial.modbus'] },
            { type: 'group', id: 'group-a', tabs: ['graph:graph-1'] },
          ],
        },
        {
          type: 'split',
          id: 'same-direction',
          direction: 'horizontal',
          children: [
            { type: 'group', id: 'group-b', tabs: ['graph:graph-2'] },
            { type: 'group', id: 'group-c', tabs: ['graph:graph-3'] },
          ],
        },
      ],
    })

    mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      expect(workspace.editorLayout.children).toEqual([
        expect.objectContaining({ id: 'group-a', tabs: ['graph:graph-1'] }),
        expect.objectContaining({ id: 'group-b', tabs: ['graph:graph-2'] }),
        expect.objectContaining({ id: 'group-c', tabs: ['graph:graph-3'] }),
      ])
    }
  })

  it('moves an only tab into another group and collapses the empty source split', async () => {
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

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-left', tabs: ['graph:graph-2'] },
        { type: 'group', id: 'group-right', tabs: ['graph:graph-1'] },
      ],
    })
    workspace.setActiveByGroup({ 'group-left': 'graph:graph-2', 'group-right': 'graph:graph-1' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetTabs = wrapper.find('[data-editor-group-id="group-right"] .editor-tabs')
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetTabs.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 10,
      clientX: 10,
      clientY: 10,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 10, clientX: 80, clientY: 20 }))
    await nextTick()

    expect(workspace.editorLayout).toMatchObject({ id: 'group-right', tabs: ['graph:graph-1', 'graph:graph-2'] })
    expect(workspace.activeByGroup['group-right']).toBe('graph:graph-2')
  })

  it('does not clear graph file state when a close handler leaves the graph loaded', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    files.markClean('/tmp/kept.portweave.json', { clean: true }, 'graph-1')
    vi.spyOn(graph, 'removeGraph').mockResolvedValueOnce(undefined)

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs: viewStubs },
    })
    await nextTick()

    await wrapper.find('[data-testid="close-graph-graph-1"]').trigger('click')
    await nextTick()

    expect(graph.graphById('graph-1')).not.toBeNull()
    expect(files.graphState('graph-1')).not.toBeNull()
  })

  it('keeps an only tab in place when dragged onto its own split edge', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '拓扑 A')],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetGroup = wrapper.find('[data-editor-group-id]')
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 0, top: 0, width: 200, height: 120 }))
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetGroup.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-1"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 11,
      clientX: 80,
      clientY: 60,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 11, clientX: 2, clientY: 60 }))
    await nextTick()

    expect(useSerialWorkspaceStore().editorLayout).toMatchObject({ type: 'group', tabs: ['graph:graph-1'] })
  })

  it('recursively inserts an edge split into nested target groups', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [
        defaultSerialGraphDocument('graph-1', '拓扑 A'),
        defaultSerialGraphDocument('graph-2', '拓扑 B'),
        defaultSerialGraphDocument('graph-3', '拓扑 C'),
        defaultSerialGraphDocument('graph-4', '拓扑 D'),
      ],
      activeGraphId: 'graph-1',
    })
    graph.renameGraph('graph-1', '拓扑 A')
    graph.renameGraph('graph-2', '拓扑 B')
    graph.renameGraph('graph-3', '拓扑 C')
    graph.renameGraph('graph-4', '拓扑 D')

    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'split',
      id: 'outer',
      direction: 'vertical',
      children: [
        { type: 'group', id: 'source', tabs: ['graph:graph-1', 'graph:graph-2'] },
        {
          type: 'split',
          id: 'nested',
          direction: 'horizontal',
          children: [
            { type: 'group', id: 'nested-left', tabs: ['graph:graph-3'] },
            { type: 'group', id: 'nested-right', tabs: ['graph:graph-4'] },
          ],
        },
      ],
    })
    workspace.setActiveByGroup({ source: 'graph:graph-1', 'nested-left': 'graph:graph-3', 'nested-right': 'graph:graph-4' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      attachTo: document.body,
      global: { stubs: viewStubs },
    })
    await nextTick()

    const targetGroup = wrapper.find('[data-editor-group-id="nested-left"]')
    vi.spyOn(targetGroup.element, 'getBoundingClientRect').mockReturnValue(rect({ left: 100, top: 0, width: 200, height: 120 }))
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: vi.fn(() => targetGroup.element),
    })

    wrapper.find('[data-testid="editor-tab-graph-2"]').element.dispatchEvent(pointerTestEvent('pointerdown', {
      button: 0,
      pointerId: 12,
      clientX: 20,
      clientY: 40,
    }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 12, clientX: 102, clientY: 40 }))
    await nextTick()

    expect(workspace.editorLayout.type).toBe('split')
    if (workspace.editorLayout.type === 'split') {
      const nested = workspace.editorLayout.children[1]
      expect(nested.type).toBe('split')
      if (nested.type === 'split') {
        expect(nested.children[0]).toMatchObject({ type: 'group', tabs: ['graph:graph-2'] })
        expect(nested.children[1]).toMatchObject({ id: 'nested-left', tabs: ['graph:graph-3'] })
        expect(nested.children[2]).toMatchObject({ id: 'nested-right', tabs: ['graph:graph-4'] })
      }
    }
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

function rect(init: { left: number; top: number; width: number; height: number }): DOMRect {
  return {
    ...init,
    right: init.left + init.width,
    bottom: init.top + init.height,
    x: init.left,
    y: init.top,
    toJSON: () => init,
  } as DOMRect
}

function pointerTestEvent(type: string, init: { button?: number; pointerId: number; clientX?: number; clientY?: number }): Event {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    button: { value: init.button ?? 0 },
    pointerId: { value: init.pointerId },
    clientX: { value: init.clientX ?? 0 },
    clientY: { value: init.clientY ?? 0 },
  })
  return event
}
