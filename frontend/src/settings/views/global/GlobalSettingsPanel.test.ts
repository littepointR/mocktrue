import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GlobalSettingsPanel from './GlobalSettingsPanel.vue'
import { useWorkspaceFileStore } from '../../../workspace/stores/workspaceFileStore'
import { __resetRegistryForTest, useRegistry } from '../../../core/registry'

vi.mock('naive-ui', () => ({
  NAlert: { props: ['type', 'closable'], template: '<div><slot /></div>' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NInputGroup: { template: '<div><slot /></div>' },
  NSelect: {
    props: {
      value: null,
      options: { type: Array, default: () => [] },
      filterable: Boolean,
      virtualScroll: { type: Boolean, default: true },
      placeholder: String,
    },
    emits: ['update:value'],
    template: `
      <select
        :value="value"
        :data-filterable="filterable ? 'true' : 'false'"
        :data-virtual-scroll="virtualScroll === false ? 'false' : 'true'"
        :data-placeholder="placeholder || ''"
        @change="$emit('update:value', $event.target.value)"
      >
        <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
    `,
  },
  NSpace: { template: '<div><slot /></div>' },
  NInput: {
    props: ['value'],
    emits: ['update:value'],
    template: '<input :value="value" @input="$emit(\'update:value\', $event.target.value)" />',
  },
  NButton: {
    props: ['disabled', 'loading'],
    emits: ['click'],
    template: '<button type="button" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
  },
}))

describe('GlobalSettingsPanel global settings and examples', () => {
  beforeEach(() => {
    localStorage.clear()
    __resetRegistryForTest()
    setActivePinia(createPinia())
  })

  it('does not expose global workspace file save or import actions', () => {
    const wrapper = mount(GlobalSettingsPanel)

    expect(wrapper.text()).not.toContain('配置文件路径')
    expect(wrapper.text()).not.toContain('导入')
    expect(wrapper.text()).not.toContain('导出副本')
    expect(wrapper.findAll('button').map(button => button.text())).not.toContain('保存')
    expect(wrapper.findAll('button').map(button => button.text())).not.toContain('另存为')
  })

  it('loads the selected example workspace from settings', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ graphIds: ['graph-1'], activeGraphId: 'graph-1' })

    const wrapper = mount(GlobalSettingsPanel)
    await wrapper.findAll('select')[1].setValue('monitor-demo')
    await wrapper.findAll('button').find(button => button.text() === '加载示例')?.trigger('click')

    expect(loadDemo).toHaveBeenCalledWith('monitor-demo')
  })

  it('updates the selected example when the dropdown changes', async () => {
    const workspace = useWorkspaceFileStore()
    const wrapper = mount(GlobalSettingsPanel)
    const exampleSelect = wrapper.findAll('select')[1]

    await exampleSelect.setValue('modbus-demo')
    expect(workspace.selectedDemoId).toBe('modbus-demo')

    await exampleSelect.setValue('fecbus-demo')
    expect(workspace.selectedDemoId).toBe('fecbus-demo')
  })

  it('makes the remote serial demo discoverable in the settings example picker', () => {
    const wrapper = mount(GlobalSettingsPanel)
    const exampleSelect = wrapper.findAll('select')[1]
    const options = exampleSelect.findAll('option')

    expect(options.some(option => option.attributes('value') === 'remote-serial-demo' && option.text() === '远端串口演示')).toBe(true)
    expect(exampleSelect.attributes('data-filterable')).toBe('true')
    expect(exampleSelect.attributes('data-virtual-scroll')).toBe('false')
    expect(exampleSelect.attributes('data-placeholder')).toContain('搜索')
  })

  it('shows selected demo purpose, hardware requirement, and docs path before loading', async () => {
    const wrapper = mount(GlobalSettingsPanel)

    expect(wrapper.text()).toContain('展示虚拟串口和打开串口操作的基础配置。')
    expect(wrapper.text()).toContain('无需外接硬件')
    expect(wrapper.text()).toContain('docs/examples.md#serial-open-demo')

    await wrapper.findAll('select')[1].setValue('remote-serial-demo')

    expect(wrapper.text()).toContain('展示一个图内 raw TCP server/client 远端串口链路')
    expect(wrapper.text()).toContain('docs/examples.md#remote-serial-demo')
  })

  it('renders optional example metadata for hardware-backed demos without docs links', () => {
    const workspace = useWorkspaceFileStore()
    vi.spyOn(workspace, 'listDemos').mockReturnValue([
      ...workspace.listDemos(),
      {
        id: 'hardware-demo',
        title: '硬件串口演示',
        description: '需要连接物理设备。',
        difficulty: 'advanced',
        requiresHardware: true,
      },
    ])
    workspace.selectedDemoId = 'hardware-demo'

    const wrapper = mount(GlobalSettingsPanel)

    expect(wrapper.text()).toContain('需要连接物理设备。')
    expect(wrapper.text()).toContain('需要外接硬件')
    expect(wrapper.text()).toContain('难度：高级')
    expect(wrapper.find('[data-testid="selected-demo-docs-path"]').exists()).toBe(false)
  })

  it('keeps the load action idle when no example is selected', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo')
    workspace.selectedDemoId = ''

    const wrapper = mount(GlobalSettingsPanel)

    expect(wrapper.find('[data-testid="selected-demo-details"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="load-demo"]').attributes('disabled')).toBeDefined()
    await (wrapper.vm as any).loadExampleWorkspace()
    expect(loadDemo).not.toHaveBeenCalled()
  })

  it('keeps the selected example after switching away from settings', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ graphIds: ['graph-1'], activeGraphId: 'graph-1' })

    const first = mount(GlobalSettingsPanel)
    await first.findAll('select')[1].setValue('bridge-demo')
    first.unmount()

    const second = mount(GlobalSettingsPanel)
    await second.findAll('button').find(button => button.text() === '加载示例')?.trigger('click')

    expect(workspace.selectedDemoId).toBe('bridge-demo')
    expect(loadDemo).toHaveBeenCalledWith('bridge-demo')
  })

  it('switches to the serial content area after loading a demo workspace', async () => {
    const registry = useRegistry()
    registry.register({ id: 'serial', activity: { icon: 'serial', title: '串口调试' }, views: [] })
    registry.register({
      id: 'settings',
      activity: { icon: 'settings', title: '设置' },
      views: [{ id: 'settings.global', title: '全局设置', component: 'settings/GlobalSettings' }],
    })
    registry.setActive('settings')
    const workspace = useWorkspaceFileStore()
    vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ graphIds: ['graph-1'], activeGraphId: 'graph-1' })

    const wrapper = mount(GlobalSettingsPanel)
    await wrapper.findAll('button').find(button => button.text() === '加载示例')?.trigger('click')

    expect(registry.active.value).toBe('serial')
  })

  it('explains that examples open as read-only graph tabs', () => {
    const wrapper = mount(GlobalSettingsPanel)

    expect(wrapper.text()).toContain('只读拓扑图标签页')
    expect(wrapper.text()).toContain('示例配置')
  })
})
