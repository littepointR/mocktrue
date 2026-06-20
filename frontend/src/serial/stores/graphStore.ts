import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  ClearSerialGraphNodeBuffer,
  GetSerialGraphStatus,
  QuerySerialGraphNodeBuffer,
  QuerySerialGraphNodeFrames,
  ResetSerialGraphNodeCounters,
  SendSerialGraphNode,
  StartSerialGraph,
  StopSerialGraph,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'
import type { Snapshot } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/buffer/models.js'
import type { Frame } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/monitor/models.js'
import type { SerialGraphRuntimeInfo, SerialGraphNodeStatus } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/models.js'
import {
  canConnect,
  cloneSerialGraphState,
  createSerialGraphNode,
  defaultSerialGraphState,
  providerByType,
  validateGraph,
  type SerialGraphEdge,
  type SerialGraphNode,
  type SerialGraphPosition,
  type SerialGraphWorkspaceState,
} from '../graph/serialGraph'

export const useSerialGraphStore = defineStore('serialGraph', () => {
  const nodes = ref<SerialGraphNode[]>([])
  const edges = ref<SerialGraphEdge[]>([])
  const selectedNodeId = ref<string | null>(null)
  const selectedEdgeId = ref<string | null>(null)
  const nextNodeSeq = ref(1)
  const nextEdgeSeq = ref(1)
  const runtimeGraphId = ref<string | null>(null)
  const runtimeStatus = ref<'idle' | 'running' | 'stopped' | 'error'>('idle')
  const runtimeError = ref<string | null>(null)
  const nodeStatuses = ref<Map<string, SerialGraphNodeStatus>>(new Map())
  const nodeBuffers = ref<Map<string, Uint8Array>>(new Map())
  const nodeBufferText = ref<Map<string, string>>(new Map())
  const nodeFrames = ref<Map<string, Frame[]>>(new Map())

  const graphState = computed<SerialGraphWorkspaceState>(() => ({
    nodes: nodes.value,
    edges: edges.value,
    selectedNodeId: selectedNodeId.value,
    selectedEdgeId: selectedEdgeId.value,
  }))
  const validation = computed(() => validateGraph(graphState.value))
  const validationErrors = computed(() => validation.value.errors)

  function addNode(type: string, position: SerialGraphPosition = defaultNodePosition()): SerialGraphNode {
    if (!providerByType(type)) {
      throw new Error(`unknown graph provider: ${type}`)
    }
    const node = createSerialGraphNode(nextNodeId(), type, position)
    nodes.value = [...nodes.value, node]
    selectedNodeId.value = node.id
    selectedEdgeId.value = null
    return node
  }

  function removeNode(id: string) {
    nodes.value = nodes.value.filter(node => node.id !== id)
    edges.value = edges.value.filter(edge => edge.source !== id && edge.target !== id)
    if (selectedNodeId.value === id) selectedNodeId.value = null
    nodeStatuses.value.delete(id)
    nodeBuffers.value.delete(id)
    nodeBufferText.value.delete(id)
    nodeFrames.value.delete(id)
  }

  function moveNode(id: string, position: SerialGraphPosition) {
    nodes.value = nodes.value.map(node => (
      node.id === id ? { ...node, position: { ...position } } : node
    ))
  }

  function updateNodeConfig(id: string, patch: Record<string, unknown>) {
    nodes.value = nodes.value.map(node => (
      node.id === id
        ? { ...node, config: { ...node.config, ...patch } }
        : node
    ))
  }

  function connect(
    source: string,
    sourceHandle: string,
    target: string,
    targetHandle: string
  ): SerialGraphEdge | null {
    const draft = { source, sourceHandle, target, targetHandle }
    const result = canConnect(graphState.value, draft)
    if (!result.valid) return null

    const edge: SerialGraphEdge = {
      id: nextEdgeId(),
      ...draft,
    }
    edges.value = [...edges.value, edge]
    selectedEdgeId.value = edge.id
    selectedNodeId.value = null
    return edge
  }

  function removeEdge(id: string) {
    edges.value = edges.value.filter(edge => edge.id !== id)
    if (selectedEdgeId.value === id) selectedEdgeId.value = null
  }

  function selectNode(id: string | null) {
    selectedNodeId.value = id
    if (id) selectedEdgeId.value = null
  }

  function selectEdge(id: string | null) {
    selectedEdgeId.value = id
    if (id) selectedNodeId.value = null
  }

  function resetWorkspace() {
    const empty = defaultSerialGraphState()
    nodes.value = empty.nodes
    edges.value = empty.edges
    selectedNodeId.value = empty.selectedNodeId
    selectedEdgeId.value = empty.selectedEdgeId
    nextNodeSeq.value = 1
    nextEdgeSeq.value = 1
    clearRuntimeState()
  }

  function exportState(): SerialGraphWorkspaceState {
    return cloneSerialGraphState(graphState.value)
  }

  function restoreState(snapshot?: SerialGraphWorkspaceState | null) {
    const next = cloneSerialGraphState(snapshot)
    nodes.value = next.nodes
    edges.value = next.edges
    selectedNodeId.value = next.selectedNodeId
    selectedEdgeId.value = next.selectedEdgeId
    nextNodeSeq.value = nextSequence(nodes.value.map(node => node.id), 'node-')
    nextEdgeSeq.value = nextSequence(edges.value.map(edge => edge.id), 'edge-')
    clearRuntimeState()
  }

  async function startRuntime(id = 'serial.graph') {
    if (!validation.value.valid) {
      throw new Error(validationErrors.value.join('; '))
    }
    const info = await StartSerialGraph({
      ID: id,
      Nodes: nodes.value.map(node => ({
        ID: node.id,
        Type: node.type,
        Config: node.config,
        Position: {
          X: node.position.x,
          Y: node.position.y,
        },
      })),
      Edges: edges.value.map(edge => ({
        ID: edge.id,
        Source: edge.source,
        SourceHandle: edge.sourceHandle,
        Target: edge.target,
        TargetHandle: edge.targetHandle,
      })),
    })
    if (!info) throw new Error('serial graph did not return runtime info')
    applyRuntimeInfo(info)
    return info
  }

  async function stopRuntime() {
    if (!runtimeGraphId.value) return
    await StopSerialGraph(runtimeGraphId.value)
    runtimeStatus.value = 'stopped'
    for (const status of nodeStatuses.value.values()) {
      status.Status = 'stopped'
    }
  }

  async function refreshRuntime() {
    if (!runtimeGraphId.value) return null
    const info = await GetSerialGraphStatus(runtimeGraphId.value)
    if (info) applyRuntimeInfo(info)
    return info
  }

  async function sendNode(nodeId: string, content: string, mode = 'ascii', encoding = 'utf-8') {
    if (!runtimeGraphId.value) {
      throw new Error('serial graph is not running')
    }
    const written = await SendSerialGraphNode({
      GraphID: runtimeGraphId.value,
      NodeID: nodeId,
      Content: content,
      Mode: mode,
      Encoding: encoding,
    })
    await refreshRuntime()
    return written
  }

  async function queryNodeBuffer(nodeId: string, offset = 0, length = 4096): Promise<Snapshot> {
    if (!runtimeGraphId.value) {
      throw new Error('serial graph is not running')
    }
    const page = await QuerySerialGraphNodeBuffer({
      GraphID: runtimeGraphId.value,
      NodeID: nodeId,
      Offset: offset,
      Length: length,
    })
    if (!page) throw new Error('serial graph did not return buffer page')
    const bytes = snapshotBytes(page)
    nodeBuffers.value.set(nodeId, bytes)
    nodeBufferText.value.set(nodeId, new TextDecoder().decode(bytes))
    return page
  }

  async function queryNodeFrames(nodeId: string, offset = 0, limit = 100) {
    if (!runtimeGraphId.value) {
      throw new Error('serial graph is not running')
    }
    const page = await QuerySerialGraphNodeFrames({
      GraphID: runtimeGraphId.value,
      NodeID: nodeId,
      Offset: offset,
      Limit: limit,
      Direction: '',
      Search: '',
    })
    if (!page) throw new Error('serial graph did not return frame page')
    nodeFrames.value.set(nodeId, page.Frames ?? [])
    return page
  }

  async function clearNodeBuffer(nodeId: string) {
    if (!runtimeGraphId.value) return
    await ClearSerialGraphNodeBuffer(runtimeGraphId.value, nodeId)
    nodeBuffers.value.delete(nodeId)
    nodeBufferText.value.delete(nodeId)
    nodeFrames.value.delete(nodeId)
  }

  async function resetNodeCounters(nodeId: string) {
    if (!runtimeGraphId.value) return
    await ResetSerialGraphNodeCounters(runtimeGraphId.value, nodeId)
    const current = nodeStatuses.value.get(nodeId)
    if (current) {
      nodeStatuses.value.set(nodeId, { ...current, RxBytes: 0, TxBytes: 0, FrameCount: 0 })
    }
  }

  function applyRuntimeInfo(info: SerialGraphRuntimeInfo) {
    runtimeGraphId.value = info.ID
    runtimeStatus.value = normalizeRuntimeStatus(info.Status)
    runtimeError.value = info.Error || null
    const next = new Map<string, SerialGraphNodeStatus>()
    for (const status of info.Nodes ?? []) {
      next.set(status.ID, status)
    }
    nodeStatuses.value = next
    nodes.value = nodes.value.map(node => {
      const status = next.get(node.id)
      return status
        ? { ...node, status: normalizeNodeStatus(status.Status), error: status.Error || undefined }
        : node
    })
  }

  function clearRuntimeState() {
    runtimeGraphId.value = null
    runtimeStatus.value = 'idle'
    runtimeError.value = null
    nodeStatuses.value = new Map()
    nodeBuffers.value = new Map()
    nodeBufferText.value = new Map()
    nodeFrames.value = new Map()
    nodes.value = nodes.value.map(node => ({ ...node, status: 'idle', error: undefined }))
  }

  function nextNodeId(): string {
    const id = `node-${nextNodeSeq.value}`
    nextNodeSeq.value += 1
    return id
  }

  function nextEdgeId(): string {
    const id = `edge-${nextEdgeSeq.value}`
    nextEdgeSeq.value += 1
    return id
  }

  function defaultNodePosition(): SerialGraphPosition {
    const index = nodes.value.length
    return {
      x: 40 + (index % 3) * 220,
      y: 48 + Math.floor(index / 3) * 140,
    }
  }

  return {
    nodes,
    edges,
    selectedNodeId,
    selectedEdgeId,
    runtimeGraphId,
    runtimeStatus,
    runtimeError,
    nodeStatuses,
    nodeBuffers,
    nodeBufferText,
    nodeFrames,
    validation,
    validationErrors,
    addNode,
    removeNode,
    moveNode,
    updateNodeConfig,
    connect,
    removeEdge,
    selectNode,
    selectEdge,
    resetWorkspace,
    exportState,
    restoreState,
    startRuntime,
    stopRuntime,
    refreshRuntime,
    sendNode,
    queryNodeBuffer,
    queryNodeFrames,
    clearNodeBuffer,
    resetNodeCounters,
  }
})

function nextSequence(ids: string[], prefix: string): number {
  let max = 0
  for (const id of ids) {
    if (!id.startsWith(prefix)) continue
    const seq = Number(id.slice(prefix.length))
    if (Number.isFinite(seq)) {
      max = Math.max(max, seq)
    }
  }
  return max + 1
}

function normalizeRuntimeStatus(status: string): 'idle' | 'running' | 'stopped' | 'error' {
  if (status === 'running' || status === 'stopped' || status === 'error') return status
  return 'idle'
}

function normalizeNodeStatus(status: string): 'idle' | 'running' | 'error' {
  if (status === 'running' || status === 'error') return status
  return 'idle'
}

function snapshotBytes(page: Snapshot): Uint8Array {
  if (!page.Data) return new Uint8Array(0)
  try {
    return new Uint8Array(Array.from(atob(page.Data), char => char.charCodeAt(0)))
  } catch {
    return new TextEncoder().encode(page.Data)
  }
}

export type { SerialGraphWorkspaceState }
