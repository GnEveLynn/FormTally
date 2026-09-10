import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import IndexPage from './index.vue'

describe('首页', () => {
  it('展示产品名称和一期用途', () => {
    const wrapper = mount(IndexPage)

    expect(wrapper.get('h1').text()).toBe('FormTally')
    expect(wrapper.text()).toContain('记录饮食，了解每天的营养摄入')
  })
})
