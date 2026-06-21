import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import SerialView from './SerialView.vue'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'
import { useSerialGraphStore } from '../stores/graphStore'
import { defaultSerialGraphDocument } from '../graph/serialGraph'
import { useWorkspaceFileStore } from '../../workspace/stores/workspaceFileStore'

describe('SerialView graph workspace layout', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
      global: { stubs },
    })
    first.unmount()
    const second = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await nextTick()

    expect(second.find('.layout-json').text()).toContain('split-1')
    expect(second.find('.layout-json').text()).toContain('group-left')
    expect(second.find('.layout-json').text()).toContain('graph:graph-1')
    expect(second.find('.layout-json').text()).toContain('graph:graph-2')
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

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await nextTick()

    expect(wrapper.find('.tabs-json').text()).toContain('"id":"graph:graph-1"')
    expect(wrapper.find('.tabs-json').text()).toContain('"id":"graph:graph-2"')
    expect(wrapper.find('.tabs-json').text()).toContain('"name":"拓扑 A"')
    expect(wrapper.find('.tabs-json').text()).toContain('"name":"拓扑 B"')
    expect(wrapper.find('[data-testid="graph-panel-graph-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="graph-panel-graph-2"]').exists()).toBe(true)

    const layout = useSerialWorkspaceStore().editorLayout
    expect(layout.type).toBe('group')
    if (layout.type === 'group') {
      expect(layout.tabs).toEqual(['graph:graph-1', 'graph:graph-2'])
      expect(useSerialWorkspaceStore().activeByGroup[layout.id]).toBe('graph:graph-2')
    }
  })

  it('shows unsaved state and path metadata on graph editor tabs', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.renameGraph('graph-1', '生产拓扑')
    files.markClean('/tmp/production.mocktrue.json', { clean: true }, 'graph-1')
    graph.addNode('serial.sender')
    files.syncGraphSnapshot('graph-1')

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await nextTick()

    expect(wrapper.find('.tabs-json').text()).toContain('生产拓扑*')
    expect(wrapper.find('.tabs-json').text()).toContain('/tmp/production.mocktrue.json')
  })

  it('filters legacy non-graph tabs from restored layout', async () => {
    const workspace = useSerialWorkspaceStore()
    workspace.setEditorLayout({
      type: 'group',
      id: 'group-main',
      tabs: ['port-1', 'monitor:mon-1', 'modbus:modbus-1', 'fecbus:fec-1', 'graph:graph-1'],
    })
    workspace.setActiveByGroup({ 'group-main': 'modbus:modbus-1' })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await nextTick()

    expect(wrapper.find('.layout-json').text()).toContain('graph:graph-1')
    expect(wrapper.find('.layout-json').text()).not.toContain('port-1')
    expect(wrapper.find('.layout-json').text()).not.toContain('monitor:mon-1')
    expect(wrapper.find('.layout-json').text()).not.toContain('modbus:modbus-1')
    expect(wrapper.find('.layout-json').text()).not.toContain('fecbus:fec-1')
    expect(wrapper.find('.tabs-json').text()).not.toContain('"kind":"serial"')
    expect(wrapper.find('.tabs-json').text()).not.toContain('"kind":"monitor"')
    expect(wrapper.find('.tabs-json').text()).not.toContain('"kind":"modbus"')
    expect(wrapper.find('.tabs-json').text()).not.toContain('"kind":"fecbus"')
  })

  it('focuses the graph tab when legacy serial module view ids are selected', async () => {
    const wrapper = mount(SerialView, {
      props: { activeViewId: 'serial.open', activeViewVersion: 1 },
      global: { stubs },
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

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
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

  it('does not delete the last graph document when its tab close button is clicked', async () => {
    const graph = useSerialGraphStore()
    graph.restoreState({
      graphs: [defaultSerialGraphDocument('graph-1', '唯一拓扑')],
      activeGraphId: 'graph-1',
    })

    const wrapper = mount(SerialView, {
      props: { activeViewId: null, activeViewVersion: 0 },
      global: { stubs },
    })
    await nextTick()

    await wrapper.find('[data-testid="close-graph-graph-1"]').trigger('click')
    await nextTick()

    expect(graph.graphList).toEqual([{ id: 'graph-1', name: '唯一拓扑' }])
    const workspace = useSerialWorkspaceStore()
    expect(workspace.editorLayout.type).toBe('group')
    if (workspace.editorLayout.type === 'group') {
      expect(workspace.editorLayout.tabs).toEqual(['graph:graph-1'])
    }
  })
})

const stubs = {
  EditorLayoutNode: {
    props: ['node', 'activeByGroup', 'tabs'],
    emits: ['setActiveTab', 'closeTab'],
    template: `
      <div>
        <pre class="layout-json">{{ JSON.stringify(node) }} {{ JSON.stringify(activeByGroup) }}</pre>
        <pre class="tabs-json">{{ JSON.stringify(tabs) }}</pre>
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :data-testid="'editor-tab-' + tab.sourceId"
          type="button"
          @click="$emit('setActiveTab', 'group-1', tab.id)"
        >
          {{ tab.name }}
        </button>
        <button
          v-for="tab in tabs"
          :key="'close-' + tab.id"
          :data-testid="'close-graph-' + tab.sourceId"
          type="button"
          @click="$emit('closeTab', tab.id)"
        >
          close
        </button>
        <div
          v-for="tab in tabs"
          :key="'panel-' + tab.id"
          :data-testid="'graph-panel-' + tab.sourceId"
        />
      </div>
    `,
  },
}
