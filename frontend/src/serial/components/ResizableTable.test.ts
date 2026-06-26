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

  it('renders default widths and optional table styling for columns without explicit sizing', () => {
    const wrapper = mount(ResizableTable, {
      props: {
        columns: [{ key: 'payload', label: '', class: 'payload-column' }],
        tableClass: 'custom-table',
        minWidth: 320,
      },
      slots: { default: '<tr><td>abc</td></tr>' },
    })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['resizable-table', 'custom-table']))
    expect(wrapper.attributes('style')).toContain('min-width: 320px')
    expect(wrapper.find('col').classes()).toContain('payload-column')
    expect(wrapper.find('col').attributes('style')).toContain('width: 80px')
    expect(wrapper.find('col').attributes('style')).toContain('min-width: 48px')
    expect(wrapper.find('[data-testid="resize-handle-payload"]').attributes('aria-label')).toBe('调整列列宽')
  })

  it('restores document resize styles when unmounted during an active drag', () => {
    document.body.style.cursor = 'default'
    document.body.style.userSelect = 'text'
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
    expect(document.body.style.cursor).toBe('col-resize')

    wrapper.unmount()

    expect(document.body.style.cursor).toBe('default')
    expect(document.body.style.userSelect).toBe('text')
  })

  it('uses column minimum widths and removes idle resize listeners on unmount', () => {
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const wrapper = mount(ResizableTable, {
      props: {
        columns: [{ key: 'payload', label: 'Payload', minWidth: 72 }],
      },
      slots: { default: '<tr><td>abc</td></tr>' },
    })

    expect(wrapper.attributes('style')).toBeUndefined()
    expect(wrapper.find('col').attributes('style')).toContain('width: 72px')

    wrapper.unmount()

    expect(removeSpy).toHaveBeenCalledWith('pointermove', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('pointerup', expect.any(Function))
    removeSpy.mockRestore()
  })

  it('falls back to configured widths and minimums when measuring columns', async () => {
    let capturedMove: ((event: Event) => void) | undefined
    const addEventListener = window.addEventListener.bind(window)
    const addSpy = vi.spyOn(window, 'addEventListener').mockImplementation((type, listener, options) => {
      if (type === 'pointermove') capturedMove = listener as (event: Event) => void
      addEventListener(type, listener, options)
    })
    const wrapper = mount(ResizableTable, {
      props: {
        columns: [{ key: 'payload', label: 'Payload', width: 120, minWidth: 72 }],
      },
      slots: { default: '<tr><td>abc</td></tr>' },
    })
    const header = wrapper.find('th')
    vi.spyOn(header.element, 'getBoundingClientRect').mockReturnValue({
      width: 0,
      height: 20,
      top: 0,
      left: 0,
      right: 0,
      bottom: 20,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect)

    dispatchPointer(wrapper.find('[data-testid="resize-handle-payload"]').element, 'pointerdown', 10)
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 20 }))
    window.dispatchEvent(new MouseEvent('pointerup'))
    await wrapper.vm.$nextTick()
    capturedMove?.(new MouseEvent('pointermove', { clientX: 40 }))

    expect(wrapper.find('col').attributes('style')).toContain('width: 130px')

    for (const cell of wrapper.findAll('th, td')) {
      Object.defineProperty(cell.element, 'scrollWidth', {
        configurable: true,
        value: 10,
      })
    }
    await wrapper.find('[data-testid="resize-handle-payload"]').trigger('dblclick')

    expect(wrapper.find('col').attributes('style')).toContain('width: 72px')
    addSpy.mockRestore()
  })
})

function dispatchPointer(target: Element, type: string, clientX: number) {
  const event = new MouseEvent(type, { bubbles: true, clientX })
  Object.defineProperty(event, 'pointerId', { value: 1 })
  target.dispatchEvent(event)
}
