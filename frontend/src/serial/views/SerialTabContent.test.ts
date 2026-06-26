import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SerialTabContent from './SerialTabContent.vue'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'
import { useSerialStore } from '../stores/serialStore'

vi.mock('naive-ui', () => naiveStubs)

describe('SerialTabContent', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('uses a larger default send panel height for new tabs', () => {
    const wrapper = mount(SerialTabContent, {
      props: { handleId: 'port-1' },
      global: {
        stubs: {
          DataDisplay: true,
          SendPanel: true,
          StatsPanel: true,
        },
      },
    })

    expect(wrapper.find('.serial-tab-content__send').attributes('style')).toContain('flex-basis: 180px')
  })

  it('uses persisted send panel height for existing tabs', () => {
    const workspace = useSerialWorkspaceStore()
    workspace.updateTabState('port-1', { sendHeight: 260 })

    const wrapper = mount(SerialTabContent, {
      props: { handleId: 'port-1' },
      global: {
        stubs: {
          DataDisplay: true,
          SendPanel: true,
          StatsPanel: true,
        },
      },
    })

    expect(wrapper.find('.serial-tab-content__send').attributes('style')).toContain('flex-basis: 260px')
  })

  it('expands the port config form and applies changed settings', async () => {
    const serial = useSerialStore()
    serial.handles.set('port-1', makeHandle('port-1'))
    const updatePortConfig = vi.spyOn(serial, 'updatePortConfig').mockResolvedValue('port-2')

    const wrapper = mount(SerialTabContent, {
      props: { handleId: 'port-1' },
      global: {
        stubs: {
          DataDisplay: true,
          SendPanel: true,
          StatsPanel: true,
        },
      },
    })

    const summary = wrapper.find('.serial-tab-config__summary')
    expect(summary.text()).toContain('/dev/ttyS1')
    expect(summary.attributes('aria-expanded')).toBe('false')

    await summary.trigger('click')

    expect(summary.attributes('aria-expanded')).toBe('true')
    const selects = wrapper.findAll('select')
    await selects[0].setValue('57600')
    await selects[1].setValue('7')
    await selects[2].setValue('2')
    await selects[3].setValue('even')
    await selects[4].setValue('hw_rtscts')
    await wrapper.find('.serial-tab-config__apply').trigger('click')
    await flushPromises()

    expect(updatePortConfig).toHaveBeenCalledWith('port-1', {
      PortName: '/dev/ttyS1',
      BaudRate: 57600,
      DataBits: 7,
      StopBits: '2',
      Parity: 'even',
      FlowMode: 'hw_rtscts',
      ReadBufKB: 64,
    })
  })

  it('clamps send panel resizing by pointer and keyboard', async () => {
    const workspace = useSerialWorkspaceStore()
    workspace.updateTabState('port-1', { sendHeight: 180 })

    const wrapper = mount(SerialTabContent, {
      props: { handleId: 'port-1' },
      attachTo: document.body,
      global: {
        stubs: {
          DataDisplay: true,
          SendPanel: true,
          StatsPanel: true,
        },
      },
    })
    Object.defineProperty(wrapper.element, 'clientHeight', { value: 360, configurable: true })
    Object.defineProperty(wrapper.find('.serial-tab-content__stats').element, 'offsetHeight', { value: 32, configurable: true })
    Object.defineProperty(wrapper.find('.serial-tab-content__resize-handle').element, 'offsetHeight', { value: 6, configurable: true })

    const handle = wrapper.find('.serial-tab-content__resize-handle')
    handle.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 1, clientY: 200 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 1, clientY: 100 }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.serial-tab-content').classes()).toContain('serial-tab-content--resizing')
    expect(workspace.tabState('port-1').sendHeight).toBe(202)
    expect(document.body.style.cursor).toBe('row-resize')

    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1, clientY: 100 }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.serial-tab-content').classes()).not.toContain('serial-tab-content--resizing')
    expect(document.body.style.cursor).toBe('')

    await handle.trigger('keydown', { key: 'ArrowDown', shiftKey: true })
    expect(workspace.tabState('port-1').sendHeight).toBe(162)
    await handle.trigger('keydown', { key: 'ArrowDown', shiftKey: true })
    await handle.trigger('keydown', { key: 'ArrowDown', shiftKey: true })
    await handle.trigger('keydown', { key: 'ArrowDown', shiftKey: true })
    expect(workspace.tabState('port-1').sendHeight).toBe(48)
    await handle.trigger('keydown', { key: 'ArrowUp' })
    expect(workspace.tabState('port-1').sendHeight).toBe(60)

    wrapper.unmount()
  })
})

const naiveStubs = vi.hoisted(() => ({
  NButton: {
    props: ['loading'],
    emits: ['click'],
    template: '<button type="button" :aria-busy="loading ? \'true\' : \'false\'" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: `
      <select
        :value="value"
        :disabled="disabled"
        @change="$emit('update:value', options.find(option => String(option.value) === $event.target.value)?.value ?? $event.target.value)"
      >
        <option v-for="option in options" :key="option.value" :value="String(option.value)">{{ option.label }}</option>
      </select>
    `,
  },
}))

function makeHandle(id: string) {
  return {
    ID: id,
    Config: {
      PortName: '/dev/ttyS1',
      BaudRate: 115200,
      DataBits: 8,
      StopBits: '1',
      Parity: 'none',
      FlowMode: 'none',
      ReadBufKB: 64,
    },
    IsOpen: true,
    RxBytes: 0,
    TxBytes: 0,
  }
}

function pointerTestEvent(type: string, init: { pointerId: number; clientY: number }): Event {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    pointerId: { value: init.pointerId },
    clientY: { value: init.clientY },
  })
  return event
}
