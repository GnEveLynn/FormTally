import type { DaySummary } from '@formtally/api-contract/days'
import * as daysApi from '../services/days'

type TodayStatus = 'idle' | 'loading' | 'empty' | 'ready' | 'error'

export function localDateAt(date: Date, offsetMinutes = -date.getTimezoneOffset()): string {
  return new Date(date.getTime() + offsetMinutes * 60_000).toISOString().slice(0, 10)
}

export function createTodayStore(client: { getDay(date: string): Promise<DaySummary> } = daysApi, today: () => string = () => localDateAt(new Date())) {
  const state: { status: TodayStatus; localDate: string; day: DaySummary | null; error: string } = {
    status: 'idle', localDate: today(), day: null, error: '',
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
    load,
    retry: () => load(state.localDate),
    async refreshAfterSave(affectedLocalDates: string[]) {
      if (affectedLocalDates.includes(state.localDate)) await load(state.localDate)
    },
  }
}

export const todayStore = createTodayStore()
