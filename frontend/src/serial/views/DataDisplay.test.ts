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
