import { describe, expect, it } from 'vitest'
import sidebarSource from './Sidebar.vue?raw'


describe('Sidebar styles', () => {
  it('uses one font size for sidebar title and view items', () => {
    expect(sidebarSource).toContain('--sidebar-font-size: 13px;')
    expect(rule('.sidebar__title')).toContain('font-size: var(--sidebar-font-size);')
    expect(rule('.sidebar__item')).toContain('font-size: var(--sidebar-font-size);')
  })

  it('does not change item font size for active or hover states', () => {
    expect(rule('.sidebar__item:hover,\n.sidebar__item.is-active')).not.toContain('font-size')
  })
})

function rule(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = sidebarSource.match(new RegExp(`${escaped}\\s*\\{([^}]*)\\}`))
  return match?.[1] ?? ''
}
