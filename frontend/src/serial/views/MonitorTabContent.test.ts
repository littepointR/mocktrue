import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import MonitorTabContent from './MonitorTabContent.vue'
import { useMonitorStore, defaultMonitorAutoSave } from '../stores/monitorStore'

describe('MonitorTabContent', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders monitor frames and switches display mode', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: '', displayMode: 'text', modbusFunction: 0 } },
      sessions: [{
        ID: 'mon-1',
        Name: '监控 A-B',
        Provider: 'bridge',
        PortA: '/tmp/a',
        PortB: '/tmp/b',
        Config: {
          PortName: '',
          BaudRate: 115200,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Encoding: 'utf-8',
        Status: 'running',
        RxBytes: 2,
        TxBytes: 3,
        FrameCount: 1,
        StartedAt: '',
        StoppedAt: '',
        Error: '',
        AutoSave: defaultMonitorAutoSave(),
      }],
      frames: {
        'mon-1': [{
          Seq: 1,
          Timestamp: '',
          Direction: 'a_to_b',
          Port: '/tmp/a',
          Length: 2,
          Data: null,
          DisplayText: 'OK',
          DisplayHex: '4f 4b',
          DisplayDec: '79 75',
          DisplayOct: '117 113',
          DisplayBin: '01001111 01001011',
          Encoding: 'utf-8',
          Modbus: null,
        }],
      },
    })

    const wrapper = mount(MonitorTabContent, {
      props: { monitorId: 'mon-1' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('监控 A-B')
    expect(wrapper.text()).toContain('OK')

    await wrapper.find('[data-testid="monitor-display-hex"]').trigger('click')
    expect(wrapper.text()).toContain('4f 4b')
  })
})

const stubs = {
  NButton: { template: '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>' },
  NInput: { props: ['value'], emits: ['update:value'], template: '<input :value="value" @input="$emit(\'update:value\', $event.target.value)" />' },
  NInputNumber: { props: ['value'], emits: ['update:value'], template: '<input type="number" :value="value" @input="$emit(\'update:value\', Number($event.target.value))" />' },
  NSelect: true,
  NSwitch: true,
}
