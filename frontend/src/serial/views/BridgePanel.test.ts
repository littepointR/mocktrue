import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BridgePanel from './BridgePanel.vue'

const virtualApi = vi.hoisted(() => {
  let ports: Array<{ ID: string; Port: string }> = []
  let bridges: Array<{ ID: string; Port1: string; Port2: string; BaudRate: number }> = []

  return {
    reset() {
      ports = [{ ID: 'vport-existing', Port: 'ttyV1' }]
      bridges = [{ ID: 'bridge-existing', Port1: '/dev/tty.usb0', Port2: 'ttyV1', BaudRate: 115200 }]
    },
    CreateVirtualPort: vi.fn(),
    DeleteVirtualPort: vi.fn(),
    ListVirtualPorts: vi.fn(async () => [...ports]),
    CreateBridge: vi.fn(async (id: string, port1: string, port2: string, baudRate: number) => {
      bridges.push({ ID: id, Port1: port1, Port2: port2, BaudRate: baudRate })
    }),
    DeleteBridge: vi.fn(async (id: string) => {
      bridges = bridges.filter(bridge => bridge.ID !== id)
    }),
    ListBridges: vi.fn(async () => [...bridges]),
    CleanupVirtual: vi.fn(),
  }
})

const serialServiceMock = vi.hoisted(() => ({
  enumeratePorts: vi.fn(async () => [
    { Name: '/dev/tty.usb0', FriendlyName: 'USB Serial' },
    { Name: 'ttyV1', FriendlyName: 'duplicate virtual from enumerator' },
  ]),
}))

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => virtualApi)
vi.mock('../services/serialService', () => ({ serialService: serialServiceMock }))
vi.mock('naive-ui', () => naiveStubs)

describe('BridgePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    virtualApi.reset()
    vi.clearAllMocks()
  })

  it('loads available ports, creates a bridge from the form, and deletes an existing bridge', async () => {
    const wrapper = mount(BridgePanel)
    await flushPromises()

    expect(virtualApi.ListBridges).toHaveBeenCalledTimes(1)
    expect(virtualApi.ListVirtualPorts).toHaveBeenCalledTimes(1)
    expect(serialServiceMock.enumeratePorts).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('已有桥接 (1)')
    expect(wrapper.text()).toContain('bridge-existing')
    expect(wrapper.text()).toContain('/dev/tty.usb0')
    expect(wrapper.text()).toContain('ttyV1')

    const selects = wrapper.findAll('select')
    expect(selects[0].findAll('option').map(option => option.attributes('value'))).toEqual([
      '/dev/tty.usb0',
      'ttyV1',
    ])

    await wrapper.find('input').setValue('bridge-new')
    await selects[0].setValue('/dev/tty.usb0')
    await selects[1].setValue('ttyV1')
    await wrapper.findAll('button').find(button => button.text() === '创建桥接')?.trigger('click')
    await flushPromises()

    expect(virtualApi.CreateBridge).toHaveBeenCalledWith('bridge-new', '/dev/tty.usb0', 'ttyV1', 115200)
    expect(wrapper.text()).toContain('已有桥接 (2)')
    expect(wrapper.text()).toContain('bridge-new')

    await wrapper.find('[data-testid="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(virtualApi.DeleteBridge).toHaveBeenCalledWith('bridge-existing')
    expect(wrapper.text()).toContain('已有桥接 (1)')
    expect(wrapper.text()).not.toContain('bridge-existing')
  })
})

const naiveStubs = vi.hoisted(() => ({
  NAlert: {
    emits: ['close'],
    template: '<div role="alert"><slot /><button type="button" data-testid="alert-close" @click="$emit(\'close\')">close</button></div>',
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
  NInputNumber: {
    props: ['value', 'disabled'],
    emits: ['update:value'],
    template: '<input type="number" :value="value" :disabled="disabled" @input="$emit(\'update:value\', Number($event.target.value))" />',
  },
  NList: { template: '<div><slot /></div>' },
  NListItem: { template: '<div><slot /></div>' },
  NPopconfirm: {
    emits: ['positive-click'],
    template: '<div><slot name="trigger" /><button type="button" data-testid="confirm-delete" @click="$emit(\'positive-click\')"><slot /></button></div>',
  },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: `
      <select :value="value" :disabled="disabled" @change="$emit('update:value', $event.target.value)">
        <option v-for="option in options" :key="option.value" :value="String(option.value)">{{ option.label }}</option>
      </select>
    `,
  },
  NSpace: { template: '<div><slot /></div>' },
  NTag: { template: '<span><slot /></span>' },
  NThing: { template: '<article><slot name="header" /><slot name="description" /><slot name="action" /><slot /></article>' },
}))
