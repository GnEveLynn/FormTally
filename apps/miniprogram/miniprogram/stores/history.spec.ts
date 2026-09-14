import type { DaySummary } from '@formtally/api-contract/days'
import { describe, expect, it, vi } from 'vitest'
import { createHistoryStore, localToday } from './history'

const emptyDay = (localDate: string): DaySummary => ({
  localDate, target: null,
  totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 },
  progress: null, mealGroups: [],
})

describe('mini-program history store', () => {
  it('uses a deterministic user-local date and switches months', async () => {
    expect(localToday('Asia/Shanghai', new Date('2026-09-10T16:30:00Z'))).toBe('2026-09-11')
    const getHistory = vi.fn().mockResolvedValue({ month: '2026-08', timezone: 'Asia/Shanghai', days: [{ localDate: '2026-08-09', mealCount: 1 }] })
    const store = createHistoryStore({ getHistory, getDay: vi.fn() }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))

    await store.loadMonth('2026-08')

    expect(store.state.month).toBe('2026-08')
    expect([...store.state.recordedDates]).toEqual(['2026-08-09'])
    expect(getHistory).toHaveBeenCalledWith('2026-08')
  })

  it('selects empty dates and exposes retryable failures', async () => {
    const getDay = vi.fn().mockRejectedValueOnce(new Error('网络失败')).mockResolvedValueOnce(emptyDay('2026-09-08'))
    const store = createHistoryStore({ getHistory: vi.fn(), getDay }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))

    await store.selectDate('2026-09-08')
    expect(store.state).toMatchObject({ selectedDate: '2026-09-08', dayStatus: 'error', error: '网络失败' })
    await store.retryDay()
    expect(store.state.dayStatus).toBe('empty')
  })

  it('reloads the month and visible date after edit or delete', async () => {
    const getHistory = vi.fn().mockResolvedValue({ month: '2026-09', timezone: 'Asia/Shanghai', days: [] })
    const getDay = vi.fn().mockImplementation(async (date: string) => emptyDay(date))
    const store = createHistoryStore({ getHistory, getDay }, 'Asia/Shanghai', new Date('2026-09-11T03:00:00Z'))
    await store.selectDate('2026-09-10')

    await store.refreshAfterMutation(['2026-09-09', '2026-09-10'])

    expect(getHistory).toHaveBeenCalledWith('2026-09')
    expect(getDay.mock.calls).toEqual([['2026-09-10'], ['2026-09-10']])
  })
})
