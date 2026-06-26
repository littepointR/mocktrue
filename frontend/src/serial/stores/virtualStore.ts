import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  CreateVirtualPort,
  DeleteVirtualPort,
  ListVirtualPorts,
  CreateBridge,
  DeleteBridge,
  ListBridges,
  CleanupVirtual,
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/service.js'

export interface VirtualPort {
  ID: string
  Port: string
}

export interface Bridge {
  ID: string
  Port1: string
  Port2: string
  BaudRate: number
}

export const useVirtualStore = defineStore('virtual', () => {
  // State
  const virtualPorts = ref<VirtualPort[]>([])
  const bridges = ref<Bridge[]>([])
  const error = ref<string | null>(null)

  // Actions
  async function refreshVirtualPorts() {
    try {
      const list = await ListVirtualPorts()
      virtualPorts.value = (list as VirtualPort[]) ?? []
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to list virtual ports'
    }
  }

  async function refreshBridges() {
    try {
      const list = await ListBridges()
      bridges.value = (list as Bridge[]) ?? []
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to list bridges'
    }
  }

  async function createVirtualPort(id: string, portName: string) {
    try {
      await CreateVirtualPort(id, portName)
      await refreshVirtualPorts()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to create virtual port'
      throw e
    }
  }

  async function deleteVirtualPort(id: string) {
    try {
      await DeleteVirtualPort(id)
      await refreshVirtualPorts()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to delete virtual port'
      throw e
    }
  }

  async function createBridge(id: string, port1: string, port2: string, baudRate: number) {
    try {
      await CreateBridge(id, port1, port2, baudRate)
      await refreshBridges()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to create bridge'
      throw e
    }
  }

  async function deleteBridge(id: string) {
    try {
      await DeleteBridge(id)
      await refreshBridges()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to delete bridge'
      throw e
    }
  }

  async function cleanupAllResources() {
    try {
      await CleanupVirtual()
      virtualPorts.value = []
      bridges.value = []
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to cleanup virtual resources'
      throw e
    }
  }

  function clearError() {
    error.value = null
  }

  return {
    virtualPorts,
    bridges,
    error,
    refreshVirtualPorts,
    refreshBridges,
    createVirtualPort,
    deleteVirtualPort,
    createBridge,
    deleteBridge,
    cleanupAllResources,
    clearError,
  }
})
