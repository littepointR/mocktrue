// Frontend module system types. Mirrors the Go side's module.FrontendContribution
// conceptually, but icons are owned by the frontend module (the Go manifest
// only carries ids/titles).

/** A module's contribution to the activity bar. */
export interface ActivityContribution {
  /** Icon key; the shell maps it to a rendered icon. */
  icon: string
  /** Hover title. */
  title: string
}

/** A single view a module contributes to the sidebar/editor area. */
export interface ViewContribution {
  id: string
  title: string
  /** Frontend component path/name for lazy loading (stage 0: not yet wired). */
  component: string
}

/** A module's full frontend contribution, as emitted by the Go shell. */
export interface ModuleContribution {
  moduleId: string
  activity: ActivityContribution
  views: ViewContribution[]
}

/** A frontend module. Implementations register via the module registry. */
export interface ModuleFrontend {
  readonly id: string
  readonly activity: ActivityContribution
  readonly views: ViewContribution[]
  /** Called when the module becomes active; optional. */
  onActivate?(): void
  /** Called when the module is deactivated; optional. */
  onDeactivate?(): void
}
