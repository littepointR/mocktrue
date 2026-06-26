import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Panel from './Panel.vue'
import Sidebar from './Sidebar.vue'
import EditorGroups from './EditorGroups.vue'
import StatsPanel from '../serial/views/StatsPanel.vue'
import { __resetRegistryForTest, useRegistry } from '../core/registry'
import { useSerialStore } from '../serial/stores/serialStore'

vi.mock('../serial/services/serialService', () => ({
  serialService: {
    resetCounters: vi.fn(async () => undefined),
  },
}))

describe('shell coverage behavior', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    __resetRegistryForTest()
  })

  it('renders the editor placeholder when no module is active', () => {
    const wrapper = mount(EditorGroups, {
      props: { activeViewId: null, activeViewVersion: 0 },
    })

    expect(wrapper.find('.editor-groups__placeholder').text()).toContain('选择左侧模块开始')
  })

  it('renders sidebar views and emits selected view IDs', async () => {
    const wrapper = mount(Sidebar, {
      props: {
        activeId: 'serial',
        activeViewId: 'serial.graph',
        contributions: [
          {
            moduleId: 'serial',
            activity: { icon: 'serial', title: '串口调试' },
            views: [
              { id: 'serial.graph', title: '串口拓扑', component: 'serial/Graph' },
              { id: 'serial.monitor', title: '串口监控', component: 'serial/Monitor' },
            ],
          },
        ],
      },
    })

    const buttons = wrapper.findAll('.sidebar__item')
    expect(wrapper.find('.sidebar__title').text()).toBe('串口调试')
    expect(buttons[0].classes()).toContain('is-active')

    await buttons[1].trigger('click')
    expect(wrapper.emitted('selectView')).toEqual([["serial.monitor"]])
  })

  it('renders the empty sidebar state when no module is active', () => {
    const wrapper = mount(Sidebar, {
      props: { activeId: null, activeViewId: null, contributions: [] },
    })

    expect(wrapper.find('.sidebar__empty').text()).toBe('选择一个模块')
  })

  it('collapses the bottom panel when the close button is clicked', async () => {
    const wrapper = mount(Panel)

    expect(wrapper.find('.panel').isVisible()).toBe(true)
    await wrapper.find('.panel__close').trigger('click')
    expect(wrapper.find('.panel').attributes('style')).toContain('display: none')
  })

  it('renders port byte counters and resets them through the serial store', async () => {
    const registry = useRegistry()
    registry.mergeBackendContributions([{ ModuleID: 'missing' }])

    const store = useSerialStore()
    store.handles.set('port-1', {
      ID: 'port-1',
      Config: { PortName: 'COM1' },
      IsOpen: true,
      RxBytes: 12,
      TxBytes: 34,
    } as any)

    const wrapper = mount(StatsPanel, { props: { handleId: 'port-1' } })

    expect(wrapper.text()).toContain('RX: 12 字节')
    expect(wrapper.text()).toContain('TX: 34 字节')

    await wrapper.find('.stats-panel__reset').trigger('click')
    expect(store.handles.get('port-1')?.RxBytes).toBe(0)
    expect(store.handles.get('port-1')?.TxBytes).toBe(0)
  })

  it('does not render stats for an unknown handle', () => {
    const wrapper = mount(StatsPanel, { props: { handleId: 'missing' } })

    expect(wrapper.find('.stats-panel').exists()).toBe(false)
  })
})
