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
      kind: 'mocktrue.graph.v1',
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

vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/core/workspace/service.js', () => workspaceServiceMock)
vi.mock('../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/service.js', () => serialBindingsMock)

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
      expect.stringContaining('mocktrue.graph.v1')
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
      expect.stringContaining('mocktrue.graph.v1')
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
    localStorage.setItem('mocktrue.open-config-files.v1', JSON.stringify(['/tmp/startup.portweave.json']))
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
    localStorage.setItem('mocktrue.open-config-files.v1', JSON.stringify(['/tmp/empty-workspace.portweave.json']))
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
    files.markClean('/tmp/current.portweave.json', { kind: 'mocktrue.graph.v1', graph: 'one' }, 'graph-1')
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
    localStorage.setItem('mocktrue.open-config-files.v1', JSON.stringify(['/tmp/first.portweave.json', '/tmp/second.portweave.json']))
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
})

function graphFile(path: string, name: string, serialSettings: Partial<typeof defaultSerialSettings> = {}) {
  return JSON.stringify({
    kind: 'mocktrue.graph.v1',
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
    kind: 'mocktrue.workspace.v1',
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
