<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useSerialGraphStore } from '../stores/graphStore'
import {
  providerByType,
  serialGraphProviders,
  type SerialGraphNode,
  type SerialGraphPortSpec,
} from '../graph/serialGraph'

const props = defineProps<{
  graphId?: string
}>()

const nodeWidth = 180
const nodeHeight = 104
const store = useSerialGraphStore()
const canvasRef = ref<HTMLElement | null>(null)
const pendingOutput = ref<{ nodeId: string; handleId: string } | null>(null)
const dragging = ref<{
  nodeId: string
  offsetX: number
  offsetY: number
} | null>(null)
const panning = ref<{
  pointerId: number
  startX: number
  startY: number
  startScrollLeft: number
  startScrollTop: number
  moved: boolean
} | null>(null)
const suppressCanvasClick = ref(false)
const runtimePollTimer = ref<number | null>(null)
const runtimePollInFlight = ref(false)
const localGraphId = ref<string | null>(null)

const providerGroups = computed(() => {
  const groups = new Map<string, typeof serialGraphProviders>()
  for (const provider of serialGraphProviders) {
    const items = groups.get(provider.category) ?? []
    items.push(provider)
    groups.set(provider.category, items)
  }
  return Array.from(groups.entries()).map(([category, providers]) => ({ category, providers }))
})
const panelGraphId = computed(() => (
  localGraphId.value ?? props.graphId ?? store.activeGraphId ?? store.graphList[0]?.id ?? null
))
const panelGraph = computed(() => {
  const id = panelGraphId.value
  return id ? store.graphById(id) : store.activeGraph
})
const panelRuntime = computed(() => store.runtimeStateForGraph(panelGraph.value?.id ?? panelGraphId.value))
const panelValidation = computed(() => store.validationForGraph(panelGraph.value?.id ?? panelGraphId.value))
const panelNodes = computed(() => panelGraph.value?.nodes ?? [])
const panelEdges = computed(() => panelGraph.value?.edges ?? [])
const panelNodeTabs = computed(() => panelGraph.value?.nodeTabs ?? [])
const panelActiveNodeTabId = computed(() => panelGraph.value?.activeNodeTabId ?? null)
const selectedNode = computed(() => (
  panelNodes.value.find(node => node.id === panelGraph.value?.selectedNodeId) ?? null
))
const selectedEdge = computed(() => (
  panelEdges.value.find(edge => edge.id === panelGraph.value?.selectedEdgeId) ?? null
))
const selectedProvider = computed(() => (
  selectedNode.value ? providerByType(selectedNode.value.type) : null
))
const selectedConfigEntries = computed(() => Object.entries(selectedNode.value?.config ?? {}))
const selectedStatus = computed(() => (
  selectedNode.value ? panelRuntime.value.nodeStatuses.get(selectedNode.value.id) ?? null : null
))
const selectedBufferText = computed(() => (
  selectedNode.value ? panelRuntime.value.nodeBufferText.get(selectedNode.value.id) ?? '' : ''
))
const selectedFrames = computed(() => (
  selectedNode.value ? panelRuntime.value.nodeFrames.get(selectedNode.value.id) ?? [] : []
))
const selectedPayload = computed(() => String(selectedNode.value?.config.payload ?? ''))
const selectedMode = computed(() => String(selectedNode.value?.config.mode ?? 'ascii'))
const runtimeRunning = computed(() => panelRuntime.value.runtimeStatus === 'running')
const activeGraphName = computed(() => panelGraph.value?.name ?? '')

function addNode(type: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.addNodeToGraph(graphId, type)
}

function createGraph() {
  const graph = store.createGraph(undefined, !props.graphId)
  localGraphId.value = props.graphId ? graph.id : null
}

function duplicateGraph() {
  const graph = store.duplicateGraph(panelGraph.value?.id ?? store.activeGraphId, !props.graphId)
  if (graph) {
    localGraphId.value = props.graphId ? graph.id : null
  }
}

async function removeGraph() {
  const id = panelGraph.value?.id
  if (!id) return
  await store.removeGraph(id)
  if (!store.graphById(id)) {
    localGraphId.value = store.activeGraphId ?? store.graphList[0]?.id ?? null
  }
}

function switchGraph(id: string) {
  if (props.graphId) {
    localGraphId.value = id
    return
  }
  localGraphId.value = null
  store.setActiveGraph(id)
}

function renameGraph(value: string) {
  const id = panelGraph.value?.id
  if (!id) return
  store.renameGraph(id, value)
}

function nodeTitle(node: SerialGraphNode): string {
  return providerByType(node.type)?.title ?? node.type
}

function inputsFor(node: SerialGraphNode): SerialGraphPortSpec[] {
  return providerByType(node.type)?.inputs ?? []
}

function outputsFor(node: SerialGraphNode): SerialGraphPortSpec[] {
  return providerByType(node.type)?.outputs ?? []
}

function selectOutput(nodeId: string, handleId: string) {
  pendingOutput.value = { nodeId, handleId }
}

function connectInput(nodeId: string, handleId: string) {
  if (!pendingOutput.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  const edge = store.connectInGraph(graphId, pendingOutput.value.nodeId, pendingOutput.value.handleId, nodeId, handleId)
  if (edge) {
    pendingOutput.value = null
  }
}

function updateConfig(key: string, value: string | boolean) {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  const current = selectedNode.value.config[key]
  store.updateNodeConfigInGraph(graphId, selectedNode.value.id, { [key]: typedConfigValue(current, value) })
}

async function startRuntime() {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.startRuntimeForGraph(graphId)
  startRuntimePolling()
}

async function stopRuntime() {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  stopRuntimePolling()
  await store.stopRuntimeForGraph(graphId)
}

async function sendSelectedNode() {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.sendNodeForGraph(
    graphId,
    selectedNode.value.id,
    String(selectedNode.value.config.payload ?? ''),
    String(selectedNode.value.config.mode ?? 'ascii'),
    String(selectedNode.value.config.encoding ?? 'utf-8')
  )
}

async function refreshSelectedBuffer() {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.queryNodeBufferForGraph(graphId, selectedNode.value.id)
}

async function refreshSelectedFrames() {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.queryNodeFramesForGraph(graphId, selectedNode.value.id)
}

async function clearSelectedBuffer() {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.clearNodeBufferForGraph(graphId, selectedNode.value.id)
}

async function resetSelectedCounters() {
  if (!selectedNode.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  await store.resetNodeCountersForGraph(graphId, selectedNode.value.id)
}

function removeSelectedEdge() {
  if (!selectedEdge.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.removeEdgeFromGraph(graphId, selectedEdge.value.id)
}

function activateNodeTab(nodeId: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.setActiveNodeTabInGraph(graphId, nodeId)
}

function closeNodeTab(nodeId: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.closeNodeTabInGraph(graphId, nodeId)
  const nextGraph = store.graphById(graphId)
  const activeNodeTabId = nextGraph?.activeNodeTabId
  if (activeNodeTabId) {
    store.selectNodeInGraph(graphId, activeNodeTabId)
    return
  }
  if (nextGraph?.selectedNodeId === nodeId) {
    store.selectNodeInGraph(graphId, null)
  }
}

function nodeRuntimeSummary(node: SerialGraphNode): string {
  const status = panelRuntime.value.nodeStatuses.get(node.id)
  if (!status) return node.status ?? 'idle'
  return `${status.Status} RX ${status.RxBytes} TX ${status.TxBytes}`
}

function supportsSend(node: SerialGraphNode): boolean {
  return [
    'serial.sender',
    'serial.modbus.master',
    'serial.modbus.slave',
    'serial.fecbus.master',
    'serial.fecbus.slave',
  ].includes(node.type)
}

function supportsBuffer(node: SerialGraphNode): boolean {
  return [
    'serial.receiver',
    'serial.physical',
    'serial.virtual',
    'serial.modbus.master',
    'serial.modbus.slave',
    'serial.fecbus.master',
    'serial.fecbus.slave',
  ].includes(node.type)
}

function supportsFrames(node: SerialGraphNode): boolean {
  return node.type === 'serial.monitor'
}

function panelGraphDocumentId(): string | null {
  return panelGraph.value?.id ?? panelGraphId.value
}

function startRuntimePolling() {
  stopRuntimePolling()
  void pollSelectedRuntimeDetails()
  runtimePollTimer.value = window.setInterval(() => {
    void pollSelectedRuntimeDetails()
  }, 250)
}

function stopRuntimePolling() {
  if (runtimePollTimer.value === null) return
  window.clearInterval(runtimePollTimer.value)
  runtimePollTimer.value = null
}

async function pollSelectedRuntimeDetails() {
  if (!runtimeRunning.value || runtimePollInFlight.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return

  runtimePollInFlight.value = true
  try {
    await store.refreshRuntimeForGraph(graphId)
    const node = selectedNode.value
    if (!runtimeRunning.value || !node) return
    if (supportsBuffer(node)) {
      await store.queryNodeBufferForGraph(graphId, node.id)
    }
    if (supportsFrames(node)) {
      await store.queryNodeFramesForGraph(graphId, node.id)
    }
  } catch {
    // Runtime polling is best-effort; explicit actions still surface errors.
  } finally {
    runtimePollInFlight.value = false
  }
}

function typedConfigValue(current: unknown, value: string | boolean): unknown {
  if (typeof current === 'number') {
    return Number(value)
  }
  if (typeof current === 'boolean') {
    return typeof value === 'boolean' ? value : value === 'true'
  }
  return value
}

function startNodeDrag(event: PointerEvent, node: SerialGraphNode) {
  if ((event.target as HTMLElement).closest('button, input, textarea, select')) return
  event.preventDefault()
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.selectNodeInGraph(graphId, node.id)
  const point = canvasContentPoint(event)
  dragging.value = {
    nodeId: node.id,
    offsetX: point.x - node.position.x,
    offsetY: point.y - node.position.y,
  }
  window.addEventListener('pointermove', handlePointerMove)
  window.addEventListener('pointerup', stopNodeDrag)
  window.addEventListener('pointercancel', stopNodeDrag)
}

function handlePointerMove(event: PointerEvent) {
  if (!dragging.value) return
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  const point = canvasContentPoint(event)
  const nextX = point.x - dragging.value.offsetX
  const nextY = point.y - dragging.value.offsetY
  store.moveNodeInGraph(graphId, dragging.value.nodeId, {
    x: Math.max(8, Math.round(nextX)),
    y: Math.max(8, Math.round(nextY)),
  })
}

function stopNodeDrag() {
  dragging.value = null
  window.removeEventListener('pointermove', handlePointerMove)
  window.removeEventListener('pointerup', stopNodeDrag)
  window.removeEventListener('pointercancel', stopNodeDrag)
}

function startCanvasPan(event: PointerEvent) {
  if (event.button !== 0 || !canvasRef.value) return
  if ((event.target as HTMLElement).closest('.serial-graph__node')) return

  event.preventDefault()
  panning.value = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    startScrollLeft: canvasRef.value.scrollLeft,
    startScrollTop: canvasRef.value.scrollTop,
    moved: false,
  }
  window.addEventListener('pointermove', handleCanvasPanMove)
  window.addEventListener('pointerup', stopCanvasPan)
  window.addEventListener('pointercancel', stopCanvasPan)
}

function handleCanvasPanMove(event: PointerEvent) {
  if (!panning.value || event.pointerId !== panning.value.pointerId || !canvasRef.value) return

  const deltaX = event.clientX - panning.value.startX
  const deltaY = event.clientY - panning.value.startY
  if (Math.abs(deltaX) > 2 || Math.abs(deltaY) > 2) {
    panning.value.moved = true
  }
  canvasRef.value.scrollLeft = panning.value.startScrollLeft - deltaX
  canvasRef.value.scrollTop = panning.value.startScrollTop - deltaY
}

function stopCanvasPan(event?: PointerEvent) {
  if (event && panning.value && event.pointerId !== panning.value.pointerId) return
  suppressCanvasClick.value = Boolean(panning.value?.moved)
  panning.value = null
  window.removeEventListener('pointermove', handleCanvasPanMove)
  window.removeEventListener('pointerup', stopCanvasPan)
  window.removeEventListener('pointercancel', stopCanvasPan)
}

function handleCanvasClick() {
  if (suppressCanvasClick.value) {
    suppressCanvasClick.value = false
    return
  }
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.selectNodeInGraph(graphId, null)
  store.selectEdgeInGraph(graphId, null)
}

function selectNode(nodeId: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.selectNodeInGraph(graphId, nodeId)
}

function selectEdge(edgeId: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.selectEdgeInGraph(graphId, edgeId)
}

function removeNode(nodeId: string) {
  const graphId = panelGraphDocumentId()
  if (!graphId) return
  store.removeNodeFromGraph(graphId, nodeId)
}

function canvasContentPoint(event: PointerEvent): { x: number; y: number } {
  const canvas = canvasRef.value
  const rect = canvas?.getBoundingClientRect()
  return {
    x: event.clientX - (rect?.left ?? 0) + (canvas?.scrollLeft ?? 0),
    y: event.clientY - (rect?.top ?? 0) + (canvas?.scrollTop ?? 0),
  }
}

function edgePath(edgeId: string): string {
  const edge = panelEdges.value.find(item => item.id === edgeId)
  if (!edge) return ''
  const source = panelNodes.value.find(node => node.id === edge.source)
  const target = panelNodes.value.find(node => node.id === edge.target)
  if (!source || !target) return ''

  const startX = source.position.x + nodeWidth
  const startY = source.position.y + nodeHeight / 2
  const endX = target.position.x
  const endY = target.position.y + nodeHeight / 2
  const controlOffset = Math.max(80, Math.abs(endX - startX) / 2)
  return `M ${startX} ${startY} C ${startX + controlOffset} ${startY}, ${endX - controlOffset} ${endY}, ${endX} ${endY}`
}

function edgeTitle(edge: { source: string; sourceHandle: string; target: string; targetHandle: string }): string {
  return `${edge.source}.${edge.sourceHandle} -> ${edge.target}.${edge.targetHandle}`
}

function isPendingOutput(nodeId: string, handleId: string): boolean {
  return pendingOutput.value?.nodeId === nodeId && pendingOutput.value.handleId === handleId
}

watch(() => panelGraph.value?.selectedNodeId, () => {
  if (runtimeRunning.value) void pollSelectedRuntimeDetails()
})

watch(
  () => props.graphId,
  () => {
    localGraphId.value = null
  }
)

onUnmounted(() => {
  stopRuntimePolling()
  stopNodeDrag()
  stopCanvasPan()
})
</script>

<template>
  <div
    class="serial-graph"
    data-testid="serial-graph-panel"
  >
    <aside
      class="serial-graph__palette"
      data-testid="serial-graph-node-palette"
    >
      <div class="serial-graph__panel-title">节点</div>
      <section
        v-for="group in providerGroups"
        :key="group.category"
        class="serial-graph__provider-group"
      >
        <h3>{{ group.category }}</h3>
        <button
          v-for="provider in group.providers"
          :key="provider.type"
          class="serial-graph__provider"
          :data-testid="`serial-graph-provider-${provider.type}`"
          type="button"
          @click="addNode(provider.type)"
        >
          <span>{{ provider.title }}</span>
          <small>{{ provider.description }}</small>
        </button>
      </section>
    </aside>

    <main class="serial-graph__workspace">
      <div class="serial-graph__toolbar">
        <select
          class="serial-graph__graph-select"
          :value="panelGraph?.id ?? ''"
          data-testid="serial-graph-switcher"
          @change="switchGraph(($event.target as HTMLSelectElement).value)"
        >
          <option
            v-for="graph in store.graphList"
            :key="graph.id"
            :value="graph.id"
          >
            {{ graph.name }}
          </option>
        </select>
        <input
          class="serial-graph__graph-name"
          :value="activeGraphName"
          data-testid="serial-graph-name"
          @change="renameGraph(($event.target as HTMLInputElement).value)"
        >
        <button
          type="button"
          data-testid="serial-graph-new"
          @click="createGraph"
        >
          新建
        </button>
        <button
          type="button"
          data-testid="serial-graph-duplicate"
          @click="duplicateGraph"
        >
          复制
        </button>
        <button
          type="button"
          data-testid="serial-graph-remove"
          :disabled="store.graphList.length <= 1"
          @click="removeGraph"
        >
          删除
        </button>
        <span>{{ panelNodes.length }} 节点</span>
        <span>{{ panelEdges.length }} 连线</span>
        <button
          type="button"
          data-testid="serial-graph-start"
          :disabled="runtimeRunning || panelValidation.errors.length > 0"
          @click="startRuntime"
        >
          启动
        </button>
        <button
          type="button"
          data-testid="serial-graph-stop"
          :disabled="!runtimeRunning"
          @click="stopRuntime"
        >
          停止
        </button>
        <span
          class="serial-graph__runtime"
          :class="`serial-graph__runtime--${panelRuntime.runtimeStatus}`"
          data-testid="serial-graph-runtime-status"
        >
          {{ panelRuntime.runtimeStatus }}
        </span>
        <span
          class="serial-graph__validation"
          :class="{ 'serial-graph__validation--error': panelValidation.errors.length > 0 }"
        >
          {{ panelValidation.errors.length > 0 ? `${panelValidation.errors.length} 个问题` : '拓扑有效' }}
        </span>
      </div>
      <div
        ref="canvasRef"
        class="serial-graph__canvas"
        :class="{ 'serial-graph__canvas--panning': panning }"
        data-testid="serial-graph-canvas"
        @pointerdown="startCanvasPan"
        @click="handleCanvasClick"
      >
        <svg
          class="serial-graph__edges"
        >
          <template
            v-for="edge in panelEdges"
            :key="edge.id"
          >
            <path
              class="serial-graph__edge-hit"
              :data-testid="`serial-graph-edge-${edge.id}`"
              :d="edgePath(edge.id)"
              @click.stop="selectEdge(edge.id)"
            />
            <path
              class="serial-graph__edge"
              :class="{ 'serial-graph__edge--selected': panelGraph?.selectedEdgeId === edge.id }"
              :d="edgePath(edge.id)"
            />
          </template>
        </svg>
        <article
          v-for="node in panelNodes"
          :key="node.id"
          class="serial-graph__node"
          :class="{ 'serial-graph__node--selected': panelGraph?.selectedNodeId === node.id }"
          :data-testid="`serial-graph-node-${node.id}`"
          :style="{ transform: `translate(${node.position.x}px, ${node.position.y}px)` }"
          @click.stop="selectNode(node.id)"
          @pointerdown="startNodeDrag($event, node)"
        >
          <header class="serial-graph__node-header">
            <strong>{{ nodeTitle(node) }}</strong>
            <button
              type="button"
              aria-label="删除节点"
              @click.stop="removeNode(node.id)"
            >
              ×
            </button>
          </header>
          <div class="serial-graph__node-status">
            {{ nodeRuntimeSummary(node) }}
          </div>
          <div class="serial-graph__ports">
            <div class="serial-graph__port-column">
              <button
                v-for="port in inputsFor(node)"
                :key="port.id"
                class="serial-graph__port serial-graph__port--input"
                :data-testid="`serial-graph-input-${node.id}-${port.id}`"
                type="button"
                @click.stop="connectInput(node.id, port.id)"
              >
                {{ port.label }}
              </button>
            </div>
            <div class="serial-graph__port-column serial-graph__port-column--right">
              <button
                v-for="port in outputsFor(node)"
                :key="port.id"
                class="serial-graph__port serial-graph__port--output"
                :class="{ 'serial-graph__port--pending': isPendingOutput(node.id, port.id) }"
                :data-testid="`serial-graph-output-${node.id}-${port.id}`"
                type="button"
                @click.stop="selectOutput(node.id, port.id)"
              >
                {{ port.label }}
              </button>
            </div>
          </div>
        </article>
      </div>
      <section
        v-if="panelNodeTabs.length > 0"
        class="serial-graph__node-workbench"
        data-testid="serial-graph-node-workbench"
      >
        <div class="serial-graph__node-tabs">
          <button
            v-for="tab in panelNodeTabs"
            :key="tab.nodeId"
            type="button"
            class="serial-graph__node-tab"
            :class="{ 'serial-graph__node-tab--active': panelActiveNodeTabId === tab.nodeId }"
            :data-testid="`serial-graph-node-tab-${tab.nodeId}`"
            @click="activateNodeTab(tab.nodeId)"
          >
            <span>{{ tab.title }}</span>
            <span
              class="serial-graph__node-tab-close"
              role="button"
              tabindex="0"
              @click.stop="closeNodeTab(tab.nodeId)"
              @keydown.enter.stop="closeNodeTab(tab.nodeId)"
            >
              ×
            </span>
          </button>
        </div>
        <div
          v-if="selectedNode"
          class="serial-graph__node-content"
          data-testid="serial-graph-node-content"
        >
          <div class="serial-graph__status-grid serial-graph__status-grid--content">
            <span>状态</span>
            <strong>{{ selectedStatus?.Status ?? selectedNode.status ?? 'idle' }}</strong>
            <span>RX</span>
            <strong>{{ selectedStatus?.RxBytes ?? 0 }}</strong>
            <span>TX</span>
            <strong>{{ selectedStatus?.TxBytes ?? 0 }}</strong>
          </div>
          <template v-if="supportsSend(selectedNode)">
            <div class="serial-graph__send-content">
              <label class="serial-graph__field">
                <span>mode</span>
                <select
                  :value="selectedMode"
                  data-testid="serial-graph-content-send-mode"
                  @change="updateConfig('mode', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="ascii">ascii</option>
                  <option value="hex">hex</option>
                </select>
              </label>
              <label class="serial-graph__field serial-graph__field--grow">
                <span>payload</span>
                <textarea
                  :value="selectedPayload"
                  data-testid="serial-graph-content-send-payload"
                  @input="updateConfig('payload', ($event.target as HTMLTextAreaElement).value)"
                />
              </label>
              <button
                type="button"
                data-testid="serial-graph-content-send"
                :disabled="!runtimeRunning"
                @click="sendSelectedNode"
              >
                发送
              </button>
            </div>
          </template>
          <template v-if="supportsBuffer(selectedNode)">
            <div class="serial-graph__button-row">
              <button
                type="button"
                data-testid="serial-graph-content-refresh-buffer"
                :disabled="!runtimeRunning"
                @click="refreshSelectedBuffer"
              >
                刷新
              </button>
              <button
                type="button"
                data-testid="serial-graph-content-clear-buffer"
                :disabled="!runtimeRunning"
                @click="clearSelectedBuffer"
              >
                清空
              </button>
              <button
                type="button"
                data-testid="serial-graph-content-reset-counters"
                :disabled="!runtimeRunning"
                @click="resetSelectedCounters"
              >
                复位计数
              </button>
            </div>
            <pre
              class="serial-graph__buffer serial-graph__buffer--content"
              data-testid="serial-graph-content-node-buffer"
            >{{ selectedBufferText }}</pre>
          </template>
          <template v-if="supportsFrames(selectedNode)">
            <button
              type="button"
              data-testid="serial-graph-content-refresh-frames"
              :disabled="!runtimeRunning"
              @click="refreshSelectedFrames"
            >
              刷新帧
            </button>
            <table
              class="serial-graph__frames"
              data-testid="serial-graph-content-node-frames"
            >
              <tbody>
                <tr
                  v-for="frame in selectedFrames"
                  :key="frame.Seq"
                >
                  <td>{{ frame.Seq }}</td>
                  <td>{{ frame.Direction }}</td>
                  <td>{{ frame.Length }}</td>
                  <td>{{ frame.DisplayHex || frame.DisplayText }}</td>
                </tr>
              </tbody>
            </table>
          </template>
        </div>
      </section>
    </main>

    <aside class="serial-graph__inspector">
      <div class="serial-graph__panel-title">配置</div>
      <template v-if="selectedNode && selectedProvider">
        <h3>{{ selectedProvider.title }}</h3>
        <p>{{ selectedProvider.description }}</p>
        <section class="serial-graph__runtime-panel">
          <div class="serial-graph__section-title">运行</div>
          <div class="serial-graph__status-grid">
            <span>状态</span>
            <strong data-testid="serial-graph-selected-status">{{ selectedStatus?.Status ?? selectedNode.status ?? 'idle' }}</strong>
            <span>RX</span>
            <strong>{{ selectedStatus?.RxBytes ?? 0 }}</strong>
            <span>TX</span>
            <strong>{{ selectedStatus?.TxBytes ?? 0 }}</strong>
          </div>
          <template v-if="supportsSend(selectedNode)">
            <label class="serial-graph__field">
              <span>mode</span>
              <select
                :value="selectedMode"
                data-testid="serial-graph-send-mode"
                @change="updateConfig('mode', ($event.target as HTMLSelectElement).value)"
              >
                <option value="ascii">ascii</option>
                <option value="hex">hex</option>
              </select>
            </label>
            <label class="serial-graph__field">
              <span>payload</span>
              <textarea
                :value="selectedPayload"
                data-testid="serial-graph-send-payload"
                @input="updateConfig('payload', ($event.target as HTMLTextAreaElement).value)"
              />
            </label>
            <button
              type="button"
              data-testid="serial-graph-send"
              :disabled="!runtimeRunning"
              @click="sendSelectedNode"
            >
              发送
            </button>
          </template>
          <template v-if="supportsBuffer(selectedNode)">
            <div class="serial-graph__button-row">
              <button
                type="button"
                data-testid="serial-graph-refresh-buffer"
                :disabled="!runtimeRunning"
                @click="refreshSelectedBuffer"
              >
                刷新
              </button>
              <button
                type="button"
                data-testid="serial-graph-clear-buffer"
                :disabled="!runtimeRunning"
                @click="clearSelectedBuffer"
              >
                清空
              </button>
              <button
                type="button"
                data-testid="serial-graph-reset-counters"
                :disabled="!runtimeRunning"
                @click="resetSelectedCounters"
              >
                复位计数
              </button>
            </div>
            <pre
              class="serial-graph__buffer"
              data-testid="serial-graph-node-buffer"
            >{{ selectedBufferText }}</pre>
          </template>
          <template v-if="supportsFrames(selectedNode)">
            <button
              type="button"
              data-testid="serial-graph-refresh-frames"
              :disabled="!runtimeRunning"
              @click="refreshSelectedFrames"
            >
              刷新帧
            </button>
            <table
              class="serial-graph__frames"
              data-testid="serial-graph-node-frames"
            >
              <tbody>
                <tr
                  v-for="frame in selectedFrames"
                  :key="frame.Seq"
                >
                  <td>{{ frame.Seq }}</td>
                  <td>{{ frame.Direction }}</td>
                  <td>{{ frame.Length }}</td>
                  <td>{{ frame.DisplayHex || frame.DisplayText }}</td>
                </tr>
              </tbody>
            </table>
          </template>
        </section>
        <div class="serial-graph__section-title">配置</div>
        <label
          v-for="[key, value] in selectedConfigEntries"
          :key="key"
          class="serial-graph__field"
        >
          <span>{{ key }}</span>
          <input
            v-if="typeof value === 'boolean'"
            type="checkbox"
            :checked="value"
            @change="updateConfig(key, ($event.target as HTMLInputElement).checked)"
          >
          <input
            v-else-if="typeof value === 'number'"
            type="number"
            :value="value"
            @input="updateConfig(key, ($event.target as HTMLInputElement).value)"
          >
          <select
            v-else-if="key === 'mode' || key === 'viewMode'"
            :value="String(value)"
            @change="updateConfig(key, ($event.target as HTMLSelectElement).value)"
          >
            <option value="ascii">ascii</option>
            <option value="hex">hex</option>
            <option value="rtu">rtu</option>
          </select>
          <input
            v-else
            :value="String(value)"
            @input="updateConfig(key, ($event.target as HTMLInputElement).value)"
          >
        </label>
      </template>
      <template v-else-if="selectedEdge">
        <h3>连接线</h3>
        <p data-testid="serial-graph-selected-edge">{{ edgeTitle(selectedEdge) }}</p>
        <button
          type="button"
          class="serial-graph__danger-button"
          data-testid="serial-graph-delete-edge"
          @click="removeSelectedEdge"
        >
          删除连接线
        </button>
      </template>
      <p
        v-else
        class="serial-graph__empty"
      >
        选择节点
      </p>
      <ul
        v-if="panelValidation.errors.length > 0"
        class="serial-graph__errors"
      >
        <li
          v-for="error in panelValidation.errors"
          :key="error"
        >
          {{ error }}
        </li>
      </ul>
    </aside>
  </div>
</template>

<style scoped>
.serial-graph {
  display: grid;
  grid-template-columns: 190px minmax(360px, 1fr) 220px;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
}
.serial-graph__palette,
.serial-graph__inspector {
  min-width: 0;
  overflow: auto;
  background: var(--app-surface, #252526);
}
.serial-graph__palette {
  border-right: 1px solid var(--app-border, #2d2d2d);
}
.serial-graph__inspector {
  border-left: 1px solid var(--app-border, #2d2d2d);
  padding: 10px;
}
.serial-graph__panel-title {
  height: 32px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
.serial-graph__provider-group {
  padding: 8px;
}
.serial-graph__provider-group h3,
.serial-graph__inspector h3 {
  margin: 4px 0 8px;
  color: var(--app-text, #cccccc);
  font-size: 13px;
  font-weight: 600;
}
.serial-graph__provider {
  display: block;
  width: 100%;
  margin-bottom: 6px;
  padding: 7px 8px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 6px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
  text-align: left;
  cursor: pointer;
}
.serial-graph__provider:hover {
  border-color: var(--app-accent, #007acc);
}
.serial-graph__provider span,
.serial-graph__provider small {
  display: block;
}
.serial-graph__provider small {
  margin-top: 3px;
  color: var(--app-text-muted, #858585);
  font-size: 11px;
  line-height: 1.3;
}
.serial-graph__workspace {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.serial-graph__toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 32px;
  padding: 0 10px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
.serial-graph__graph-select,
.serial-graph__graph-name {
  min-width: 0;
  height: 24px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
  font-size: 12px;
}
.serial-graph__graph-select {
  width: 110px;
}
.serial-graph__graph-name {
  width: 132px;
  padding: 0 6px;
}
.serial-graph__toolbar button,
.serial-graph__runtime-panel button {
  padding: 4px 8px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-surface, #252526);
  color: var(--app-text, #cccccc);
  font-size: 12px;
  cursor: pointer;
}
.serial-graph__toolbar button:disabled,
.serial-graph__runtime-panel button:disabled {
  opacity: 0.45;
  cursor: default;
}
.serial-graph__danger-button {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--app-danger, #f85149);
  border-radius: 4px;
  background: transparent;
  color: var(--app-danger, #f85149);
  cursor: pointer;
}
.serial-graph__runtime {
  color: var(--app-text-muted, #858585);
}
.serial-graph__runtime--running {
  color: var(--app-success, #3fb950);
}
.serial-graph__runtime--error {
  color: var(--app-danger, #f85149);
}
.serial-graph__validation {
  margin-left: auto;
  color: var(--app-success, #3fb950);
}
.serial-graph__validation--error {
  color: var(--app-danger, #f85149);
}
.serial-graph__canvas {
  position: relative;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  cursor: grab;
  background-image:
    linear-gradient(var(--app-border, #2d2d2d) 1px, transparent 1px),
    linear-gradient(90deg, var(--app-border, #2d2d2d) 1px, transparent 1px);
  background-size: 24px 24px;
}
.serial-graph__canvas--panning {
  cursor: grabbing;
}
.serial-graph__node-workbench {
  flex: 0 0 260px;
  display: flex;
  flex-direction: column;
  min-height: 180px;
  border-top: 1px solid var(--app-border, #2d2d2d);
  background: var(--app-bg, #1e1e1e);
}
.serial-graph__node-tabs {
  display: flex;
  flex: 0 0 auto;
  min-height: 31px;
  overflow-x: auto;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  background: var(--app-surface, #252526);
}
.serial-graph__node-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 180px;
  height: 31px;
  padding: 0 9px;
  border: 0;
  border-right: 1px solid var(--app-border, #2d2d2d);
  background: var(--app-hover-bg, #2d2d2d);
  color: var(--app-text, #cccccc);
  font: inherit;
  font-size: 12px;
  cursor: default;
}
.serial-graph__node-tab--active {
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #ffffff);
}
.serial-graph__node-tab span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.serial-graph__node-tab-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 15px;
  height: 15px;
  border-radius: 3px;
  color: var(--app-text-muted, #858585);
}
.serial-graph__node-tab-close:hover {
  background: var(--app-hover-bg, #3c3c3c);
  color: var(--app-text, #ffffff);
}
.serial-graph__node-content {
  flex: 1;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}
.serial-graph__send-content {
  display: grid;
  grid-template-columns: 120px minmax(220px, 1fr) auto;
  align-items: end;
  gap: 10px;
  margin-bottom: 10px;
}
.serial-graph__field--grow textarea {
  min-height: 112px;
}
.serial-graph__edges {
  position: absolute;
  inset: 0;
  width: 2400px;
  height: 1600px;
  pointer-events: none;
}
.serial-graph__edge-hit {
  fill: none;
  stroke: transparent;
  stroke-width: 14;
  cursor: pointer;
  pointer-events: stroke;
}
.serial-graph__edge {
  fill: none;
  stroke: var(--app-accent, #007acc);
  stroke-width: 2;
  pointer-events: none;
}
.serial-graph__edge--selected {
  stroke: var(--app-warning, #d7ba7d);
  stroke-width: 3;
}
.serial-graph__node {
  position: absolute;
  width: 180px;
  min-height: 104px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 6px;
  background: var(--app-surface, #252526);
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.18);
}
.serial-graph__node--selected {
  border-color: var(--app-accent, #007acc);
}
.serial-graph__node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 30px;
  padding: 0 8px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  cursor: grab;
}
.serial-graph__node-header strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.serial-graph__node-header button {
  border: 0;
  background: transparent;
  color: var(--app-text-muted, #858585);
  cursor: pointer;
}
.serial-graph__node-status {
  padding: 5px 8px 0;
  color: var(--app-text-muted, #858585);
  font-size: 11px;
  line-height: 1.2;
}
.serial-graph__ports {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 8px;
}
.serial-graph__port-column {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.serial-graph__port-column--right {
  align-items: flex-end;
}
.serial-graph__port {
  max-width: 76px;
  padding: 4px 6px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 999px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
  font-size: 11px;
  line-height: 1.2;
  cursor: pointer;
}
.serial-graph__port--pending,
.serial-graph__port:hover {
  border-color: var(--app-accent, #007acc);
  color: var(--app-text, #ffffff);
}
.serial-graph__inspector p {
  margin: 0 0 10px;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
  line-height: 1.4;
}
.serial-graph__field {
  display: block;
  margin-bottom: 8px;
}
.serial-graph__field span {
  display: block;
  margin-bottom: 4px;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
.serial-graph__field input {
  width: 100%;
  box-sizing: border-box;
  padding: 5px 6px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
}
.serial-graph__field select,
.serial-graph__field textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 5px 6px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
}
.serial-graph__field textarea {
  min-height: 96px;
  resize: vertical;
  font-family: var(--terminal-font-family, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 12px;
}
.serial-graph__runtime-panel {
  margin: 10px 0 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
}
.serial-graph__section-title {
  margin: 8px 0;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
  font-weight: 600;
}
.serial-graph__status-grid {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 5px 10px;
  margin-bottom: 10px;
  font-size: 12px;
}
.serial-graph__status-grid--content {
  grid-template-columns: repeat(3, auto auto);
  justify-content: start;
}
.serial-graph__status-grid span {
  color: var(--app-text-muted, #858585);
}
.serial-graph__button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.serial-graph__buffer {
  min-height: 120px;
  max-height: 220px;
  margin: 8px 0 0;
  padding: 8px;
  overflow: auto;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-bg, #1e1e1e);
  color: var(--app-text, #cccccc);
  font-family: var(--terminal-font-family, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 12px;
  white-space: pre-wrap;
}
.serial-graph__buffer--content {
  max-height: none;
}
.serial-graph__frames {
  width: 100%;
  margin-top: 8px;
  border-collapse: collapse;
  font-size: 12px;
}
.serial-graph__frames td {
  padding: 4px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  color: var(--app-text, #cccccc);
}
.serial-graph__empty {
  padding: 10px 0;
}
.serial-graph__errors {
  margin: 12px 0 0;
  padding-left: 18px;
  color: var(--app-danger, #f85149);
  font-size: 12px;
}
</style>
