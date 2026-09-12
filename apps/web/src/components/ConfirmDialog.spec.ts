import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ConfirmDialog from './ConfirmDialog.vue'

describe('ConfirmDialog', () => {
  it('does not confirm when the user cancels', async () => {
    const wrapper = mount(ConfirmDialog, { props: { open: true, title: '删除这餐？', confirmLabel: '删除整餐' } })
    await wrapper.get('[data-action="cancel"]').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  it('uses the explicit destructive action label', async () => {
    const wrapper = mount(ConfirmDialog, { props: { open: true, title: '移除图片？', confirmLabel: '仅移除图片' } })
    expect(wrapper.get('[data-action="confirm"]').text()).toBe('仅移除图片')
    await wrapper.get('[data-action="confirm"]').trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })
})
