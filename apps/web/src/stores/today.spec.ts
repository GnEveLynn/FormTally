import { describe, expect, it } from 'vitest'
import { createTodayStore } from './today'
import type { DaySummary, MetricProgress } from '@formtally/api-contract/days'

const zeroProgress: MetricProgress = { consumed: 0, target: 0, remaining: 0, overBy: 0, percent: 0, status: 'unavailable' }
const day: DaySummary = { localDate: '2026-09-11', target: { energyKcal: 2200, proteinGrams: 130, carbGrams: 250, fatGrams: 65 }, totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 }, progress: { energy: zeroProgress, protein: zeroProgress, carb: zeroProgress, fat: zeroProgress }, mealGroups: [] }

describe('today store', () => {
  it('distinguishes loading empty ready and error', async () => {
    let resolve!: (value: any) => void
    const store = createTodayStore({ getDay: () => new Promise((done) => { resolve = done }) })
    const pending = store.load('2026-09-11')
    expect(store.state.status).toBe('loading')
    resolve(day)
    await pending
    expect(store.state.status).toBe('empty')
    const ready = createTodayStore({ getDay: async () => ({ ...day, mealGroups: [{ mealType: 'lunch', meals: [{ id: 'meal_1', occurredAt: '2026-09-11T12:00:00+08:00', totals: day.totals }] }] }) })
    await ready.load('2026-09-11')
    expect(ready.state.status).toBe('ready')
    const failed = createTodayStore({ getDay: async () => { throw new Error('网络失败') } })
    await failed.load('2026-09-11')
    expect(failed.state.status).toBe('error')
    expect(failed.state.day).toBeNull()
  })
})
