import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { computed, nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HexClassicView from './HexClassicView.vue'
import { useSerialStore } from '../stores/serialStore'

const pageData = vi.hoisted(() => Uint8Array.from([
  0x41, 0x00, 0x7e, 0x7f, 0x20, 0x5a, 0x30, 0x31,
  0x02, 0xff, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66,
  0x67, 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e,
  0x6f, 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76,
]))

const serviceApi = vi.hoisted(() => ({
  QueryPage: vi.fn(async (_port: string, _offset: number, length: number) => ({
    Data: length === 0 ? null : pageData,
    Total: pageData.length,
  })),
}))

const runtimeApi = vi.hoisted(() => {
  const state: {
    handler: ((event: { data: unknown }) => void) | null
    cancel: ReturnType<typeof vi.fn>
    On: ReturnType<typeof vi.fn>
    reset: () => void
  } = {
    handler: null,
    cancel: vi.fn(),
    On: vi.fn(),
    reset() {
      state.handler = null
      state.cancel.mockClear()
      state.On.mockClear()
    },
  }
  state.On.mockImplementation((_eventName: string, handler: (event: { data: unknown }) => void) => {
    state.handler = handler
    return state.cancel
  })
  return state
})

const virtualWindowApi = vi.hoisted(() => {
  const scrollToIndex = vi.fn()
  return {
    scrollToIndex,
    useVirtualWindow: vi.fn((options: { rowCount: { value: number }; rowHeight: number }) => ({
      virtualItems: computed(() => Array.from({ length: Math.min(options.rowCount.value, 2) }, (_value, index) => ({
        index,
        key: index,
        start: index * options.rowHeight,
        size: options.rowHeight,
      }))),
      totalSize: computed(() => options.rowCount.value * options.rowHeight),
      scrollToIndex,
    })),
  }
})

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => serviceApi)
vi.mock('../services/serialService', () => ({ serialService: {} }))
vi.mock('@wailsio/runtime', () => ({ Events: { On: runtimeApi.On } }))
vi.mock('../../shared/composables/useVirtualWindow', () => ({ useVirtualWindow: virtualWindowApi.useVirtualWindow }))

describe('HexClassicView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    runtimeApi.reset()
    vi.clearAllMocks()
    serviceApi.QueryPage.mockImplementation(async (_port: string, _offset: number, length: number) => ({
      Data: length === 0 ? null : pageData,
      Total: pageData.length,
    }))
  })

  it('formats rows only after matching serial data updates the reachable virtual window', async () => {
    const serial = useSerialStore()
    serial.setActivePort('port-1')

    const wrapper = mount(HexClassicView)
    await flushPromises()

    expect(virtualWindowApi.useVirtualWindow).toHaveBeenCalledWith(expect.objectContaining({
      rowHeight: 20,
    }))
    expect(runtimeApi.On).toHaveBeenCalledWith('serial:data', expect.any(Function))
    expect(serviceApi.QueryPage).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('00000000')

    runtimeApi.handler?.({ data: { PortID: 'other-port' } })
    await flushPromises()

    expect(serviceApi.QueryPage).not.toHaveBeenCalled()

    runtimeApi.handler?.({ data: { PortID: 'port-1' } })
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    expect(serviceApi.QueryPage).toHaveBeenCalledWith('port-1', 0, 0)
    expect(serviceApi.QueryPage).toHaveBeenCalledWith('port-1', 0, 64 * 1024)
    expect(wrapper.text()).toContain('00000000')
    expect(wrapper.text()).toContain('41 00 7e 7f 20 5a 30 31 02 ff 61 62 63 64 65 66')
    expect(wrapper.text()).toContain('A.~. Z01..abcdef')
    expect(wrapper.text()).toContain('00000010')
    expect(wrapper.text()).toContain('67 68 69 6a 6b 6c 6d 6e 6f 70 71 72 73 74 75 76')
    expect(wrapper.text()).toContain('ghijklmnopqrstuv')

    wrapper.unmount()
    expect(runtimeApi.cancel).toHaveBeenCalledTimes(1)
  })
})
