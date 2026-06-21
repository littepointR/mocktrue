export type SerialGraphPortKind = 'bytes' | 'frame' | 'registers' | 'status' | 'control'
export type SerialGraphPortDirection = 'input' | 'output'
export type SerialGraphNodeStatus = 'idle' | 'running' | 'error'

export interface SerialGraphPosition {
  x: number
  y: number
}

export interface SerialGraphPortSpec {
  id: string
  label: string
  kind: SerialGraphPortKind
  direction: SerialGraphPortDirection
  multiple?: boolean
}

export interface SerialGraphNodeProvider {
  type: string
  title: string
  category: string
  description: string
  inputs: SerialGraphPortSpec[]
  outputs: SerialGraphPortSpec[]
  defaultConfig: Record<string, unknown>
  resourceOwner?: boolean
  resourceKeys?: string[]
}

export interface SerialGraphNode {
  id: string
  type: string
  name?: string
  position: SerialGraphPosition
  config: Record<string, unknown>
  status?: SerialGraphNodeStatus
  error?: string
}

export interface SerialGraphEdge {
  id: string
  source: string
  sourceHandle: string
  target: string
  targetHandle: string
}

export interface SerialGraphNodeTab {
  nodeId: string
  title: string
}

export interface SerialGraphDocument {
  id: string
  name: string
  nodes: SerialGraphNode[]
  edges: SerialGraphEdge[]
  selectedNodeId: string | null
  selectedEdgeId: string | null
  nodeTabs: SerialGraphNodeTab[]
  activeNodeTabId: string | null
}

export interface SerialGraphWorkspaceState {
  graphs: SerialGraphDocument[]
  activeGraphId: string | null
}

export type SerialGraphTopologyState = Pick<SerialGraphDocument, 'nodes' | 'edges'>

export interface SerialGraphConnectionDraft {
  id?: string
  source: string
  sourceHandle: string
  target: string
  targetHandle: string
}

export interface SerialGraphValidationResult {
  valid: boolean
  errors: string[]
}

const serialDefaults = {
  portName: '',
  baudRate: 115200,
  dataBits: 8,
  stopBits: '1',
  parity: 'none',
  flowMode: 'none',
  readBufKB: 32,
}

const bytesIn: SerialGraphPortSpec = { id: 'in', label: '接收', kind: 'bytes', direction: 'input' }
const bytesOut: SerialGraphPortSpec = { id: 'out', label: '发送', kind: 'bytes', direction: 'output' }

export const serialGraphProviders: SerialGraphNodeProvider[] = [
  {
    type: 'serial.physical',
    title: '物理串口',
    category: '串口',
    description: '系统中已有的真实串口资源。',
    inputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'input' }],
    outputs: [{ id: 'rx', label: '接收', kind: 'bytes', direction: 'output' }],
    defaultConfig: { ...serialDefaults },
    resourceOwner: true,
    resourceKeys: ['portName'],
  },
  {
    type: 'serial.virtual',
    title: '虚拟串口',
    category: '串口',
    description: '由 MockTrue 创建和管理的单端虚拟串口。',
    inputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'input' }],
    outputs: [{ id: 'rx', label: '接收', kind: 'bytes', direction: 'output' }],
    defaultConfig: { ...serialDefaults, portName: 'mocktrue-vport' },
    resourceOwner: true,
    resourceKeys: ['portName'],
  },
  {
    type: 'serial.bridge',
    title: '串口桥接',
    category: '串口',
    description: '把两路字节流双向桥接。',
    inputs: [
      { id: 'a-in', label: '接收 A', kind: 'bytes', direction: 'input' },
      { id: 'b-in', label: '接收 B', kind: 'bytes', direction: 'input' },
    ],
    outputs: [
      { id: 'a-out', label: '发送 A', kind: 'bytes', direction: 'output' },
      { id: 'b-out', label: '发送 B', kind: 'bytes', direction: 'output' },
    ],
    defaultConfig: {},
  },
  {
    type: 'serial.monitor',
    title: '串口监控',
    category: '工具',
    description: '监听一路串口字节流并生成监控帧。',
    inputs: [bytesIn],
    outputs: [],
    defaultConfig: { displayMode: 'hex' },
  },
  {
    type: 'serial.tap',
    title: '分流器',
    category: '工具',
    description: '允许一路字节流分发到多个下游节点。',
    inputs: [bytesIn],
    outputs: [{ ...bytesOut, multiple: true }],
    defaultConfig: {},
  },
  {
    type: 'serial.tee',
    title: 'T 型分支',
    category: '工具',
    description: '分流器的别名，用于表达串联链路中的并联分支。',
    inputs: [bytesIn],
    outputs: [{ ...bytesOut, multiple: true }],
    defaultConfig: {},
  },
  {
    type: 'serial.sender',
    title: '发送器',
    category: '终端',
    description: '从编辑框或发送历史输出字节流。',
    inputs: [],
    outputs: [bytesOut],
    defaultConfig: { mode: 'ascii', encoding: 'utf-8', payload: '', autoSend: false, intervalMs: 1000 },
  },
  {
    type: 'serial.receiver',
    title: '接收器',
    category: '终端',
    description: '显示接收到的字节流。',
    inputs: [bytesIn],
    outputs: [],
    defaultConfig: { viewMode: 'ascii', autoScroll: true },
  },
  {
    type: 'serial.modbus.master',
    title: 'Modbus 主站',
    category: '协议',
    description: 'Modbus RTU/ASCII 主站请求和寄存器视图。',
    inputs: [{ id: 'rx', label: '接收', kind: 'bytes', direction: 'input' }],
    outputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'output' }],
    defaultConfig: { mode: 'rtu', unitIds: '1', addressMode: 'zero-based', functionCode: 3, address: 0, quantity: 1, value: 0 },
  },
  {
    type: 'serial.modbus.slave',
    title: 'Modbus 从站',
    category: '协议',
    description: 'Modbus RTU/ASCII 多 Unit ID 从站数据区。',
    inputs: [
      { id: 'rx', label: '接收', kind: 'bytes', direction: 'input' },
    ],
    outputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'output' }],
    defaultConfig: { mode: 'rtu', unitIds: '1' },
  },
  {
    type: 'serial.fecbus.master',
    title: 'FECbus 主控',
    category: '协议',
    description: 'FECbus 主控请求和帧解析。',
    inputs: [{ id: 'rx', label: '接收', kind: 'bytes', direction: 'input' }],
    outputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'output' }],
    defaultConfig: { sourceAddress: 1, targetAddress: 2, priority: 3, messageNumber: 1, groupNumber: 0, functionCode: 44, dataHex: '' },
  },
  {
    type: 'serial.fecbus.slave',
    title: 'FECbus 从机',
    category: '协议',
    description: 'FECbus 从机应答和设备状态。',
    inputs: [{ id: 'rx', label: '接收', kind: 'bytes', direction: 'input' }],
    outputs: [{ id: 'tx', label: '发送', kind: 'bytes', direction: 'output' }],
    defaultConfig: { address: 2, defaultStatus: 10, autoStatusAnswer: true },
  },
]

export function providerByType(type: string): SerialGraphNodeProvider | null {
  return serialGraphProviders.find(provider => provider.type === type) ?? null
}

export function createSerialGraphNode(
  id: string,
  type: string,
  position: SerialGraphPosition,
  config: Record<string, unknown> = {}
): SerialGraphNode {
  const provider = providerByType(type)
  return {
    id,
    type,
    name: provider?.title ?? type,
    position,
    config: {
      ...(provider?.defaultConfig ?? {}),
      ...config,
    },
    status: 'idle',
  }
}

export function defaultSerialGraphDocument(id = 'graph-1', name = '拓扑图 1'): SerialGraphDocument {
  return {
    id,
    name,
    nodes: [],
    edges: [],
    selectedNodeId: null,
    selectedEdgeId: null,
    nodeTabs: [],
    activeNodeTabId: null,
  }
}

export function defaultSerialGraphState(): SerialGraphWorkspaceState {
  const graph = defaultSerialGraphDocument()
  return {
    graphs: [graph],
    activeGraphId: graph.id,
  }
}

export function nodeTabTitle(node: SerialGraphNode): string {
  return node.name || providerByType(node.type)?.title || node.id
}

export function cloneSerialGraphDocument(graph: Partial<SerialGraphDocument> & SerialGraphTopologyState): SerialGraphDocument {
  const nodes = graph.nodes.map(node => ({
    ...node,
    name: node.name ?? providerByType(node.type)?.title ?? node.type,
    position: { ...node.position },
    config: sanitizeNodeConfig(node.type, node.config),
  }))
  const nodeIds = new Set(nodes.map(node => node.id))
  const nodeTabs = (graph.nodeTabs ?? [])
    .filter(tab => nodeIds.has(tab.nodeId))
    .map(tab => ({ ...tab }))
  const activeNodeTabId = graph.activeNodeTabId && nodeIds.has(graph.activeNodeTabId)
    ? graph.activeNodeTabId
    : nodeTabs[0]?.nodeId ?? null

  return {
    id: graph.id ?? 'graph-1',
    name: graph.name ?? '拓扑图 1',
    nodes,
    edges: graph.edges.map(edge => ({ ...edge })),
    selectedNodeId: graph.selectedNodeId && nodeIds.has(graph.selectedNodeId) ? graph.selectedNodeId : null,
    selectedEdgeId: graph.selectedEdgeId ?? null,
    nodeTabs,
    activeNodeTabId,
  }
}

export function cloneSerialGraphState(state: SerialGraphWorkspaceState | SerialGraphDocument | null | undefined): SerialGraphWorkspaceState {
  if (!state) return defaultSerialGraphState()
  if (Array.isArray((state as SerialGraphWorkspaceState).graphs)) {
    const graphs = (state as SerialGraphWorkspaceState).graphs.map(graph => cloneSerialGraphDocument(graph))
    if (graphs.length === 0) return defaultSerialGraphState()
    const activeGraphId = graphs.some(graph => graph.id === (state as SerialGraphWorkspaceState).activeGraphId)
      ? (state as SerialGraphWorkspaceState).activeGraphId
      : graphs[0].id
    return { graphs, activeGraphId }
  }

  const legacy = state as SerialGraphDocument
  const graph = cloneSerialGraphDocument({
    ...legacy,
    id: legacy.id ?? 'graph-1',
    name: legacy.name ?? '拓扑图 1',
    nodeTabs: legacy.nodeTabs ?? legacy.nodes.map(node => ({ nodeId: node.id, title: nodeTabTitle(node) })),
    activeNodeTabId: legacy.activeNodeTabId ?? legacy.selectedNodeId ?? legacy.nodes[0]?.id ?? null,
  })
  return {
    graphs: [graph],
    activeGraphId: graph.id,
  }
}

export function canConnect(
  state: SerialGraphTopologyState,
  draft: SerialGraphConnectionDraft
): SerialGraphValidationResult {
  const errors: string[] = []
  const sourceNode = state.nodes.find(node => node.id === draft.source)
  const targetNode = state.nodes.find(node => node.id === draft.target)
  if (!sourceNode) errors.push(`source node not found: ${draft.source}`)
  if (!targetNode) errors.push(`target node not found: ${draft.target}`)
  if (!sourceNode || !targetNode) return { valid: false, errors }
  if (sourceNode.id === targetNode.id) {
    errors.push('node cannot connect to itself')
  }

  const sourceProvider = providerByType(sourceNode.type)
  const targetProvider = providerByType(targetNode.type)
  if (!sourceProvider) errors.push(`provider not found: ${sourceNode.type}`)
  if (!targetProvider) errors.push(`provider not found: ${targetNode.type}`)
  if (!sourceProvider || !targetProvider) return { valid: false, errors }

  const sourcePort = sourceProvider.outputs.find(port => port.id === draft.sourceHandle)
  const targetPort = targetProvider.inputs.find(port => port.id === draft.targetHandle)
  if (!sourcePort) errors.push(`output port not found: ${sourceNode.type}.${draft.sourceHandle}`)
  if (!targetPort) errors.push(`input port not found: ${targetNode.type}.${draft.targetHandle}`)
  if (!sourcePort || !targetPort) return { valid: false, errors }

  if (!compatibleKinds(sourcePort.kind, targetPort.kind)) {
    errors.push(`incompatible port kinds: ${sourcePort.kind} -> ${targetPort.kind}`)
  }

  const otherEdges = state.edges.filter(edge => edge.id !== draft.id)
  if (!targetPort.multiple && otherEdges.some(edge => (
    edge.target === draft.target && edge.targetHandle === draft.targetHandle
  ))) {
    errors.push(`input already connected: ${targetNode.id}.${targetPort.id}`)
  }

  const fanOutAllowed = sourcePort.multiple || sourceProvider.type === 'serial.tap' || sourceProvider.type === 'serial.tee'
  if (!fanOutAllowed && otherEdges.some(edge => (
    edge.source === draft.source && edge.sourceHandle === draft.sourceHandle
  ))) {
    errors.push(`fan-out requires a tap node: ${sourceNode.id}.${sourcePort.id}`)
  }

  if (wouldCreateDirectedCycle(draft, otherEdges)) {
    errors.push('directed cycle not allowed')
  }

  return { valid: errors.length === 0, errors }
}

export function validateGraph(state: SerialGraphTopologyState): SerialGraphValidationResult {
  const errors: string[] = []

  for (const node of state.nodes) {
    if (!providerByType(node.type)) {
      errors.push(`provider not found: ${node.type}`)
    }
  }

  errors.push(...validateResourceOwners(state))

  for (const edge of state.edges) {
    const result = canConnect(state, edge)
    if (!result.valid) {
      errors.push(...result.errors.map(error => `${edge.id}: ${error}`))
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

function validateResourceOwners(state: SerialGraphTopologyState): string[] {
  const errors: string[] = []
  const used = new Map<string, SerialGraphNode>()

  for (const node of state.nodes) {
    const provider = providerByType(node.type)
    if (!provider?.resourceOwner) continue

    for (const key of provider.resourceKeys ?? []) {
      const value = String(node.config[key] ?? '').trim()
      if (!value) continue
      const resourceKey = `${key}:${value}`
      const previous = used.get(resourceKey)
      if (previous) {
        errors.push(`resource port duplicated: ${value} (${previous.id}, ${node.id})`)
      } else {
        used.set(resourceKey, node)
      }
    }
  }

  return errors
}

function compatibleKinds(source: SerialGraphPortKind, target: SerialGraphPortKind): boolean {
  return source === target
}

function sanitizeNodeConfig(type: string, config: Record<string, unknown> = {}): Record<string, unknown> {
  const next = { ...config }
  if (type === 'serial.monitor') {
    delete next.mode
  }
  if (type === 'serial.bridge') {
    delete next.baudRate
  }
  if (type === 'serial.modbus.slave') {
    delete next.addressMode
  }
  return next
}

function wouldCreateDirectedCycle(
  draft: SerialGraphConnectionDraft,
  edges: SerialGraphEdge[]
): boolean {
  const visited = new Set<string>()
  const stack = [draft.target]

  while (stack.length > 0) {
    const nodeId = stack.pop()
    if (!nodeId || visited.has(nodeId)) continue
    if (nodeId === draft.source) return true
    visited.add(nodeId)

    for (const edge of edges) {
      if (edge.source === nodeId) {
        stack.push(edge.target)
      }
    }
  }

  return false
}
