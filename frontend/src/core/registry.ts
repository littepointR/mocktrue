import { computed, ref } from 'vue'
import type { ModuleContribution, ModuleFrontend } from './module/types'

/**
 * ModuleRegistry tracks registered frontend modules and the active module.
 * State is reactive (Vue refs) so the shell can render activity bar etc.
 */
class ModuleRegistry {
  private modules = new Map<string, ModuleFrontend>()
  private activeId = ref<string | null>(null)

  /** Register a frontend module. Throws on empty or duplicate id. */
  register(m: ModuleFrontend): void {
    if (m.id === '') throw new Error('module id must not be empty')
    if (this.modules.has(m.id)) throw new Error(`module already registered: ${m.id}`)
    this.modules.set(m.id, m)
  }

  /** List all registered modules as contributions (snapshot). */
  list(): ModuleContribution[] {
    return Array.from(this.modules.values()).map((m) => ({
      moduleId: m.id,
      activity: m.activity,
      views: m.views,
    }))
  }

  /** The active module id (reactive). */
  readonly active = computed(() => this.activeId.value)

  /** Get the currently active module frontend, or null. */
  activeModule(): ModuleFrontend | null {
    return this.activeId.value ? this.modules.get(this.activeId.value) ?? null : null
  }

  /** Activate a module by id, or pass null to deactivate. Throws on unknown id. */
  setActive(id: string | null): void {
    if (id !== null && !this.modules.has(id)) throw new Error(`unknown module: ${id}`)
    const prev = this.activeId.value ? this.modules.get(this.activeId.value) ?? null : null
    prev?.onDeactivate?.()
    this.activeId.value = id
    if (id) this.modules.get(id)?.onActivate?.()
  }

  /**
   * Merge backend-emitted contributions. Stage 0: icons are owned by the
   * frontend module, so this only validates/records; it tolerates null and
   * missing fields without throwing.
   */
  mergeBackendContributions(items: unknown): void {
    if (!Array.isArray(items)) return
    for (const item of items) {
      if (!item || typeof item !== 'object') continue
      const rec = item as { ModuleID?: string }
      if (!rec.ModuleID || !this.modules.has(rec.ModuleID)) continue
      // Stage 0: no-op beyond validation. Later stages may augment views.
    }
  }
}

let _registry: ModuleRegistry | null = null

/** Get the singleton module registry. */
export function useRegistry(): ModuleRegistry {
  if (!_registry) _registry = new ModuleRegistry()
  return _registry
}

/** Reset the singleton registry (tests only). */
export function __resetRegistryForTest(): void {
  _registry = null
}
