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
    bindings.StartSerialGraph.mockImplementation(async (req: { ID: string }) => ({
      ID: req.ID,
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    }))
    bindings.GetSerialGraphStatus.mockImplementation(async (id: string) => ({
      ID: id,
      Status: 'running',
      Error: '',
      Nodes: [
        { ID: 'node-1', Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 5, FrameCount: 0, ResourceID: '', Error: '' },
        { ID: 'node-2', Type: 'serial.receiver', Status: 'running', RxBytes: 5, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
      ],
    }))
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

  it('manages graphs from the workbench toolbar', async () => {
    const wrapper = mount(SerialGraphPanel)
    const store = useSerialGraphStore()

    expect(wrapper.find('[data-testid="serial-graph-switcher"]').findAll('option')).toHaveLength(1)
    expect((wrapper.find('[data-testid="serial-graph-name"]').element as HTMLInputElement).value).toBe('拓扑图 1')

    await wrapper.find('[data-testid="serial-graph-new"]').trigger('click')

    expect(store.graphList.map(graph => graph.name)).toEqual(['拓扑图 1', '拓扑图 2'])
    expect(store.activeGraphId).toBe('graph-2')
    expect(wrapper.find('[data-testid="serial-graph-switcher"]').findAll('option')).toHaveLength(2)

    const nameInput = wrapper.find('[data-testid="serial-graph-name"]')
    ;(nameInput.element as HTMLInputElement).value = '生产拓扑'
    await nameInput.trigger('change')

    expect(store.activeGraph?.name).toBe('生产拓扑')

    await wrapper.find('[data-testid="serial-graph-duplicate"]').trigger('click')

    expect(store.graphList.map(graph => graph.name)).toEqual(['拓扑图 1', '生产拓扑', '生产拓扑 副本'])
    expect(store.activeGraphId).toBe('graph-3')

    await wrapper.find('[data-testid="serial-graph-switcher"]').setValue('graph-1')

    expect(store.activeGraphId).toBe('graph-1')

    await wrapper.find('[data-testid="serial-graph-remove"]').trigger('click')
    await flushPromises()

    expect(store.graphList.map(graph => graph.id)).toEqual(['graph-2', 'graph-3'])
    expect(store.activeGraphId).toBe('graph-2')
  })

  it('renders the graph passed by tab id without switching the active graph', () => {
    const store = useSerialGraphStore()
    store.createGraph('辅助拓扑')
    store.setActiveGraph('graph-2')

    mount(SerialGraphPanel, {
      props: { graphId: 'graph-1' },
    })

    expect(store.activeGraphId).toBe('graph-2')
  })

  it('renders the graph scoped by its graphId when multiple panels are mounted', () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    store.createGraph('辅助拓扑')
    store.addNode('serial.receiver')

    const first = mount(SerialGraphPanel, {
      props: { graphId: 'graph-1' },
    })
    const second = mount(SerialGraphPanel, {
      props: { graphId: 'graph-2' },
    })

    expect(store.activeGraphId).toBe('graph-2')
    expect((first.find('[data-testid="serial-graph-switcher"]').element as HTMLSelectElement).value).toBe('graph-1')
    expect((second.find('[data-testid="serial-graph-switcher"]').element as HTMLSelectElement).value).toBe('graph-2')
    expect(first.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('发送器')
    expect(first.find('[data-testid="serial-graph-node-node-1"]').text()).not.toContain('接收器')
    expect(second.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('接收器')
  })

  it('updates rendered graph content when the tab graphId prop changes', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const secondGraph = store.createGraph('辅助拓扑')
    const receiver = store.addNode('serial.receiver')
    expect(sender.id).toBe('node-1')
    expect(receiver.id).toBe('node-1')

    const wrapper = mount(SerialGraphPanel, {
      props: { graphId: 'graph-1' },
    })
    expect((wrapper.find('[data-testid="serial-graph-switcher"]').element as HTMLSelectElement).value).toBe('graph-1')
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('发送器')

    await wrapper.setProps({ graphId: secondGraph.id })
    await nextTick()

    expect((wrapper.find('[data-testid="serial-graph-switcher"]').element as HTMLSelectElement).value).toBe(secondGraph.id)
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('接收器')
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).not.toContain('发送器')
  })

  it('starts a scoped graph panel runtime without switching the active graph', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    const second = store.createGraph('辅助拓扑')
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.setActiveGraph('graph-1')
    const wrapper = mount(SerialGraphPanel, {
      props: { graphId: second.id },
    })

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()

    expect(store.activeGraphId).toBe('graph-1')
    expect(bindings.StartSerialGraph).toHaveBeenCalledWith(expect.objectContaining({
      ID: second.id,
      Nodes: [
        expect.objectContaining({ ID: sender.id, Type: 'serial.sender' }),
        expect.objectContaining({ ID: receiver.id, Type: 'serial.receiver' }),
      ],
    }))
    expect(wrapper.find('[data-testid="serial-graph-runtime-status"]').text()).toBe('running')

    wrapper.unmount()
  })

  it('keeps toolbar graph switching local to a scoped panel', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    store.createGraph('辅助拓扑')
    store.addNode('serial.receiver')
    store.setActiveGraph('graph-1')
    const wrapper = mount(SerialGraphPanel, {
      props: { graphId: 'graph-1' },
    })

    await wrapper.find('[data-testid="serial-graph-switcher"]').setValue('graph-2')

    expect(store.activeGraphId).toBe('graph-1')
    expect((wrapper.find('[data-testid="serial-graph-switcher"]').element as HTMLSelectElement).value).toBe('graph-2')
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('接收器')
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).not.toContain('发送器')
  })

  it('opens node content tabs when nodes are created and keeps nodes when tabs close', async () => {
    const wrapper = mount(SerialGraphPanel)
    const store = useSerialGraphStore()

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')

    const sender = store.nodes[0]
    expect(wrapper.find('.serial-graph__inspector').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(true)
    expect(wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"]`).text()).toContain('发送器')
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('payload')

    await wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"] .serial-graph__node-tab-close`).trigger('click')

    expect(store.nodes.map(node => node.id)).toEqual([sender.id])
    expect(store.nodeTabs).toHaveLength(0)
    expect(wrapper.find(`[data-testid="serial-graph-node-${sender.id}"]`).exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(false)

    await wrapper.find(`[data-testid="serial-graph-node-${sender.id}"]`).trigger('click')

    expect(store.nodeTabs.map(tab => tab.nodeId)).toEqual([sender.id])
    expect(wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"]`).exists()).toBe(true)
  })

  it('switches between content, split, and topology views from the toolbar', async () => {
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
    const store = useSerialGraphStore()

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')
    const sender = store.nodes[0]
    const contentButton = wrapper.find('[data-testid="serial-graph-view-content"]')
    const splitButton = wrapper.find('[data-testid="serial-graph-view-split"]')
    const topologyButton = wrapper.find('[data-testid="serial-graph-view-topology"]')
    const validation = wrapper.find('.serial-graph__validation')

    expect(contentButton.exists()).toBe(true)
    expect(splitButton.exists()).toBe(true)
    expect(topologyButton.exists()).toBe(true)
    expect(validation.element.nextElementSibling).toBe(wrapper.find('.serial-graph__view-switcher').element)
    expect(splitButton.attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="serial-graph-canvas"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(true)

    await contentButton.trigger('click')

    expect(contentButton.attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="serial-graph-canvas"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').classes()).toContain('serial-graph__node-workbench--full')
    expect(wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"]`).exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(false)

    await topologyButton.trigger('click')

    expect(topologyButton.attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="serial-graph-canvas"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(false)

    await splitButton.trigger('click')

    expect(splitButton.attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="serial-graph-canvas"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(true)

    wrapper.unmount()
  })

  it('resizes the split view with pointer and keyboard controls', async () => {
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')
    const workspace = wrapper.find<HTMLElement>('.serial-graph__workspace')
    Object.defineProperty(workspace.element, 'clientHeight', { value: 720, configurable: true })
    const toolbar = wrapper.find<HTMLElement>('.serial-graph__toolbar')
    Object.defineProperty(toolbar.element, 'offsetHeight', { value: 32, configurable: true })

    const workbench = wrapper.find<HTMLElement>('[data-testid="serial-graph-node-workbench"]')
    expect(workbench.attributes('style')).toContain('320px')

    const handle = wrapper.find<HTMLElement>('[data-testid="serial-graph-split-resize-handle"]')
    handle.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 1, clientY: 400 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 1, clientY: 360 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1 }))
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').attributes('style')).toContain('360px')

    await handle.trigger('keydown', { key: 'ArrowDown' })
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').attributes('style')).toContain('348px')

    await handle.trigger('keydown', { key: 'ArrowUp', shiftKey: true })
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').attributes('style')).toContain('388px')

    handle.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 1, clientY: 400 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 1, clientY: -200 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1 }))
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').attributes('style')).toContain('562px')

    wrapper.unmount()
  })

  it('shows the remaining node content when the active node tab is closed', async () => {
    const wrapper = mount(SerialGraphPanel)
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    await nextTick()

    expect(store.activeNodeTabId).toBe(receiver.id)
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('刷新')

    await wrapper.find(`[data-testid="serial-graph-node-tab-${receiver.id}"] .serial-graph__node-tab-close`).trigger('click')

    expect(store.nodes.map(node => node.id)).toEqual([sender.id, receiver.id])
    expect(store.nodeTabs.map(tab => tab.nodeId)).toEqual([sender.id])
    expect(store.activeNodeTabId).toBe(sender.id)
    expect(store.selectedNodeId).toBe(sender.id)
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('payload')
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

  it('selects and deletes a connection line from the bottom details area', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const edge = store.connect(sender.id, 'out', receiver.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find(`[data-testid="serial-graph-edge-${edge?.id}"]`).trigger('click')

    expect(store.selectedEdgeId).toBe(edge?.id)
    expect(wrapper.find('.serial-graph__inspector').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-selected-edge"]').text()).toContain(`${sender.id}.out`)

    await wrapper.find('[data-testid="serial-graph-delete-edge"]').trigger('click')

    expect(store.edges).toHaveLength(0)
    expect(store.selectedEdgeId).toBeNull()
  })

  it('preserves number and boolean config types when editing', async () => {
    const store = useSerialGraphStore()
    const port = store.addNode('serial.physical')
    const wrapper = mount(SerialGraphPanel)

    const baudRateInput = wrapper.find('[data-testid="serial-graph-config-baudRate"]')
    expect(baudRateInput.exists()).toBe(true)

    await baudRateInput.setValue('57600')

    expect(store.nodes.find(node => node.id === port.id)?.config.baudRate).toBe(57600)
    expect(typeof store.nodes.find(node => node.id === port.id)?.config.baudRate).toBe('number')

    const receiver = store.addNode('serial.receiver')
    await nextTick()

    const autoScrollInput = wrapper.find('[data-testid="serial-graph-config-autoScroll"]')
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

    await wrapper.find('[data-testid="serial-graph-content-send-payload"]').setValue('hello')
    await wrapper.find('[data-testid="serial-graph-content-send"]').trigger('click')

    expect(bindings.SendSerialGraphNode).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'graph-1',
      NodeID: sender.id,
      Content: 'hello',
    }))

    store.selectNode(receiver.id)
    await nextTick()
    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').text()).toContain('hello')
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').classes()).toContain('serial-graph__buffer--content')
    expect(wrapper.find('[data-testid="serial-graph-send"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-buffer"]').exists()).toBe(false)

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
      GraphID: 'graph-1',
      NodeID: receiver.id,
    }))
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').text()).toContain('hello')

    wrapper.unmount()
  })

  it('keeps node card counters polling when no node is selected', async () => {
    vi.useFakeTimers()
    try {
      const store = useSerialGraphStore()
      const sender = store.addNode('serial.sender')
      const receiver = store.addNode('serial.receiver')
      store.connect(sender.id, 'out', receiver.id, 'in')
      const wrapper = mount(SerialGraphPanel)

      await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
      await flushPromises()
      bindings.GetSerialGraphStatus.mockClear()
      bindings.GetSerialGraphStatus.mockResolvedValue({
        ID: 'graph-1',
        Status: 'running',
        Error: '',
        Nodes: [
          { ID: sender.id, Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 25, FrameCount: 0, ResourceID: '', Error: '' },
          { ID: receiver.id, Type: 'serial.receiver', Status: 'running', RxBytes: 25, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
        ],
      })
      store.selectNode(null)
      await nextTick()

      await vi.advanceTimersByTimeAsync(250)
      await flushPromises()

      expect(bindings.GetSerialGraphStatus).toHaveBeenCalledWith('graph-1')
      expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).toContain('RX 25')

      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
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
