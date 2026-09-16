import type { DaySummary } from '@formtally/api-contract/days'
import * as daysApi from '../services/days'

type TodayStatus = 'idle' | 'loading' | 'guest' | 'empty' | 'ready' | 'error'

export function localDateAt(date: Date, offsetMinutes = -date.getTimezoneOffset()): string {
  return new Date(date.getTime() + offsetMinutes * 60_000).toISOString().slice(0, 10)
}

export function dateTabsAt(localDate: string) {
  const center = new Date(`${localDate}T12:00:00Z`)
  return [-1, 0, 1].map((offset, index) => {
    const date = new Date(center)
    date.setUTCDate(date.getUTCDate() + offset)
    const value = date.toISOString().slice(0, 10)
    return { label: ['昨日', '今日', '明日'][index]!, date: value, displayDate: `${date.getUTCMonth() + 1}月${date.getUTCDate()}日` }
  })
}

export function createTodayStore(client: { getDay(date: string): Promise<DaySummary> } = daysApi, today: () => string = () => localDateAt(new Date())) {
  const todayDate = today()
  const state: { status: TodayStatus; todayDate: string; localDate: string; day: DaySummary | null; error: string } = {
    status: 'idle', todayDate, localDate: todayDate, day: null, error: '',
  }

  async function load(date = today()) {
    state.localDate = date
    state.status = 'loading'
    state.error = ''
    try {
      state.day = await client.getDay(date)
      state.status = state.day.mealGroups.length ? 'ready' : 'empty'
    } catch (error) {
      state.day = null
      state.status = 'error'
      state.error = error instanceof Error ? error.message : '加载失败，请重试'
    }
  }

  return {
    state,
    browse(date = today()) {
      state.localDate = date
      state.status = 'guest'
      state.day = null
      state.error = ''
    },
    load,
    retry: () => load(state.localDate),
    async refreshAfterSave(affectedLocalDates: string[]) {
      if (affectedLocalDates.includes(state.localDate)) await load(state.localDate)
    },
  }
}

export const todayStore = createTodayStore()
