import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SerialGraphPanel from './SerialGraphPanel.vue'
import { useSerialGraphStore } from '../stores/graphStore'

const bindings = vi.hoisted(() => ({
  StartSerialGraph: vi.fn(),
  StopSerialGraph: vi.fn(),
  GetSerialGraphStatus: vi.fn(),
  SendSerialGraphNode: vi.fn(),
  QuerySerialGraphNodeBuffer: vi.fn(),
  QuerySerialGraphNodeFrames: vi.fn(),
  ClearSerialGraphNodeBuffer: vi.fn(),
  ResetSerialGraphNodeCounters: vi.fn(),
}))

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => bindings)

describe('SerialGraphPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    bindings.StartSerialGraph.mockResolvedValue({
      ID: 'serial.graph',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    bindings.GetSerialGraphStatus.mockResolvedValue({
      ID: 'serial.graph',
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    })
    bindings.SendSerialGraphNode.mockResolvedValue(5)
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa('hello'),
      Total: 5,
      EOF: false,
    })
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [{ Seq: 1, Direction: '接收', Length: 5, DisplayText: 'hello', DisplayHex: '68 65 6c 6c 6f' }],
      Total: 1,
      NextOffset: 1,
    })
  })

  it('renders the graph workbench and adds nodes from the provider palette', async () => {
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-graph-panel"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-palette"]').text()).toContain('发送器')

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')

    const store = useSerialGraphStore()
    expect(store.nodes[0].type).toBe('serial.sender')
    expect(wrapper.find(`[data-testid="serial-graph-node-${store.nodes[0].id}"]`).exists()).toBe(true)
  })

  it('connects an output handle to an input handle through the UI', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find(`[data-testid="serial-graph-output-${sender.id}-out"]`).trigger('click')
    await wrapper.find(`[data-testid="serial-graph-input-${receiver.id}-in"]`).trigger('click')

    expect(store.edges).toHaveLength(1)
    expect(wrapper.find('[data-testid="serial-graph-edge-edge-1"]').exists()).toBe(true)
  })

  it('selects and deletes a connection line from the inspector', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const edge = store.connect(sender.id, 'out', receiver.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find(`[data-testid="serial-graph-edge-${edge?.id}"]`).trigger('click')

    expect(store.selectedEdgeId).toBe(edge?.id)
    expect(wrapper.find('[data-testid="serial-graph-selected-edge"]').text()).toContain(`${sender.id}.out`)

    await wrapper.find('[data-testid="serial-graph-delete-edge"]').trigger('click')

    expect(store.edges).toHaveLength(0)
    expect(store.selectedEdgeId).toBeNull()
  })

  it('preserves number and boolean config types when editing', async () => {
    const store = useSerialGraphStore()
    const port = store.addNode('serial.physical')
    const wrapper = mount(SerialGraphPanel)

    const baudRateInput = wrapper.findAll('input').find(input => (
      (input.element as HTMLInputElement).value === '115200'
    ))
    expect(baudRateInput).toBeTruthy()

    await baudRateInput!.setValue('57600')

    expect(store.nodes.find(node => node.id === port.id)?.config.baudRate).toBe(57600)
    expect(typeof store.nodes.find(node => node.id === port.id)?.config.baudRate).toBe('number')

    const receiver = store.addNode('serial.receiver')
    await nextTick()

    const autoScrollInput = wrapper.find('input[type="checkbox"]')
    expect(autoScrollInput.exists()).toBe(true)

    await autoScrollInput.setValue(false)

    expect(store.nodes.find(node => node.id === receiver.id)?.config.autoScroll).toBe(false)
    expect(typeof store.nodes.find(node => node.id === receiver.id)?.config.autoScroll).toBe('boolean')
  })

  it('pans the canvas by dragging the empty background with the left mouse button', async () => {
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
    const canvas = wrapper.find<HTMLElement>('[data-testid="serial-graph-canvas"]')
    Object.defineProperty(canvas.element, 'scrollLeft', { value: 120, writable: true })
    Object.defineProperty(canvas.element, 'scrollTop', { value: 80, writable: true })

    const downEvent = pointerTestEvent('pointerdown', { button: 0, pointerId: 1, clientX: 240, clientY: 180 })
    const preventDefault = vi.spyOn(downEvent, 'preventDefault')
    canvas.element.dispatchEvent(downEvent)
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 1, clientX: 210, clientY: 140 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1 }))

    expect(preventDefault).toHaveBeenCalled()
    expect(canvas.element.scrollLeft).toBe(150)
    expect(canvas.element.scrollTop).toBe(120)

    wrapper.unmount()
  })

  it('starts runtime, sends payload, and renders receiver buffer without screenshot checks', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(sender.id)
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await nextTick()

    expect(bindings.StartSerialGraph).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="serial-graph-runtime-status"]').text()).toBe('running')

    await wrapper.find('[data-testid="serial-graph-send-payload"]').setValue('hello')
    await wrapper.find('[data-testid="serial-graph-send"]').trigger('click')

    expect(bindings.SendSerialGraphNode).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'serial.graph',
      NodeID: sender.id,
      Content: 'hello',
    }))

    store.selectNode(receiver.id)
    await nextTick()
    await wrapper.find('[data-testid="serial-graph-refresh-buffer"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-node-buffer"]').text()).toContain('hello')

    wrapper.unmount()
  })

  it('polls the selected receiver buffer automatically after runtime start', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()

    expect(bindings.QuerySerialGraphNodeBuffer).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'serial.graph',
      NodeID: receiver.id,
    }))
    expect(wrapper.find('[data-testid="serial-graph-node-buffer"]').text()).toContain('hello')

    wrapper.unmount()
  })
})

function pointerTestEvent(type: string, init: { button?: number; pointerId: number; clientX?: number; clientY?: number }): Event {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    button: { value: init.button ?? 0 },
    pointerId: { value: init.pointerId },
    clientX: { value: init.clientX ?? 0 },
    clientY: { value: init.clientY ?? 0 },
  })
  return event
}
