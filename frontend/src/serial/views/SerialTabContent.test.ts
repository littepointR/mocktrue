import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import SerialTabContent from './SerialTabContent.vue'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'

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
})
