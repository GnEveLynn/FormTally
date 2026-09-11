import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-router', () => ({ useRouter: () => ({ replace: vi.fn() }) }))

import IndexPage from './index.vue'

describe('启动页', () => {
  it('会话探测期间展示明确的恢复状态', () => {
    const wrapper = mount(IndexPage)

    expect(wrapper.get('h1').text()).toBe('FormTally')
    expect(wrapper.text()).toContain('正在恢复你的登录状态')
  })
})
