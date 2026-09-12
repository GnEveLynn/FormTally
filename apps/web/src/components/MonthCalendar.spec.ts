import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import MonthCalendar from './MonthCalendar.vue'

describe('MonthCalendar', () => {
  it('marks only recorded dates and emits a selected local date', async () => {
    const wrapper = mount(MonthCalendar, { props: { month: '2026-09', selected: '2026-09-11', recordedDates: new Set(['2026-09-09']) } })
    expect(wrapper.get('[data-date="2026-09-09"]').classes()).toContain('recorded')
    expect(wrapper.get('[data-date="2026-09-10"]').classes()).not.toContain('recorded')
    await wrapper.get('[data-date="2026-09-09"]').trigger('click')
    expect(wrapper.emitted('select')).toEqual([['2026-09-09']])
  })

  it('emits adjacent calendar months', async () => {
    const wrapper = mount(MonthCalendar, { props: { month: '2026-01', selected: '2026-01-01', recordedDates: new Set<string>() } })
    await wrapper.get('[data-action="previous-month"]').trigger('click')
    await wrapper.get('[data-action="next-month"]').trigger('click')
    expect(wrapper.emitted('change-month')).toEqual([['2025-12'], ['2026-02']])
  })

  it('aligns the first date with its weekday', () => {
    const wrapper = mount(MonthCalendar, { props: { month: '2026-09', selected: '2026-09-01', recordedDates: new Set<string>() } })
    expect(wrapper.get('[data-date="2026-09-01"]').attributes('style') ?? '').toContain('grid-column-start: 2')
  })
})
