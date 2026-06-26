import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatusBar from './StatusBar.vue'

const wailsRuntime = vi.hoisted(() => {
  const off = vi.fn()
  return {
    off,
    on: vi.fn((_event: string, _callback: (event: any) => void) => off),
  }
})

vi.mock('@wailsio/runtime', () => ({
  Call: {
    ByID: vi.fn(async () => undefined),
  },
  CancellablePromise: Promise,
  Create: {
    Any: (value: any) => value,
    Array: (createItem: (value: any) => any) => (value: any[]) => Array.isArray(value) ? value.map(createItem) : [],
    ByteSlice: (value: any) => value ?? '',
    Nullable: (createValue: (value: any) => any) => (value: any) => value == null ? null : createValue(value),
  },
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

  it('does not mark the app name dirty from workspace state', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'serial',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__left').text()).toContain('PortWeave v0.1.0')
  })

  it('does not render graph config file state in the status bar', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'settings',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__config').exists()).toBe(false)
  })

  it('formats fallback runtime values and disabled MCP state', async () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: null,
        runtimeMetrics: {
          CPUPercent: Number.NaN,
          MemoryBytes: 512,
        },
        mcpStatus: {
          Enabled: false,
          Running: false,
          Address: '127.0.0.1:39391',
          Path: '/mcp',
          Error: '',
        },
      },
    })

    expect(wrapper.find('.status-bar__metrics').text()).toContain('CPU 0.0%')
    expect(wrapper.find('.status-bar__metrics').text()).toContain('内存 512 B')
    expect(wrapper.find('.status-bar__active').text()).toBe('—')
    expect(wrapper.find('.status-bar__mcp').exists()).toBe(false)

    await wrapper.setProps({
      runtimeMetrics: {
        CPUPercent: -5,
        MemoryBytes: 1536,
      },
      mcpStatus: {
        Enabled: true,
        Running: false,
        Address: '127.0.0.1:39391',
        Path: '/mcp',
        Error: 'listen failed',
      },
    })

    expect(wrapper.find('.status-bar__metrics').text()).toContain('CPU 0.0%')
    expect(wrapper.find('.status-bar__metrics').text()).toContain('内存 1.5 KB')
    expect(wrapper.find('.status-bar__mcp').text()).toBe('MCP 错误')

    await wrapper.setProps({
      runtimeMetrics: {
        CPUPercent: Number.POSITIVE_INFINITY,
        MemoryBytes: 2 * 1024 * 1024 * 1024,
      },
      mcpStatus: {
        Enabled: true,
        Running: false,
        Address: '127.0.0.1:39391',
        Path: '/mcp',
        Error: '',
      },
    })

    expect(wrapper.find('.status-bar__metrics').text()).toContain('CPU 0.0%')
    expect(wrapper.find('.status-bar__metrics').text()).toContain('内存 2.0 GB')
    expect(wrapper.find('.status-bar__mcp').text()).toBe('MCP 停止')
  })

  it('samples runtime metrics and unsubscribes MCP status events when props are omitted', async () => {
    vi.useFakeTimers()
    try {
      wailsRuntime.on.mockImplementationOnce((_event: string, callback: (event: any) => void) => {
        callback({
          data: {
            Enabled: true,
            Running: true,
            Address: '127.0.0.1:39391',
            Path: '/mcp',
            Error: '',
          },
        })
        return wailsRuntime.off
      })

      const wrapper = mount(StatusBar, {
        props: {
          activeId: 'serial',
        },
      })
      await wrapper.vm.$nextTick()

      expect(wailsRuntime.on).toHaveBeenCalledWith('mcp:status', expect.any(Function))
      expect(wrapper.find('.status-bar__mcp').text()).toBe('MCP 127.0.0.1:39391/mcp')
      expect(vi.getTimerCount()).toBe(1)

      wrapper.unmount()

      expect(wailsRuntime.off).toHaveBeenCalled()
      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })
})
