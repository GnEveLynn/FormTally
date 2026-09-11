import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ImagePicker from './ImagePicker.vue'

describe('ImagePicker', () => {
  it('重新选图不会改变父级保存的日期和餐别', async () => {
    const wrapper = mount(ImagePicker, { props: { occurredAt: '2026-09-11T12:00', mealType: 'lunch' } })
    const input = wrapper.get('input[type="file"]')
    expect(input.attributes('accept')).toBe('image/jpeg,image/png,image/webp')
    expect(input.attributes('capture')).toBe('environment')
    expect(wrapper.text()).toContain('2026-09-11T12:00')
    expect(wrapper.text()).toContain('lunch')
  })
})
