import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DataDisplay from './DataDisplay.vue'
import { useBufferStore } from '../stores/bufferStore'
import { useSettingsStore } from '../../settings/stores/settingsStore'

vi.mock('naive-ui', () => naiveStubs)

describe('DataDisplay', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('applies configured terminal font settings', () => {
    const settings = useSettingsStore()
    settings.updateSerial({
      TerminalFontFamily: 'System Mono',
      TerminalFontSize: 16,
    })

    const wrapper = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })

    const content = wrapper.find('.data-display__content')
    expect(wrapper.find('.data-display').attributes('style')).toContain('--serial-terminal-font-family: "System Mono", monospace')
    expect(wrapper.find('.data-display').attributes('style')).toContain('--serial-terminal-font-size: 16px')
    expect(content.classes()).toContain('data-display__content')
  })

  it('decodes received bytes with the selected encoding', () => {
    const settings = useSettingsStore()
    settings.updateSerial({ TextEncoding: 'utf-16le' })
    const buffer = useBufferStore()
    buffer.appendData('port-1', [0x32, 0x4e, 0xe3, 0x53], 1)

    const wrapper = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })

    expect(wrapper.text()).toContain('串口')
  })

  it('keeps receive display options across remounts', async () => {
    const first = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })

    await first.findAll('select').at(0)?.setValue('hexClassic')
    await first.findAll('select').at(1)?.setValue('split')
    await first.findAll('input[type="checkbox"]').at(0)?.setValue(false)
    await first.findAll('input[type="checkbox"]').at(1)?.setValue(false)
    first.unmount()

    const second = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })

    expect((second.findAll('select').at(0)?.element as HTMLSelectElement).value).toBe('hexClassic')
    expect((second.findAll('select').at(1)?.element as HTMLSelectElement).value).toBe('split')
    expect((second.findAll('input[type="checkbox"]').at(0)?.element as HTMLInputElement).checked).toBe(false)
    expect((second.findAll('input[type="checkbox"]').at(1)?.element as HTMLInputElement).checked).toBe(false)
  })

  it('renders timestamped ASCII chunks, hex views, and auto-scroll behavior', async () => {
    const buffer = useBufferStore()
    buffer.appendData('port-1', [0x41, 0x0a, 0x00], new Date(2026, 0, 2, 3, 4, 5, 6).getTime())
    const wrapper = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })
    const content = wrapper.find('.data-display__content').element as HTMLElement
    Object.defineProperty(content, 'scrollHeight', { configurable: true, value: 500 })

    expect(wrapper.text()).toContain('03:04:05.006')
    expect(wrapper.text()).toContain('A\n\u0000')

    buffer.appendData('port-1', [0x42], Date.UTC(2026, 0, 2, 3, 4, 6, 7))
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    expect(content.scrollTop).toBe(500)

    await wrapper.findAll('select').at(0)?.setValue('hexClassic')
    expect(wrapper.text()).toContain('00000000')
    expect(wrapper.text()).toContain('41 0a 00 42')
    expect(wrapper.text()).toContain('A..B')

    await wrapper.findAll('select').at(0)?.setValue('hexTable')
    expect(wrapper.text()).toContain('HEX 表格视图')
  })

  it('skips auto-scroll when the persisted receive option disables it', async () => {
    const buffer = useBufferStore()
    const wrapper = mount(DataDisplay, {
      props: { handleId: 'port-1' },
      global: { stubs: naiveStubs },
    })
    const content = wrapper.find('.data-display__content').element as HTMLElement
    Object.defineProperty(content, 'scrollHeight', { configurable: true, value: 800 })

    await wrapper.findAll('input[type="checkbox"]').at(1)?.setValue(false)
    content.scrollTop = 0
    buffer.appendData('port-1', [0x43], Date.now())
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(content.scrollTop).toBe(0)
  })
})

const naiveStubs = vi.hoisted(() => ({
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
  NSwitch: {
    props: ['value'],
    emits: ['update:value'],
    template: '<input type="checkbox" :checked="value" @change="$emit(\'update:value\', $event.target.checked)" />',
  },
}))
