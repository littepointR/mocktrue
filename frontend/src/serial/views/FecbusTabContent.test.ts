import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FecbusTabContent from './FecbusTabContent.vue'

describe('FecbusTabContent', () => {
  it('renders the FECbus protocol panel in tab mode for the provided session', () => {
    const wrapper = mount(FecbusTabContent, {
      props: { sessionId: 'fec-1' },
      global: { stubs },
    })

    const panel = wrapper.find('[data-testid="fecbus-panel"]')
    expect(panel.exists()).toBe(true)
    expect(panel.attributes('data-variant')).toBe('tab')
    expect(panel.attributes('data-session-id')).toBe('fec-1')
  })
})

const stubs = {
  FecbusPanel: {
    props: ['variant', 'sessionId'],
    template: '<div data-testid="fecbus-panel" :data-variant="variant" :data-session-id="sessionId" />',
  },
}
