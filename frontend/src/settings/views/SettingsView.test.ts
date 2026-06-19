import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import SettingsView from './SettingsView.vue'

describe('SettingsView', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('switches between global and serial settings editors', () => {
    const global = mount(SettingsView, {
      props: { activeViewId: 'settings.global' },
      global: { stubs: naiveStubs },
    })
    expect(global.text()).toContain('全局设置')
    expect(global.text()).toContain('主题')
    expect(global.text()).not.toContain('MCP 服务')
    expect(global.text()).not.toContain('MCP 地址')

    const serial = mount(SettingsView, {
      props: { activeViewId: 'settings.serial' },
      global: { stubs: naiveStubs },
    })
    expect(serial.text()).toContain('串口')
    expect(serial.text()).toContain('默认波特率')
  })
})

const naiveStubs = {
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NSelect: { template: '<select />' },
  NInput: { template: '<input />' },
  NInputNumber: { template: '<input type="number" />' },
  NSwitch: { template: '<input type="checkbox" />' },
  NButton: { template: '<button><slot /></button>' },
}
