import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useVirtualStore } from './virtualStore'

const bindings = vi.hoisted(() => ({
  CreateVirtualPort: vi.fn(),
  DeleteVirtualPort: vi.fn(),
  ListVirtualPorts: vi.fn(),
  CreateBridge: vi.fn(),
  DeleteBridge: vi.fn(),
  ListBridges: vi.fn(),
  CleanupVirtual: vi.fn(),
}))

vi.mock('../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js', () => bindings)

describe('virtual store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    bindings.ListVirtualPorts.mockResolvedValue([])
    bindings.ListBridges.mockResolvedValue([])
    bindings.CreateVirtualPort.mockResolvedValue(undefined)
    bindings.DeleteVirtualPort.mockResolvedValue(undefined)
    bindings.CreateBridge.mockResolvedValue(undefined)
    bindings.DeleteBridge.mockResolvedValue(undefined)
    bindings.CleanupVirtual.mockResolvedValue(undefined)
  })

  it('refreshes virtual ports and bridges, coercing null lists to empty arrays', async () => {
    bindings.ListVirtualPorts.mockResolvedValueOnce(null)
    bindings.ListBridges.mockResolvedValueOnce(null)
    const store = useVirtualStore()
    store.error = 'stale error'

    await store.refreshVirtualPorts()
    await store.refreshBridges()

    expect(store.virtualPorts).toEqual([])
    expect(store.bridges).toEqual([])
    expect(store.error).toBeNull()
  })

  it('creates and deletes virtual ports while refreshing cached state', async () => {
    bindings.ListVirtualPorts
      .mockResolvedValueOnce([{ ID: 'vp-1', Port: '/tmp/mock-vp-1' }])
      .mockResolvedValueOnce([])
    const store = useVirtualStore()

    await store.createVirtualPort('vp-1', '/tmp/mock-vp-1')
    expect(bindings.CreateVirtualPort).toHaveBeenCalledWith('vp-1', '/tmp/mock-vp-1')
    expect(store.virtualPorts).toEqual([{ ID: 'vp-1', Port: '/tmp/mock-vp-1' }])

    await store.deleteVirtualPort('vp-1')
    expect(bindings.DeleteVirtualPort).toHaveBeenCalledWith('vp-1')
    expect(store.virtualPorts).toEqual([])
  })

  it('creates and deletes bridges while refreshing cached state', async () => {
    const bridge = { ID: 'bridge-1', Port1: '/tmp/a', Port2: '/tmp/b', BaudRate: 115200 }
    bindings.ListBridges
      .mockResolvedValueOnce([bridge])
      .mockResolvedValueOnce([])
    const store = useVirtualStore()

    await store.createBridge('bridge-1', '/tmp/a', '/tmp/b', 115200)
    expect(bindings.CreateBridge).toHaveBeenCalledWith('bridge-1', '/tmp/a', '/tmp/b', 115200)
    expect(store.bridges).toEqual([bridge])

    await store.deleteBridge('bridge-1')
    expect(bindings.DeleteBridge).toHaveBeenCalledWith('bridge-1')
    expect(store.bridges).toEqual([])
  })

  it('records refresh errors without throwing and can clear them', async () => {
    bindings.ListVirtualPorts.mockRejectedValueOnce(undefined)
    const store = useVirtualStore()

    await store.refreshVirtualPorts()
    expect(store.error).toBe('Failed to list virtual ports')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('rethrows create bridge errors and stores the backend message', async () => {
    bindings.CreateBridge.mockRejectedValueOnce(new Error('bridge already exists'))
    const store = useVirtualStore()

    await expect(store.createBridge('bridge-1', '/tmp/a', '/tmp/b', 9600)).rejects.toThrow('bridge already exists')

    expect(store.error).toBe('bridge already exists')
    expect(bindings.ListBridges).not.toHaveBeenCalled()
  })

  it('rethrows failed virtual port and bridge deletes without refreshing stale lists', async () => {
    bindings.CreateVirtualPort.mockRejectedValueOnce(undefined)
    bindings.DeleteVirtualPort.mockRejectedValueOnce(new Error('virtual port busy'))
    bindings.DeleteBridge.mockRejectedValueOnce(undefined)
    const store = useVirtualStore()

    await expect(store.createVirtualPort('vp-1', '/tmp/mock-vp-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to create virtual port')
    expect(bindings.ListVirtualPorts).not.toHaveBeenCalled()

    await expect(store.deleteVirtualPort('vp-1')).rejects.toThrow('virtual port busy')
    expect(store.error).toBe('virtual port busy')
    expect(bindings.ListVirtualPorts).not.toHaveBeenCalled()

    await expect(store.deleteBridge('bridge-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to delete bridge')
    expect(bindings.ListBridges).not.toHaveBeenCalled()
  })

  it('cleans up all resources and leaves state unchanged when cleanup fails', async () => {
    const store = useVirtualStore()
    store.virtualPorts = [{ ID: 'vp-1', Port: '/tmp/mock-vp-1' }]
    store.bridges = [{ ID: 'bridge-1', Port1: '/tmp/a', Port2: '/tmp/b', BaudRate: 115200 }]

    await store.cleanupAllResources()

    expect(bindings.CleanupVirtual).toHaveBeenCalledTimes(1)
    expect(store.virtualPorts).toEqual([])
    expect(store.bridges).toEqual([])
    expect(store.error).toBeNull()

    store.virtualPorts = [{ ID: 'vp-2', Port: '/tmp/mock-vp-2' }]
    bindings.CleanupVirtual.mockRejectedValueOnce(undefined)

    await expect(store.cleanupAllResources()).rejects.toBeUndefined()

    expect(store.virtualPorts).toEqual([{ ID: 'vp-2', Port: '/tmp/mock-vp-2' }])
    expect(store.error).toBe('Failed to cleanup virtual resources')
  })
})
