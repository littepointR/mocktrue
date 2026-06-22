import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { graphTabKind, stableStringify } from '../workspaceSnapshot'
import {
  DefaultWorkspacePath,
  ExportWorkspace,
  LoadLastWorkspace,
  ReadWorkspace,
  RememberLastWorkspace,
  SaveWorkspace,
  SelectWorkspaceOpenPath,
  SelectWorkspaceSavePath,
} from '../../../bindings/github.com/littepointR/mocktrue/internal/core/workspace/service.js'
import {
  buildGraphTabSnapshot,
  graphTabSnapshotsFromUnknown,
  restoreGraphTabSnapshot,
} from '../workspaceSession'
import { createDemoWorkspaceSnapshot, listDemoWorkspaces } from '../demoWorkspaces'
import { useSerialGraphStore } from '../../serial/stores/graphStore'
import { useSettingsStore, type SerialModuleSettings } from '../../settings/stores/settingsStore'

type WorkspaceSourceKind = 'empty' | 'file' | 'demo' | 'legacy'

interface OpenGraphOptions {
  cancelPendingStartup?: boolean
  shouldAbort?: () => boolean
}

interface GraphFileState {
  graphId: string
  path: string
  savedSnapshot: string
  currentSnapshot: string
  sourceKind: WorkspaceSourceKind
  readOnly: boolean
  title?: string
}

const recentFilesStorageKey = 'mocktrue.open-config-files.v1'

export const useWorkspaceFileStore = defineStore('workspaceFile', () => {
  const demos = listDemoWorkspaces()
  const states = ref<Record<string, GraphFileState>>({})
  const lastError = ref<string | null>(null)
  const selectedDemoId = ref(demos[0]?.id ?? '')
  const startupRestoreVersion = ref(0)

  const graphStore = useSerialGraphStore()
  const settingsStore = useSettingsStore()

  const activeGraphId = computed(() => graphStore.activeGraphId ?? graphStore.graphList[0]?.id ?? '')
  const activeState = computed(() => activeGraphId.value ? states.value[activeGraphId.value] ?? null : null)
  const currentPath = computed(() => activeState.value?.path ?? '')
  const sourceKind = computed(() => activeState.value?.sourceKind ?? 'empty')
  const isDirty = computed(() => Boolean(activeState.value && activeState.value.currentSnapshot !== activeState.value.savedSnapshot))

  function graphState(graphId: string): GraphFileState | null {
    return states.value[graphId] ?? null
  }

  function isGraphDirty(graphId: string): boolean {
    const state = graphState(graphId)
    return Boolean(state && state.currentSnapshot !== state.savedSnapshot)
  }

  function graphPath(graphId: string): string {
    return graphState(graphId)?.path ?? ''
  }

  function graphTitle(graphId: string, fallback: string): string {
    const state = graphState(graphId)
    const suffix = isGraphDirty(graphId) ? '*' : ''
    return `${state?.title ?? fallback}${suffix}`
  }

  function graphTooltip(graphId: string): string {
    return graphState(graphId)?.path || '未保存'
  }

  function markClean(path: string, snapshot: unknown, graphId = activeGraphId.value) {
    if (!graphId) return
    setGraphFileState(graphId, {
      path,
      savedSnapshot: stableStringify(snapshot),
      currentSnapshot: stableStringify(snapshot),
      sourceKind: path ? 'file' : 'empty',
      readOnly: false,
    })
    lastError.value = null
    persistRecentFiles()
  }

  function updateCurrentSnapshot(snapshot: unknown, graphId = activeGraphId.value) {
    if (!graphId) return
    const serialized = stableStringify(snapshot)
    const current = states.value[graphId]
    states.value = {
      ...states.value,
      [graphId]: current
        ? { ...current, currentSnapshot: serialized }
        : {
            graphId,
            path: '',
            savedSnapshot: serialized,
            currentSnapshot: serialized,
            sourceKind: 'empty',
            readOnly: false,
          },
    }
  }

  function syncGraphSnapshot(graphId: string, options: { useCurrentSerialSettings?: boolean } = {}) {
    try {
      updateCurrentSnapshot(buildSnapshotForGraph(graphId, options), graphId)
    } catch {
      // Graph may have been removed between watcher ticks.
    }
  }

  function syncAllGraphSnapshots() {
    for (const graph of graphStore.graphList) {
      syncGraphSnapshot(graph.id)
    }
  }

  function markDirty(graphId = activeGraphId.value) {
    if (!graphId) return
    const current = states.value[graphId]
    if (!current) return
    states.value = {
      ...states.value,
      [graphId]: { ...current, currentSnapshot: `${current.currentSnapshot}:${Date.now()}` },
    }
  }

  function setPath(path: string, graphId = activeGraphId.value) {
    if (!graphId) return
    const current = states.value[graphId]
    states.value = {
      ...states.value,
      [graphId]: {
        graphId,
        path,
        savedSnapshot: current?.savedSnapshot ?? '',
        currentSnapshot: current?.currentSnapshot ?? '',
        sourceKind: path ? 'file' : 'empty',
        readOnly: false,
        title: current?.title,
      },
    }
    persistRecentFiles()
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
    return saveGraph(activeGraphId.value, path)
  }

  async function saveGraph(graphId: string, path?: string): Promise<string> {
    try {
      const current = states.value[graphId]
      if (current?.readOnly && !path) {
        return await saveGraphAs(graphId)
      }
      const snapshot = buildSnapshotForGraph(graphId, { useCurrentSerialSettings: graphId === activeGraphId.value })
      const targetPath = (path ?? current?.path) || await DefaultWorkspacePath()
      assertPathCanBind(targetPath, graphId)
      await SaveWorkspace(targetPath, stableStringify(snapshot))
      setGraphFileState(graphId, {
        path: targetPath,
        savedSnapshot: stableStringify(snapshot),
        currentSnapshot: stableStringify(snapshot),
        sourceKind: 'file',
        readOnly: false,
      })
      await rememberPath(targetPath)
      persistRecentFiles()
      lastError.value = null
      return targetPath
    } catch (e: any) {
      setError(e?.message ?? 'Save config failed')
      throw e
    }
  }

  async function selectOpenPath(): Promise<string> {
    try {
      const path = await SelectWorkspaceOpenPath(currentPath.value)
      lastError.value = null
      return path
    } catch (e: any) {
      setError(e?.message ?? 'Select config file failed')
      throw e
    }
  }

  async function saveAs(): Promise<string> {
    return saveGraphAs(activeGraphId.value)
  }

  async function saveGraphAs(graphId: string): Promise<string> {
    try {
      const path = await SelectWorkspaceSavePath(states.value[graphId]?.path ?? '')
      if (!path) return ''
      return await saveGraph(graphId, path)
    } catch (e: any) {
      setError(e?.message ?? 'Save config failed')
      throw e
    }
  }

  async function importFromPath(path: string) {
    return openFromPath(path)
  }

  async function openFromPath(path: string, options: OpenGraphOptions = {}) {
    try {
      if (options.cancelPendingStartup !== false) {
        cancelPendingStartupRestore()
      }
      const existing = graphIdByPath(path)
      if (existing) {
        graphStore.setActiveGraph(existing)
        lastError.value = null
        return { graphIds: [existing], activeGraphId: existing, reused: true }
      }

      const file = await ReadWorkspace(path)
      if (options.shouldAbort?.()) {
        return { graphIds: [], activeGraphId: null, reused: false, aborted: true }
      }
      if (!file) throw new Error('config file is empty')
      const snapshot = JSON.parse(file.Content)
      const originalKind = snapshot?.kind
      const tabSnapshots = graphTabSnapshotsFromUnknown(snapshot)
      const opened: string[] = []
      for (const [index, tabSnapshot] of tabSnapshots.entries()) {
        const result = await restoreGraphTabSnapshot(tabSnapshot, { activate: true })
        for (const graphId of result.graphIds) {
          const cleanSnapshot = buildGraphTabSnapshot(graphId, { serialSettings: tabSnapshot.settings.serial })
          setGraphFileState(graphId, {
            path: index === 0 ? file.Path : '',
            savedSnapshot: stableStringify(cleanSnapshot),
            currentSnapshot: stableStringify(cleanSnapshot),
            sourceKind: originalKind === 'mocktrue.graph.v1' ? 'file' : 'legacy',
            readOnly: false,
          })
          opened.push(graphId)
        }
      }
      if (opened.length > 0) {
        graphStore.setActiveGraph(opened[opened.length - 1])
      }
      await rememberPath(file.Path)
      persistRecentFiles()
      lastError.value = null
      return { graphIds: opened, activeGraphId: opened[opened.length - 1] ?? null, reused: false }
    } catch (e: any) {
      setError(e?.message ?? 'Open config failed')
      throw e
    }
  }

  async function importSelected() {
    try {
      const path = await SelectWorkspaceOpenPath(currentPath.value)
      if (!path) return null
      return await openFromPath(path)
    } catch (e: any) {
      setError(e?.message ?? 'Open config failed')
      throw e
    }
  }

  async function exportCopy(): Promise<string> {
    try {
      const path = await SelectWorkspaceSavePath(currentPath.value)
      if (!path) return ''
      const snapshot = buildSnapshotForGraph(activeGraphId.value, { useCurrentSerialSettings: true })
      await ExportWorkspace(path, stableStringify(snapshot))
      lastError.value = null
      return path
    } catch (e: any) {
      setError(e?.message ?? 'Export config failed')
      throw e
    }
  }

  async function loadLast(): Promise<boolean> {
    try {
      const restoreVersion = startupRestoreVersion.value
      const shouldAbort = () => startupRestoreVersion.value !== restoreVersion
      const paths = loadRecentFiles()
      if (paths.length > 0) {
        let opened = false
        for (const path of paths) {
          try {
            if (shouldAbort()) return opened
            const result = await openFromPath(path, { cancelPendingStartup: false, shouldAbort })
            opened = opened || result.graphIds.length > 0
          } catch {
            // Continue opening the rest of the last session.
          }
        }
        return opened
      }

      const file = await LoadLastWorkspace()
      if (shouldAbort()) return false
      if (!file?.Found) return false
      const result = await openFromPath(file.Path, { cancelPendingStartup: false, shouldAbort })
      return result.graphIds.length > 0
    } catch (e: any) {
      setError(e?.message ?? 'Load last config failed')
      throw e
    }
  }

  async function loadDemo(demoId: string) {
    try {
      const snapshot = createDemoWorkspaceSnapshot(demoId)
      if (!snapshot) {
        throw new Error(`Unknown demo workspace: ${demoId}`)
      }
      cancelPendingStartupRestore()
      await removePristineEmptyGraphs()
      const tabSnapshots = graphTabSnapshotsFromUnknown(snapshot)
      const result = await restoreGraphTabSnapshot(snapshot, { activate: true })
      for (const [index, graphId] of result.graphIds.entries()) {
        const cleanSnapshot = buildGraphTabSnapshot(graphId, { serialSettings: tabSnapshots[index]?.settings.serial })
        setGraphFileState(graphId, {
          path: '',
          savedSnapshot: stableStringify(cleanSnapshot),
          currentSnapshot: stableStringify(cleanSnapshot),
          sourceKind: 'demo',
          readOnly: true,
        })
      }
      lastError.value = null
      return result
    } catch (e: any) {
      setError(e?.message ?? 'Load demo config failed')
      throw e
    }
  }

  function removeGraphState(graphId: string) {
    if (!states.value[graphId]) return
    const next = { ...states.value }
    delete next[graphId]
    states.value = next
    persistRecentFiles()
  }

  function cancelPendingStartupRestore() {
    startupRestoreVersion.value += 1
  }

  async function removePristineEmptyGraphs() {
    const emptyGraphIds = graphStore.graphList
      .map(graph => graph.id)
      .filter(isPristineEmptyGraph)
    for (const graphId of emptyGraphIds) {
      await graphStore.removeGraph(graphId)
      removeGraphState(graphId)
    }
  }

  function isPristineEmptyGraph(graphId: string): boolean {
    const graph = graphStore.graphById(graphId)
    if (!graph || graph.nodes.length > 0 || graph.edges.length > 0) {
      return false
    }
    const state = states.value[graphId]
    if (!state) {
      return true
    }
    return (state.sourceKind === 'empty' || state.sourceKind === 'file' || state.sourceKind === 'legacy')
      && state.currentSnapshot === state.savedSnapshot
      && serializedSnapshotIsEmptyGraph(state.currentSnapshot)
  }

  function serializedSnapshotIsEmptyGraph(serialized: string): boolean {
    if (!serialized) return true
    try {
      const snapshot = JSON.parse(serialized)
      const graphs = graphTabSnapshotsFromUnknown(snapshot).map(item => item.graph)
      return graphs.length > 0 && graphs.every(graph => graph.nodes.length === 0 && graph.edges.length === 0)
    } catch {
      return false
    }
  }

  function applyGraphSerialSettings(graphId = activeGraphId.value): boolean {
    if (!graphId) return false
    const serialSettings = serialSettingsForGraph(graphId)
    if (!serialSettings) return false
    settingsStore.replaceSerialSettings(serialSettings)
    return true
  }

  function setEditableSource(path: string) {
    setPath(path)
  }

  function markSnapshotClean(snapshot: unknown) {
    markClean(currentPath.value, snapshot)
  }

  function setGraphFileState(graphId: string, patch: Omit<Partial<GraphFileState>, 'graphId'>) {
    const current = states.value[graphId]
    states.value = {
      ...states.value,
      [graphId]: {
        graphId,
        path: patch.path ?? current?.path ?? '',
        savedSnapshot: patch.savedSnapshot ?? current?.savedSnapshot ?? '',
        currentSnapshot: patch.currentSnapshot ?? current?.currentSnapshot ?? '',
        sourceKind: patch.sourceKind ?? current?.sourceKind ?? 'empty',
        readOnly: patch.readOnly ?? current?.readOnly ?? false,
        title: patch.title ?? current?.title,
      },
    }
  }

  function buildSnapshotForGraph(graphId: string, options: { useCurrentSerialSettings?: boolean } = {}) {
    const serialSettings = options.useCurrentSerialSettings ? settingsStore.snapshot().serial : serialSettingsForGraph(graphId)
    return buildGraphTabSnapshot(graphId, { serialSettings })
  }

  function serialSettingsForGraph(graphId: string): SerialModuleSettings | undefined {
    const state = states.value[graphId]
    return serialSettingsFromSerializedSnapshot(state?.currentSnapshot)
      ?? serialSettingsFromSerializedSnapshot(state?.savedSnapshot)
  }

  function serialSettingsFromSerializedSnapshot(serialized: string | undefined): SerialModuleSettings | undefined {
    if (!serialized) return undefined
    try {
      const snapshot = JSON.parse(serialized)
      return snapshot?.kind === graphTabKind ? snapshot.settings?.serial : undefined
    } catch {
      return undefined
    }
  }

  function graphIdByPath(path: string): string | null {
    const normalized = normalizePath(path)
    for (const state of Object.values(states.value)) {
      if (state.path && normalizePath(state.path) === normalized) {
        return state.graphId
      }
    }
    return null
  }

  function assertPathCanBind(path: string, graphId: string) {
    const existing = graphIdByPath(path)
    if (existing && existing !== graphId) {
      throw new Error(`config file already open: ${path}`)
    }
  }

  async function rememberPath(path: string) {
    if (!path) return
    try {
      await RememberLastWorkspace(path)
    } catch {
      // Remembering the path is non-critical; saving/opening already succeeded.
    }
  }

  function persistRecentFiles() {
    if (typeof localStorage === 'undefined') return
    const files = Object.values(states.value)
      .filter(state => state.path && state.sourceKind === 'file')
      .map(state => state.path)
    localStorage.setItem(recentFilesStorageKey, JSON.stringify(Array.from(new Set(files))))
  }

  function loadRecentFiles(): string[] {
    if (typeof localStorage === 'undefined') return []
    const raw = localStorage.getItem(recentFilesStorageKey)
    if (!raw) return []
    try {
      const parsed = JSON.parse(raw)
      return Array.isArray(parsed) ? parsed.filter(item => typeof item === 'string') : []
    } catch {
      return []
    }
  }

  function normalizePath(path: string): string {
    return path.trim()
  }

  return {
    states,
    currentPath,
    lastError,
    sourceKind,
    isDirty,
    activeState,
    graphState,
    isGraphDirty,
    graphPath,
    graphTitle,
    graphTooltip,
    markClean,
    updateCurrentSnapshot,
    syncGraphSnapshot,
    syncAllGraphSnapshots,
    markDirty,
    setPath,
    setError,
    selectedDemoId,
    setSelectedDemo,
    save,
    saveGraph,
    selectOpenPath,
    saveAs,
    saveGraphAs,
    importFromPath,
    openFromPath,
    importSelected,
    exportCopy,
    loadLast,
    loadDemo,
    removeGraphState,
    applyGraphSerialSettings,
    setEditableSource,
    markSnapshotClean,
    listDemos: () => demos.map(demo => ({ ...demo })),
  }
})
