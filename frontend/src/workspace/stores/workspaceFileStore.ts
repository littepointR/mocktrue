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

export const useWorkspaceFileStore = defineStore('workspaceFile', () => {
  const currentPath = ref('')
  const savedSnapshot = ref('')
  const currentSnapshot = ref('')
  const lastError = ref<string | null>(null)

  const isDirty = computed(() => currentSnapshot.value !== savedSnapshot.value)

  function markClean(path: string, snapshot: unknown) {
    currentPath.value = path
    const serialized = stableStringify(snapshot)
    savedSnapshot.value = serialized
    currentSnapshot.value = serialized
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
  }

  function setError(message: string | null) {
    lastError.value = message
  }

  async function save(path = currentPath.value): Promise<string> {
    try {
      const snapshot = buildWorkspaceSnapshot()
      const targetPath = path || await DefaultWorkspacePath()
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

  return {
    currentPath,
    lastError,
    isDirty,
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
  }
})
