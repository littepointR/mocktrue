import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ActivityBar from './ActivityBar.vue'
import type { ModuleContribution } from '../core/module/types'

const contributions: ModuleContribution[] = [
  {
    moduleId: 'serial',
    activity: { icon: 'serial', title: '串口调试' },
    views: [],
  },
  {
    moduleId: 'settings',
    activity: { icon: 'settings', title: '设置' },
    views: [],
  },
]

describe('ActivityBar', () => {
  it('pins the settings activity to the bottom', () => {
    const wrapper = mount(ActivityBar, {
      props: {
        contributions,
        activeId: 'settings',
      },
    })

    const buttons = wrapper.findAll('.activity-bar__item')
    expect(buttons).toHaveLength(2)
    expect(buttons[1].attributes('title')).toBe('设置')
    expect(buttons[1].classes()).toContain('activity-bar__item--bottom')
    expect(buttons[1].classes()).toContain('is-active')
    expect(buttons[1].text()).toContain('⚙')
  })

  it('emits the selected module id when an activity is clicked', async () => {
    const selected: string[] = []
    const wrapper = mount({
      components: { ActivityBar },
      setup: () => ({ contributions, activeId: null, selected }),
      template: '<ActivityBar :contributions="contributions" :active-id="activeId" @select="selected.push($event)" />',
    })

    await wrapper.findAll('.activity-bar__item')[0].trigger('click')

    expect(selected).toEqual(['serial'])
  })
})
