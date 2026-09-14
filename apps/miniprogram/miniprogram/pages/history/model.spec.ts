import { describe, expect, it, vi } from 'vitest'
import { loadHistoryPage } from './model'

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
})
