import type { DaySummary } from '@formtally/api-contract/days'
import * as daysApi from '../services/days'
import * as historyApi from '../services/history'

interface HistoryClient {
  getHistory(month: string): Promise<{ month: string; timezone: string; days: Array<{ localDate: string }> }>
  getDay(date: string): Promise<DaySummary>
}

export function localToday(timezone: string, now = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(now)
  const value = Object.fromEntries(parts.map(({ type, value }) => [type, value]))
  return `${value.year}-${value.month}-${value.day}`
}

export function createHistoryStore(client: HistoryClient = { ...historyApi, ...daysApi }, timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai', now = new Date()) {
  const selectedDate = localToday(timezone, now)
  const state = {
    month: selectedDate.slice(0, 7), selectedDate, recordedDates: new Set<string>(),
    monthStatus: 'idle' as 'idle' | 'loading' | 'ready' | 'error',
    dayStatus: 'idle' as 'idle' | 'loading' | 'empty' | 'ready' | 'error',
    day: null as DaySummary | null, error: '',
  }

  async function loadMonth(month = state.month) {
    state.monthStatus = 'loading'; state.error = ''
    try {
      const result = await client.getHistory(month)
      state.month = result.month
      state.recordedDates = new Set(result.days.map(({ localDate }) => localDate))
      state.monthStatus = 'ready'
    } catch (error) {
      state.monthStatus = 'error'; state.error = error instanceof Error ? error.message : '历史记录加载失败'
    }
  }

  async function selectDate(date: string) {
    state.selectedDate = date; state.dayStatus = 'loading'; state.error = ''
    try {
      state.day = await client.getDay(date)
      state.dayStatus = state.day.mealGroups.length ? 'ready' : 'empty'
    } catch (error) {
      state.day = null; state.dayStatus = 'error'; state.error = error instanceof Error ? error.message : '当日记录加载失败'
    }
  }

  return {
    state, loadMonth, selectDate, retryDay: () => selectDate(state.selectedDate),
    async refreshAfterMutation(dates: string[]) {
      await loadMonth(state.month)
      if (dates.includes(state.selectedDate)) await selectDate(state.selectedDate)
    },
  }
}

export const historyStore = createHistoryStore()
