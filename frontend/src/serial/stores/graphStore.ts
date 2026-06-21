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
  cloneSerialGraphDocument,
  cloneSerialGraphState,
  createSerialGraphNode,
  defaultSerialGraphDocument,
  defaultSerialGraphState,
  nodeTabTitle,
  providerByType,
  validateGraph,
  type SerialGraphDocument,
  type SerialGraphEdge,
  type SerialGraphNode,
  type SerialGraphPosition,
  type SerialGraphWorkspaceState,
} from '../graph/serialGraph'

type RuntimeStatus = 'idle' | 'running' | 'stopped' | 'error'

interface GraphRuntimeState {
  runtimeGraphId: string | null
  runtimeStatus: RuntimeStatus
  runtimeError: string | null
  nodeStatuses: Map<string, SerialGraphNodeStatus>
  nodeBuffers: Map<string, Uint8Array>
  nodeBufferText: Map<string, string>
  nodeFrames: Map<string, Frame[]>
}

const defaultNodeFramePageSize = 1000

export interface SerialGraphRuntimeSnapshot {
  nodeBuffers: Record<string, Uint8Array>
  nodeFrames: Record<string, Frame[]>
}

export const useSerialGraphStore = defineStore('serialGraph', () => {
  const initial = defaultSerialGraphState()
  const graphs = ref<SerialGraphDocument[]>(initial.graphs)
  const activeGraphId = ref<string | null>(initial.activeGraphId)
  const runtimeStates = ref<Record<string, GraphRuntimeState>>({})

  const activeGraph = computed(() => (
    graphs.value.find(graph => graph.id === activeGraphId.value) ?? graphs.value[0] ?? null
  ))
  const graphList = computed(() => graphs.value.map(graph => ({ id: graph.id, name: graph.name })))
  const nodes = computed(() => activeGraph.value?.nodes ?? [])
  const edges = computed(() => activeGraph.value?.edges ?? [])
  const selectedNodeId = computed(() => activeGraph.value?.selectedNodeId ?? null)
  const selectedEdgeId = computed(() => activeGraph.value?.selectedEdgeId ?? null)
  const nodeTabs = computed(() => activeGraph.value?.nodeTabs ?? [])
  const activeNodeTabId = computed(() => activeGraph.value?.activeNodeTabId ?? null)
  const runtimeGraphId = computed(() => activeRuntimeState().runtimeGraphId)
  const runtimeStatus = computed(() => activeRuntimeState().runtimeStatus)
  const runtimeError = computed(() => activeRuntimeState().runtimeError)
  const nodeStatuses = computed(() => activeRuntimeState().nodeStatuses)
  const nodeBuffers = computed(() => activeRuntimeState().nodeBuffers)
  const nodeBufferText = computed(() => activeRuntimeState().nodeBufferText)
  const nodeFrames = computed(() => activeRuntimeState().nodeFrames)
  const validation = computed(() => activeGraph.value ? validateGraph(activeGraph.value) : { valid: true, errors: [] })
  const validationErrors = computed(() => validation.value.errors)

  function graphById(id: string | null | undefined): SerialGraphDocument | null {
    if (!id) return null
    return graphs.value.find(graph => graph.id === id) ?? null
  }

  function runtimeStateForGraph(id: string | null | undefined): GraphRuntimeState {
    return runtimeFor(id ?? activeGraph.value?.id ?? activeGraphId.value ?? 'graph-1')
  }

  function validationForGraph(id: string | null | undefined) {
    const graph = graphById(id)
    return graph ? validateGraph(graph) : { valid: true, errors: [] }
  }

  function createGraph(name?: string, activate = true): SerialGraphDocument {
    const id = nextGraphId()
    const graph = defaultSerialGraphDocument(id, name ?? `拓扑图 ${graphNumber(id)}`)
    graphs.value = [...graphs.value, graph]
    if (activate) {
      activeGraphId.value = graph.id
    }
    return graph
  }

  function duplicateGraph(id = activeGraphId.value, activate = true): SerialGraphDocument | null {
    const source = graphs.value.find(graph => graph.id === id)
    if (!source) return null
    const nextId = nextGraphId()
    const graph = cloneSerialGraphDocument({
      ...source,
      id: nextId,
      name: `${source.name} 副本`,
      nodes: source.nodes.map(node => {
        const { status, error, ...rest } = node
        void status
        void error
        return rest
      }),
    })
    graphs.value = [...graphs.value, graph]
    if (activate) {
      activeGraphId.value = graph.id
    }
    return graph
  }

  function importGraphDocument(document: SerialGraphDocument, activate = true): SerialGraphDocument {
    const usedIds = new Set(graphs.value.map(graph => graph.id))
    const requested = cloneSerialGraphDocument(document)
    const id = usedIds.has(requested.id) ? nextGraphId() : requested.id
    const graph = cloneSerialGraphDocument({
      ...requested,
      id,
      name: requested.name || `拓扑图 ${graphNumber(id)}`,
      nodes: requested.nodes.map(node => {
        const { status, error, ...rest } = node
        void status
        void error
        return rest
      }),
    })
    graphs.value = [...graphs.value, graph]
    if (activate) {
      activeGraphId.value = graph.id
    }
    return graph
  }

  async function removeGraph(id: string) {
    const runtime = runtimeStates.value[id]
    if (runtime?.runtimeGraphId && runtime.runtimeStatus === 'running') {
      await StopSerialGraph(runtime.runtimeGraphId)
    }
    if (!graphs.value.some(graph => graph.id === id)) return
    removeGraphLocal(id)
  }

  function removeGraphLocal(id: string) {
    const remaining = graphs.value.filter(graph => graph.id !== id)
    delete runtimeStates.value[id]
    graphs.value = remaining
    if (activeGraphId.value === id || (activeGraphId.value && !remaining.some(graph => graph.id === activeGraphId.value))) {
      activeGraphId.value = remaining[0]?.id ?? null
    }
  }

  function renameGraph(id: string, name: string) {
    const trimmed = name.trim()
    if (!trimmed) return
    replaceGraph(id, graph => ({ ...graph, name: trimmed }))
  }

  function setActiveGraph(id: string | null) {
    if (!id) return
    if (graphs.value.some(graph => graph.id === id)) {
      activeGraphId.value = id
    }
  }

  function addNode(type: string, position: SerialGraphPosition = defaultNodePosition()): SerialGraphNode {
    const graph = ensureActiveGraph()
    return addNodeToGraph(graph.id, type, position)
  }

  function addNodeToGraph(
    graphId: string,
    type: string,
    position?: SerialGraphPosition
  ): SerialGraphNode {
    if (!providerByType(type)) {
      throw new Error(`unknown graph provider: ${type}`)
    }
    const graph = graphByIdOrThrow(graphId)
    const node = createSerialGraphNode(nextNodeId(graph), type, position ?? defaultNodePositionForGraph(graph))
    replaceGraph(graph.id, current => ({
      ...current,
      nodes: [...current.nodes, node],
      selectedNodeId: node.id,
      selectedEdgeId: null,
      nodeTabs: [...current.nodeTabs, { nodeId: node.id, title: nodeTabTitle(node) }],
      activeNodeTabId: node.id,
    }))
    return node
  }

  function removeNode(id: string) {
    const graph = ensureActiveGraph()
    removeNodeFromGraph(graph.id, id)
  }

  function removeNodeFromGraph(graphId: string, id: string) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => {
      const nodeTabsNext = current.nodeTabs.filter(tab => tab.nodeId !== id)
      return {
        ...current,
        nodes: current.nodes.filter(node => node.id !== id),
        edges: current.edges.filter(edge => edge.source !== id && edge.target !== id),
        selectedNodeId: current.selectedNodeId === id ? null : current.selectedNodeId,
        selectedEdgeId: null,
        nodeTabs: nodeTabsNext,
        activeNodeTabId: current.activeNodeTabId === id ? nodeTabsNext[0]?.nodeId ?? null : current.activeNodeTabId,
      }
    })
    const runtime = runtimeFor(graph.id)
    runtime.nodeStatuses.delete(id)
    runtime.nodeBuffers.delete(id)
    runtime.nodeBufferText.delete(id)
    runtime.nodeFrames.delete(id)
  }

  function moveNode(id: string, position: SerialGraphPosition) {
    const graph = ensureActiveGraph()
    moveNodeInGraph(graph.id, id, position)
  }

  function moveNodeInGraph(graphId: string, id: string, position: SerialGraphPosition) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => ({
      ...current,
      nodes: current.nodes.map(node => (
        node.id === id ? { ...node, position: { ...position } } : node
      )),
    }))
  }

  function updateNodeConfig(id: string, patch: Record<string, unknown>) {
    const graph = ensureActiveGraph()
    updateNodeConfigInGraph(graph.id, id, patch)
  }

  function updateNodeConfigInGraph(graphId: string, id: string, patch: Record<string, unknown>) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => ({
      ...current,
      nodes: current.nodes.map(node => (
        node.id === id
          ? { ...node, config: { ...node.config, ...patch } }
          : node
      )),
    }))
  }

  function connect(
    source: string,
    sourceHandle: string,
    target: string,
    targetHandle: string
  ): SerialGraphEdge | null {
    const graph = ensureActiveGraph()
    return connectInGraph(graph.id, source, sourceHandle, target, targetHandle)
  }

  function connectInGraph(
    graphId: string,
    source: string,
    sourceHandle: string,
    target: string,
    targetHandle: string
  ): SerialGraphEdge | null {
    const graph = graphByIdOrThrow(graphId)
    const draft = { source, sourceHandle, target, targetHandle }
    const result = canConnect(graph, draft)
    if (!result.valid) return null

    const edge: SerialGraphEdge = {
      id: nextEdgeId(graph),
      ...draft,
    }
    replaceGraph(graph.id, current => ({
      ...current,
      edges: [...current.edges, edge],
      selectedEdgeId: edge.id,
      selectedNodeId: null,
    }))
    return edge
  }

  function removeEdge(id: string) {
    const graph = ensureActiveGraph()
    removeEdgeFromGraph(graph.id, id)
  }

  function removeEdgeFromGraph(graphId: string, id: string) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => ({
      ...current,
      edges: current.edges.filter(edge => edge.id !== id),
      selectedEdgeId: current.selectedEdgeId === id ? null : current.selectedEdgeId,
    }))
  }

  function selectNode(id: string | null) {
    const graph = ensureActiveGraph()
    selectNodeInGraph(graph.id, id)
  }

  function selectNodeInGraph(graphId: string, id: string | null) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => {
      const node = id ? current.nodes.find(item => item.id === id) : null
      const hasTab = Boolean(id && current.nodeTabs.some(tab => tab.nodeId === id))
      return {
        ...current,
        selectedNodeId: id,
        selectedEdgeId: id ? null : current.selectedEdgeId,
        nodeTabs: node && !hasTab
          ? [...current.nodeTabs, { nodeId: node.id, title: nodeTabTitle(node) }]
          : current.nodeTabs,
        activeNodeTabId: node ? node.id : current.activeNodeTabId,
      }
    })
  }

  function selectEdge(id: string | null) {
    const graph = ensureActiveGraph()
    selectEdgeInGraph(graph.id, id)
  }

  function selectEdgeInGraph(graphId: string, id: string | null) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => ({
      ...current,
      selectedEdgeId: id,
      selectedNodeId: id ? null : current.selectedNodeId,
    }))
  }

  function setActiveNodeTab(nodeId: string) {
    selectNode(nodeId)
  }

  function setActiveNodeTabInGraph(graphId: string, nodeId: string) {
    selectNodeInGraph(graphId, nodeId)
  }

  function closeNodeTab(nodeId: string) {
    const graph = ensureActiveGraph()
    closeNodeTabInGraph(graph.id, nodeId)
  }

  function closeNodeTabInGraph(graphId: string, nodeId: string) {
    const graph = graphByIdOrThrow(graphId)
    replaceGraph(graph.id, current => {
      const tabs = current.nodeTabs.filter(tab => tab.nodeId !== nodeId)
      return {
        ...current,
        nodeTabs: tabs,
        activeNodeTabId: current.activeNodeTabId === nodeId ? tabs[0]?.nodeId ?? null : current.activeNodeTabId,
      }
    })
  }

  function resetWorkspace() {
    const empty = defaultSerialGraphState()
    graphs.value = empty.graphs
    activeGraphId.value = empty.activeGraphId
    runtimeStates.value = {}
  }

  function exportRuntimeSnapshot(graphId: string): SerialGraphRuntimeSnapshot {
    const runtime = runtimeFor(graphId)
    return {
      nodeBuffers: Object.fromEntries(Array.from(runtime.nodeBuffers.entries()).map(([nodeId, bytes]) => [
        nodeId,
        new Uint8Array(bytes),
      ])),
      nodeFrames: Object.fromEntries(Array.from(runtime.nodeFrames.entries()).map(([nodeId, frames]) => [
        nodeId,
        frames.map(frame => ({ ...frame })),
      ])),
    }
  }

  function restoreRuntimeSnapshot(graphId: string, snapshot?: Partial<SerialGraphRuntimeSnapshot> | null) {
    const runtime = runtimeFor(graphId)
    const nodeBuffers = new Map<string, Uint8Array>()
    const nodeBufferText = new Map<string, string>()
    for (const [nodeId, bytes] of Object.entries(snapshot?.nodeBuffers ?? {})) {
      const nextBytes = bytes instanceof Uint8Array ? new Uint8Array(bytes) : new Uint8Array(bytes as any)
      nodeBuffers.set(nodeId, nextBytes)
      nodeBufferText.set(nodeId, new TextDecoder().decode(nextBytes))
    }
    const nodeFrames = new Map<string, Frame[]>()
    for (const [nodeId, frames] of Object.entries(snapshot?.nodeFrames ?? {})) {
      nodeFrames.set(nodeId, (frames ?? []).map(frame => ({ ...frame })))
    }
    runtimeStates.value = {
      ...runtimeStates.value,
      [graphId]: {
        ...runtime,
        nodeBuffers,
        nodeBufferText,
        nodeFrames,
      },
    }
  }

  function exportState(): SerialGraphWorkspaceState {
    return cloneSerialGraphState({
      graphs: graphs.value,
      activeGraphId: activeGraphId.value,
    })
  }

  function restoreState(snapshot?: SerialGraphWorkspaceState | SerialGraphDocument | null) {
    const next = cloneSerialGraphState(snapshot)
    graphs.value = next.graphs
    activeGraphId.value = next.activeGraphId
    runtimeStates.value = {}
  }

  async function startRuntime(runtimeId = activeGraph.value?.id ?? 'graph-1') {
    const graph = ensureActiveGraph()
    return startRuntimeForGraph(graph.id, runtimeId)
  }

  async function startRuntimeForGraph(graphId: string, runtimeId = graphId) {
    const graph = graphByIdOrThrow(graphId)
    const graphValidation = validateGraph(graph)
    if (!graphValidation.valid) {
      throw new Error(graphValidation.errors.join('; '))
    }
    const info = await StartSerialGraph({
      ID: runtimeId,
      Nodes: graph.nodes.map(node => ({
        ID: node.id,
        Type: node.type,
        Config: node.config,
        Position: {
          X: node.position.x,
          Y: node.position.y,
        },
      })),
      Edges: graph.edges.map(edge => ({
        ID: edge.id,
        Source: edge.source,
        SourceHandle: edge.sourceHandle,
        Target: edge.target,
        TargetHandle: edge.targetHandle,
      })),
    })
    if (!info) throw new Error('serial graph did not return runtime info')
    applyRuntimeInfo(graph.id, info)
    return info
  }

  async function stopRuntime() {
    const graph = ensureActiveGraph()
    await stopRuntimeForGraph(graph.id)
  }

  async function stopRuntimeForGraph(graphId: string) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) return
    await StopSerialGraph(runtime.runtimeGraphId)
    runtimeStates.value = {
      ...runtimeStates.value,
      [graph.id]: {
        ...runtime,
        runtimeStatus: 'stopped',
        nodeStatuses: stoppedStatuses(runtime.nodeStatuses),
      },
    }
  }

  async function refreshRuntime() {
    const graph = ensureActiveGraph()
    return refreshRuntimeForGraph(graph.id)
  }

  async function refreshRuntimeForGraph(graphId: string) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) return null
    const info = await GetSerialGraphStatus(runtime.runtimeGraphId)
    if (info) applyRuntimeInfo(graph.id, info)
    return info
  }

  async function sendNode(nodeId: string, content: string, mode = 'ascii', encoding = 'utf-8') {
    const graph = ensureActiveGraph()
    return sendNodeForGraph(graph.id, nodeId, content, mode, encoding)
  }

  async function sendNodeForGraph(
    graphId: string,
    nodeId: string,
    content: string,
    mode = 'ascii',
    encoding = 'utf-8'
  ) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) {
      throw new Error('serial graph is not running')
    }
    const written = await SendSerialGraphNode({
      GraphID: runtime.runtimeGraphId,
      NodeID: nodeId,
      Content: content,
      Mode: mode,
      Encoding: encoding,
    })
    await refreshRuntimeForGraph(graph.id)
    return written
  }

  async function queryNodeBuffer(nodeId: string, offset = 0, length = 4096): Promise<Snapshot> {
    const graph = ensureActiveGraph()
    return queryNodeBufferForGraph(graph.id, nodeId, offset, length)
  }

  async function queryNodeBufferForGraph(
    graphId: string,
    nodeId: string,
    offset = 0,
    length = 4096
  ): Promise<Snapshot> {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) {
      throw new Error('serial graph is not running')
    }
    const page = await QuerySerialGraphNodeBuffer({
      GraphID: runtime.runtimeGraphId,
      NodeID: nodeId,
      Offset: offset,
      Length: length,
    })
    if (!page) throw new Error('serial graph did not return buffer page')
    const bytes = snapshotBytes(page)
    const latestRuntime = runtimeFor(graph.id)
    runtimeStates.value = {
      ...runtimeStates.value,
      [graph.id]: {
        ...latestRuntime,
        nodeBuffers: new Map(latestRuntime.nodeBuffers).set(nodeId, bytes),
        nodeBufferText: new Map(latestRuntime.nodeBufferText).set(nodeId, new TextDecoder().decode(bytes)),
      },
    }
    return page
  }

  async function queryNodeFrames(nodeId: string, offset = -defaultNodeFramePageSize, limit = defaultNodeFramePageSize) {
    const graph = ensureActiveGraph()
    return queryNodeFramesForGraph(graph.id, nodeId, offset, limit)
  }

  async function queryNodeFramesForGraph(graphId: string, nodeId: string, offset = -defaultNodeFramePageSize, limit = defaultNodeFramePageSize) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) {
      throw new Error('serial graph is not running')
    }
    const page = await QuerySerialGraphNodeFrames({
      GraphID: runtime.runtimeGraphId,
      NodeID: nodeId,
      Offset: offset,
      Limit: limit,
      Direction: '',
      Search: '',
    })
    if (!page) throw new Error('serial graph did not return frame page')
    const latestRuntime = runtimeFor(graph.id)
    runtimeStates.value = {
      ...runtimeStates.value,
      [graph.id]: {
        ...latestRuntime,
        nodeFrames: new Map(latestRuntime.nodeFrames).set(nodeId, page.Frames ?? []),
      },
    }
    return page
  }

  async function clearNodeBuffer(nodeId: string) {
    const graph = ensureActiveGraph()
    await clearNodeBufferForGraph(graph.id, nodeId)
  }

  async function clearNodeBufferForGraph(graphId: string, nodeId: string) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) return
    await ClearSerialGraphNodeBuffer(runtime.runtimeGraphId, nodeId)
    const latestRuntime = runtimeFor(graph.id)
    const nodeBuffers = new Map(latestRuntime.nodeBuffers)
    const nodeBufferText = new Map(latestRuntime.nodeBufferText)
    const nodeFrames = new Map(latestRuntime.nodeFrames)
    nodeBuffers.delete(nodeId)
    nodeBufferText.delete(nodeId)
    nodeFrames.delete(nodeId)
    runtimeStates.value = {
      ...runtimeStates.value,
      [graph.id]: { ...latestRuntime, nodeBuffers, nodeBufferText, nodeFrames },
    }
  }

  async function resetNodeCounters(nodeId: string) {
    const graph = ensureActiveGraph()
    await resetNodeCountersForGraph(graph.id, nodeId)
  }

  async function resetNodeCountersForGraph(graphId: string, nodeId: string) {
    const graph = graphByIdOrThrow(graphId)
    const runtime = runtimeFor(graph.id)
    if (!runtime.runtimeGraphId) return
    await ResetSerialGraphNodeCounters(runtime.runtimeGraphId, nodeId)
    const latestRuntime = runtimeFor(graph.id)
    const nodeStatuses = new Map(latestRuntime.nodeStatuses)
    const current = nodeStatuses.get(nodeId)
    if (current) {
      nodeStatuses.set(nodeId, { ...current, RxBytes: 0, TxBytes: 0, FrameCount: 0 })
      runtimeStates.value = {
        ...runtimeStates.value,
        [graph.id]: { ...latestRuntime, nodeStatuses },
      }
    }
  }

  function applyRuntimeInfo(graphId: string, info: SerialGraphRuntimeInfo) {
    const runtime = runtimeFor(graphId)
    const nodeStatuses = new Map<string, SerialGraphNodeStatus>()
    for (const status of info.Nodes ?? []) {
      nodeStatuses.set(status.ID, status)
    }
    runtimeStates.value = {
      ...runtimeStates.value,
      [graphId]: {
        ...runtime,
        runtimeGraphId: info.ID,
        runtimeStatus: normalizeRuntimeStatus(info.Status),
        runtimeError: info.Error || null,
        nodeStatuses,
      },
    }
    replaceGraph(graphId, graph => ({
      ...graph,
      nodes: graph.nodes.map(node => {
        const status = nodeStatuses.get(node.id)
        return status
          ? { ...node, status: normalizeNodeStatus(status.Status), error: status.Error || undefined }
          : node
      }),
    }))
  }

  function ensureActiveGraph(): SerialGraphDocument {
    const current = activeGraph.value
    if (current) return current
    return createGraph()
  }

  function graphByIdOrThrow(id: string): SerialGraphDocument {
    const graph = graphById(id)
    if (!graph) {
      throw new Error(`serial graph not found: ${id}`)
    }
    return graph
  }

  function replaceGraph(id: string, update: (graph: SerialGraphDocument) => SerialGraphDocument) {
    graphs.value = graphs.value.map(graph => (
      graph.id === id ? cloneSerialGraphDocument(update(graph)) : graph
    ))
  }

  function activeRuntimeState(): GraphRuntimeState {
    return runtimeFor(activeGraph.value?.id ?? activeGraphId.value ?? 'graph-1')
  }

  function runtimeFor(graphId: string): GraphRuntimeState {
    if (!runtimeStates.value[graphId]) {
      runtimeStates.value = {
        ...runtimeStates.value,
        [graphId]: emptyRuntimeState(),
      }
    }
    return runtimeStates.value[graphId]
  }

  function nextGraphId(): string {
    const ids = new Set(graphs.value.map(graph => graph.id))
    let seq = 1
    while (ids.has(`graph-${seq}`)) seq += 1
    return `graph-${seq}`
  }

  function nextNodeId(graph: SerialGraphDocument): string {
    return `node-${nextSequence(graph.nodes.map(node => node.id), 'node-')}`
  }

  function nextEdgeId(graph: SerialGraphDocument): string {
    return `edge-${nextSequence(graph.edges.map(edge => edge.id), 'edge-')}`
  }

  function defaultNodePosition(): SerialGraphPosition {
    return defaultNodePositionForGraph(ensureActiveGraph())
  }

  function defaultNodePositionForGraph(graph: SerialGraphDocument): SerialGraphPosition {
    const index = graph.nodes.length
    return {
      x: 40 + (index % 3) * 220,
      y: 48 + Math.floor(index / 3) * 140,
    }
  }

  return {
    graphs,
    graphList,
    activeGraphId,
    activeGraph,
    nodes,
    edges,
    selectedNodeId,
    selectedEdgeId,
    nodeTabs,
    activeNodeTabId,
    runtimeGraphId,
    runtimeStatus,
    runtimeError,
    nodeStatuses,
    nodeBuffers,
    nodeBufferText,
    nodeFrames,
    validation,
    validationErrors,
    graphById,
    runtimeStateForGraph,
    validationForGraph,
    createGraph,
    duplicateGraph,
    importGraphDocument,
    removeGraph,
    renameGraph,
    setActiveGraph,
    addNode,
    addNodeToGraph,
    removeNode,
    removeNodeFromGraph,
    moveNode,
    moveNodeInGraph,
    updateNodeConfig,
    updateNodeConfigInGraph,
    connect,
    connectInGraph,
    removeEdge,
    removeEdgeFromGraph,
    selectNode,
    selectNodeInGraph,
    selectEdge,
    selectEdgeInGraph,
    setActiveNodeTab,
    setActiveNodeTabInGraph,
    closeNodeTab,
    closeNodeTabInGraph,
    resetWorkspace,
    exportState,
    restoreState,
    exportRuntimeSnapshot,
    restoreRuntimeSnapshot,
    startRuntime,
    startRuntimeForGraph,
    stopRuntime,
    stopRuntimeForGraph,
    refreshRuntime,
    refreshRuntimeForGraph,
    sendNode,
    sendNodeForGraph,
    queryNodeBuffer,
    queryNodeBufferForGraph,
    queryNodeFrames,
    queryNodeFramesForGraph,
    clearNodeBuffer,
    clearNodeBufferForGraph,
    resetNodeCounters,
    resetNodeCountersForGraph,
  }
})

function emptyRuntimeState(): GraphRuntimeState {
  return {
    runtimeGraphId: null,
    runtimeStatus: 'idle',
    runtimeError: null,
    nodeStatuses: new Map(),
    nodeBuffers: new Map(),
    nodeBufferText: new Map(),
    nodeFrames: new Map(),
  }
}

function stoppedStatuses(statuses: Map<string, SerialGraphNodeStatus>): Map<string, SerialGraphNodeStatus> {
  const next = new Map<string, SerialGraphNodeStatus>()
  for (const [id, status] of statuses.entries()) {
    next.set(id, { ...status, Status: 'stopped' })
  }
  return next
}

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

function graphNumber(id: string): number {
  const value = Number(id.replace(/^graph-/, ''))
  return Number.isFinite(value) ? value : 1
}

function normalizeRuntimeStatus(status: string): RuntimeStatus {
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
