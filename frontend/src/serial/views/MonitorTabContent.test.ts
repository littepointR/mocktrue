import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MonitorTabContent from './MonitorTabContent.vue'
import { useMonitorStore } from '../stores/monitorStore'

vi.mock('naive-ui', () => ({
  NButton: { props: ['disabled'], emits: ['click'], template: '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>' },
  NInput: { inheritAttrs: false, props: ['value', 'placeholder'], emits: ['update:value'], template: '<input :class="$attrs.class" :value="value" :placeholder="placeholder" @input="$emit(\'update:value\', $event.target.value)" />' },
  NSelect: { inheritAttrs: false, props: ['value', 'options'], emits: ['update:value'], template: '<select :class="$attrs.class" :value="value" @change="$emit(\'update:value\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>' },
}))

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
          Error: '',
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
          Error: '',
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
    const receiveContainer = wrapper.find('[data-testid="monitor-receive-container"]')
    expect(receiveContainer.exists()).toBe(true)
    expect(receiveContainer.find('[data-testid="monitor-frame-list"]').exists()).toBe(true)
    expect(receiveContainer.find('[data-testid="monitor-frame-detail"]').exists()).toBe(true)
    expect(receiveContainer.find('.monitor-table').exists()).toBe(true)
    expect(wrapper.find('.monitor-tab > .monitor-detail').exists()).toBe(false)
    expect(wrapper.find('.monitor-table colgroup').exists()).toBe(true)
    expect(wrapper.find('.monitor-table__data-column').exists()).toBe(true)
    expect(wrapper.find('[data-testid="resize-handle-data"]').exists()).toBe(true)

    await wrapper.find('[data-testid="monitor-display-hex"]').trigger('click')
    expect(wrapper.text()).toContain('4f 4b')
  })

  it('refreshes filters, stop, clear, and fallback frame formatting controls', async () => {
    const stopMonitor = vi.fn(async () => undefined)
    const clearFrames = vi.fn(async () => undefined)
    const refreshFrames = vi.fn(async () => undefined)
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: '', displayMode: 'hex' } },
      sessions: [{
        ID: 'mon-1',
        Name: '本地监控',
        Provider: 'bridge',
        PortA: '/tmp/a',
        PortB: '/tmp/b',
        ExternalPort: '/tmp/ext',
        AutoVirtualPortID: '',
        Config: {
          PortName: '',
          BaudRate: 9600,
          DataBits: 8,
          StopBits: '1',
          Parity: 'none',
          FlowMode: 'none',
          ReadBufKB: 32,
        },
        Encoding: 'utf-8',
        Status: 'stopped',
        RxBytes: 4,
        TxBytes: 5,
        FrameCount: 1,
        StartedAt: '',
        StoppedAt: '',
        Error: '',
      }],
      frames: {
        'mon-1': [{
          Seq: 7,
          Timestamp: 'not-a-date',
          Direction: 'loopback',
          Port: '/tmp/ext',
          Length: 1,
          Data: '',
          DisplayText: 'TXT',
          DisplayHex: 'aa',
          DisplayDec: '170',
          DisplayOct: '252',
          DisplayBin: '10101010',
          Encoding: 'utf-8',
          Error: '',
        }],
      },
    })
    store.stopMonitor = stopMonitor as any
    store.clearFrames = clearFrames as any
    store.refreshFrames = refreshFrames as any

    const wrapper = mount(MonitorTabContent, {
      props: { monitorId: 'mon-1' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('监听 /tmp/a，外部 /tmp/ext')
    expect(wrapper.text()).toContain('not-a-date')
    expect(wrapper.text()).toContain('loopback')
    expect(wrapper.find('button').attributes('disabled')).toBe('')

    await wrapper.get('.monitor-tab__control').setValue('b_to_a')
    await wrapper.get('.monitor-tab__search').setValue('aa')
    await wrapper.find('[data-testid="monitor-display-dec"]').trigger('click')
    await wrapper.findAll('button').find(button => button.text() === '停止')?.trigger('click')
    await wrapper.findAll('button').find(button => button.text() === '清空')?.trigger('click')

    expect(refreshFrames).toHaveBeenCalledWith('mon-1', { direction: 'b_to_a' })
    expect(refreshFrames).toHaveBeenCalledWith('mon-1', { search: 'aa' })
    expect(store.filterFor('mon-1').displayMode).toBe('dec')
    expect(stopMonitor).not.toHaveBeenCalled()
    expect(clearFrames).toHaveBeenCalledWith('mon-1')
  })

  it('refreshes running monitors on mount and when switching monitor ids', async () => {
    const store = useMonitorStore()
    const refreshFrames = vi.fn(async () => undefined)
    store.filters = {
      'mon-1': { direction: 'all', search: '', displayMode: 'hex' },
      'mon-2': { direction: 'all', search: '', displayMode: 'hex' },
    }
    store.sessions.set('mon-1', { ...sampleSession('mon-1'), Status: 'running' })
    store.sessions.set('mon-2', { ...sampleSession('mon-2'), Status: 'running', Name: '第二监控' })
    store.refreshFrames = refreshFrames as any

    const wrapper = mount(MonitorTabContent, {
      props: { monitorId: 'mon-1' },
      global: { stubs },
    })
    await wrapper.vm.$nextTick()

    expect(refreshFrames).toHaveBeenCalledWith('mon-1')

    await wrapper.setProps({ monitorId: 'mon-2' })
    await wrapper.vm.$nextTick()

    expect(refreshFrames).toHaveBeenCalledWith('mon-2')
  })

  it('renders empty monitor fallback state and every frame display mode', async () => {
    const store = useMonitorStore()
    store.restoreState({
      activeMonitorId: 'mon-1',
      filters: { 'mon-1': { direction: 'all', search: '', displayMode: 'oct' } },
      sessions: [sampleSession('mon-1')],
      frames: {
        'mon-1': [{
          Seq: 3,
          Timestamp: 0 as any,
          Direction: 'a_to_b',
          Port: '/tmp/a',
          Length: 1,
          Data: '',
          DisplayText: 'TXT',
          DisplayHex: 'ff',
          DisplayDec: '255',
          DisplayOct: '377',
          DisplayBin: '11111111',
          Encoding: 'utf-8',
          Error: '',
        }],
      },
    })

    const wrapper = mount(MonitorTabContent, {
      props: { monitorId: 'mon-1' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('377')
    expect(wrapper.text()).toContain('TXT')
    await wrapper.find('[data-testid="monitor-display-bin"]').trigger('click')
    expect(wrapper.text()).toContain('11111111')

    await wrapper.setProps({ monitorId: 'missing-monitor' })

    expect(wrapper.text()).toContain('missing-monitor')
    expect(wrapper.text()).toContain('stopped')
    expect(wrapper.find('[data-testid="monitor-frame-detail"]').exists()).toBe(false)
  })
})

const stubs = {
  NButton: { template: '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>' },
  NInput: { inheritAttrs: false, props: ['value'], emits: ['update:value'], template: '<input :class="$attrs.class" :value="value" @input="$emit(\'update:value\', $event.target.value)" />' },
  NInputNumber: { props: ['value'], emits: ['update:value'], template: '<input type="number" :value="value" @input="$emit(\'update:value\', Number($event.target.value))" />' },
  NSelect: {
    props: ['value', 'options'],
    emits: ['update:value'],
    template: '<select :class="$attrs.class" :value="value" @change="$emit(\'update:value\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
  },
  NSwitch: true,
}

function sampleSession(id: string) {
  return {
    ID: id,
    Name: '测试监控',
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
    Status: 'stopped',
    RxBytes: 0,
    TxBytes: 0,
    FrameCount: 0,
    StartedAt: '',
    StoppedAt: '',
    Error: '',
  }
}
