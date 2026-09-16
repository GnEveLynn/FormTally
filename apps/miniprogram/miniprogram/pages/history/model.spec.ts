import { describe, expect, it, vi } from 'vitest'
import { buildCalendar, dateForMonth, loadHistoryPage, shiftMonth } from './model'

describe('history page loading', () => {
  it('renders loading before requests settle and renders results afterward', async () => {
    let finish!: () => void
    const pending = new Promise<void>((resolve) => { finish = resolve })
    const store = { state: { selectedDate: '2026-09-14' }, loadMonth: vi.fn(() => pending), selectDate: vi.fn(() => pending) }
    const sync = vi.fn()

    const loading = loadHistoryPage(store, sync)
    expect(sync).toHaveBeenCalledTimes(1)
    finish()
    await loading
    expect(sync).toHaveBeenCalledTimes(2)
  })

  it('builds a monday-first calendar and marks recorded and selected dates', () => {
    const cells = buildCalendar('2026-09', new Set(['2026-09-03']), '2026-09-16')

    expect(cells).toHaveLength(42)
    expect(cells[0]).toMatchObject({ day: 31, inMonth: false })
    expect(cells.find(({ date }) => date === '2026-09-03')).toMatchObject({ recorded: true, selected: false })
    expect(cells.find(({ date }) => date === '2026-09-16')).toMatchObject({ selected: true })
  })

  it('shifts months across year boundaries', () => {
    expect(shiftMonth('2026-01', -1)).toBe('2025-12')
    expect(shiftMonth('2026-12', 1)).toBe('2027-01')
  })

  it('opens the first recorded day when browsing another month', () => {
    expect(dateForMonth('2026-08', new Set(['2026-08-21', '2026-08-03']))).toBe('2026-08-03')
    expect(dateForMonth('2026-08', new Set())).toBe('2026-08-01')
  })
})
