import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SerialSettingsPanel from './SerialSettingsPanel.vue'
import { useSettingsStore } from '../../stores/settingsStore'
import { ListSystemFonts } from '../../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'

vi.mock('naive-ui', () => ({
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NSelect: {
    props: ['value', 'options', 'renderLabel'],
    emits: ['update:value'],
    template: `
      <select :value="value" @change="$emit('update:value', $event.target.value)">
        <option
          v-for="option in options"
          :key="option.value"
          :value="option.value"
          :style="renderLabel ? renderLabel(option).props.style : undefined"
        >{{ renderLabel ? renderLabel(option).children : option.label }}</option>
      </select>
    `,
  },
  NInputNumber: {
    props: ['value'],
    emits: ['update:value'],
    template: '<input type="number" :value="value" @input="$emit(\'update:value\', Number($event.target.value))" />',
  },
  NButton: { template: '<button type="button"><slot /></button>' },
}))

vi.mock('../../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => ({
  ListSystemFonts: vi.fn(async () => ['System Mono', 'PingFang SC']),
}))

describe('SerialSettingsPanel', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('updates terminal display and text settings', async () => {
    const wrapper = mount(SerialSettingsPanel)

    const selects = wrapper.findAll('select')
    await selects.at(5)?.setValue('Monaco')
    await wrapper.findAll('input[type="number"]').at(1)?.setValue('16')
    await selects.at(6)?.setValue('gbk')
    await selects.at(7)?.setValue('\r\n')

    const store = useSettingsStore()
    expect(store.serial).toMatchObject({
      TerminalFontFamily: 'Monaco',
      TerminalFontSize: 16,
      TextEncoding: 'gbk',
      EnterString: '\r\n',
    })
  })

  it('loads font options from the system font list', async () => {
    const wrapper = mount(SerialSettingsPanel)
    await vi.dynamicImportSettled()

    const fontOptions = wrapper.findAll('select').at(5)?.findAll('option').map(option => option.text())

    expect(ListSystemFonts).toHaveBeenCalled()
    expect(fontOptions).toContain('System Mono')
    expect(fontOptions).toContain('PingFang SC')
  })

  it('previews each font option with its own font family', async () => {
    const wrapper = mount(SerialSettingsPanel)
    await vi.dynamicImportSettled()

    const options = wrapper.findAll('select').at(5)?.findAll('option') ?? []
    const systemMonoOption = options.find(option => option.text() === 'System Mono')

    expect(systemMonoOption?.element.style.fontFamily).toBe('"System Mono", monospace')
  })
})
