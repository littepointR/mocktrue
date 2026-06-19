import type { SettingsSnapshot } from '../settings/stores/settingsStore'
import type { SerialConfig } from '../../bindings/github.com/suyue/mocktrue/internal/modules/serial/port/models.js'
import type { Bridge, VirtualPort } from '../serial/stores/virtualStore'
import type { SerialWorkspaceState } from '../serial/stores/workspaceStore'

export const workspaceKind = 'mocktrue.workspace.v1'

export type JsonRecord = Record<string, unknown>

export interface WorkspaceBufferChunk {
  timestamp: number
  data: string
}

export interface WorkspaceHandleSnapshot {
  id: string
  config: SerialConfig
  isOpen: boolean
  rxBytes: number
  txBytes: number
}

export interface WorkspaceSnapshot {
  kind: typeof workspaceKind
  settings: SettingsSnapshot
  serial: {
    activePortId: string | null
    handles: WorkspaceHandleSnapshot[]
    virtualPorts: VirtualPort[]
    bridges: Bridge[]
    buffers: Record<string, WorkspaceBufferChunk[]>
    workspace: SerialWorkspaceState
  }
}

export function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  const chunkSize = 0x8000
  for (let offset = 0; offset < bytes.length; offset += chunkSize) {
    const chunk = bytes.subarray(offset, offset + chunkSize)
    binary += String.fromCharCode(...chunk)
  }
  return btoa(binary)
}

export function base64ToBytes(value: string): Uint8Array {
  const binary = atob(value)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

export function stableStringify(value: unknown): string {
  return JSON.stringify(sortJson(value))
}

function sortJson(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortJson)
  }
  if (!value || typeof value !== 'object') {
    return value
  }
  const input = value as JsonRecord
  const output: JsonRecord = {}
  for (const key of Object.keys(input).sort()) {
    output[key] = sortJson(input[key])
  }
  return output
}
