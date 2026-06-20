import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ResizableTable, { type ResizableTableColumn } from './ResizableTable.vue'

const columns: ResizableTableColumn[] = [
  { key: 'seq', label: '#', width: 50, minWidth: 40 },
  { key: 'data', label: '数据', width: 120, minWidth: 60 },
]

function mountTable() {
  return mount(ResizableTable, {
    props: { columns },
    slots: {
      default: '<tr><td>1</td><td>abcdef</td></tr>',
    },
    attachTo: document.body,
  })
}

describe('ResizableTable', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('resizes a column by dragging its header splitter', async () => {
    const wrapper = mountTable()
    const firstHeader = wrapper.find('th')
    vi.spyOn(firstHeader.element, 'getBoundingClientRect').mockReturnValue({
      width: 50,
      height: 20,
      top: 0,
      left: 0,
      right: 50,
      bottom: 20,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect)

    dispatchPointer(wrapper.find('[data-testid="resize-handle-seq"]').element, 'pointerdown', 50)
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 90 }))
    window.dispatchEvent(new MouseEvent('pointerup'))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('col').attributes('style')).toContain('width: 90px')
  })

  it('keeps a dragged column above its minimum width', async () => {
    const wrapper = mountTable()
    const firstHeader = wrapper.find('th')
    vi.spyOn(firstHeader.element, 'getBoundingClientRect').mockReturnValue({
      width: 50,
      height: 20,
      top: 0,
      left: 0,
      right: 50,
      bottom: 20,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect)

    dispatchPointer(wrapper.find('[data-testid="resize-handle-seq"]').element, 'pointerdown', 50)
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 10 }))
    window.dispatchEvent(new MouseEvent('pointerup'))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('col').attributes('style')).toContain('width: 40px')
  })

  it('auto fits a column when the header splitter is double clicked', async () => {
    const wrapper = mountTable()
    const dataCells = wrapper.findAll('th, td').filter(node => (node.element as HTMLTableCellElement).cellIndex === 1)
    for (const [index, cell] of dataCells.entries()) {
      Object.defineProperty(cell.element, 'scrollWidth', {
        configurable: true,
        value: index === 0 ? 42 : 180,
      })
    }

    await wrapper.find('[data-testid="resize-handle-data"]').trigger('dblclick')

    expect(wrapper.findAll('col')[1].attributes('style')).toContain('width: 196px')
  })
})

function dispatchPointer(target: Element, type: string, clientX: number) {
  const event = new MouseEvent(type, { bubbles: true, clientX })
  Object.defineProperty(event, 'pointerId', { value: 1 })
  target.dispatchEvent(event)
}
