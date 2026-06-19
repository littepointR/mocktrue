import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWorkspaceFileStore } from './workspaceFileStore'
import { useBufferStore } from '../../serial/stores/bufferStore'
import { useSerialStore } from '../../serial/stores/serialStore'

const workspaceServiceMock = vi.hoisted(() => ({
  DefaultWorkspacePath: vi.fn(async () => '/tmp/default.mocktrue.json'),
  ExportWorkspace: vi.fn(async () => undefined),
  SaveWorkspace: vi.fn(async () => undefined),
  ReadWorkspace: vi.fn(async () => ({ Path: '/tmp/import.mocktrue.json', Content: '{"kind":"mocktrue.workspace.v1","value":3}' })),
  LoadLastWorkspace: vi.fn(async () => ({ Found: true, Path: '/tmp/last.mocktrue.json', Content: '{"kind":"mocktrue.workspace.v1","value":4}' })),
  RememberLastWorkspace: vi.fn(async () => undefined),
  SelectWorkspaceOpenPath: vi.fn(async () => '/tmp/selected-open.mocktrue.json'),
  SelectWorkspaceSavePath: vi.fn(async () => '/tmp/selected-save.mocktrue.json'),
}))

const sessionMock = vi.hoisted(() => ({
  buildWorkspaceSnapshot: vi.fn(() => ({ kind: 'mocktrue.workspace.v1', value: 2 })),
  restoreWorkspaceSnapshot: vi.fn(async () => ({ errors: [], handleMap: {} })),
}))

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/core/workspace/service.js', () => workspaceServiceMock)
vi.mock('../workspaceSession', () => sessionMock)
vi.mock('../../serial/services/serialEvents', () => ({
  serialEvents: {
    onData: vi.fn(() => vi.fn()),
  },
}))
vi.mock('../../serial/services/serialService', () => ({
  serialService: {
    closePort: vi.fn(async () => undefined),
    enumeratePorts: vi.fn(async () => []),
    listPorts: vi.fn(async () => []),
    openPort: vi.fn(async () => {
      throw new Error('not used in workspace file store tests')
    }),
    resetCounters: vi.fn(async () => undefined),
    restoreCounters: vi.fn(async () => undefined),
  },
}))

describe('workspaceFileStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('tracks dirty state against the last saved snapshot', () => {
    const store = useWorkspaceFileStore()
    const saved = { kind: 'mocktrue.workspace.v1', value: 1 }

    store.markClean('/tmp/session.mocktrue.json', saved)
    expect(store.isDirty).toBe(false)
    expect(store.currentPath).toBe('/tmp/session.mocktrue.json')

    store.updateCurrentSnapshot({ kind: 'mocktrue.workspace.v1', value: 2 })
    expect(store.isDirty).toBe(true)

    store.markClean('/tmp/session.mocktrue.json', { kind: 'mocktrue.workspace.v1', value: 2 })
    expect(store.isDirty).toBe(false)
  })

  it('saves current snapshot to the default path when no file is active', async () => {
    const store = useWorkspaceFileStore()

    await store.save()

    expect(workspaceServiceMock.DefaultWorkspacePath).toHaveBeenCalled()
    expect(workspaceServiceMock.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/default.mocktrue.json',
      '{"kind":"mocktrue.workspace.v1","value":2}'
    )
    expect(store.currentPath).toBe('/tmp/default.mocktrue.json')
    expect(store.isDirty).toBe(false)
  })

  it('selects a workspace path for the path input', async () => {
    const store = useWorkspaceFileStore()
    store.setPath('/tmp/current.mocktrue.json')

    const selected = await store.selectOpenPath()

    expect(workspaceServiceMock.SelectWorkspaceOpenPath).toHaveBeenCalledWith('/tmp/current.mocktrue.json')
    expect(selected).toBe('/tmp/selected-open.mocktrue.json')
    expect(store.currentPath).toBe('/tmp/selected-open.mocktrue.json')
  })

  it('saves as a selected file and marks that file clean', async () => {
    const store = useWorkspaceFileStore()
    store.setPath('/tmp/current.mocktrue.json')

    const savedPath = await store.saveAs()

    expect(workspaceServiceMock.SelectWorkspaceSavePath).toHaveBeenCalledWith('/tmp/current.mocktrue.json')
    expect(workspaceServiceMock.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/selected-save.mocktrue.json',
      '{"kind":"mocktrue.workspace.v1","value":2}'
    )
    expect(savedPath).toBe('/tmp/selected-save.mocktrue.json')
    expect(store.currentPath).toBe('/tmp/selected-save.mocktrue.json')
    expect(store.isDirty).toBe(false)
  })

  it('does nothing when save-as path selection is cancelled', async () => {
    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('')
    const store = useWorkspaceFileStore()
    store.markDirty()

    const savedPath = await store.saveAs()

    expect(savedPath).toBe('')
    expect(workspaceServiceMock.SaveWorkspace).not.toHaveBeenCalled()
    expect(store.isDirty).toBe(true)
  })

  it('imports a workspace file and marks the imported snapshot clean', async () => {
    const store = useWorkspaceFileStore()

    await store.importFromPath('/tmp/import.mocktrue.json')

    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/import.mocktrue.json')
    expect(workspaceServiceMock.RememberLastWorkspace).toHaveBeenCalledWith('/tmp/import.mocktrue.json')
    expect(sessionMock.restoreWorkspaceSnapshot).toHaveBeenCalledWith({ kind: 'mocktrue.workspace.v1', value: 3 })
    expect(store.currentPath).toBe('/tmp/import.mocktrue.json')
    expect(store.isDirty).toBe(false)
  })

  it('imports from a selected path when no path is configured', async () => {
    const store = useWorkspaceFileStore()

    await store.importSelected()

    expect(workspaceServiceMock.SelectWorkspaceOpenPath).toHaveBeenCalledWith('')
    expect(workspaceServiceMock.ReadWorkspace).toHaveBeenCalledWith('/tmp/selected-open.mocktrue.json')
    expect(store.currentPath).toBe('/tmp/import.mocktrue.json')
    expect(store.isDirty).toBe(false)
  })

  it('does nothing when import path selection is cancelled', async () => {
    workspaceServiceMock.SelectWorkspaceOpenPath.mockResolvedValueOnce('')
    const store = useWorkspaceFileStore()
    store.markDirty()

    const result = await store.importSelected()

    expect(result).toBeNull()
    expect(workspaceServiceMock.ReadWorkspace).not.toHaveBeenCalled()
    expect(store.isDirty).toBe(true)
  })

  it('exports a copy without changing the active workspace or clean baseline', async () => {
    const store = useWorkspaceFileStore()
    store.markClean('/tmp/current.mocktrue.json', { kind: 'mocktrue.workspace.v1', value: 1 })
    store.updateCurrentSnapshot({ kind: 'mocktrue.workspace.v1', value: 2 })

    const exportedPath = await store.exportCopy()

    expect(workspaceServiceMock.SelectWorkspaceSavePath).toHaveBeenCalledWith('/tmp/current.mocktrue.json')
    expect(workspaceServiceMock.ExportWorkspace).toHaveBeenCalledWith(
      '/tmp/selected-save.mocktrue.json',
      '{"kind":"mocktrue.workspace.v1","value":2}'
    )
    expect(exportedPath).toBe('/tmp/selected-save.mocktrue.json')
    expect(store.currentPath).toBe('/tmp/current.mocktrue.json')
    expect(store.isDirty).toBe(true)
  })

  it('does nothing when export-copy path selection is cancelled', async () => {
    workspaceServiceMock.SelectWorkspaceSavePath.mockResolvedValueOnce('')
    const store = useWorkspaceFileStore()
    store.markDirty()

    const exportedPath = await store.exportCopy()

    expect(exportedPath).toBe('')
    expect(workspaceServiceMock.ExportWorkspace).not.toHaveBeenCalled()
    expect(store.isDirty).toBe(true)
  })

  it('auto-loads the last workspace when it is available', async () => {
    const store = useWorkspaceFileStore()

    const loaded = await store.loadLast()

    expect(loaded).toBe(true)
    expect(sessionMock.restoreWorkspaceSnapshot).toHaveBeenCalledWith({ kind: 'mocktrue.workspace.v1', value: 4 })
    expect(store.currentPath).toBe('/tmp/last.mocktrue.json')
    expect(store.isDirty).toBe(false)
  })

  it('loads a readonly demo workspace and exposes it as the displayed config source', async () => {
    const store = useWorkspaceFileStore()

    const result = await store.loadDemo('monitor-demo')

    expect(result).toEqual({ errors: [], handleMap: {} })
    expect(sessionMock.restoreWorkspaceSnapshot).toHaveBeenCalledWith(expect.objectContaining({
      kind: 'mocktrue.workspace.v1',
      serial: expect.objectContaining({
        monitors: expect.objectContaining({
          sessions: expect.arrayContaining([expect.objectContaining({ ID: expect.stringContaining('demo-monitor') })]),
        }),
      }),
    }))
    expect(store.sourceKind).toBe('demo')
    expect(store.readonly).toBe(true)
    expect(store.currentDemoId).toBe('monitor-demo')
    expect(store.currentPath).toBe('')
    expect(store.displayPath).toContain('Demo:')
    expect(store.isDirty).toBe(false)
  })

  it('adds local demo handles when real demo ports cannot be opened', async () => {
    sessionMock.restoreWorkspaceSnapshot.mockResolvedValueOnce({
      errors: [{ target: 'port:demo-open', message: 'open failed' }],
      handleMap: {},
    } as any)
    const store = useWorkspaceFileStore()

    await store.loadDemo('serial-open-demo')

    const serial = useSerialStore()
    const buffer = useBufferStore()
    const [handle] = serial.openHandles
    expect(handle.ID).toContain('demo-open-handle')
    expect(handle.Config.PortName).toContain('/tmp/mocktrue-demo-open')
    expect(handle.RxBytes).toBeGreaterThan(0)
    expect(buffer.getChunks(handle.ID)).toHaveLength(2)
    expect(serial.activePortId).toBe(handle.ID)
  })

  it('rejects direct save while a readonly demo is active', async () => {
    const store = useWorkspaceFileStore()
    await store.loadDemo('bridge-demo')

    await expect(store.save()).rejects.toThrow('Demo 配置为只读，请使用另存为')

    expect(workspaceServiceMock.SaveWorkspace).not.toHaveBeenCalled()
    expect(store.lastError).toBe('Demo 配置为只读，请使用另存为')
  })

  it('keeps a demo readonly when the path input changes before save-as', async () => {
    const store = useWorkspaceFileStore()
    await store.loadDemo('bridge-demo')

    store.setPath('/tmp/manual-target.mocktrue.json')

    expect(store.currentPath).toBe('/tmp/manual-target.mocktrue.json')
    expect(store.displayPath).toContain('Demo:')
    expect(store.readonly).toBe(true)
    await expect(store.save()).rejects.toThrow('Demo 配置为只读，请使用另存为')
    expect(workspaceServiceMock.SaveWorkspace).not.toHaveBeenCalled()
  })

  it('saves a readonly demo as a normal file and clears the readonly demo source', async () => {
    const store = useWorkspaceFileStore()
    await store.loadDemo('full-workspace-demo')

    const savedPath = await store.saveAs()

    expect(savedPath).toBe('/tmp/selected-save.mocktrue.json')
    expect(workspaceServiceMock.SaveWorkspace).toHaveBeenCalledWith(
      '/tmp/selected-save.mocktrue.json',
      '{"kind":"mocktrue.workspace.v1","value":2}'
    )
    expect(store.sourceKind).toBe('file')
    expect(store.readonly).toBe(false)
    expect(store.currentDemoId).toBe('')
    expect(store.currentPath).toBe('/tmp/selected-save.mocktrue.json')
    expect(store.displayPath).toBe('/tmp/selected-save.mocktrue.json')
  })

  it('imports a file as an editable file source after a demo has been loaded', async () => {
    const store = useWorkspaceFileStore()
    await store.loadDemo('virtual-port-demo')

    await store.importFromPath('/tmp/import.mocktrue.json')

    expect(store.sourceKind).toBe('file')
    expect(store.readonly).toBe(false)
    expect(store.currentDemoId).toBe('')
    expect(store.currentPath).toBe('/tmp/import.mocktrue.json')
    expect(store.displayPath).toBe('/tmp/import.mocktrue.json')
  })
})
