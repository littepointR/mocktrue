import { useSettingsStore, type SerialModuleSettings } from '../settings/stores/settingsStore'
import { useSerialStore } from '../serial/stores/serialStore'
import { useBufferStore } from '../serial/stores/bufferStore'
import type { SerializableBufferChunk } from '../serial/stores/bufferStore'
import { useVirtualStore } from '../serial/stores/virtualStore'
import { useSerialWorkspaceStore } from '../serial/stores/workspaceStore'
import { useMonitorStore } from '../serial/stores/monitorStore'
import { useModbusStore } from '../serial/stores/modbusStore'
import { useFecbusStore } from '../serial/stores/fecbusStore'
import { useSerialGraphStore } from '../serial/stores/graphStore'
import {
  base64ToBytes,
  bytesToBase64,
  graphTabKind,
  type GraphTabSnapshot,
  workspaceKind,
  type WorkspaceSnapshot,
} from './workspaceSnapshot'
import { cloneSerialGraphDocument, cloneSerialGraphState, type SerialGraphDocument } from '../serial/graph/serialGraph'

export interface WorkspaceRestoreError {
  target: string
  message: string
}

export interface WorkspaceRestoreResult {
  errors: WorkspaceRestoreError[]
  handleMap: Record<string, string>
}

export interface GraphTabRestoreResult {
  graphIds: string[]
  activeGraphId: string | null
}

export function buildWorkspaceSnapshot(): WorkspaceSnapshot {
  const settingsStore = useSettingsStore()
  const serialStore = useSerialStore()
  const bufferStore = useBufferStore()
  const virtualStore = useVirtualStore()
  const workspaceStore = useSerialWorkspaceStore()
  const monitorStore = useMonitorStore()
  const modbusStore = useModbusStore()
  const fecbusStore = useFecbusStore()
  const graphStore = useSerialGraphStore()

  return {
    kind: workspaceKind,
    settings: {
      serial: settingsStore.snapshot().serial,
    },
    serial: {
      activePortId: serialStore.activePortId,
      handles: Array.from(serialStore.handles.values()).filter(handle => handle.IsOpen).map(handle => ({
        id: handle.ID,
        config: handle.Config,
        isOpen: handle.IsOpen,
        rxBytes: handle.RxBytes,
        txBytes: handle.TxBytes,
      })),
      virtualPorts: virtualStore.virtualPorts.map(port => ({ ...port })),
      bridges: virtualStore.bridges.map(bridge => ({ ...bridge })),
      buffers: exportBuffers(bufferStore.exportChunks()),
      monitors: monitorStore.exportState(),
      modbus: modbusStore.exportState(),
      fecbus: fecbusStore.exportState(),
      graph: graphStore.exportState(),
      workspace: workspaceStore.exportState(),
    },
  }
}

export function buildGraphTabSnapshot(
  graphId?: string | null,
  options: { serialSettings?: SerialModuleSettings } = {}
): GraphTabSnapshot {
  const settingsStore = useSettingsStore()
  const graphStore = useSerialGraphStore()
  const graph = graphStore.graphById(graphId ?? graphStore.activeGraphId) ?? graphStore.activeGraph
  if (!graph) {
    throw new Error('serial graph not found')
  }
  const runtime = graphStore.exportRuntimeSnapshot(graph.id)

  return {
    kind: graphTabKind,
    settings: {
      serial: options.serialSettings ?? settingsStore.snapshot().serial,
    },
    graph: exportGraphDocument(graph),
    runtime: {
      nodeBuffers: Object.fromEntries(Object.entries(runtime.nodeBuffers).map(([nodeId, bytes]) => [
        nodeId,
        [{ timestamp: 0, data: bytesToBase64(bytes) }],
      ])),
      nodeFrames: Object.fromEntries(Object.entries(runtime.nodeFrames).map(([nodeId, frames]) => [
        nodeId,
        frames.map(frame => ({ ...frame })),
      ])),
    },
  }
}

export async function restoreGraphTabSnapshot(
  snapshot: GraphTabSnapshot | WorkspaceSnapshot,
  options: { activate?: boolean } = {}
): Promise<GraphTabRestoreResult> {
  const settingsStore = useSettingsStore()
  const graphStore = useSerialGraphStore()
  const graphSnapshots = graphTabSnapshotsFromUnknown(snapshot)
  const graphIds: string[] = []
  for (const [index, graphSnapshot] of graphSnapshots.entries()) {
    settingsStore.replaceSerialSettings(graphSnapshot.settings?.serial)
    const graph = graphStore.importGraphDocument(graphSnapshot.graph, options.activate !== false && index === graphSnapshots.length - 1)
    graphStore.restoreRuntimeSnapshot(graph.id, importGraphRuntime(graphSnapshot))
    graphIds.push(graph.id)
  }
  if (options.activate !== false && graphIds.length > 0) {
    graphStore.setActiveGraph(graphIds[graphIds.length - 1])
  }
  return {
    graphIds,
    activeGraphId: graphIds.length > 0 ? graphIds[graphIds.length - 1] : graphStore.activeGraphId,
  }
}

export function graphTabSnapshotsFromUnknown(snapshot: GraphTabSnapshot | WorkspaceSnapshot): GraphTabSnapshot[] {
  const kind = (snapshot as GraphTabSnapshot | WorkspaceSnapshot | undefined)?.kind
  if (kind === graphTabKind) {
    return [normalizeGraphTabSnapshot(snapshot as GraphTabSnapshot)]
  }
  if (kind !== workspaceKind && !(snapshot as WorkspaceSnapshot | undefined)?.serial?.graph) {
    throw new Error('unsupported MockTrue config file')
  }
  return graphTabSnapshotsFromWorkspace(snapshot as WorkspaceSnapshot)
}

export async function restoreWorkspaceSnapshot(snapshot: WorkspaceSnapshot): Promise<WorkspaceRestoreResult> {
  const errors: WorkspaceRestoreError[] = []
  const handleMap: Record<string, string> = {}
  const settingsStore = useSettingsStore()
  const serialStore = useSerialStore()
  const bufferStore = useBufferStore()
  const virtualStore = useVirtualStore()
  const workspaceStore = useSerialWorkspaceStore()
  const monitorStore = useMonitorStore()
  const modbusStore = useModbusStore()
  const fecbusStore = useFecbusStore()
  const graphStore = useSerialGraphStore()

  await captureError(errors, 'serial.closeAll', () => serialStore.closeAllPorts())
  await captureError(errors, 'virtual.cleanup', () => virtualStore.cleanupAllResources())
  monitorStore.restoreState()
  serialStore.clearLocalHandles()
  bufferStore.clearAll()
  workspaceStore.resetWorkspace()
  modbusStore.resetWorkspace()
  fecbusStore.resetWorkspace()
  graphStore.resetWorkspace()
  settingsStore.replaceSerialSettings(snapshot.settings?.serial)

  for (const vport of snapshot.serial.virtualPorts) {
    await captureError(errors, `virtual:${vport.ID}`, () => virtualStore.createVirtualPort(vport.ID, vport.Port))
  }

  for (const handle of snapshot.serial.handles) {
    if (!handle.isOpen) continue
    await captureError(errors, `port:${handle.id}`, async () => {
      handleMap[handle.id] = await serialStore.openConfig(handle.config)
    })
  }

  for (const bridge of snapshot.serial.bridges) {
    await captureError(errors, `bridge:${bridge.ID}`, () => (
      virtualStore.createBridge(bridge.ID, bridge.Port1, bridge.Port2, bridge.BaudRate)
    ))
  }

  for (const handle of snapshot.serial.handles) {
    const nextId = handleMap[handle.id]
    if (!nextId) continue
    await captureError(errors, `counters:${handle.id}`, () => (
      serialStore.restoreCounters(nextId, handle.rxBytes, handle.txBytes)
    ))
  }

  bufferStore.restoreChunks(importBuffers(snapshot.serial.buffers, handleMap))
  monitorStore.restoreState(snapshot.serial.monitors)
  modbusStore.restoreState(snapshot.serial.modbus)
  fecbusStore.restoreState(snapshot.serial.fecbus)
  graphStore.restoreState(snapshot.serial.graph)
  workspaceStore.restoreState(snapshot.serial.workspace, handleMap)
  serialStore.setActivePort(remapID(snapshot.serial.activePortId, handleMap))

  return { errors, handleMap }
}

function normalizeGraphTabSnapshot(snapshot: GraphTabSnapshot): GraphTabSnapshot {
  return {
    kind: graphTabKind,
    settings: {
      serial: snapshot.settings?.serial ?? useSettingsStore().snapshot().serial,
    },
    graph: cloneSerialGraphDocument(snapshot.graph),
    runtime: {
      nodeBuffers: snapshot.runtime?.nodeBuffers ?? {},
      nodeFrames: snapshot.runtime?.nodeFrames ?? {},
    },
  }
}

function graphTabSnapshotsFromWorkspace(snapshot: WorkspaceSnapshot): GraphTabSnapshot[] {
  const state = cloneSerialGraphState(snapshot.serial?.graph as any)
  const activeGraphId = state.activeGraphId ?? state.graphs[0]?.id ?? null
  const serialSettings = snapshot.settings?.serial ?? useSettingsStore().snapshot().serial
  return state.graphs.map(graph => ({
    kind: graphTabKind,
    settings: { serial: serialSettings },
    graph: cloneSerialGraphDocument({
      ...graph,
      id: graph.id === activeGraphId ? graph.id : graph.id,
    }),
    runtime: { nodeBuffers: {}, nodeFrames: {} },
  }))
}

function exportGraphDocument(graph: SerialGraphDocument): SerialGraphDocument {
  return cloneSerialGraphDocument({
    ...graph,
    nodes: graph.nodes.map(node => {
      const { status, error, ...rest } = node
      void status
      void error
      return rest
    }),
  })
}

function importGraphRuntime(snapshot: GraphTabSnapshot) {
  return {
    nodeBuffers: Object.fromEntries(Object.entries(snapshot.runtime?.nodeBuffers ?? {}).map(([nodeId, chunks]) => [
      nodeId,
      concatBufferChunks(chunks),
    ])),
    nodeFrames: Object.fromEntries(Object.entries(snapshot.runtime?.nodeFrames ?? {}).map(([nodeId, frames]) => [
      nodeId,
      frames.map(frame => ({ ...frame })),
    ])),
  }
}

function concatBufferChunks(chunks: WorkspaceSnapshot['serial']['buffers'][string]): Uint8Array {
  const parts = chunks.map(chunk => base64ToBytes(chunk.data))
  const length = parts.reduce((sum, bytes) => sum + bytes.length, 0)
  const result = new Uint8Array(length)
  let offset = 0
  for (const part of parts) {
    result.set(part, offset)
    offset += part.length
  }
  return result
}

function exportBuffers(buffers: Record<string, SerializableBufferChunk[]>): WorkspaceSnapshot['serial']['buffers'] {
  return Object.fromEntries(Object.entries(buffers).map(([portId, chunks]) => [
    portId,
    chunks.map(chunk => ({
      timestamp: chunk.timestamp,
      data: bytesToBase64(new Uint8Array(chunk.data)),
    })),
  ]))
}

function importBuffers(
  buffers: WorkspaceSnapshot['serial']['buffers'],
  handleMap: Record<string, string>
): Parameters<ReturnType<typeof useBufferStore>['restoreChunks']>[0] {
  return Object.fromEntries(Object.entries(buffers).map(([portId, chunks]) => [
    handleMap[portId] ?? portId,
    chunks.map(chunk => ({
      timestamp: chunk.timestamp,
      data: Array.from(base64ToBytes(chunk.data)),
    })),
  ]))
}

async function captureError(errors: WorkspaceRestoreError[], target: string, fn: () => Promise<unknown>) {
  try {
    await fn()
  } catch (e: any) {
    errors.push({ target, message: e?.message ?? String(e) })
  }
}

function remapID(id: string | null, handleMap: Record<string, string>): string | null {
  if (!id) return id
  return handleMap[id] ?? id
}
