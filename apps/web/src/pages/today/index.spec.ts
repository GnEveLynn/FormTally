import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/days', () => ({ getDay: vi.fn() }))

import { getDay } from '../../api/days'
import TodayPage from './index.vue'

const metric = { consumed: 0, target: 100, remaining: 100, overBy: 0, percent: 0, status: 'under' as const }
const emptyDay = {
  localDate: '2026-09-11',
  target: { energyKcal: 2000, proteinGrams: 100, carbGrams: 250, fatGrams: 60 },
  totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 },
  progress: { energy: metric, protein: metric, carb: metric, fat: metric },
  mealGroups: [],
}

describe('TodayPage', () => {
  beforeEach(() => vi.mocked(getDay).mockResolvedValue(emptyDay))

  it('shows the explicit empty state and meal entry', async () => {
    const wrapper = mount(TodayPage)
    await flushPromises()
    expect(wrapper.text()).toContain('今天还没有记录')
    expect(wrapper.get('a[href="/meals/new"]').text()).toContain('记录一餐')
  })

  it('renders recorded meals and nutrition progress', async () => {
    vi.mocked(getDay).mockResolvedValue({
      ...emptyDay,
      totals: { energyKcal: 520, proteinGrams: 28, carbGrams: 65, fatGrams: 17 },
      mealGroups: [{ mealType: 'lunch', meals: [{ id: 'meal_1', occurredAt: '2026-09-11T12:10:00+08:00', totals: { energyKcal: 520, proteinGrams: 28, carbGrams: 65, fatGrams: 17 } }] }],
    })
    const wrapper = mount(TodayPage)
    await flushPromises()
    expect(wrapper.text()).toContain('午餐')
    expect(wrapper.text()).toContain('520 千卡')
  })
})
