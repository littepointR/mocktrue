import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  CreateVirtualPair,
  DeleteVirtualPair,
  ListVirtualPairs,
  CreateBridge,
  DeleteBridge,
  ListBridges,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'

export interface VirtualPair {
  ID: string
  Port1: string
  Port2: string
}

export interface Bridge {
  ID: string
  Port1: string
  Port2: string
  BaudRate: number
}

export const useVirtualStore = defineStore('virtual', () => {
  // State
  const pairs = ref<VirtualPair[]>([])
  const bridges = ref<Bridge[]>([])
  const error = ref<string | null>(null)

  // Actions
  async function refreshPairs() {
    try {
      const list = await ListVirtualPairs()
      pairs.value = (list as VirtualPair[]) ?? []
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to list virtual pairs'
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

  async function createPair(id: string, port1Name: string, port2Name: string) {
    try {
      await CreateVirtualPair(id, port1Name, port2Name)
      await refreshPairs()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to create pair'
      throw e
    }
  }

  async function deletePair(id: string) {
    try {
      await DeleteVirtualPair(id)
      await refreshPairs()
      error.value = null
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to delete pair'
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

  function clearError() {
    error.value = null
  }

  return {
    pairs,
    bridges,
    error,
    refreshPairs,
    refreshBridges,
    createPair,
    deletePair,
    createBridge,
    deleteBridge,
    clearError,
  }
})
