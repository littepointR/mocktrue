import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ConnectView from './ConnectView.vue'

describe('ConnectView', () => {
  it('renders the serial connection placeholder', () => {
    const wrapper = mount(ConnectView)

    expect(wrapper.find('.serial-connect-placeholder').text()).toBe('串口连接（占位）')
  })
})
