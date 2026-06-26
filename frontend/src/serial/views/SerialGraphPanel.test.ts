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

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => bindings)
vi.mock('./ScriptEditor.vue', () => ({
  default: {
    props: ['modelValue', 'nodeType'],
    emits: ['script-change', 'update:modelValue'],
    template: '<div class="serial-script-editor" data-testid="serial-script-editor">script editor</div>',
  },
}))

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
    expect(wrapper.find('[data-testid="serial-graph-provider-serial.modbus.slave"]').text()).toContain('协议合法的零值/默认响应')
    expect(wrapper.find('[data-testid="serial-graph-provider-serial.modbus.slave"]').text()).toContain('完整可编辑的多 Unit 数据模型请使用 Modbus 会话面板')

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')

    const store = useSerialGraphStore()
    expect(store.nodes[0].type).toBe('serial.sender')
    expect(wrapper.find(`[data-testid="serial-graph-node-${store.nodes[0].id}"]`).exists()).toBe(true)
  })

  it('manages graphs from the workbench toolbar', async () => {
    const wrapper = mount(SerialGraphPanel)
    const store = useSerialGraphStore()

    expect(wrapper.find('[data-testid="serial-graph-switcher"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-demo-select"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-load-demo"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-new"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-open-config"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-save-config"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-save-config-as"]').exists()).toBe(false)
    expect((wrapper.find('[data-testid="serial-graph-name"]').element as HTMLInputElement).value).toBe('拓扑图 1')

    store.createGraph('第二拓扑')
    await nextTick()
    expect(store.activeGraphId).toBe('graph-2')

    const nameInput = wrapper.find('[data-testid="serial-graph-name"]')
    ;(nameInput.element as HTMLInputElement).value = '生产拓扑'
    await nameInput.trigger('change')

    expect(store.activeGraph?.name).toBe('生产拓扑')

    await wrapper.find('[data-testid="serial-graph-duplicate"]').trigger('click')

    expect(store.graphList.map(graph => graph.name)).toEqual(['拓扑图 1', '生产拓扑', '生产拓扑 副本'])
    expect(store.activeGraphId).toBe('graph-3')

    await wrapper.find('[data-testid="serial-graph-remove"]').trigger('click')
    await flushPromises()

    expect(store.graphList.map(graph => graph.id)).toEqual(['graph-1', 'graph-2'])
    expect(store.activeGraphId).toBe('graph-1')
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
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('发送器')

    await wrapper.setProps({ graphId: secondGraph.id })
    await nextTick()

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

  it('duplicates graphs locally from a scoped panel without switching the active graph', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.sender')
    store.createGraph('辅助拓扑')
    store.addNode('serial.receiver')
    store.setActiveGraph('graph-1')
    const wrapper = mount(SerialGraphPanel, {
      props: { graphId: 'graph-1' },
    })

    await wrapper.find('[data-testid="serial-graph-duplicate"]').trigger('click')

    expect(store.activeGraphId).toBe('graph-1')
    expect(store.graphList.map(graph => graph.id)).toEqual(['graph-1', 'graph-2', 'graph-3'])
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).toContain('发送器')
    expect(wrapper.find('[data-testid="serial-graph-node-node-1"]').text()).not.toContain('接收器')
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
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').classes()).toContain('serial-graph__node-workbench--empty')

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

  it('keeps an empty content pane visible in split view before a node is selected', async () => {
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })

    expect(wrapper.find('[data-testid="serial-graph-view-split"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="serial-graph-canvas"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').classes()).toContain('serial-graph__node-workbench--empty')
    expect(wrapper.find('[data-testid="serial-graph-empty-content"]').text()).toContain('内容区')
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(true)

    await wrapper.find('[data-testid="serial-graph-view-topology"]').trigger('click')

    expect(wrapper.find('[data-testid="serial-graph-canvas"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-split-resize-handle"]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('keeps graphless toolbar palette and log actions as no-ops', async () => {
    const store = useSerialGraphStore()
    await store.removeGraph('graph-1')
    const wrapper = mount(SerialGraphPanel)

    expect(store.graphList).toEqual([])
    expect((wrapper.find('[data-testid="serial-graph-name"]').element as HTMLInputElement).value).toBe('')

    await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')

    expect(store.graphList).toEqual([])

    const nameInput = wrapper.find('[data-testid="serial-graph-name"]')
    ;(nameInput.element as HTMLInputElement).value = '无图拓扑'
    await nameInput.trigger('change')
    await wrapper.find('[data-testid="serial-graph-duplicate"]').trigger('click')

    expect(store.graphList).toEqual([])

    await wrapper.find('[data-testid="serial-graph-open-operation-log"]').trigger('click')

    expect(wrapper.find('[data-testid="serial-operation-log-empty"]').text()).toContain('没有匹配的操作日志')

    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('payload')
    await wrapper.find('[data-testid="serial-operation-log-clear"]').trigger('click')

    expect(store.operationLogs).toEqual([])

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

    await handle.trigger('keydown', { key: 'Escape' })
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

  it('ignores input clicks without a pending output and duplicate connection attempts', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find(`[data-testid="serial-graph-input-${receiver.id}-in"]`).trigger('click')

    expect(store.edges).toHaveLength(0)

    await wrapper.find(`[data-testid="serial-graph-output-${sender.id}-out"]`).trigger('click')
    await wrapper.find(`[data-testid="serial-graph-input-${receiver.id}-in"]`).trigger('click')

    expect(store.edges).toHaveLength(1)

    await wrapper.find(`[data-testid="serial-graph-output-${sender.id}-out"]`).trigger('click')
    await wrapper.find(`[data-testid="serial-graph-input-${receiver.id}-in"]`).trigger('click')

    expect(store.edges).toHaveLength(1)
    wrapper.unmount()
  })

  it('uses matching colors for connected edges and their port buttons', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    const edge = wrapper.find('[data-testid="serial-graph-edge-line-edge-1"]')
    const output = wrapper.find(`[data-testid="serial-graph-output-${sender.id}-out"]`)
    const input = wrapper.find(`[data-testid="serial-graph-input-${receiver.id}-in"]`)

    expect(edge.attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(output.attributes('style')).toContain('--port-edge-color: #4fc3f7')
    expect(input.attributes('style')).toContain('--port-edge-color: #4fc3f7')
    expect(output.classes()).toContain('serial-graph__port--connected')
    expect(input.classes()).toContain('serial-graph__port--connected')
  })

  it('changes colors at each node endpoint while keeping each endpoint consistent', () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const virtualPort = store.addNode('serial.virtual')
    const tap = store.addNode('serial.tap')
    const receiverA = store.addNode('serial.receiver')
    const receiverB = store.addNode('serial.receiver')
    const senderToVirtual = store.connect(sender.id, 'out', virtualPort.id, 'tx')
    const virtualToTap = store.connect(virtualPort.id, 'rx', tap.id, 'in')
    const tapToReceiverA = store.connect(tap.id, 'out', receiverA.id, 'in')
    const tapToReceiverB = store.connect(tap.id, 'out', receiverB.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${senderToVirtual?.id}"]`).attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${virtualToTap?.id}"]`).attributes('style')).toContain('--edge-color: #ffb74d')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${tapToReceiverA?.id}"]`).attributes('style')).toContain('--edge-color: #81c784')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${tapToReceiverB?.id}"]`).attributes('style')).toContain('--edge-color: #81c784')
    expect(wrapper.find(`[data-testid="serial-graph-input-${virtualPort.id}-tx"]`).attributes('style')).toContain('--port-edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-output-${virtualPort.id}-rx"]`).attributes('style')).toContain('--port-edge-color: #ffb74d')
    expect(wrapper.find(`[data-testid="serial-graph-input-${tap.id}-in"]`).attributes('style')).toContain('--port-edge-color: #ffb74d')
    expect(wrapper.find(`[data-testid="serial-graph-output-${tap.id}-out"]`).attributes('style')).toContain('--port-edge-color: #81c784')
    expect(wrapper.find(`[data-testid="serial-graph-output-${tap.id}-out"]`).attributes('style')).toContain('--port-edge-marker: #81c784')
  })

  it('uses different colors for independent data paths', () => {
    const store = useSerialGraphStore()
    const senderA = store.addNode('serial.sender')
    const receiverA = store.addNode('serial.receiver')
    const senderB = store.addNode('serial.sender')
    const receiverB = store.addNode('serial.receiver')
    const first = store.connect(senderA.id, 'out', receiverA.id, 'in')
    const second = store.connect(senderB.id, 'out', receiverB.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${first?.id}"]`).attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${second?.id}"]`).attributes('style')).toContain('--edge-color: #ffb74d')
  })

  it('keeps every bridge endpoint visually distinct unless the same endpoint has multiple lines', () => {
    const store = useSerialGraphStore()
    const senderA = store.addNode('serial.sender')
    const receiverA = store.addNode('serial.receiver')
    const senderB = store.addNode('serial.sender')
    const receiverB = store.addNode('serial.receiver')
    const bridge = store.addNode('serial.bridge')
    const firstInput = store.connect(senderA.id, 'out', bridge.id, 'a-in')
    const firstOutput = store.connect(bridge.id, 'b-out', receiverA.id, 'in')
    const secondInput = store.connect(senderB.id, 'out', bridge.id, 'b-in')
    const secondOutput = store.connect(bridge.id, 'a-out', receiverB.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${firstInput?.id}"]`).attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${firstOutput?.id}"]`).attributes('style')).toContain('--edge-color: #ffb74d')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${secondInput?.id}"]`).attributes('style')).toContain('--edge-color: #81c784')
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${secondOutput?.id}"]`).attributes('style')).toContain('--edge-color: #ba68c8')
    expect(wrapper.find(`[data-testid="serial-graph-input-${bridge.id}-a-in"]`).attributes('style')).toContain('--port-edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-output-${bridge.id}-b-out"]`).attributes('style')).toContain('--port-edge-color: #ffb74d')
    expect(wrapper.find(`[data-testid="serial-graph-input-${bridge.id}-b-in"]`).attributes('style')).toContain('--port-edge-color: #81c784')
    expect(wrapper.find(`[data-testid="serial-graph-output-${bridge.id}-a-out"]`).attributes('style')).toContain('--port-edge-color: #ba68c8')
  })

  it('shows the shared endpoint color on ports with multiple connections', () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const tap = store.addNode('serial.tap')
    const receiverA = store.addNode('serial.receiver')
    const receiverB = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', tap.id, 'in')
    store.connect(tap.id, 'out', receiverA.id, 'in')
    store.connect(tap.id, 'out', receiverB.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    const output = wrapper.find(`[data-testid="serial-graph-output-${tap.id}-out"]`)

    expect(output.classes()).toContain('serial-graph__port--multi-connected')
    expect(output.attributes('style')).toContain('--port-edge-color: #ffb74d')
    expect(output.attributes('style')).toContain('--port-edge-marker: #ffb74d')
  })

  it('selects and deletes a connection line from the bottom details area', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const edge = store.connect(sender.id, 'out', receiver.id, 'in')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find(`[data-testid="serial-graph-edge-${edge?.id}"]`).trigger('click')

    expect(store.selectedEdgeId).toBe(edge?.id)
    expect(wrapper.find(`[data-testid="serial-graph-edge-line-${edge?.id}"]`).attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-edge-selection-${edge?.id}"]`).attributes('style')).toContain('--edge-color: #4fc3f7')
    expect(wrapper.find(`[data-testid="serial-graph-edge-selection-${edge?.id}"]`).exists()).toBe(true)
    expect(wrapper.find('.serial-graph__inspector').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-selected-edge"]').text()).toContain(`${sender.id}.out`)

    await wrapper.find('[data-testid="serial-graph-delete-edge"]').trigger('click')

    expect(store.edges).toHaveLength(0)
    expect(store.selectedEdgeId).toBeNull()
  })

  it('renders validation errors for selected edges and graph-level validation panes', async () => {
    const store = useSerialGraphStore()
    store.restoreState({
      graphs: [{
        id: 'graph-1',
        name: '坏拓扑',
        nodes: [{
          id: 'sender',
          type: 'serial.sender',
          position: { x: 24, y: 32 },
          config: { mode: 'ascii', encoding: 'utf-8', payload: '' },
        }],
        edges: [{ id: 'edge-bad', source: 'sender', sourceHandle: 'out', target: 'missing', targetHandle: 'in' }],
        selectedNodeId: null,
        selectedEdgeId: 'edge-bad',
        nodeTabs: [],
        activeNodeTabId: null,
      }],
      activeGraphId: 'graph-1',
    } as any)
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-graph-edge-content"]').text()).toContain('sender.out -> missing.in')
    expect(wrapper.find('[data-testid="serial-graph-edge-content"] .serial-graph__errors').text()).toContain('target node not found')

    store.selectEdgeInGraph('graph-1', null)
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-validation-content"]').text()).toContain('target node not found')

    wrapper.unmount()
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

  it('renders sender and receiver logging controls with stable test ids', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    const wrapper = mount(SerialGraphPanel)

    store.selectNode(sender.id)
    await nextTick()

    const enableLogging = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-enableLogging"]')
    const logLevel = wrapper.find<HTMLSelectElement>('[data-testid="serial-graph-config-logLevel"]')
    const logFormat = wrapper.find<HTMLTextAreaElement>('[data-testid="serial-graph-config-logFormat"]')
    expect(enableLogging.element.type).toBe('checkbox')
    expect(logLevel.element.tagName).toBe('SELECT')
    expect(logLevel.findAll('option').map(option => option.attributes('value'))).toEqual(['debug', 'info', 'warn', 'error'])
    expect(logFormat.element.tagName).toBe('TEXTAREA')

    await enableLogging.setValue(true)
    await logLevel.setValue('warn')
    await logFormat.setValue('{direction}:{text}')

    expect(store.nodes.find(node => node.id === sender.id)?.config).toEqual(expect.objectContaining({
      enableLogging: true,
      logLevel: 'warn',
      logFormat: '{direction}:{text}',
    }))

    store.selectNode(receiver.id)
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-config-enableLogging"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-config-logLevel"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-config-logFormat"]').exists()).toBe(true)

    wrapper.unmount()
  })

  it('renders logging controls for restored sender and receiver nodes that omit logging config', async () => {
    const store = useSerialGraphStore()
    store.restoreState({
      graphs: [
        {
          id: 'legacy-graph',
          name: '旧拓扑',
          nodes: [
            {
              id: 'legacy-sender',
              type: 'serial.sender',
              position: { x: 24, y: 32 },
              config: { mode: 'ascii', encoding: 'utf-8', payload: '' },
            },
            {
              id: 'legacy-receiver',
              type: 'serial.receiver',
              position: { x: 320, y: 32 },
              config: { viewMode: 'ascii' },
            },
          ],
          edges: [],
          selectedNodeId: 'legacy-sender',
          selectedEdgeId: null,
          nodeTabs: [
            { nodeId: 'legacy-sender', title: '发送器' },
            { nodeId: 'legacy-receiver', title: '接收器' },
          ],
          activeNodeTabId: 'legacy-sender',
        },
      ],
      activeGraphId: 'legacy-graph',
    })
    const wrapper = mount(SerialGraphPanel)

    const senderEnableLogging = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-enableLogging"]')
    expect(senderEnableLogging.exists()).toBe(true)
    expect(senderEnableLogging.element.checked).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-config-logLevel"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-config-logFormat"]').exists()).toBe(true)

    await senderEnableLogging.setValue(true)

    expect(store.graphById('legacy-graph')?.nodes.find(node => node.id === 'legacy-sender')?.config.enableLogging).toBe(true)

    store.selectNodeInGraph('legacy-graph', 'legacy-receiver')
    await nextTick()

    const receiverEnableLogging = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-enableLogging"]')
    expect(receiverEnableLogging.exists()).toBe(true)
    expect(receiverEnableLogging.element.checked).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-config-logLevel"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-graph-config-logFormat"]').exists()).toBe(true)

    wrapper.unmount()
  })

  it('renders filter controls with stable test ids and writes matcher options', async () => {
    const store = useSerialGraphStore()
    const filter = store.addNode('serial.filter')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-graph-provider-serial.filter"]').text()).toContain('过滤')

    const mode = wrapper.find<HTMLSelectElement>('[data-testid="serial-graph-config-mode"]')
    const expression = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-expression"]')
    const caseSensitive = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-caseSensitive"]')
    const wholeWord = wrapper.find<HTMLInputElement>('[data-testid="serial-graph-config-wholeWord"]')
    expect(mode.element.tagName).toBe('SELECT')
    expect(mode.findAll('option').map(option => option.attributes('value'))).toEqual(['plain', 'regex', 'expression'])
    expect(expression.exists()).toBe(true)
    expect(caseSensitive.element.type).toBe('checkbox')
    expect(wholeWord.element.type).toBe('checkbox')

    await mode.setValue('regex')
    await expression.setValue('TEMP\\d+')
    await caseSensitive.setValue(true)
    await wholeWord.setValue(true)

    expect(store.nodes.find(node => node.id === filter.id)?.config).toEqual(expect.objectContaining({
      mode: 'regex',
      expression: 'TEMP\\d+',
      caseSensitive: true,
      wholeWord: true,
    }))

    wrapper.unmount()
  })

  it('opens the operation log tab and filters rendered entries by level and keyword options', async () => {
    const store = useSerialGraphStore()
    store.appendOperationLogForGraph('graph-1', {
      timestamp: '2026-06-23T02:01:02.003Z',
      level: 'info',
      source: 'serial.sender',
      action: 'send-command',
      category: 'serial.graph',
      message: 'Sent TEMP OK payload',
      details: 'manual send',
      nodeId: 'node-1',
      direction: 'tx',
      payloadText: 'TEMP OK',
      payloadHex: '54 45 4d 50 20 4f 4b',
      byteLength: 7,
    })
    store.appendOperationLogForGraph('graph-1', {
      timestamp: '2026-06-23T02:01:03.004Z',
      level: 'error',
      source: 'serial.receiver',
      action: 'template',
      category: 'serial.graph',
      message: 'Log template failed',
      details: 'fallback used',
      nodeId: 'node-2',
      direction: 'rx',
      byteLength: 7,
    })
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').exists()).toBe(false)

    await wrapper.find('[data-testid="serial-graph-open-operation-log"]').trigger('click')

    expect(wrapper.find('[data-testid="serial-graph-operation-log-tab"]').text()).toContain('操作日志')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-operation-log-empty"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(2)
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('2026-06-23')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('serial.sender/send-command')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('TEMP OK')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('node-1')

    await wrapper.find('[data-testid="serial-operation-log-level"]').setValue('error')
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('Log template failed')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).not.toContain('Sent TEMP OK payload')

    await wrapper.find('[data-testid="serial-operation-log-level"]').setValue('all')
    await wrapper.find('[data-testid="serial-operation-log-mode"]').setValue('plain')
    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('temp ok')
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('Sent TEMP OK payload')

    await wrapper.find('[data-testid="serial-operation-log-case-sensitive"]').setValue(true)
    expect(wrapper.find('[data-testid="serial-operation-log-empty"]').text()).toContain('没有匹配的操作日志')

    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('TEMP')
    await wrapper.find('[data-testid="serial-operation-log-whole-word"]').setValue(true)
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(1)

    await wrapper.find('[data-testid="serial-operation-log-mode"]').setValue('regex')
    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('template\\s+failed')
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('Log template failed')

    await wrapper.find('[data-testid="serial-operation-log-mode"]').setValue('expression')
    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('level == "info" and action == "send-command"')
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('Sent TEMP OK payload')

    await wrapper.find('[data-testid="serial-operation-log-mode"]').setValue('regex')
    await wrapper.find('[data-testid="serial-operation-log-expression"]').setValue('[broken')
    expect(wrapper.findAll('[data-testid="serial-operation-log-entry"]')).toHaveLength(0)
    expect(wrapper.find('[data-testid="serial-operation-log-error"]').text()).toContain('invalid regex')

    wrapper.unmount()
  })

  it('uses actual receiver viewMode options and writes the selected mode to node config', async () => {
    const store = useSerialGraphStore()
    const receiver = store.addNode('serial.receiver')
    const wrapper = mount(SerialGraphPanel)

    const viewModeSelect = wrapper.find('[data-testid="serial-graph-config-viewMode"]')
    expect(viewModeSelect.exists()).toBe(true)
    expect(viewModeSelect.findAll('option').map(option => ({
      value: option.attributes('value'),
      text: option.text(),
    }))).toEqual([
      { value: 'ascii', text: 'ASCII' },
      { value: 'hexClassic', text: 'HEX 经典' },
      { value: 'hexTable', text: 'HEX 表格' },
    ])

    await viewModeSelect.setValue('hexClassic')

    expect(store.nodes.find(node => node.id === receiver.id)?.config.viewMode).toBe('hexClassic')

    await viewModeSelect.setValue('hexTable')

    expect(store.nodes.find(node => node.id === receiver.id)?.config.viewMode).toBe('hexTable')
  })

  it('renders receiver buffer according to the selected viewMode', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa(String.fromCharCode(0x41, 0x00, 0xff)),
      Total: 3,
      EOF: true,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await flushPromises()

    const buffer = () => wrapper.find('[data-testid="serial-graph-content-node-buffer"]').element.textContent ?? ''
    expect(buffer()).toContain('A')
    expect(buffer()).not.toContain('41 00 ff')

    await wrapper.find('[data-testid="serial-graph-config-viewMode"]').setValue('hexClassic')
    await nextTick()

    expect(buffer()).toContain('00000000')
    expect(buffer()).toContain('41 00 ff')
    expect(buffer()).toContain('A..')

    await wrapper.find('[data-testid="serial-graph-config-viewMode"]').setValue('hexTable')
    await nextTick()

    expect(buffer()).toContain('Offset')
    expect(buffer()).toContain('HEX')
    expect(buffer()).toContain('ASCII')
    expect(buffer()).toContain('41 00 ff')
    expect(buffer()).toContain('A..')

    wrapper.unmount()
  })

  it('renders modbus node data as grouped protocol frames instead of offset buffers', async () => {
    const store = useSerialGraphStore()
    const master = store.addNode('serial.modbus.master')
    store.selectNode(master.id)
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [
        {
          Seq: 1,
          Direction: '发送',
          Length: 8,
          DisplayText: 'Unit 3 | FC 03 Read Holding Registers | Address 0 Quantity 2 | CRC c5 e9 | Raw 03 03 00 00 00 02 c5 e9',
          DisplayHex: '03 03 00 00 00 02 c5 e9',
        },
        {
          Seq: 2,
          Direction: '接收',
          Length: 9,
          DisplayText: 'Unit 3 | FC 03 Read Holding Registers | Byte Count 4 Values 0, 0 | CRC f0 33 | Raw 03 03 04 00 00 00 00 f0 33',
          DisplayHex: '03 03 04 00 00 00 00 f0 33',
        },
      ],
      Total: 2,
      NextOffset: 2,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').exists()).toBe(false)
    const tableText = wrapper.find('[data-testid="serial-graph-content-node-frames"]').text()
    expect(tableText).toContain('Unit 3')
    expect(tableText).toContain('Read Holding Registers')
    expect(tableText).toContain('Address 0 Quantity 2')
    expect(tableText).toContain('Byte Count 4 Values 0, 0')
    expect(tableText).toContain('原始数据')
    expect(tableText).toContain('03 03 00 00 00 02 c5 e9')
    expect(tableText).not.toContain('Raw 03 03 00 00 00 02 c5 e9')
    expect(tableText).not.toContain('Offset')

    const rows = wrapper.findAll('[data-testid="serial-graph-content-node-frame-row"]')
    expect(rows).toHaveLength(2)
    const firstCells = rows[0].findAll('td').map(cell => cell.text())
    expect(firstCells).toEqual([
      '1',
      '包 1',
      '发送',
      '8',
      'Unit 3 | FC 03 Read Holding Registers | Address 0 Quantity 2 | CRC c5 e9',
      '03 03 00 00 00 02 c5 e9',
    ])
    expect(rows[0].classes()).toContain('serial-graph__frame-row--tx')
    expect(rows[1].classes()).toContain('serial-graph__frame-row--rx')
    const packetClass = rows[0].classes().find(item => item.startsWith('serial-graph__frame-row--packet-'))
    expect(packetClass).toBeTruthy()
    expect(rows[1].classes()).toContain(packetClass as string)

    wrapper.unmount()
  })

  it('marks invalid modbus frames while still showing raw data', async () => {
    const store = useSerialGraphStore()
    const master = store.addNode('serial.modbus.master')
    store.selectNode(master.id)
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [
        {
          Seq: 1,
          Direction: '接收',
          Length: 3,
          DisplayText: 'Modbus RTU | Invalid frame: rtu frame too short | Raw 01 03 00',
          DisplayHex: '01 03 00',
          Error: 'rtu frame too short',
        } as any,
      ],
      Total: 1,
      NextOffset: 1,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()

    const row = wrapper.find('[data-testid="serial-graph-content-node-frame-row"]')
    const cells = row.findAll('td').map(cell => cell.text())
    expect(row.classes()).toContain('serial-graph__frame-row--error')
    expect(cells[4]).toContain('错误')
    expect(cells[4]).toContain('Modbus RTU | Invalid frame: rtu frame too short')
    expect(cells[4]).not.toContain('Raw 01 03 00')
    expect(cells[5]).toBe('01 03 00')

    wrapper.unmount()
  })

  it('shows mode options that match the selected node type', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const master = store.addNode('serial.modbus.master')
    const slave = store.addNode('serial.modbus.slave')
    const wrapper = mount(SerialGraphPanel)

    store.selectNode(sender.id)
    await nextTick()

    const senderMode = wrapper.find('[data-testid="serial-graph-config-mode"]')
    expect(senderMode.findAll('option').map(option => option.attributes('value'))).toEqual(['ascii', 'hex'])

    store.selectNode(master.id)
    await nextTick()

    const modbusMode = wrapper.find('[data-testid="serial-graph-config-mode"]')
    expect(modbusMode.findAll('option').map(option => option.attributes('value'))).toEqual(['rtu', 'ascii'])

    store.selectNode(slave.id)
    await nextTick()

    const slaveMode = wrapper.find('[data-testid="serial-graph-config-mode"]')
    expect(slaveMode.findAll('option').map(option => option.attributes('value'))).toEqual(['rtu', 'ascii'])

    wrapper.unmount()
  })

  it('shows only node-appropriate operation panes and counters for every graph node type', async () => {
    const cases = [
      { type: 'serial.physical', send: false, buffer: true, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.virtual', send: false, buffer: true, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.bridge', send: false, buffer: false, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.monitor', send: false, buffer: false, frames: true, counters: ['RX'] },
      { type: 'serial.tap', send: false, buffer: false, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.tee', send: false, buffer: false, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.sender', send: true, buffer: false, frames: false, counters: ['TX'] },
      { type: 'serial.receiver', send: false, buffer: true, frames: false, counters: ['RX'] },
      { type: 'serial.script.transform', send: false, buffer: false, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.script.generator', send: false, buffer: false, frames: false, counters: ['TX'] },
      { type: 'serial.script.analyzer', send: false, buffer: false, frames: true, counters: ['RX'] },
      { type: 'serial.modbus.master', send: true, buffer: false, frames: true, counters: ['RX', 'TX'] },
      { type: 'serial.modbus.slave', send: false, buffer: false, frames: true, counters: ['RX', 'TX'] },
      { type: 'serial.fecbus.master', send: true, buffer: true, frames: false, counters: ['RX', 'TX'] },
      { type: 'serial.fecbus.slave', send: false, buffer: true, frames: false, counters: ['RX', 'TX'] },
    ]
    const store = useSerialGraphStore()
    const nodes = cases.map(item => store.addNode(item.type))
    const statuses = nodes.map((node, index) => ({
      ID: node.id,
      Type: node.type,
      Status: 'running',
      RxBytes: 100 + index,
      TxBytes: 200 + index,
      FrameCount: node.type === 'serial.monitor' || node.type === 'serial.script.analyzer' ? 3 : 0,
      ResourceID: '',
      Error: '',
    }))
    const info = {
      ID: 'graph-1',
      Status: 'running',
      Error: '',
      Nodes: statuses,
    }
    bindings.StartSerialGraph.mockResolvedValue(info)
    bindings.GetSerialGraphStatus.mockResolvedValue(info)
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()

    for (const [index, item] of cases.entries()) {
      const node = nodes[index]
      const status = statuses[index]
      store.selectNode(node.id)
      await nextTick()

      const content = wrapper.find('[data-testid="serial-graph-node-content"]')
      const statusGridText = content.find('.serial-graph__status-grid--content').text()
      const cardStatusText = wrapper.find(`[data-testid="serial-graph-node-${node.id}"] .serial-graph__node-status`).text()

      expect(wrapper.find('[data-testid="serial-graph-content-send"]').exists(), `${item.type} send pane`).toBe(item.send)
      expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').exists(), `${item.type} receive buffer`).toBe(item.buffer)
      expect(wrapper.find('[data-testid="serial-graph-config-autoScroll"]').exists(), `${item.type} auto scroll`).toBe(item.buffer)
      expect(wrapper.find('[data-testid="serial-graph-config-showTimestamp"]').exists(), `${item.type} timestamp`).toBe(item.buffer)
      expect(wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').exists(), `${item.type} frame pane`).toBe(item.frames)

      for (const label of ['RX', 'TX']) {
        const expected = item.counters.includes(label)
        expect(statusGridText.includes(label), `${item.type} content ${label}`).toBe(expected)
        expect(cardStatusText.includes(label), `${item.type} card ${label}`).toBe(expected)
      }
      if (item.counters.includes('RX')) {
        expect(statusGridText).toContain(String(status.RxBytes))
        expect(cardStatusText).toContain(`RX ${status.RxBytes}`)
      }
      if (item.counters.includes('TX')) {
        expect(statusGridText).toContain(String(status.TxBytes))
        expect(cardStatusText).toContain(`TX ${status.TxBytes}`)
      }
    }

    wrapper.unmount()
  })

  it('auto-scrolls receiver buffers only when autoScroll is enabled', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()

    const buffer = wrapper.find<HTMLElement>('[data-testid="serial-graph-content-node-buffer"]').element
    Object.defineProperty(buffer, 'scrollHeight', { value: 480, configurable: true })
    Object.defineProperty(buffer, 'scrollTop', { value: 0, writable: true, configurable: true })

    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(buffer.scrollTop).toBe(480)

    await wrapper.find('[data-testid="serial-graph-config-autoScroll"]').setValue(false)
    buffer.scrollTop = 0
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa('next'),
      Total: 4,
      EOF: true,
    })

    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(buffer.scrollTop).toBe(0)

    wrapper.unmount()
  })

  it('renders receiver buffer timestamps only when showTimestamp is enabled', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)
    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa('hi!'),
      Total: 3,
      EOF: false,
      Chunks: [
        { Offset: 0, Timestamp: '2026-06-23T02:01:02.003', Data: btoa('hi') },
        { Offset: 2, Timestamp: '2026-06-23T02:01:03.004', Data: btoa('!') },
      ],
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()

    const bufferText = () => wrapper.find('[data-testid="serial-graph-content-node-buffer"]').text()
    expect(bufferText()).toContain('hi!')
    expect(bufferText()).not.toContain('2026-06-23 02:01:02.003')

    await wrapper.find('[data-testid="serial-graph-config-showTimestamp"]').setValue(true)
    await nextTick()

    expect(bufferText()).toContain('[2026-06-23 02:01:02.003] hi')
    expect(bufferText()).toContain('[2026-06-23 02:01:03.004] !')

    await wrapper.find('[data-testid="serial-graph-config-viewMode"]').setValue('hexClassic')
    await nextTick()

    expect(bufferText()).toContain('[2026-06-23 02:01:02.003]')
    expect(bufferText()).toContain('00000000')
    expect(bufferText()).toContain('68 69')

    bindings.QuerySerialGraphNodeBuffer.mockResolvedValue({
      Offset: 0,
      Data: btoa('xy'),
      Total: 2,
      EOF: true,
      Chunks: [
        { Offset: 0, Timestamp: '', Data: btoa('x') },
        { Offset: 1, Timestamp: 'not-a-date', Data: btoa('y') },
      ],
    })
    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await flushPromises()

    expect(bufferText()).toContain('[未知时间]')
    expect(bufferText()).toContain('[not-a-date]')

    wrapper.unmount()
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

  it('ignores secondary canvas pans and tiny pans do not suppress selection clearing', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
    const canvas = wrapper.find<HTMLElement>('[data-testid="serial-graph-canvas"]')
    Object.defineProperty(canvas.element, 'scrollLeft', { value: 40, writable: true, configurable: true })
    Object.defineProperty(canvas.element, 'scrollTop', { value: 50, writable: true, configurable: true })

    const rightDown = pointerTestEvent('pointerdown', { button: 2, pointerId: 9, clientX: 100, clientY: 100 })
    const preventDefault = vi.spyOn(rightDown, 'preventDefault')
    canvas.element.dispatchEvent(rightDown)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(canvas.element.scrollLeft).toBe(40)
    expect(canvas.element.scrollTop).toBe(50)

    canvas.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 10, clientX: 100, clientY: 100 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 10, clientX: 101, clientY: 101 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 11 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 10 }))
    await nextTick()

    expect(store.selectedNodeId).toBe(sender.id)

    await canvas.trigger('click')

    expect(store.selectedNodeId).toBeNull()
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
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('TX')
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).not.toContain('RX')
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').exists()).toBe(false)
    expect(wrapper.find(`[data-testid="serial-graph-node-${sender.id}"]`).text()).toContain('TX 5')
    expect(wrapper.find(`[data-testid="serial-graph-node-${sender.id}"]`).text()).not.toContain('RX')

    store.selectNode(receiver.id)
    await nextTick()
    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('RX')
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).not.toContain('TX')
    expect(wrapper.find('[data-testid="serial-graph-content-send"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').text()).toContain('hello')
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').classes()).toContain('serial-graph__buffer--content')
    expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).toContain('RX 5')
    expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).not.toContain('TX')
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

  it('renders monitor node frames inside a fixed content container', async () => {
    const store = useSerialGraphStore()
    const monitor = store.addNode('serial.monitor')
    store.selectNode(monitor.id)
    const wrapper = mount(SerialGraphPanel)

    expect(store.nodes.find(node => node.id === monitor.id)?.config).toEqual({ displayMode: 'hex' })

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()

    const frameContainer = wrapper.find('[data-testid="serial-graph-content-node-frame-container"]')
    expect(frameContainer.exists()).toBe(true)
    expect(frameContainer.classes()).toContain('serial-graph__frames-container')
    expect(frameContainer.find('[data-testid="serial-graph-content-node-frames"]').exists()).toBe(true)
    expect(frameContainer.find('[data-testid="serial-graph-content-node-frames"]').text()).toContain('68 65 6c 6c 6f')
    expect(wrapper.find('.serial-graph__node-content > .serial-graph__frames').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-config-mode"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-config-displayMode"]').exists()).toBe(true)

    wrapper.unmount()
  })

  it('renders monitor frames using the selected displayMode', async () => {
    const store = useSerialGraphStore()
    const monitor = store.addNode('serial.monitor')
    store.selectNode(monitor.id)
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [{
        Seq: 1,
        Direction: '接收',
        Length: 1,
        DisplayText: 'A',
        DisplayHex: '41',
        DisplayDec: '65',
        DisplayOct: '101',
        DisplayBin: '01000001',
      }],
      Total: 1,
      NextOffset: 1,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()

    const displayMode = wrapper.find('[data-testid="serial-graph-config-displayMode"]')
    expect(displayMode.findAll('option').map(option => ({
      value: option.attributes('value'),
      text: option.text(),
    }))).toEqual([
      { value: 'text', text: 'text' },
      { value: 'hex', text: 'hex' },
      { value: 'dec', text: 'dec' },
      { value: 'oct', text: 'oct' },
      { value: 'bin', text: 'bin' },
    ])

    const tableText = () => wrapper.find('[data-testid="serial-graph-content-node-frames"]').text()
    expect(tableText()).toContain('41')
    expect(tableText()).not.toContain('65')

    await displayMode.setValue('text')
    await nextTick()

    expect(tableText()).toContain('A')
    expect(tableText()).not.toContain('41')

    await displayMode.setValue('dec')
    await nextTick()

    expect(tableText()).toContain('65')
    expect(tableText()).not.toContain('41')

    await displayMode.setValue('bin')
    await nextTick()

    expect(tableText()).toContain('01000001')
    expect(tableText()).not.toContain('65')

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

  it('keeps node card counters polling while selected node buffer refresh is pending', async () => {
    vi.useFakeTimers()
    let resolveBuffer: (value: { Offset: number; Data: string; Total: number; EOF: boolean }) => void = () => {}
    try {
      const store = useSerialGraphStore()
      const sender = store.addNode('serial.sender')
      const receiver = store.addNode('serial.receiver')
      store.connect(sender.id, 'out', receiver.id, 'in')
      store.selectNode(receiver.id)
      bindings.QuerySerialGraphNodeBuffer.mockImplementation(() => new Promise(resolve => {
        resolveBuffer = resolve
      }))
      const wrapper = mount(SerialGraphPanel)

      await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
      await flushPromises()
      expect(bindings.QuerySerialGraphNodeBuffer).toHaveBeenCalledWith(expect.objectContaining({
        GraphID: 'graph-1',
        NodeID: receiver.id,
      }))

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

      await vi.advanceTimersByTimeAsync(250)
      await flushPromises()

      expect(bindings.GetSerialGraphStatus).toHaveBeenCalledWith('graph-1')
      expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).toContain('RX 25')

      resolveBuffer({ Offset: 0, Data: btoa('late'), Total: 4, EOF: true })
      await flushPromises()

      expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).toContain('RX 25')
      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('resumes node card counter polling when mounted for an already running graph', async () => {
    vi.useFakeTimers()
    try {
      const store = useSerialGraphStore()
      const sender = store.addNode('serial.sender')
      const receiver = store.addNode('serial.receiver')
      store.connect(sender.id, 'out', receiver.id, 'in')
      await store.startRuntime()
      bindings.GetSerialGraphStatus.mockClear()
      bindings.GetSerialGraphStatus.mockResolvedValue({
        ID: 'graph-1',
        Status: 'running',
        Error: '',
        Nodes: [
          { ID: sender.id, Type: 'serial.sender', Status: 'running', RxBytes: 0, TxBytes: 35, FrameCount: 0, ResourceID: '', Error: '' },
          { ID: receiver.id, Type: 'serial.receiver', Status: 'running', RxBytes: 35, TxBytes: 0, FrameCount: 0, ResourceID: '', Error: '' },
        ],
      })

      const wrapper = mount(SerialGraphPanel)
      await flushPromises()
      await vi.advanceTimersByTimeAsync(250)
      await flushPromises()

      expect(bindings.GetSerialGraphStatus).toHaveBeenCalledWith('graph-1')
      expect(wrapper.find(`[data-testid="serial-graph-node-${receiver.id}"]`).text()).toContain('RX 35')

      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('renders script nodes with Monaco-backed script editors and frozen config fields', async () => {
    const store = useSerialGraphStore()
    const transform = store.addNode('serial.script.transform')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-graph-provider-serial.script.transform"]').text()).toContain('脚本转换')
    expect(wrapper.find('[data-testid="serial-graph-provider-serial.script.generator"]').text()).toContain('脚本生成')
    expect(wrapper.find('[data-testid="serial-graph-provider-serial.script.analyzer"]').text()).toContain('脚本分析')
    expect(wrapper.find(`[data-testid="serial-graph-input-${transform.id}-in"]`).exists()).toBe(true)
    expect(wrapper.find(`[data-testid="serial-graph-output-${transform.id}-out"]`).exists()).toBe(true)
    expect(wrapper.find('[data-testid="serial-script-editor"]').exists()).toBe(true)

    const configKeys = wrapper.findAll('.serial-graph__field > span').map(item => item.text())
    expect(configKeys).toEqual(expect.arrayContaining([
      'timeoutMs',
      'maxOutputBytes',
      'maxStateBytes',
      'onError',
      'encoding',
      'autoRun',
      'intervalMs',
      'displayMode',
    ]))
    expect(wrapper.find('[data-testid="serial-graph-config-script"]').exists()).toBe(false)

    wrapper.findComponent({ name: 'ScriptEditor' }).vm.$emit('script-change', 'output.bytes(input.bytes())')
    await nextTick()

    expect(store.nodes.find(node => node.id === transform.id)?.config.script).toBe('output.bytes(input.bytes())')
  })

  it('shows script restart guidance when editing while the graph runtime is running', async () => {
    const store = useSerialGraphStore()
    store.addNode('serial.script.generator')
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    wrapper.findComponent({ name: 'ScriptEditor' }).vm.$emit('script-change', 'output.bytes([1])')
    await nextTick()

    expect(wrapper.find('[data-testid="serial-script-restart-notice"]').text()).toContain('停止重启生效')

    wrapper.unmount()
  })

  it('uses script node type to omit generator input and analyzer output handles', () => {
    const store = useSerialGraphStore()
    const generator = store.addNode('serial.script.generator')
    const analyzer = store.addNode('serial.script.analyzer')
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find(`[data-testid="serial-graph-input-${generator.id}-in"]`).exists()).toBe(false)
    expect(wrapper.find(`[data-testid="serial-graph-output-${generator.id}-out"]`).exists()).toBe(true)
    expect(wrapper.find(`[data-testid="serial-graph-input-${analyzer.id}-in"]`).exists()).toBe(true)
    expect(wrapper.find(`[data-testid="serial-graph-output-${analyzer.id}-out"]`).exists()).toBe(false)
  })

  it('reactivates clears and closes the operation log tab without losing node tabs', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    store.appendOperationLogForGraph('graph-1', {
      level: 'info',
      source: 'serial.sender',
      action: 'send',
      message: 'manual send',
      nodeId: sender.id,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-open-operation-log"]').trigger('click')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').text()).toContain('manual send')

    await wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"]`).trigger('click')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('payload')

    await wrapper.find('[data-testid="serial-graph-operation-log-tab"]').trigger('click')
    expect(wrapper.find('[data-testid="serial-operation-log-panel"]').exists()).toBe(true)

    await wrapper.find('[data-testid="serial-operation-log-clear"]').trigger('click')
    expect(store.operationLogs).toEqual([])
    expect(wrapper.find('[data-testid="serial-operation-log-empty"]').text()).toContain('操作日志')

    await wrapper.find('[data-testid="serial-graph-operation-log-tab"] .serial-graph__node-tab-close').trigger('click')
    expect(wrapper.find('[data-testid="serial-graph-operation-log-tab"]').exists()).toBe(false)
    expect(wrapper.find(`[data-testid="serial-graph-node-tab-${sender.id}"]`).exists()).toBe(true)
  })

  it('stops runtime and clears selected receiver buffer controls from the panel', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    const receiver = store.addNode('serial.receiver')
    store.connect(sender.id, 'out', receiver.id, 'in')
    store.selectNode(receiver.id)
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-buffer"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="serial-graph-content-node-buffer"]').text()).toContain('hello')

    await wrapper.find('[data-testid="serial-graph-content-clear-buffer"]').trigger('click')
    await wrapper.find('[data-testid="serial-graph-content-reset-counters"]').trigger('click')
    await wrapper.find('[data-testid="serial-graph-stop"]').trigger('click')
    await flushPromises()

    expect(bindings.ClearSerialGraphNodeBuffer).toHaveBeenCalledWith('graph-1', receiver.id)
    expect(bindings.ResetSerialGraphNodeCounters).toHaveBeenCalledWith('graph-1', receiver.id)
    expect(bindings.StopSerialGraph).toHaveBeenCalledWith('graph-1')
    expect(wrapper.find('[data-testid="serial-graph-runtime-status"]').text()).toBe('stopped')
  })

  it('drags node cards, clears canvas selection, and deletes nodes from card controls', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender', { x: 24, y: 32 })
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
    const canvas = wrapper.find<HTMLElement>('[data-testid="serial-graph-canvas"]')
    Object.defineProperty(canvas.element, 'scrollLeft', { value: 10, writable: true, configurable: true })
    Object.defineProperty(canvas.element, 'scrollTop', { value: 5, writable: true, configurable: true })
    vi.spyOn(canvas.element, 'getBoundingClientRect').mockReturnValue({
      left: 4,
      top: 6,
      right: 804,
      bottom: 606,
      width: 800,
      height: 600,
      x: 4,
      y: 6,
      toJSON: () => ({}),
    } as DOMRect)
    const node = wrapper.find<HTMLElement>(`[data-testid="serial-graph-node-${sender.id}"]`)

    node.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 1, clientX: 40, clientY: 50 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 1, clientX: 90, clientY: 120 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 1 }))
    await nextTick()

    expect(store.nodes.find(item => item.id === sender.id)?.position).toEqual({ x: 74, y: 102 })
    expect(store.selectedNodeId).toBe(sender.id)

    await canvas.trigger('click')
    expect(store.selectedNodeId).toBeNull()

    await node.find('header button').trigger('click')
    expect(store.nodes).toEqual([])
    wrapper.unmount()
  })

  it('pans the canvas without clearing selection on the suppressed click', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender', { x: 24, y: 32 })
    const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
    const canvas = wrapper.find<HTMLElement>('[data-testid="serial-graph-canvas"]')
    Object.defineProperty(canvas.element, 'scrollLeft', { value: 40, writable: true, configurable: true })
    Object.defineProperty(canvas.element, 'scrollTop', { value: 50, writable: true, configurable: true })

    canvas.element.dispatchEvent(pointerTestEvent('pointerdown', { pointerId: 7, clientX: 100, clientY: 100 }))
    window.dispatchEvent(pointerTestEvent('pointermove', { pointerId: 7, clientX: 130, clientY: 140 }))
    window.dispatchEvent(pointerTestEvent('pointerup', { pointerId: 7 }))
    await nextTick()

    expect(canvas.element.scrollLeft).toBe(10)
    expect(canvas.element.scrollTop).toBe(10)

    await canvas.trigger('click')
    expect(store.selectedNodeId).toBe(sender.id)
    wrapper.unmount()
  })

  it('renders unknown node fallbacks validation errors and sparse operation log fields', async () => {
    const store = useSerialGraphStore()
    store.restoreState({
      graphs: [{
        id: 'graph-1',
        name: '坏拓扑',
        nodes: [{ id: 'mystery', type: 'serial.unknown', position: { x: 12, y: 18 }, config: {} }],
        edges: [{ id: 'edge-bad', source: 'mystery', sourceHandle: 'out', target: 'missing', targetHandle: 'in' }],
        selectedNodeId: 'mystery',
        selectedEdgeId: null,
        nodeTabs: [{ nodeId: 'mystery', title: 'Mystery' }],
        activeNodeTabId: 'mystery',
      }],
      activeGraphId: 'graph-1',
    } as any)
    store.appendOperationLogForGraph('graph-1', {
      timestamp: 'not-a-date',
      level: 'warn',
      message: 'sparse log',
    })
    const wrapper = mount(SerialGraphPanel)

    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('serial.unknown')
    expect(wrapper.find('[data-testid="serial-graph-node-content"]').text()).toContain('RX')
    expect(wrapper.find('.serial-graph__errors').text()).toContain('provider not found: serial.unknown')
    expect(wrapper.find('.serial-graph__errors').text()).toContain('edge-bad: target node not found: missing')

    await wrapper.find('[data-testid="serial-graph-open-operation-log"]').trigger('click')
    const rowText = wrapper.find('[data-testid="serial-operation-log-entry"]').text()
    expect(rowText).toContain('not-a-date')
    expect(rowText).toContain('unknown/event')
    expect(rowText).toContain('sparse log')
  })

  it('renders octal monitor frames and raw fallback text for modbus slave packets', async () => {
    const store = useSerialGraphStore()
    const monitor = store.addNode('serial.monitor')
    store.selectNode(monitor.id)
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [{
        Seq: 1,
        Direction: '接收',
        Length: 1,
        DisplayText: 'A',
        DisplayHex: '41',
        DisplayDec: '65',
        DisplayOct: '101',
        DisplayBin: '01000001',
      }],
      Total: 1,
      NextOffset: 1,
    })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()
    const displayMode = wrapper.find('[data-testid="serial-graph-config-displayMode"]')
    await displayMode.setValue('oct')
    await nextTick()

    expect(wrapper.find('[data-testid="serial-graph-content-node-frames"]').text()).toContain('101')

    const slave = store.addNode('serial.modbus.slave')
    store.selectNode(slave.id)
    bindings.QuerySerialGraphNodeFrames.mockResolvedValue({
      Frames: [
        { Seq: 1, Direction: '发送', Length: 3, DisplayText: 'Response | Raw aa bb cc', DisplayHex: '' },
        { Seq: 2, Direction: '接收', Length: 4, DisplayText: 'Request | Raw 01 03 00 00', DisplayHex: '' },
      ],
      Total: 2,
      NextOffset: 2,
    })
    await nextTick()
    await wrapper.find('[data-testid="serial-graph-content-refresh-frames"]').trigger('click')
    await flushPromises()

    const rows = wrapper.findAll('[data-testid="serial-graph-content-node-frame-row"]')
    expect(rows[0].findAll('td').at(-1)?.text()).toBe('aa bb cc')
    expect(rows[1].findAll('td').at(-1)?.text()).toBe('01 03 00 00')
    expect(rows[0].classes().find(item => item.startsWith('serial-graph__frame-row--packet-')))
      .not.toBe(rows[1].classes().find(item => item.startsWith('serial-graph__frame-row--packet-')))
    wrapper.unmount()
  })

  it('clamps compact split layouts through ResizeObserver and disconnects on unmount', async () => {
    const resizeObservers: Array<{
      callback: ResizeObserverCallback
      observe: ReturnType<typeof vi.fn>
      disconnect: ReturnType<typeof vi.fn>
    }> = []
    class ResizeObserverMock {
      callback: ResizeObserverCallback
      observe = vi.fn()
      disconnect = vi.fn()

      constructor(callback: ResizeObserverCallback) {
        this.callback = callback
        resizeObservers.push(this)
      }
    }
    vi.stubGlobal('ResizeObserver', ResizeObserverMock)
    try {
      const wrapper = mount(SerialGraphPanel, { attachTo: document.body })
      await wrapper.find('[data-testid="serial-graph-provider-serial.sender"]').trigger('click')
      const workspace = wrapper.find<HTMLElement>('.serial-graph__workspace')
      Object.defineProperty(workspace.element, 'clientHeight', { value: 150, configurable: true })

      await nextTick()
      resizeObservers[0].callback([], resizeObservers[0] as unknown as ResizeObserver)
      await nextTick()

      expect(resizeObservers[0].observe).toHaveBeenCalledWith(workspace.element)
      expect(wrapper.find('[data-testid="serial-graph-node-workbench"]').attributes('style')).toContain('64px')

      wrapper.unmount()
      expect(resizeObservers[0].disconnect).toHaveBeenCalled()
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('sends manual node payload defaults when legacy node config values are absent', async () => {
    const store = useSerialGraphStore()
    const sender = store.addNode('serial.sender')
    store.updateNodeConfig(sender.id, { payload: undefined, mode: undefined, encoding: undefined })
    const wrapper = mount(SerialGraphPanel)

    await wrapper.find('[data-testid="serial-graph-start"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-testid="serial-graph-content-send"]').trigger('click')
    await flushPromises()

    expect(bindings.SendSerialGraphNode).toHaveBeenCalledWith(expect.objectContaining({
      GraphID: 'graph-1',
      NodeID: sender.id,
      Content: '',
      Mode: 'ascii',
      Encoding: 'utf-8',
    }))
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
