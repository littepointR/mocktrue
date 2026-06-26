import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWorkspaceFileStore } from './workspaceFileStore'
import { useSerialGraphStore } from '../../serial/stores/graphStore'
import { defaultSerialSettings, useSettingsStore } from '../../settings/stores/settingsStore'

const workspaceServiceMock = vi.hoisted(() => {
  const readFiles = new Map<string, string>()
  return {
    readFiles,
    DefaultWorkspacePath: vi.fn(async () => '/tmp/default.portweave.json'),
    ExportWorkspace: vi.fn(async (_path: string, _content: string) => undefined),
    SaveWorkspace: vi.fn(async (_path: string, _content: string) => undefined),
    ReadWorkspace: vi.fn(async (path: string) => ({ Path: path, Content: readFiles.get(path) ?? graphFile(path, '导入拓扑') })),
    LoadLastWorkspace: vi.fn(async () => ({ Found: true, Path: '/tmp/last.portweave.json', Content: graphFile('/tmp/last.portweave.json', '最近拓扑') })),
    RememberLastWorkspace: vi.fn(async () => undefined),
    SelectWorkspaceOpenPath: vi.fn(async () => '/tmp/selected-open.portweave.json'),
    SelectWorkspaceSavePath: vi.fn(async () => '/tmp/selected-save.portweave.json'),
  }

  function graphFile(path: string, name: string) {
    return JSON.stringify({
      kind: 'portweave.graph.v1',
      settings: { serial: defaultSerialSettings },
      graph: {
        id: path.includes('last') ? 'last-graph' : 'imported-graph',
        name,
        nodes: [],
        edges: [],
        selectedNodeId: null,
        selectedEdgeId: null,
        nodeTabs: [],
        activeNodeTabId: null,
      },
      runtime: { nodeBuffers: {}, nodeFrames: {} },
    })
  }
})

const serialBindingsMock = vi.hoisted(() => ({
  StartSerialGraph: vi.fn(),
  StopSerialGraph: vi.fn(),
  GetSerialGraphStatus: vi.fn(),
  SendSerialGraphNode: vi.fn(),
  QuerySerialGraphNodeBuffer: vi.fn(),
  QuerySerialGraphNodeFrames: vi.fn(),
  ClearSerialGraphNodeBuffer: vi.fn(),
  ResetSerialGraphNodeCounters: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/core/workspace/service.js', () => workspaceServiceMock)
vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => serialBindingsMock)

describe('workspaceFileStore graph tab files', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    workspaceServiceMock.readFiles.clear()
  })

  it('tracks dirty state per graph tab', () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    files.syncAllGraphSnapshots()

    expect(files.isDirty).toBe(false)

    graph.addNode('serial.sender')
    files.syncGraphSnapshot('graph-1')

    expect(files.isDirty).toBe(true)
    expect(files.isGraphDirty('graph-1')).toBe(true)

    const second = graph.createGraph('第二拓扑')
    files.syncGraphSnapshot(second.id)

    expect(files.isGraphDirty(second.id)).toBe(false)
    expect(files.isGraphDirty('graph-1')).toBe(true)
  })

  it('saves only the active graph tab and marks it clean', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.renameGraph('graph-1', '主拓扑')
    graph.addNode('serial.sender')
    files.syncAllGraphSnapshots()

    const path = await files.save()

    expect(path).toBe('/tmp/default.portweave.json')
    expect(workspaceServiceMock.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/default.portweave.json',
      expect.stringContaining('portweave.graph.v1')
    )
    const savedContent = workspaceServiceMock.SaveWorkspace.mock.calls[0]?.[1] as string
    expect(savedContent).toContain('主拓扑')
    expect(files.currentPath).toBe('/tmp/default.portweave.json')
    expect(files.isDirty).toBe(false)
  })

  it('preserves a non-active graph tab serial settings when saving it', async () => {
    const graph = useSerialGraphStore()
    const settings = useSettingsStore()
    const files = useWorkspaceFileStore()
    const second = graph.createGraph('第二拓扑')
    files.markClean('/tmp/first.portweave.json', JSON.parse(graphFile('/tmp/first.portweave.json', '第一拓扑', { TerminalFontSize: 15 })), 'graph-1')
    files.markClean('/tmp/second.portweave.json', JSON.parse(graphFile('/tmp/second.portweave.json', '第二拓扑', { TerminalFontSize: 21 })), second.id)
    graph.setActiveGraph('graph-1')
    settings.updateSerial({ TerminalFontSize: 33 })
    files.syncGraphSnapshot('graph-1', { useCurrentSerialSettings: true })

    await files.saveGraph(second.id, '/tmp/second-save.portweave.json')

    const savedContent = workspaceServiceMock.SaveWorkspace.mock.calls[0]?.[1] as string
    expect(JSON.parse(savedContent).settings.serial.TerminalFontSize).toBe(21)
    expect(files.isGraphDirty('graph-1')).toBe(true)
    expect(files.isGraphDirty(second.id)).toBe(false)
  })

  it('opens a config file as a new graph tab and reuses it on the next open', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    const first = await files.openFromPath('/tmp/import.portweave.json')

    expect(first.reused).toBe(false)
    expect(graph.graphList.map(item => item.name)).toEqual(['拓扑图 1', '导入拓扑'])
    expect(files.currentPath).toBe('/tmp/import.portweave.json')
    expect(files.isDirty).toBe(false)

    const second = await files.openFromPath('/tmp/import.portweave.json')

    expect(second.reused).toBe(true)
    expect(graph.graphList.map(item => item.name)).toEqual(['拓扑图 1', '导入拓扑'])
    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledTimes(1)
  })

  it('loads examples as read-only graph tabs that save via save-as', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    const result = await files.loadDemo('bridge-demo')
    const graphId = result.activeGraphId!

    expect(graph.activeGraphId).toBe(graphId)
    expect(files.graphState(graphId)?.sourceKind).toBe('demo')
    expect(files.graphState(graphId)?.readOnly).toBe(true)

    await files.save()

    expect(workspaceServiceMock.SelectWorkspaceSavePath).toHaveBeenCalled()
    expect(workspaceServiceMock.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/selected-save.portweave.json',
      expect.stringContaining('portweave.graph.v1')
    )
    expect(files.graphState(graphId)?.readOnly).toBe(false)
    expect(files.graphState(graphId)?.sourceKind).toBe('file')
  })

  it('replaces existing clean empty topology tabs when loading an example', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    graph.createGraph('拓扑图 2')
    files.syncAllGraphSnapshots()
    files.markClean('/tmp/empty.portweave.json', JSON.parse(graphFile('/tmp/empty.portweave.json', '空拓扑')), 'graph-1')

    const result = await files.loadDemo('bridge-demo')

    expect(result.graphIds).toHaveLength(1)
    expect(graph.graphList.map(item => item.id)).toEqual([result.activeGraphId])
    expect(graph.graphList.map(item => item.name)).toEqual(['串口桥接演示'])
  })

  it('keeps only the demo graph when startup restore races with loading an example', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/startup.portweave.json']))
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    const releaseRead = deferred<void>()
    workspaceServiceMock.ReadWorkspace.mockImplementationOnce(async (path: string) => {
      await releaseRead.promise
      return { Path: path, Content: graphFile(path, '拓扑图 1') }
    })

    const startupRestore = files.loadLast()
    const result = await files.loadDemo('bridge-demo')
    releaseRead.resolve()
    await startupRestore

    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/startup.portweave.json')
    expect(result.graphIds).toHaveLength(1)
    expect(graph.graphList.map(item => item.id)).toEqual([result.activeGraphId])
    expect(graph.graphList.map(item => item.name)).toEqual(['串口桥接演示'])
  })

  it('replaces a clean empty topology restored from a workspace without graph state when loading an example', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/empty-workspace.portweave.json']))
    workspaceServiceMock.readFiles.set('/tmp/empty-workspace.portweave.json', workspaceFileWithoutGraph())
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    await files.loadLast()
    expect(graph.graphList.map(item => item.name)).toContain('拓扑图 1')

    const result = await files.loadDemo('bridge-demo')

    expect(result.graphIds).toHaveLength(1)
    expect(graph.graphList.map(item => item.id)).toEqual([result.activeGraphId])
    expect(graph.graphList.map(item => item.name)).toEqual(['串口桥接演示'])
  })

  it('prevents two graph tabs from binding to the same file path', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    files.markClean('/tmp/current.portweave.json', { kind: 'portweave.graph.v1', graph: 'one' }, 'graph-1')
    const second = graph.createGraph('第二拓扑')
    files.syncGraphSnapshot(second.id)

    await expect(files.saveGraph(second.id, '/tmp/current.portweave.json')).rejects.toThrow('already open')
  })

  it('rejects unsupported JSON files instead of opening an empty graph', async () => {
    workspaceServiceMock.readFiles.set('/tmp/unsupported.portweave.json', '{"kind":"other"}')
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    await expect(files.openFromPath('/tmp/unsupported.portweave.json')).rejects.toThrow('unsupported PortWeave config file')

    expect(graph.graphList.map(item => item.name)).toEqual(['拓扑图 1'])
  })

  it('loads recent files from local storage as graph tabs on startup', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/first.portweave.json', '/tmp/second.portweave.json']))
    workspaceServiceMock.readFiles.set('/tmp/first.portweave.json', graphFile('/tmp/first.portweave.json', '第一拓扑'))
    workspaceServiceMock.readFiles.set('/tmp/second.portweave.json', graphFile('/tmp/second.portweave.json', '第二拓扑'))
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    const loaded = await files.loadLast()

    expect(loaded).toBe(true)
    expect(graph.graphList.map(item => item.name)).toEqual(['拓扑图 1', '第一拓扑', '第二拓扑'])
    expect(files.currentPath).toBe('/tmp/second.portweave.json')
    expect(workspaceServiceMock.LoadLastWorkspace).not.toHaveBeenCalled()
  })

  it('exposes graph path labels and mutates explicit graph file state', () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    expect(files.graphPath('missing')).toBe('')
    expect(files.graphTooltip('missing')).toBe('未保存')
    expect(files.graphTitle('missing', 'Fallback')).toBe('Fallback')

    files.markDirty('graph-1')
    expect(files.graphState('graph-1')).toBeNull()

    files.markClean('/tmp/first.portweave.json', JSON.parse(graphFile('/tmp/first.portweave.json', '第一拓扑')), 'graph-1')
    expect(files.graphPath('graph-1')).toBe('/tmp/first.portweave.json')
    expect(files.graphTooltip('graph-1')).toBe('/tmp/first.portweave.json')
    expect(files.graphTitle('graph-1', '第一拓扑')).toBe('第一拓扑')

    files.markDirty('graph-1')
    expect(files.graphTitle('graph-1', '第一拓扑')).toBe('第一拓扑*')

    const second = graph.createGraph('第二拓扑', false)
    files.setPath('/tmp/second.portweave.json', second.id)
    expect(files.graphState(second.id)).toEqual(expect.objectContaining({
      path: '/tmp/second.portweave.json',
      sourceKind: 'file',
      readOnly: false,
    }))

    files.setEditableSource('/tmp/editable.portweave.json')
    files.markSnapshotClean({ kind: 'clean-snapshot' })
    expect(files.currentPath).toBe('/tmp/editable.portweave.json')
    expect(files.isDirty).toBe(false)
  })

  it('handles picker cancellations and service errors for file actions', async () => {
    const files = useWorkspaceFileStore()

    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('')
    await expect(files.saveAs()).resolves.toBe('')

    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('')
    await expect(files.exportCopy()).resolves.toBe('')

    workspaceServiceMock.SelectWorkspaceOpenPath.mockResolvedValueOnce('')
    await expect(files.importSelected()).resolves.toBeNull()

    workspaceServiceMock.SelectWorkspaceOpenPath.mockRejectedValueOnce(new Error('open picker failed'))
    await expect(files.selectOpenPath()).rejects.toThrow('open picker failed')
    expect(files.lastError).toBe('open picker failed')

    workspaceServiceMock.SelectWorkspaceSavePath.mockRejectedValueOnce(new Error('save picker failed'))
    await expect(files.saveAs()).rejects.toThrow('save picker failed')
    expect(files.lastError).toBe('save picker failed')

    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('/tmp/export-copy.portweave.json')
    workspaceServiceMock.ExportWorkspace.mockRejectedValueOnce(new Error('export denied'))
    await expect(files.exportCopy()).rejects.toThrow('export denied')
    expect(files.lastError).toBe('export denied')
  })

  it('imports selected paths exports copies and mutates demo and graph state helpers', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    const firstDemo = files.listDemos()[0]
    const secondDemo = files.listDemos()[1]

    files.setSelectedDemo(42)
    expect(files.selectedDemoId).toBe(firstDemo?.id ?? '')
    files.setSelectedDemo(secondDemo?.id ?? firstDemo?.id ?? '')
    expect(files.selectedDemoId).toBe(secondDemo?.id ?? firstDemo?.id ?? '')

    const imported = await files.importFromPath('/tmp/direct-import.portweave.json')
    expect(imported.activeGraphId).toEqual(expect.any(String))
    expect(files.currentPath).toBe('/tmp/direct-import.portweave.json')

    files.removeGraphState(imported.activeGraphId!)
    files.removeGraphState(imported.activeGraphId!)
    expect(files.graphState(imported.activeGraphId!)).toBeNull()

    workspaceServiceMock.SelectWorkspaceOpenPath.mockResolvedValueOnce('/tmp/picked-import.portweave.json')
    const selected = await files.importSelected()
    expect(selected?.activeGraphId).toEqual(expect.any(String))
    expect(files.currentPath).toBe('/tmp/picked-import.portweave.json')

    const exported = await files.exportCopy()
    expect(exported).toBe('/tmp/selected-save.portweave.json')
    expect(workspaceServiceMock.ExportWorkspace).toHaveBeenCalledWith(
      '/tmp/selected-save.portweave.json',
      expect.stringContaining('portweave.graph.v1')
    )
    expect(graph.graphList.map(item => item.name)).toContain('导入拓扑')
  })

  it('falls back from invalid recent storage and filters saved recent file lists', async () => {
    localStorage.setItem('portweave.open-config-files.v1', 'not json')
    workspaceServiceMock.LoadLastWorkspace.mockResolvedValueOnce({ Found: false, Path: '', Content: '' })
    const files = useWorkspaceFileStore()

    await expect(files.loadLast()).resolves.toBe(false)
    expect(workspaceServiceMock.LoadLastWorkspace).toHaveBeenCalled()

    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/recent.portweave.json', 42, null]))
    workspaceServiceMock.readFiles.set('/tmp/recent.portweave.json', graphFile('/tmp/recent.portweave.json', '最近拓扑'))

    await expect(files.loadLast()).resolves.toBe(true)
    expect(files.currentPath).toBe('/tmp/recent.portweave.json')
  })

  it('returns false when all recent workspace restores fail during startup', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/broken-only.portweave.json']))
    workspaceServiceMock.ReadWorkspace.mockRejectedValueOnce(new Error('broken recent file'))
    const files = useWorkspaceFileStore()

    await expect(files.loadLast()).resolves.toBe(false)

    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/broken-only.portweave.json')
    expect(workspaceServiceMock.LoadLastWorkspace).not.toHaveBeenCalled()
  })

  it('opens aborted and empty files with structured results or errors', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    workspaceServiceMock.ReadWorkspace.mockResolvedValueOnce({ Path: '/tmp/abort.portweave.json', Content: graphFile('/tmp/abort.portweave.json', '取消拓扑') })
    await expect(files.openFromPath('/tmp/abort.portweave.json', { shouldAbort: () => true })).resolves.toEqual({
      graphIds: [],
      activeGraphId: null,
      reused: false,
      aborted: true,
    })
    expect(graph.graphList.map(item => item.name)).toEqual(['拓扑图 1'])

    workspaceServiceMock.ReadWorkspace.mockResolvedValueOnce(null as any)
    await expect(files.openFromPath('/tmp/empty.portweave.json')).rejects.toThrow('config file is empty')
    expect(files.lastError).toBe('config file is empty')

    workspaceServiceMock.RememberLastWorkspace.mockRejectedValueOnce(new Error('non-critical remember failure'))
    await expect(files.openFromPath('/tmp/remember-fail.portweave.json')).resolves.toEqual(expect.objectContaining({
      reused: false,
      activeGraphId: expect.any(String),
    }))
  })

  it('applies saved serial settings and reports unknown demos', async () => {
    const settings = useSettingsStore()
    const files = useWorkspaceFileStore()
    files.markClean('/tmp/settings.portweave.json', JSON.parse(graphFile('/tmp/settings.portweave.json', '设置拓扑', { TerminalFontSize: 22 })), 'graph-1')
    files.states['graph-1'].currentSnapshot = '{bad json'
    settings.updateSerial({ TerminalFontSize: 11 })

    expect(files.applyGraphSerialSettings('graph-1')).toBe(true)
    expect(settings.serial.TerminalFontSize).toBe(22)
    expect(files.applyGraphSerialSettings('missing')).toBe(false)

    await expect(files.loadDemo('missing-demo')).rejects.toThrow('Unknown demo workspace: missing-demo')
    expect(files.lastError).toBe('Unknown demo workspace: missing-demo')
  })

  it('loads the last workspace directly and aborts stale startup restores', async () => {
    const files = useWorkspaceFileStore()

    await expect(files.loadLast()).resolves.toBe(true)
    expect(files.currentPath).toBe('/tmp/last.portweave.json')

    localStorage.clear()
    setActivePinia(createPinia())
    const racedFiles = useWorkspaceFileStore()
    const releaseLoad = deferred<{ Found: boolean; Path: string; Content: string }>()
    workspaceServiceMock.LoadLastWorkspace.mockImplementationOnce(() => releaseLoad.promise)

    const staleRestore = racedFiles.loadLast()
    await racedFiles.loadDemo('bridge-demo')
    releaseLoad.resolve({ Found: true, Path: '/tmp/stale-last.portweave.json', Content: graphFile('/tmp/stale-last.portweave.json', '过期拓扑') })

    await expect(staleRestore).resolves.toBe(false)
    expect(racedFiles.currentPath).toBe('')
  })

  it('keeps empty-workspace file helpers as no-ops when no graph is active', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    await graph.removeGraph('graph-1')
    files.markClean('/tmp/ignored.portweave.json', { ignored: true })
    files.updateCurrentSnapshot({ ignored: true })
    files.markDirty()
    files.setPath('/tmp/ignored.portweave.json')

    expect(graph.graphList).toEqual([])
    expect(files.currentPath).toBe('')
    expect(files.sourceKind).toBe('empty')
    expect(files.activeState).toBeNull()
    expect(files.isDirty).toBe(false)
    expect(files.graphPath('missing')).toBe('')
    expect(files.graphTooltip('missing')).toBe('未保存')
  })

  it('continues startup restore after recent-file failures and keeps dirty graphs while loading demos', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify(['/tmp/broken.portweave.json', '/tmp/good.portweave.json']))
    workspaceServiceMock.ReadWorkspace.mockRejectedValueOnce(new Error('broken recent file'))
    workspaceServiceMock.readFiles.set('/tmp/good.portweave.json', graphFile('/tmp/good.portweave.json', '恢复拓扑'))
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    await expect(files.loadLast()).resolves.toBe(true)

    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/broken.portweave.json')
    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/good.portweave.json')
    expect(files.currentPath).toBe('/tmp/good.portweave.json')
    expect(graph.graphList.map(item => item.name)).toContain('恢复拓扑')

    graph.setActiveGraph('graph-1')
    files.syncGraphSnapshot('graph-1')
    graph.addNode('serial.sender')
    files.syncGraphSnapshot('graph-1')
    expect(files.isGraphDirty('graph-1')).toBe(true)

    const demo = await files.loadDemo('bridge-demo')

    expect(graph.graphList.map(item => item.id)).toContain('graph-1')
    expect(graph.graphList.map(item => item.id)).toContain(demo.activeGraphId)
  })

  it('tracks empty paths custom titles and non-graph snapshots through helper fallbacks', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    const settings = useSettingsStore()

    files.markClean('', { kind: 'empty-snapshot' }, 'graph-1')
    expect(files.currentPath).toBe('')
    expect(files.sourceKind).toBe('empty')

    files.setPath('', 'graph-1')
    expect(files.graphState('graph-1')).toEqual(expect.objectContaining({
      path: '',
      sourceKind: 'empty',
    }))

    files.states['graph-1'].title = '自定义标题'
    files.setPath('/tmp/titled.portweave.json', 'graph-1')
    expect(files.graphTitle('graph-1', 'Fallback')).toBe('自定义标题')

    files.updateCurrentSnapshot({ kind: 'not-a-graph-tab' }, 'graph-1')
    settings.updateSerial({ TerminalFontSize: 17 })
    expect(files.applyGraphSerialSettings('graph-1')).toBe(false)
    expect(settings.serial.TerminalFontSize).toBe(17)

    await graph.removeGraph('graph-1')
    expect(files.applyGraphSerialSettings()).toBe(false)
  })

  it('uses fallback error messages when workspace services reject non-error values', async () => {
    const files = useWorkspaceFileStore()

    workspaceServiceMock.SaveWorkspace.mockRejectedValueOnce('save rejected')
    await expect(files.save()).rejects.toBe('save rejected')
    expect(files.lastError).toBe('Save config failed')

    workspaceServiceMock.SelectWorkspaceOpenPath.mockRejectedValueOnce('open picker rejected')
    await expect(files.selectOpenPath()).rejects.toBe('open picker rejected')
    expect(files.lastError).toBe('Select config file failed')

    workspaceServiceMock.SelectWorkspaceSavePath.mockRejectedValueOnce('save picker rejected')
    await expect(files.saveAs()).rejects.toBe('save picker rejected')
    expect(files.lastError).toBe('Save config failed')

    workspaceServiceMock.ReadWorkspace.mockRejectedValueOnce('open rejected')
    await expect(files.openFromPath('/tmp/open-error.portweave.json')).rejects.toBe('open rejected')
    expect(files.lastError).toBe('Open config failed')

    workspaceServiceMock.SelectWorkspaceOpenPath.mockRejectedValueOnce('import picker rejected')
    await expect(files.importSelected()).rejects.toBe('import picker rejected')
    expect(files.lastError).toBe('Open config failed')

    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('/tmp/export-error.portweave.json')
    workspaceServiceMock.ExportWorkspace.mockRejectedValueOnce('export rejected')
    await expect(files.exportCopy()).rejects.toBe('export rejected')
    expect(files.lastError).toBe('Export config failed')

    workspaceServiceMock.LoadLastWorkspace.mockRejectedValueOnce('load rejected')
    await expect(files.loadLast()).rejects.toBe('load rejected')
    expect(files.lastError).toBe('Load last config failed')
  })

  it('treats blank clean graph states and object-shaped recent storage as fallback startup cases', async () => {
    localStorage.setItem('portweave.open-config-files.v1', JSON.stringify({ path: '/tmp/not-a-list.portweave.json' }))
    workspaceServiceMock.LoadLastWorkspace.mockResolvedValueOnce({ Found: false, Path: '', Content: '' })
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    await expect(files.loadLast()).resolves.toBe(false)
    expect(workspaceServiceMock.LoadLastWorkspace).toHaveBeenCalled()

    files.setPath('/tmp/blank-empty.portweave.json', 'graph-1')
    const demo = await files.loadDemo('bridge-demo')

    expect(graph.graphList.map(item => item.id)).toEqual([demo.activeGraphId])
  })

  it('keeps clean empty graph states when their serialized snapshot cannot be parsed', async () => {
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()
    files.setPath('/tmp/bad-empty.portweave.json', 'graph-1')
    files.states['graph-1'].savedSnapshot = '{bad json'
    files.states['graph-1'].currentSnapshot = '{bad json'

    const demo = await files.loadDemo('bridge-demo')

    expect(graph.graphList.map(item => item.id)).toContain('graph-1')
    expect(graph.graphList.map(item => item.id)).toContain(demo.activeGraphId)
  })

  it('opens service results with blank paths without remembering an empty recent file', async () => {
    workspaceServiceMock.ReadWorkspace.mockResolvedValueOnce({
      Path: '',
      Content: graphFile('/tmp/blank-service-path.portweave.json', '空路径拓扑'),
    })
    const graph = useSerialGraphStore()
    const files = useWorkspaceFileStore()

    const result = await files.openFromPath('/tmp/blank-service-path.portweave.json')

    expect(result.graphIds).toHaveLength(1)
    expect(files.currentPath).toBe('')
    expect(graph.graphList.map(item => item.name)).toContain('空路径拓扑')
    expect(workspaceServiceMock.RememberLastWorkspace).not.toHaveBeenCalled()
  })
})

function graphFile(path: string, name: string, serialSettings: Partial<typeof defaultSerialSettings> = {}) {
  return JSON.stringify({
    kind: 'portweave.graph.v1',
    settings: { serial: { ...defaultSerialSettings, ...serialSettings } },
    graph: {
      id: path.includes('second') ? 'second-graph' : 'first-graph',
      name,
      nodes: [],
      edges: [],
      selectedNodeId: null,
      selectedEdgeId: null,
      nodeTabs: [],
      activeNodeTabId: null,
    },
    runtime: { nodeBuffers: {}, nodeFrames: {} },
  })
}

function workspaceFileWithoutGraph() {
  return JSON.stringify({
    kind: 'portweave.workspace.v1',
    settings: { serial: defaultSerialSettings },
    serial: {
      activePortId: '',
      handles: [],
      virtualPorts: [],
      bridges: [],
      buffers: {},
      monitors: undefined,
      modbus: undefined,
      fecbus: undefined,
      workspace: {
        selectedOperation: null,
        editorLayout: { type: 'group', id: 'group-empty', tabs: [] },
        activeByGroup: { 'group-empty': null },
        tabStates: {},
      },
    },
  })
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}
