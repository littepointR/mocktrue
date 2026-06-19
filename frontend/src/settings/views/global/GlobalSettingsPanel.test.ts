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

  it('loads the selected readonly demo workspace from settings', async () => {
    const workspace = useWorkspaceFileStore()
    const loadDemo = vi.spyOn(workspace, 'loadDemo').mockResolvedValue({ errors: [], handleMap: {} })

    const wrapper = mount(GlobalSettingsPanel)
    await wrapper.findAll('select')[1].setValue('monitor-demo')
    await wrapper.findAll('button').find(button => button.text() === '加载 Demo')?.trigger('click')

    expect(loadDemo).toHaveBeenCalledWith('monitor-demo')
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
    await wrapper.findAll('button').find(button => button.text() === '加载 Demo')?.trigger('click')

    expect(registry.active.value).toBe('serial')
  })

  it('disables direct save for readonly demo workspaces but keeps save-as available', () => {
    const workspace = useWorkspaceFileStore()
    workspace.readonly = true

    const wrapper = mount(GlobalSettingsPanel)
    const save = wrapper.findAll('button').find(button => button.text() === '保存')
    const saveAs = wrapper.findAll('button').find(button => button.text() === '另存为')

    expect(save?.attributes('disabled')).toBeDefined()
    expect(saveAs?.attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).toContain('只读 Demo')
  })
})
