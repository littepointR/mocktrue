import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SendPanel from './SendPanel.vue'
import { useSettingsStore } from '../../settings/stores/settingsStore'
import {
  DecodeHexToText,
  EncodeTextToHex,
  Send,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => ({
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

  it('ignores non-enter editor shortcuts and falls back for non-error conversion failures', async () => {
    vi.mocked(EncodeTextToHex).mockRejectedValueOnce('encoding worker crashed')
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('abc')
    await textarea.trigger('keydown', { key: 'Escape' })
    await wrapper.find('select').setValue('hex')
    await flushPromises()

    expect(Send).not.toHaveBeenCalled()
    expect((textarea.element as HTMLTextAreaElement).value).toBe('abc')
    expect(wrapper.find('.send-panel__error').text()).toContain('Convert send content failed')
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

  it('formats hex editor input and sends it without text encoding', async () => {
    vi.mocked(EncodeTextToHex).mockResolvedValueOnce('')
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })
    const textarea = wrapper.find('textarea')

    await wrapper.find('select').setValue('hex')
    await flushPromises()
    await textarea.setValue('0a0b0c')
    await flushPromises()

    expect((textarea.element as HTMLTextAreaElement).value).toBe('0a 0b 0c')

    await wrapper.find('button').trigger('click')

    expect(Send).toHaveBeenCalledWith({
      PortID: 'port-1',
      Content: '0a 0b 0c',
      Mode: 'hex',
      Encoding: '',
    })
  })

  it('starts restarts and stops auto sending as content and interval change', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(SendPanel, {
        props: { handleId: 'port-1' },
      })
      const textarea = wrapper.find('textarea')
      const interval = wrapper.find('input[type="number"]')
      const autoSend = wrapper.find<HTMLInputElement>('input[type="checkbox"]')

      await textarea.setValue('ping')
      await interval.setValue('20')
      await autoSend.setValue(true)

      await vi.advanceTimersByTimeAsync(20)
      expect(Send).toHaveBeenCalledTimes(1)
      expect(Send).toHaveBeenLastCalledWith({
        PortID: 'port-1',
        Content: 'ping',
        Mode: 'ascii',
        Encoding: 'utf-8',
      })

      await interval.setValue('15')
      await vi.advanceTimersByTimeAsync(15)
      expect(Send).toHaveBeenCalledTimes(2)

      await textarea.setValue('')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(30)

      expect(Send).toHaveBeenCalledTimes(2)
      expect(autoSend.element.checked).toBe(false)
      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('clamps zero auto-send intervals to the minimum timer delay', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(SendPanel, {
        props: { handleId: 'port-1' },
      })

      await wrapper.find('textarea').setValue('ping')
      await wrapper.find('input[type="number"]').setValue('0')
      await wrapper.find<HTMLInputElement>('input[type="checkbox"]').setValue(true)

      await vi.advanceTimersByTimeAsync(9)
      expect(Send).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(1)
      expect(Send).toHaveBeenCalledWith(expect.objectContaining({ Content: 'ping' }))
      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('resends history entries and surfaces send and conversion errors', async () => {
    const wrapper = mount(SendPanel, {
      props: { handleId: 'port-1' },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('first')
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.find('.send-panel__history-item').text()).toContain('first')

    vi.mocked(Send).mockRejectedValueOnce(new Error('permission denied'))
    await wrapper.find('.send-panel__history-item').trigger('click')
    await flushPromises()

    expect(wrapper.find('.send-panel__error').text()).toContain('permission denied')

    vi.mocked(EncodeTextToHex).mockRejectedValueOnce(new Error('encode unavailable'))
    await wrapper.find('select').setValue('hex')
    await flushPromises()

    expect(wrapper.find('.send-panel__error').text()).toContain('encode unavailable')
  })

  it('ignores empty sends restarts auto send on edits and discards stale conversions', async () => {
    vi.useFakeTimers()
    try {
      let resolveHex!: (value: string) => void
      const pendingHex = new Promise<string>(resolve => {
        resolveHex = resolve
      }) as ReturnType<typeof EncodeTextToHex>
      vi.mocked(EncodeTextToHex).mockReturnValueOnce(pendingHex)
      const wrapper = mount(SendPanel, {
        props: { handleId: '' },
      })
      const textarea = wrapper.find('textarea')

      await textarea.setValue('ignored')
      await wrapper.find('button').trigger('click')
      expect(Send).not.toHaveBeenCalled()

      await wrapper.setProps({ handleId: 'port-1' })
      await textarea.setValue('')
      await wrapper.find('button').trigger('click')
      expect(Send).not.toHaveBeenCalled()

      await textarea.setValue('slow')
      await wrapper.find('select').setValue('hex')
      await textarea.setValue('00')
      resolveHex('73 6c 6f 77')
      await flushPromises()
      expect((textarea.element as HTMLTextAreaElement).value).toBe('00')

      await wrapper.find('select').setValue('ascii')
      await flushPromises()
      await textarea.setValue('alpha')
      await wrapper.find('input[type="number"]').setValue('10')
      await wrapper.find<HTMLInputElement>('input[type="checkbox"]').setValue(true)
      await vi.advanceTimersByTimeAsync(10)
      expect(Send).toHaveBeenCalledTimes(1)
      expect(Send).toHaveBeenLastCalledWith(expect.objectContaining({ Content: 'alpha' }))

      await textarea.setValue('beta')
      await vi.advanceTimersByTimeAsync(10)
      expect(Send).toHaveBeenCalledTimes(2)
      expect(Send).toHaveBeenLastCalledWith(expect.objectContaining({ Content: 'beta' }))
      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
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
