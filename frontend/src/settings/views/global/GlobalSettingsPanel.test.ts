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
    props: ['value', 'options'],
    emits: ['update:value'],
    template: `
      <select :value="value" @change="$emit('update:value', $event.target.value)">
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

describe('GlobalSettingsPanel workspace file actions', () => {
  beforeEach(() => {
    localStorage.clear()
    __resetRegistryForTest()
    setActivePinia(createPinia())
  })

  it('keeps the path input wired to the workspace file store', async () => {
    const workspace = useWorkspaceFileStore()
    const wrapper = mount(GlobalSettingsPanel)

    await wrapper.find('input').setValue('/tmp/session.mocktrue.json')

    expect(workspace.currentPath).toBe('/tmp/session.mocktrue.json')
  })

  it('selects a workspace file from the path button', async () => {
    const workspace = useWorkspaceFileStore()
    const selectOpenPath = vi.spyOn(workspace, 'selectOpenPath').mockResolvedValue('/tmp/session.mocktrue.json')
    const wrapper = mount(GlobalSettingsPanel)

    await wrapper.findAll('button').find(button => button.text() === '选择')?.trigger('click')

    expect(selectOpenPath).toHaveBeenCalled()
  })

  it('runs the four workspace file actions without requiring a prefilled path', async () => {
    const workspace = useWorkspaceFileStore()
    const save = vi.spyOn(workspace, 'save').mockResolvedValue('/tmp/session.mocktrue.json')
    const saveAs = vi.spyOn(workspace, 'saveAs').mockResolvedValue('/tmp/session-as.mocktrue.json')
    const importSelected = vi.spyOn(workspace, 'importSelected').mockResolvedValue({ errors: [], handleMap: {} })
    const exportCopy = vi.spyOn(workspace, 'exportCopy').mockResolvedValue('/tmp/export.mocktrue.json')

    const wrapper = mount(GlobalSettingsPanel)
    const buttons = wrapper.findAll('button')
    await buttons.find(button => button.text() === '保存')?.trigger('click')
    await buttons.find(button => button.text() === '另存为')?.trigger('click')
    await buttons.find(button => button.text() === '导入')?.trigger('click')
    await buttons.find(button => button.text() === '导出副本')?.trigger('click')

    expect(save).toHaveBeenCalled()
    expect(saveAs).toHaveBeenCalled()
    expect(importSelected).toHaveBeenCalled()
    expect(exportCopy).toHaveBeenCalled()
  })

  it('loads the selected example workspace from settings', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ errors: [], handleMap: {} })

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

  it('keeps the selected example after switching away from settings', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ errors: [], handleMap: {} })

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
    vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ errors: [], handleMap: {} })

    const wrapper = mount(GlobalSettingsPanel)
    await wrapper.findAll('button').find(button => button.text() === '加载示例')?.trigger('click')

    expect(registry.active.value).toBe('serial')
  })

  it('keeps direct save enabled for loaded example workspaces', () => {
    const wrapper = mount(GlobalSettingsPanel)
    const save = wrapper.findAll('button').find(button => button.text() === '保存')
    const saveAs = wrapper.findAll('button').find(button => button.text() === '另存为')

    expect(save?.attributes('disabled')).toBeUndefined()
    expect(saveAs?.attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).not.toContain('只读 Demo')
    expect(wrapper.text()).toContain('示例配置')
  })
})
