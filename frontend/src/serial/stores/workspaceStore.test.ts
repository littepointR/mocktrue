import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useSerialWorkspaceStore } from './workspaceStore'

describe('serial workspace store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('keeps editor layout and tab UI state outside component lifetime', () => {
    const store = useSerialWorkspaceStore()
    store.setEditorLayout({
      type: 'split',
      id: 'split-1',
      direction: 'horizontal',
      children: [
        { type: 'group', id: 'group-1', tabs: ['port-1'] },
        { type: 'group', id: 'group-2', tabs: ['port-2'] },
      ],
    })
    store.setActiveByGroup({ 'group-1': 'port-1', 'group-2': 'port-2' })
    store.updateTabState('port-1', {
      sendHeight: 260,
      dataDisplay: { viewMode: 'hexClassic', layoutMode: 'split', showTimestamp: false, autoScroll: false },
      sendPanel: {
        sendData: 'aa bb',
        sendMode: 'hex',
        autoSend: true,
        sendIntervalMs: 20,
        sendHistory: [{ id: 1, content: 'aa bb', mode: 'hex' }],
      },
    })

    expect(store.editorLayout.type).toBe('split')
    expect(store.activeByGroup['group-2']).toBe('port-2')
    expect(store.tabState('port-1').sendHeight).toBe(260)
    expect(store.tabState('port-1').dataDisplay.viewMode).toBe('hexClassic')
    expect(store.tabState('port-1').sendPanel.sendHistory[0].content).toBe('aa bb')
  })

  it('remaps handle ids when restoring imported layout state', () => {
    const store = useSerialWorkspaceStore()
    store.restoreState({
      selectedOperation: 'open',
      editorLayout: { type: 'group', id: 'group-1', tabs: ['old-port'] },
      activeByGroup: { 'group-1': 'old-port' },
      tabStates: {
        'old-port': {
          showConfig: true,
          sendHeight: 220,
          dataDisplay: { viewMode: 'ascii', layoutMode: 'combined', showTimestamp: true, autoScroll: true },
          sendPanel: {
            sendData: 'hello',
            sendMode: 'ascii',
            autoSend: false,
            sendIntervalMs: 100,
            sendHistory: [],
          },
        },
      },
    }, { 'old-port': 'new-port' })

    expect(store.editorLayout).toEqual({ type: 'group', id: 'group-1', tabs: ['new-port'] })
    expect(store.activeByGroup['group-1']).toBe('new-port')
    expect(store.selectedOperation).toBeNull()
    expect(store.tabState('new-port').sendPanel.sendData).toBe('hello')
  })
})
