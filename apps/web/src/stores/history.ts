import type { DaySummary } from '@formtally/api-contract/days'
import { reactive } from 'vue'

import * as dayApi from '../api/days'
import * as historyApi from '../api/history'

export interface HistoryClient {
  getHistory(month: string): Promise<{ month: string; timezone: string; days: Array<{ localDate: string }> }>
  getDay(date: string): Promise<DaySummary>
}

export function localToday(timezone: string, now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(now)
}

export function createHistoryStore(
  client: HistoryClient = { ...historyApi, ...dayApi },
  timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
  now = new Date(),
) {
  const selectedDate = localToday(timezone, now)
  const state = reactive({
    month: selectedDate.slice(0, 7),
    selectedDate,
    recordedDates: new Set<string>(),
    monthStatus: 'idle' as 'idle' | 'loading' | 'ready' | 'error',
    dayStatus: 'idle' as 'idle' | 'loading' | 'empty' | 'ready' | 'error',
    day: null as DaySummary | null,
    error: '',
  })

  async function loadMonth(month = state.month) {
    state.monthStatus = 'loading'
    state.error = ''
    try {
      const result = await client.getHistory(month)
      state.month = result.month
      state.recordedDates = new Set(result.days.map((item) => item.localDate))
      state.monthStatus = 'ready'
    } catch (error) {
      state.monthStatus = 'error'
      state.error = error instanceof Error ? error.message : '历史记录加载失败'
    }
  }

  async function selectDate(date: string) {
    state.selectedDate = date
    state.dayStatus = 'loading'
    state.error = ''
    try {
      state.day = await client.getDay(date)
      state.dayStatus = state.day.mealGroups.length ? 'ready' : 'empty'
    } catch (error) {
      state.day = null
      state.dayStatus = 'error'
      state.error = error instanceof Error ? error.message : '当日记录加载失败'
    }
  }

  async function refreshDates(dates: string[]) {
    await Promise.all(dates.map((date) => client.getDay(date)))
    if (dates.includes(state.selectedDate)) await selectDate(state.selectedDate)
  }

  return { state, loadMonth, selectDate, refreshDates }
}

export const historyStore = createHistoryStore()
