import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatusBar from './StatusBar.vue'

describe('StatusBar', () => {
  it('renders runtime CPU and memory usage', async () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'serial',
        runtimeMetrics: {
          CPUPercent: 12.34,
          MemoryBytes: 52 * 1024 * 1024,
        },
      },
    })

    expect(wrapper.find('.status-bar__metrics').text()).toContain('CPU 12.3%')
    expect(wrapper.find('.status-bar__metrics').text()).toContain('内存 52.0 MB')
  })
})
