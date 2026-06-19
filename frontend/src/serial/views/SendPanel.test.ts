import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SendPanel from './SendPanel.vue'
import { useSettingsStore } from '../../settings/stores/settingsStore'
import {
  DecodeHexToText,
  EncodeTextToHex,
  Send,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => ({
  Send: vi.fn(async () => 2),
  EncodeTextToHex: vi.fn(async () => 'b4 ae bf da'),
  DecodeHexToText: vi.fn(async () => '串口'),
}))

vi.mock('naive-ui', () => naiveStubs)

describe('SendPanel', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('sends ascii content with selected encoding', async () => {
    const settings = useSettingsStore()
    settings.updateSerial({ TextEncoding: 'gbk' })
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })

    await wrapper.find('textarea').setValue('串口')
    await wrapper.find('button').trigger('click')

    expect(Send).toHaveBeenCalledWith({
      PortID: 'port-1',
      Content: '串口',
      Mode: 'ascii',
      Encoding: 'gbk',
    })
  })

  it('applies configured terminal font settings to the send editor scope', () => {
    const settings = useSettingsStore()
    settings.updateSerial({
      TerminalFontFamily: 'System Mono',
      TerminalFontSize: 18,
    })
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })

    expect(wrapper.find('.send-panel').attributes('style')).toContain('--serial-terminal-font-family: "System Mono", monospace')
    expect(wrapper.find('.send-panel').attributes('style')).toContain('--serial-terminal-font-size: 18px')
  })

  it('inserts configured enter string at the cursor without sending', async () => {
    const settings = useSettingsStore()
    settings.updateSerial({ EnterString: '\r\n' })
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })
    const textarea = wrapper.find('textarea')
    await textarea.setValue('ab')
    const el = textarea.element as HTMLTextAreaElement
    el.setSelectionRange(1, 1)

    await textarea.trigger('keydown', { key: 'Enter' })

    expect(Send).not.toHaveBeenCalled()
    await wrapper.find('button').trigger('click')
    expect(Send).toHaveBeenCalledWith({
      PortID: 'port-1',
      Content: 'a\r\nb',
      Mode: 'ascii',
      Encoding: 'utf-8',
    })
  })

  it('converts editor content with the selected encoding when switching modes', async () => {
    const settings = useSettingsStore()
    settings.updateSerial({ TextEncoding: 'gbk' })
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('串口')
    await wrapper.find('select').setValue('hex')
    await flushPromises()

    expect(EncodeTextToHex).toHaveBeenCalledWith({
      Content: '串口',
      Encoding: 'gbk',
    })
    expect((textarea.element as HTMLTextAreaElement).value).toBe('b4 ae bf da')

    await wrapper.find('select').setValue('ascii')
    await flushPromises()

    expect(DecodeHexToText).toHaveBeenCalledWith({
      Content: 'b4 ae bf da',
      Encoding: 'gbk',
    })
    expect((textarea.element as HTMLTextAreaElement).value).toBe('串口')
  })

  it('keeps editor content and send history across remounts', async () => {
    const first = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })

    await first.find('textarea').setValue('persist me')
    await first.find('button').trigger('click')
    first.unmount()

    const second = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })

    expect((second.find('textarea').element as HTMLTextAreaElement).value).toBe('persist me')
    expect(second.text()).toContain('persist me')
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
  NInputNumber: {
    props: ['value'],
    emits: ['update:value'],
    template: '<input type="number" :value="value" @input="$emit(\'update:value\', Number($event.target.value))" />',
  },
  NSwitch: {
    props: ['value'],
    emits: ['update:value'],
    template: '<input type="checkbox" :checked="value" @change="$emit(\'update:value\', $event.target.checked)" />',
  },
  NButton: {
    template: '<button type="button" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  NInput: {
    props: ['value'],
    emits: ['update:value', 'keydown'],
    template: '<textarea :value="value" @input="$emit(\'update:value\', $event.target.value)" @keydown="$emit(\'keydown\', $event)" />',
  },
}))
