import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpenPortDialog from './OpenPortDialog.vue'
import { useSerialStore } from '../stores/serialStore'

const serialServiceMock = vi.hoisted(() => ({
  enumeratePorts: vi.fn(async () => [
    { Name: '/dev/tty.usb0', FriendlyName: 'USB Serial' },
  ]),
}))

vi.mock('../services/serialService', () => ({ serialService: serialServiceMock }))
vi.mock('naive-ui', () => naiveStubs)

describe('OpenPortDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    serialServiceMock.enumeratePorts.mockResolvedValue([
      { Name: '/dev/tty.usb0', FriendlyName: 'USB Serial' },
    ])
  })

  it('refreshes available ports and emits success after opening the selected port with form options', async () => {
    const serial = useSerialStore()
    const openPort = vi.spyOn(serial, 'openPort').mockResolvedValue('handle-1')
    const wrapper = mount(OpenPortDialog, { props: { show: true } })
    await flushPromises()

    expect(serialServiceMock.enumeratePorts).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[role="dialog"]').text()).toContain('打开串口')

    const selects = wrapper.findAll('select')
    await selects[0].setValue('/dev/tty.usb0')
    await selects[1].setValue('57600')
    await selects[2].setValue('7')
    await selects[3].setValue('2')
    await selects[4].setValue('even')
    await selects[5].setValue('hw_rtscts')
    await wrapper.findAll('button').find(button => button.text() === '打开')?.trigger('click')
    await flushPromises()

    expect(openPort).toHaveBeenCalledWith('/dev/tty.usb0', 57600, {
      dataBits: 7,
      stopBits: '2',
      parity: 'even',
      flowMode: 'hw_rtscts',
    })
    expect(wrapper.emitted('update:show')?.at(-1)).toEqual([false])
    expect(wrapper.emitted('success')).toHaveLength(1)
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
  NCard: {
    props: ['title'],
    template: '<section v-bind="$attrs"><h4>{{ title }}</h4><slot /><footer><slot name="footer" /></footer></section>',
  },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NModal: {
    props: ['show'],
    emits: ['update:show'],
    template: '<div v-if="show" data-testid="modal"><slot /></div>',
  },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: `
      <select
        :value="value"
        :disabled="disabled"
        @change="$emit('update:value', options.find(option => String(option.value) === $event.target.value)?.value ?? $event.target.value)"
      >
        <option value=""></option>
        <option v-for="option in options" :key="option.value" :value="String(option.value)">{{ option.label }}</option>
      </select>
    `,
  },
  NSpace: { template: '<div><slot /></div>' },
}))
