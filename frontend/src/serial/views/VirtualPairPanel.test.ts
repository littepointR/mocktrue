import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VirtualPairPanel from './VirtualPairPanel.vue'

const virtualApi = vi.hoisted(() => {
  let ports: Array<{ ID: string; Port: string }> = []
  let backendStatus = {
    Name: 'fake',
    Available: true,
    Message: 'ready',
    Reason: '',
    RequiresAdmin: false,
  }
  return {
    reset(nextPorts: Array<{ ID: string; Port: string }> = [], nextStatus = {
      Name: 'fake',
      Available: true,
      Message: 'ready',
      Reason: '',
      RequiresAdmin: false,
    }) {
      ports = [...nextPorts]
      backendStatus = { ...nextStatus }
    },
    CreateVirtualPort: vi.fn(async (id: string, portName: string) => {
      ports.push({ ID: id, Port: portName })
    }),
    DeleteVirtualPort: vi.fn(async (id: string) => {
      ports = ports.filter(port => port.ID !== id)
    }),
    ListVirtualPorts: vi.fn(async () => [...ports]),
    GetVirtualSerialBackendStatus: vi.fn(async () => ({ ...backendStatus })),
    CreateBridge: vi.fn(),
    DeleteBridge: vi.fn(),
    ListBridges: vi.fn(async () => []),
    CleanupVirtual: vi.fn(),
  }
})

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => virtualApi)
vi.mock('naive-ui', () => naiveStubs)

describe('VirtualPairPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    virtualApi.reset([{ ID: 'vport-existing', Port: 'ttyV1' }])
    vi.clearAllMocks()
  })

  it('loads existing virtual ports and creates a new port from the form', async () => {
    const wrapper = mount(VirtualPairPanel)
    await flushPromises()

    expect(virtualApi.GetVirtualSerialBackendStatus).toHaveBeenCalledTimes(1)
    expect(virtualApi.ListVirtualPorts).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('fake')
    expect(wrapper.text()).toContain('ready')
    expect(wrapper.text()).toContain('已有虚拟串口 (1)')
    expect(wrapper.text()).toContain('vport-existing')
    expect(wrapper.text()).toContain('ttyV1')

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('vport-new')
    await inputs[1].setValue('ttyNEW')
    await wrapper.findAll('button').find(button => button.text() === '创建')?.trigger('click')
    await flushPromises()

    expect(virtualApi.CreateVirtualPort).toHaveBeenCalledWith('vport-new', 'ttyNEW')
    expect((inputs[0].element as HTMLInputElement).value).toBe('')
    expect((inputs[1].element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain('已有虚拟串口 (2)')
    expect(wrapper.text()).toContain('vport-new')
    expect(wrapper.text()).toContain('ttyNEW')
  })

  it('refreshes, deletes ports, and clears alerts', async () => {
    const wrapper = mount(VirtualPairPanel)
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '刷新')?.trigger('click')
    await flushPromises()
    expect(virtualApi.GetVirtualSerialBackendStatus).toHaveBeenCalledTimes(2)
    expect(virtualApi.ListVirtualPorts).toHaveBeenCalledTimes(2)

    await wrapper.find('[data-testid="confirm-delete"]').trigger('click')
    await flushPromises()
    expect(virtualApi.DeleteVirtualPort).toHaveBeenCalledWith('vport-existing')
    expect(wrapper.text()).toContain('已有虚拟串口 (0)')
    expect(wrapper.text()).toContain('暂无虚拟串口')

    const store = await import('../stores/virtualStore').then(mod => mod.useVirtualStore())
    store.error = 'boom'
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('boom')

    await wrapper.find('[data-testid="alert-close"]').trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).not.toContain('boom')
  })

  it('shows unavailable backend status and disables creation', async () => {
    virtualApi.reset([], {
      Name: 'com0com',
      Available: false,
      Message: 'com0com setupc.exe was not found',
      Reason: 'checked PORTWEAVE_COM0COM_SETUPC',
      RequiresAdmin: true,
    })
    const wrapper = mount(VirtualPairPanel)
    await flushPromises()

    expect(wrapper.text()).toContain('com0com')
    expect(wrapper.text()).toContain('com0com setupc.exe was not found')
    expect(wrapper.text()).toContain('需要管理员权限')

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('vport-new')
    await inputs[1].setValue('ttyNEW')
    const createButton = wrapper.findAll('button').find(button => button.text() === '创建')
    expect(createButton?.attributes('disabled')).toBeDefined()
  })
})

const naiveStubs = vi.hoisted(() => ({
  NAlert: {
    props: { closable: Boolean },
    emits: ['close'],
    template: '<div role="alert"><slot /><button v-if="closable" type="button" data-testid="alert-close" @click="$emit(\'close\')">close</button></div>',
  },
  NButton: {
    props: ['disabled', 'loading'],
    emits: ['click'],
    template: '<button type="button" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  NCard: { props: ['title'], template: '<section><h4>{{ title }}</h4><slot /></section>' },
  NDivider: { template: '<hr />' },
  NEmpty: { props: ['description'], template: '<div>{{ description }}</div>' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NInput: {
    props: ['value', 'placeholder', 'disabled'],
    emits: ['update:value'],
    template: '<input :value="value" :placeholder="placeholder" :disabled="disabled" @input="$emit(\'update:value\', $event.target.value)" />',
  },
  NList: { template: '<div><slot /></div>' },
  NListItem: { template: '<div><slot /></div>' },
  NPopconfirm: {
    emits: ['positive-click'],
    template: '<div><slot name="trigger" /><button type="button" data-testid="confirm-delete" @click="$emit(\'positive-click\')"><slot /></button></div>',
  },
  NSpace: { template: '<div><slot /></div>' },
  NTag: { template: '<span><slot /></span>' },
  NThing: { template: '<article><slot name="header" /><slot name="description" /><slot name="action" /><slot /></article>' },
}))
