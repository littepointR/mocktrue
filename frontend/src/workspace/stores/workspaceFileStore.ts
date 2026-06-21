import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { stableStringify } from '../workspaceSnapshot'
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
import { createDemoWorkspaceSnapshot, listDemoWorkspaces } from '../demoWorkspaces'

type WorkspaceSourceKind = 'empty' | 'file'

export const useWorkspaceFileStore = defineStore('workspaceFile', () => {
  const demos = listDemoWorkspaces()
  const currentPath = ref('')
  const savedSnapshot = ref('')
  const currentSnapshot = ref('')
  const lastError = ref<string | null>(null)
  const sourceKind = ref<WorkspaceSourceKind>('empty')
  const selectedDemoId = ref(demos[0]?.id ?? '')

  const isDirty = computed(() => currentSnapshot.value !== savedSnapshot.value)
  const displayPath = computed(() => currentPath.value)

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
    sourceKind.value = path ? 'file' : 'empty'
  }

  function setError(message: string | null) {
    lastError.value = message
  }

  function setSelectedDemo(id: unknown) {
    const nextId = typeof id === 'string' ? id : ''
    if (demos.some(demo => demo.id === nextId)) {
      selectedDemoId.value = nextId
    }
  }

  async function save(path?: string): Promise<string> {
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
      const snapshot = createDemoWorkspaceSnapshot(demoId)
      if (!snapshot) {
        throw new Error(`Unknown demo workspace: ${demoId}`)
      }
      const result = await restoreWorkspaceSnapshot(snapshot)
      setEditableSource('')
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
  }

  function markSnapshotClean(snapshot: unknown) {
    const serialized = stableStringify(snapshot)
    savedSnapshot.value = serialized
    currentSnapshot.value = serialized
  }

  return {
    currentPath,
    lastError,
    sourceKind,
    isDirty,
    displayPath,
    markClean,
    updateCurrentSnapshot,
    markDirty,
    setPath,
    setError,
    selectedDemoId,
    setSelectedDemo,
    save,
    selectOpenPath,
    saveAs,
    importFromPath,
    importSelected,
    exportCopy,
    loadLast,
    loadDemo,
    listDemos: () => demos.map(demo => ({ ...demo })),
  }
})
