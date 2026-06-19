import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { EditorLayoutNode } from '../views/editorLayout'

export type SerialOperation = 'open' | 'virtual' | 'bridge' | 'monitor'
export type ReceiveViewMode = 'ascii' | 'hexClassic' | 'hexTable'
export type ReceiveLayoutMode = 'combined' | 'split'
export type SendMode = 'ascii' | 'hex'

export interface SendHistoryItem {
  id: number
  content: string
  mode: SendMode
}

export interface DataDisplayState {
  viewMode: ReceiveViewMode
  layoutMode: ReceiveLayoutMode
  showTimestamp: boolean
  autoScroll: boolean
}

export interface SendPanelState {
  sendData: string
  sendMode: SendMode
  autoSend: boolean
  sendIntervalMs: number
  sendHistory: SendHistoryItem[]
}

export interface SerialTabWorkspaceState {
  showConfig: boolean
  sendHeight: number
  dataDisplay: DataDisplayState
  sendPanel: SendPanelState
}

export type SerialTabWorkspacePatch =
  Omit<Partial<SerialTabWorkspaceState>, 'dataDisplay' | 'sendPanel'> & {
    dataDisplay?: Partial<DataDisplayState>
    sendPanel?: Partial<SendPanelState>
  }

export interface SerialWorkspaceState {
  selectedOperation: SerialOperation | null
  editorLayout: EditorLayoutNode
  activeByGroup: Record<string, string | null>
  tabStates: Record<string, SerialTabWorkspaceState>
}

export const defaultEditorLayout: EditorLayoutNode = { type: 'group', id: 'group-1', tabs: [] }

export function defaultTabState(): SerialTabWorkspaceState {
  return {
    showConfig: false,
    sendHeight: 180,
    dataDisplay: {
      viewMode: 'ascii',
      layoutMode: 'combined',
      showTimestamp: true,
      autoScroll: true,
    },
    sendPanel: {
      sendData: '',
      sendMode: 'ascii',
      autoSend: false,
      sendIntervalMs: 1000,
      sendHistory: [],
    },
  }
}

export const useSerialWorkspaceStore = defineStore('serialWorkspace', () => {
  const selectedOperation = ref<SerialOperation | null>(null)
  const editorLayout = ref<EditorLayoutNode>({ ...defaultEditorLayout })
  const activeByGroup = ref<Record<string, string | null>>({ 'group-1': null })
  const tabStates = ref<Record<string, SerialTabWorkspaceState>>({})

  function tabState(handleId: string): SerialTabWorkspaceState {
    if (!tabStates.value[handleId]) {
      tabStates.value[handleId] = defaultTabState()
    }
    return tabStates.value[handleId]
  }

  function updateTabState(handleId: string, next: SerialTabWorkspacePatch) {
    const current = tabState(handleId)
    tabStates.value[handleId] = {
      ...current,
      ...next,
      dataDisplay: { ...current.dataDisplay, ...next.dataDisplay },
      sendPanel: { ...current.sendPanel, ...next.sendPanel },
    }
  }

  function setSelectedOperation(operation: SerialOperation | null) {
    selectedOperation.value = operation
  }

  function setEditorLayout(next: EditorLayoutNode) {
    editorLayout.value = next
  }

  function setActiveByGroup(next: Record<string, string | null>) {
    activeByGroup.value = next
  }

  function resetWorkspace() {
    selectedOperation.value = null
    editorLayout.value = { ...defaultEditorLayout }
    activeByGroup.value = { 'group-1': null }
    tabStates.value = {}
  }

  function exportState(): SerialWorkspaceState {
    return {
      selectedOperation: selectedOperation.value,
      editorLayout: editorLayout.value,
      activeByGroup: activeByGroup.value,
      tabStates: tabStates.value,
    }
  }

  function restoreState(snapshot: SerialWorkspaceState, handleMap: Record<string, string> = {}) {
    selectedOperation.value = snapshot.selectedOperation
    editorLayout.value = remapLayout(snapshot.editorLayout, handleMap)
    activeByGroup.value = remapActiveByGroup(snapshot.activeByGroup, handleMap)
    tabStates.value = remapTabStates(snapshot.tabStates, handleMap)
  }

  return {
    selectedOperation,
    editorLayout,
    activeByGroup,
    tabStates,
    tabState,
    updateTabState,
    setSelectedOperation,
    setEditorLayout,
    setActiveByGroup,
    resetWorkspace,
    exportState,
    restoreState,
  }
})

function remapID(id: string | null, handleMap: Record<string, string>): string | null {
  if (!id) return id
  return handleMap[id] ?? id
}

function remapLayout(node: EditorLayoutNode, handleMap: Record<string, string>): EditorLayoutNode {
  if (node.type === 'group') {
    return { ...node, tabs: node.tabs.map(id => remapID(id, handleMap) ?? id) }
  }
  return { ...node, children: node.children.map(child => remapLayout(child, handleMap)) }
}

function remapActiveByGroup(active: Record<string, string | null>, handleMap: Record<string, string>): Record<string, string | null> {
  return Object.fromEntries(Object.entries(active).map(([groupId, handleId]) => [groupId, remapID(handleId, handleMap)]))
}

function remapTabStates(
  states: Record<string, SerialTabWorkspaceState>,
  handleMap: Record<string, string>
): Record<string, SerialTabWorkspaceState> {
  const result: Record<string, SerialTabWorkspaceState> = {}
  for (const [handleId, state] of Object.entries(states)) {
    result[handleMap[handleId] ?? handleId] = state
  }
  return result
}
