import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/days', () => ({ getDay: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

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
  beforeEach(() => { vi.mocked(getDay).mockReset(); vi.mocked(getDay).mockResolvedValue(emptyDay) })

  it('shows the explicit empty state and meal entry', async () => {
    const wrapper = mount(TodayPage)
    await flushPromises()
    expect(wrapper.text()).toContain('今天还没有记录')
    await wrapper.get('[data-test="open-record"]').trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('记录饮食')
    expect(wrapper.findAll('textarea')).toHaveLength(1)
    expect(wrapper.get('textarea').attributes('placeholder')).toContain('少油')
  })

  it('loads yesterday and tomorrow from the date switcher', async () => {
    const wrapper = mount(TodayPage)
    await flushPromises()
    await wrapper.get('[data-test="date-yesterday"]').trigger('click')
    await wrapper.get('[data-test="date-tomorrow"]').trigger('click')
    expect(vi.mocked(getDay).mock.calls.map(([date]) => date)).toEqual([
      expect.any(String), expect.any(String), expect.any(String),
    ])
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
