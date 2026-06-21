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

    expect(wrapper.find('.status-bar__left').text()).toContain('MockTrue v0.1.0')
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

    expect(wrapper.find('.status-bar__config').text()).toBe('配置 未保存')
  })

  it('renders the current config location when provided', () => {
    const wrapper = mount(StatusBar, {
      props: {
        activeId: 'settings',
        configPath: '/tmp/example.mocktrue.json',
        runtimeMetrics: {
          CPUPercent: 0,
          MemoryBytes: 0,
        },
      },
    })

    expect(wrapper.find('.status-bar__config').text()).toBe('配置 /tmp/example.mocktrue.json')
    expect(wrapper.find('.status-bar__config').attributes('title')).toBe('/tmp/example.mocktrue.json')
  })
})
