import type { DaySummary } from '@formtally/api-contract/days'
import { describe, expect, it, vi } from 'vitest'
import { createTodayStore, dateTabsAt, localDateAt } from './today'

const emptyDay: DaySummary = {
  localDate: '2026-09-14', target: null,
  totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 }, progress: null, mealGroups: [],
}

describe('mini-program today store', () => {
  it('builds yesterday today and tomorrow tabs around the selected date', () => {
    expect(dateTabsAt('2026-09-15')).toEqual([
      { label: '昨日', date: '2026-09-14', displayDate: '9月14日' },
      { label: '今日', date: '2026-09-15', displayDate: '9月15日' },
      { label: '明日', date: '2026-09-16', displayDate: '9月16日' },
    ])
  })

  it('uses the user device offset instead of UTC for the selected date', () => {
    expect(localDateAt(new Date('2026-09-13T16:30:00Z'), 8 * 60)).toBe('2026-09-14')
    expect(localDateAt(new Date('2026-09-14T02:00:00Z'), -7 * 60)).toBe('2026-09-13')
  })

  it('exposes loading, empty, ready, and retryable failure states', async () => {
    let resolve!: (day: DaySummary) => void
    const getDay = vi.fn(() => new Promise<DaySummary>((done) => { resolve = done }))
    const store = createTodayStore({ getDay }, () => '2026-09-14')
    const pending = store.load()
    expect(store.state.status).toBe('loading')
    resolve(emptyDay)
    await pending
    expect(store.state.status).toBe('empty')

    getDay.mockRejectedValueOnce(new Error('网络失败'))
    await store.load()
    expect(store.state).toMatchObject({ status: 'error', day: null, error: '网络失败' })

    getDay.mockResolvedValueOnce({ ...emptyDay, mealGroups: [{ mealType: 'lunch', meals: [{ id: 'meal_1', occurredAt: '2026-09-14T12:00:00+08:00', totals: emptyDay.totals }] }] })
    await store.retry()
    expect(store.state.status).toBe('ready')
  })

  it('refreshes the visible day after a save affects it', async () => {
    const getDay = vi.fn().mockResolvedValue(emptyDay)
    const store = createTodayStore({ getDay }, () => '2026-09-14')
    await store.load()
    await store.refreshAfterSave(['2026-09-14'])
    await store.refreshAfterSave(['2026-09-13'])
    expect(getDay).toHaveBeenCalledTimes(2)
  })

  it('keeps the real today fixed when the selected day changes', async () => {
    const getDay = vi.fn().mockResolvedValue(emptyDay)
    const store = createTodayStore({ getDay }, () => '2026-09-15')
    await store.load('2026-09-14')
    expect(store.state.localDate).toBe('2026-09-14')
    expect(store.state.todayDate).toBe('2026-09-15')
  })
})
