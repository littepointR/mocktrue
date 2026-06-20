import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import MonitorTabContent from './MonitorTabContent.vue'
import { useMonitorStore } from '../stores/monitorStore'

describe('MonitorTabContent', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders monitor frames and switches display mode', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: '', displayMode: 'text' } },
      sessions: [{
        ID: 'mon-1',
        Name: '串口监控',
        Provider: 'bridge',
        PortA: '/tmp/a',
        PortB: '/tmp/b',
        ExternalPort: '',
        AutoVirtualPortID: '',
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
      }],
      frames: {
        'mon-1': [{
          Seq: 1,
          Timestamp: '2026-06-20T12:34:56.789',
          Direction: 'a_to_b',
          Port: '/tmp/a',
          Length: 2,
          Data: '',
          DisplayText: 'OK',
          DisplayHex: '4f 4b',
          DisplayDec: '79 75',
          DisplayOct: '117 113',
          DisplayBin: '01001111 01001011',
          Encoding: 'utf-8',
        }, {
          Seq: 2,
          Timestamp: '',
          Direction: 'b_to_a',
          Port: '/tmp/b',
          Length: 2,
          Data: '',
          DisplayText: 'GO',
          DisplayHex: '47 4f',
          DisplayDec: '71 79',
          DisplayOct: '107 117',
          DisplayBin: '01000111 01001111',
          Encoding: 'utf-8',
        }],
      },
    })

    const wrapper = mount(MonitorTabContent, {
      props: { monitorId: 'mon-1' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('串口监控')
    expect(wrapper.text()).toContain('接收')
    expect(wrapper.text()).toContain('发送')
    expect(wrapper.text()).not.toContain('A → B')
    expect(wrapper.text()).not.toContain('B → A')
    expect(wrapper.text()).not.toContain('a_to_b')
    expect(wrapper.text()).not.toContain('b_to_a')
    expect(wrapper.text()).not.toContain('Modbus')
    expect(wrapper.text()).not.toContain('功能码')
    expect(wrapper.text()).not.toContain('导出')
    expect(wrapper.text()).not.toContain('自动保存')
    expect(wrapper.text()).toContain('OK')
    expect(wrapper.text()).toContain('2026-06-20 12:34:56.789')
    expect(wrapper.find('.monitor-table colgroup').exists()).toBe(true)
    expect(wrapper.find('.monitor-table__data-column').exists()).toBe(true)
    expect(wrapper.find('[data-testid="resize-handle-data"]').exists()).toBe(true)

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
