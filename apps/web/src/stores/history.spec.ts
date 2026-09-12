import { describe, expect, it, vi } from 'vitest'

import { createHistoryStore, localToday } from './history'

const day = (localDate: string) => ({
  localDate,
  target: { energyKcal: 2000, proteinGrams: 120, carbGrams: 250, fatGrams: 60 },
  totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 },
  progress: {
    energy: { consumed: 0, target: 2000, remaining: 2000, overBy: 0, percent: 0, status: 'under' as const },
    protein: { consumed: 0, target: 120, remaining: 120, overBy: 0, percent: 0, status: 'under' as const },
    carb: { consumed: 0, target: 250, remaining: 250, overBy: 0, percent: 0, status: 'under' as const },
    fat: { consumed: 0, target: 60, remaining: 60, overBy: 0, percent: 0, status: 'under' as const },
  },
  mealGroups: [],
})

describe('history store', () => {
  it('uses the user timezone for the initial selected day', () => {
    expect(localToday('Asia/Shanghai', new Date('2026-09-10T16:30:00Z'))).toBe('2026-09-11')
  })

  it('loads only recorded dates as calendar marks and switches months', async () => {
    const getHistory = vi.fn().mockResolvedValue({ month: '2026-09', timezone: 'Asia/Shanghai', days: [{ localDate: '2026-09-09', mealCount: 1 }] })
    const getDay = vi.fn().mockResolvedValue(day('2026-09-11'))
    const store = createHistoryStore({ getHistory, getDay }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))

    await store.loadMonth('2026-09')
    expect([...store.state.recordedDates]).toEqual(['2026-09-09'])
    expect(store.state.month).toBe('2026-09')
    expect(getHistory).toHaveBeenCalledWith('2026-09')
  })

  it('reads an empty day without creating or mutating history', async () => {
    const getHistory = vi.fn().mockResolvedValue({ month: '2026-09', timezone: 'Asia/Shanghai', days: [] })
    const getDay = vi.fn().mockResolvedValue(day('2026-09-08'))
    const store = createHistoryStore({ getHistory, getDay }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))

    await store.selectDate('2026-09-08')
    expect(store.state.dayStatus).toBe('empty')
    expect(store.state.recordedDates.size).toBe(0)
    expect(getDay).toHaveBeenCalledTimes(1)
    expect(getHistory).not.toHaveBeenCalled()
  })

  it('refreshes only dates returned by a meal mutation', async () => {
    const getDay = vi.fn().mockImplementation(async (date: string) => day(date))
    const store = createHistoryStore({ getHistory: vi.fn(), getDay }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))

    await store.refreshDates(['2026-09-09', '2026-09-10'])
    expect(getDay.mock.calls).toEqual([['2026-09-09'], ['2026-09-10']])
  })
})
