import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatusBar from './StatusBar.vue'

const wailsRuntime = vi.hoisted(() => {
  const off = vi.fn()
  return {
    off,
    on: vi.fn(() => off),
  }
})

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: wailsRuntime.on,
  },
}))

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

  it('renders MCP server status when provided', async () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'serial',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
        mcpStatus: {
          Enabled: true,
          Running: true,
          Address: '127.0.0.1:39391',
          Path: '/mcp',
          Error: '',
        },
      },
    })

    expect(wrapper.find('.status-bar__mcp').text()).toContain('MCP 127.0.0.1:39391')
  })

  it('marks the app name dirty when workspace has unsaved changes', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'serial',
        dirty: true,
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__left').text()).toContain('MockTrue* v0.1.0')
  })

  it('renders the current workspace file path', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'settings',
        configPath: '/tmp/mocktrue-session.json',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__config').text()).toBe('配置 /tmp/mocktrue-session.json')
    expect(wrapper.find('.status-bar__config').attributes('title')).toBe('/tmp/mocktrue-session.json')
  })

  it('shows an empty config path placeholder before a workspace file is selected', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'settings',
        configPath: '',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__config').text()).toBe('配置 未指定')
  })
})
