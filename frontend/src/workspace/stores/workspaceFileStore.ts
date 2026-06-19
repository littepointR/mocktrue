import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { base64ToBytes, stableStringify, type WorkspaceSnapshot } from '../workspaceSnapshot'
import {
  DefaultWorkspacePath,
  ExportWorkspace,
  LoadLastWorkspace,
  ReadWorkspace,
  RememberLastWorkspace,
  SaveWorkspace,
  SelectWorkspaceOpenPath,
  SelectWorkspaceSavePath,
} from '../../../bindings/github.com/suyue/mocktrue/internal/core/workspace/service.js'
import { buildWorkspaceSnapshot, restoreWorkspaceSnapshot } from '../workspaceSession'
import { createDemoWorkspaceSnapshot, getDemoWorkspace, listDemoWorkspaces } from '../demoWorkspaces'
import { useBufferStore } from '../../serial/stores/bufferStore'
import { useSerialStore } from '../../serial/stores/serialStore'

type WorkspaceSourceKind = 'empty' | 'file' | 'demo'

export const useWorkspaceFileStore = defineStore('workspaceFile', () => {
  const currentPath = ref('')
  const savedSnapshot = ref('')
  const currentSnapshot = ref('')
  const lastError = ref<string | null>(null)
  const sourceKind = ref<WorkspaceSourceKind>('empty')
  const readonly = ref(false)
  const currentDemoId = ref('')
  const currentDemoTitle = ref('')

  const isDirty = computed(() => currentSnapshot.value !== savedSnapshot.value)
  const displayPath = computed(() => {
    if (sourceKind.value === 'demo') {
      return `Demo: ${currentDemoTitle.value || currentDemoId.value}`
    }
    return currentPath.value
  })
  const canSaveDirectly = computed(() => !readonly.value)

  function markClean(path: string, snapshot: unknown) {
    setEditableSource(path)
    markSnapshotClean(snapshot)
    lastError.value = null
  }

  function updateCurrentSnapshot(snapshot: unknown) {
    currentSnapshot.value = stableStringify(snapshot)
  }

  function markDirty() {
    currentSnapshot.value = `${currentSnapshot.value}:${Date.now()}`
  }

  function setPath(path: string) {
    currentPath.value = path
    if (sourceKind.value === 'demo') return
    sourceKind.value = path ? 'file' : 'empty'
    readonly.value = false
    currentDemoId.value = ''
    currentDemoTitle.value = ''
  }

  function setError(message: string | null) {
    lastError.value = message
  }

  async function save(path?: string): Promise<string> {
    if (readonly.value && !path) {
      const error = new Error('Demo 配置为只读，请使用另存为')
      setError(error.message)
      throw error
    }
    try {
      const snapshot = buildWorkspaceSnapshot()
      const targetPath = (path ?? currentPath.value) || await DefaultWorkspacePath()
      await SaveWorkspace(targetPath, stableStringify(snapshot))
      markClean(targetPath, snapshot)
      return targetPath
    } catch (e: any) {
      setError(e?.message ?? 'Save workspace failed')
      throw e
    }
  }

  async function selectOpenPath(): Promise<string> {
    try {
      const path = await SelectWorkspaceOpenPath(currentPath.value)
      if (path) setPath(path)
      lastError.value = null
      return path
    } catch (e: any) {
      setError(e?.message ?? 'Select workspace file failed')
      throw e
    }
  }

  async function saveAs(): Promise<string> {
    try {
      const path = await SelectWorkspaceSavePath(currentPath.value)
      if (!path) return ''
      return await save(path)
    } catch (e: any) {
      setError(e?.message ?? 'Save workspace failed')
      throw e
    }
  }

  async function importFromPath(path: string) {
    try {
      const file = await ReadWorkspace(path)
      if (!file) throw new Error('workspace file is empty')
      const snapshot = JSON.parse(file.Content)
      const result = await restoreWorkspaceSnapshot(snapshot)
      await RememberLastWorkspace(file.Path)
      markClean(file.Path, snapshot)
      return result
    } catch (e: any) {
      setError(e?.message ?? 'Import workspace failed')
      throw e
    }
  }

  async function importSelected() {
    try {
      const path = currentPath.value || await SelectWorkspaceOpenPath(currentPath.value)
      if (!path) return null
      return await importFromPath(path)
    } catch (e: any) {
      setError(e?.message ?? 'Import workspace failed')
      throw e
    }
  }

  async function exportCopy(): Promise<string> {
    try {
      const path = await SelectWorkspaceSavePath(currentPath.value)
      if (!path) return ''
      const snapshot = buildWorkspaceSnapshot()
      await ExportWorkspace(path, stableStringify(snapshot))
      lastError.value = null
      return path
    } catch (e: any) {
      setError(e?.message ?? 'Export workspace failed')
      throw e
    }
  }

  async function loadLast(): Promise<boolean> {
    try {
      const file = await LoadLastWorkspace()
      if (!file?.Found) return false
      const snapshot = JSON.parse(file.Content)
      await restoreWorkspaceSnapshot(snapshot)
      markClean(file.Path, snapshot)
      return true
    } catch (e: any) {
      setError(e?.message ?? 'Load last workspace failed')
      throw e
    }
  }

  async function loadDemo(demoId: string) {
    try {
      const demo = getDemoWorkspace(demoId)
      const snapshot = createDemoWorkspaceSnapshot(demoId)
      if (!demo || !snapshot) {
        throw new Error(`Unknown demo workspace: ${demoId}`)
      }
      const result = await restoreWorkspaceSnapshot(snapshot)
      restoreDemoDisplayFallback(snapshot, result.handleMap)
      currentPath.value = ''
      sourceKind.value = 'demo'
      readonly.value = true
      currentDemoId.value = demo.id
      currentDemoTitle.value = demo.title
      markSnapshotClean(buildWorkspaceSnapshot())
      lastError.value = null
      return result
    } catch (e: any) {
      setError(e?.message ?? 'Load demo workspace failed')
      throw e
    }
  }

  function setEditableSource(path: string) {
    currentPath.value = path
    sourceKind.value = path ? 'file' : 'empty'
    readonly.value = false
    currentDemoId.value = ''
    currentDemoTitle.value = ''
  }

  function markSnapshotClean(snapshot: unknown) {
    const serialized = stableStringify(snapshot)
    savedSnapshot.value = serialized
    currentSnapshot.value = serialized
  }

  function restoreDemoDisplayFallback(snapshot: WorkspaceSnapshot, handleMap: Record<string, string>) {
    const serialStore = useSerialStore()
    const bufferStore = useBufferStore()

    for (const handle of snapshot.serial.handles) {
      if (!handle.isOpen) continue
      const targetId = handleMap[handle.id] ?? handle.id
      if (!serialStore.handles.has(targetId)) {
        serialStore.restoreDemoHandle({
          ID: targetId,
          Config: handle.config,
          IsOpen: true,
          RxBytes: handle.rxBytes,
          TxBytes: handle.txBytes,
        })
      }

      const chunks = snapshot.serial.buffers[handle.id]
      if (chunks) {
        bufferStore.restorePortChunks(targetId, chunks.map(chunk => ({
          timestamp: chunk.timestamp,
          data: Array.from(base64ToBytes(chunk.data)),
        })))
      }
    }

    if (snapshot.serial.activePortId) {
      const activeId = handleMap[snapshot.serial.activePortId] ?? snapshot.serial.activePortId
      if (serialStore.handles.has(activeId)) {
        serialStore.setActivePort(activeId)
      }
    }
  }

  return {
    currentPath,
    lastError,
    sourceKind,
    readonly,
    currentDemoId,
    currentDemoTitle,
    isDirty,
    displayPath,
    canSaveDirectly,
    markClean,
    updateCurrentSnapshot,
    markDirty,
    setPath,
    setError,
    save,
    selectOpenPath,
    saveAs,
    importFromPath,
    importSelected,
    exportCopy,
    loadLast,
    loadDemo,
    listDemos: listDemoWorkspaces,
  }
})
