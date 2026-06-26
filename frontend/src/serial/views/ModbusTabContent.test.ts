import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ModbusTabContent from './ModbusTabContent.vue'

describe('ModbusTabContent', () => {
  it('renders the Modbus protocol panel in tab mode for the provided session', () => {
    const wrapper = mount(ModbusTabContent, {
      props: { sessionId: 'modbus-1' },
      global: { stubs },
    })

    const panel = wrapper.find('[data-testid="modbus-panel"]')
    expect(panel.exists()).toBe(true)
    expect(panel.attributes('data-variant')).toBe('tab')
    expect(panel.attributes('data-session-id')).toBe('modbus-1')
  })
})

const stubs = {
  ModbusPanel: {
    props: ['variant', 'sessionId'],
    template: '<div data-testid="modbus-panel" :data-variant="variant" :data-session-id="sessionId" />',
  },
}
