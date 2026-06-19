import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PortConfigPanel from './PortConfigPanel.vue'
import { useSerialStore } from '../stores/serialStore'
import { useSettingsStore } from '../../settings/stores/settingsStore'

vi.mock('naive-ui', () => ({
  NAlert: { template: '<div><slot /></div>' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: `
      <select :value="value" :disabled="disabled" @change="$emit('update:value', $event.target.value)">
        <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
    `,
  },
  NButton: { template: '<button type="button"><slot /></button>' },
}))

vi.mock('../services/serialService', () => ({
  serialService: {
    enumeratePorts: vi.fn(async () => [{ Name: '/tmp/ttyS1', FriendlyName: 'ttyS1' }]),
    openPort: vi.fn(async () => ({
      ID: 'port-1',
      Config: {
        PortName: '/tmp/ttyS1',
        BaudRate: 57600,
        DataBits: 7,
        StopBits: '2',
        Parity: 'even',
        FlowMode: 'hw_rtscts',
        ReadBufKB: 64,
      },
      IsOpen: true,
      RxBytes: 0,
      TxBytes: 0,
    })),
  },
}))

describe('PortConfigPanel settings effects', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('opens ports with configured serial defaults', async () => {
    const settings = useSettingsStore()
    settings.updateSerial({
      BaudRate: 57600,
      DataBits: 7,
      StopBits: '2',
      Parity: 'even',
      FlowMode: 'hw_rtscts',
      ReadBufKB: 64,
    })
    const serial = useSerialStore()
    const openSpy = vi.spyOn(serial, 'openPort')

    const wrapper = mount(PortConfigPanel)
    await flushPromises()
    await wrapper.find('select').setValue('/tmp/ttyS1')
    await wrapper.vm.$nextTick()
    await wrapper.findAll('button').at(1)?.trigger('click')

    expect(openSpy).toHaveBeenCalledWith('/tmp/ttyS1', 57600, {
      dataBits: 7,
      stopBits: '2',
      parity: 'even',
      flowMode: 'hw_rtscts',
      readBufKB: 64,
    })
  })
})
