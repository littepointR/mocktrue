import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import HexTableView from './HexTableView.vue'
import { useSerialStore } from '../stores/serialStore'

const serviceApi = vi.hoisted(() => ({
  QueryPage: vi.fn(),
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

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => serviceApi)
vi.mock('../services/serialService', () => ({ serialService: {} }))
vi.mock('@wailsio/runtime', () => ({ Events: { On: runtimeApi.On } }))
vi.mock('naive-ui', () => ({
  NDataTable: {
    props: ['columns', 'data'],
    template: `
      <table data-testid="hex-table">
        <thead>
          <tr><th v-for="column in columns" :key="column.key">{{ column.title }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="row in data" :key="row.offset">
            <td v-for="column in columns" :key="column.key">{{ row[column.key] }}</td>
          </tr>
        </tbody>
      </table>
    `,
  },
}))

describe('HexTableView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    runtimeApi.reset()
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 23, 1, 2, 3, 4))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('appends matching serial data events as formatted rows and clears rows when the active port changes', async () => {
    const serial = useSerialStore()
    serial.setActivePort('port-1')
    const wrapper = mount(HexTableView)

    expect(runtimeApi.On).toHaveBeenCalledWith('serial:data', expect.any(Function))
    runtimeApi.handler?.({ data: { PortID: 'other-port', Data: btoa('skip') } })
    await nextTick()
    expect(wrapper.text()).not.toContain('skip')

    runtimeApi.handler?.({ data: { PortID: 'port-1', Data: btoa(String.fromCharCode(0x41, 0x00, 0x7e)) } })
    await nextTick()

    expect(wrapper.text()).toContain('Offset')
    expect(wrapper.text()).toContain('方向')
    expect(wrapper.text()).toContain('HEX')
    expect(wrapper.text()).toContain('ASCII')
    expect(wrapper.text()).toContain('时间')
    expect(wrapper.text()).toContain('RX')
    expect(wrapper.text()).toContain('41 00 7e')
    expect(wrapper.text()).toContain('A.~')
    expect(wrapper.text()).toContain('01:02:03.004')

    serial.setActivePort('port-2')
    await nextTick()

    expect(wrapper.text()).not.toContain('41 00 7e')
    expect(wrapper.text()).not.toContain('A.~')

    wrapper.unmount()
    expect(runtimeApi.cancel).toHaveBeenCalledTimes(1)
  })
})
