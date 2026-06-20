import { useSettingsStore } from '../settings/stores/settingsStore'
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
  workspaceKind,
  type WorkspaceSnapshot,
} from './workspaceSnapshot'

export interface WorkspaceRestoreError {
  target: string
  message: string
}

export interface WorkspaceRestoreResult {
  errors: WorkspaceRestoreError[]
  handleMap: Record<string, string>
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
