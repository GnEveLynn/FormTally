import type { DaySummary } from '@formtally/api-contract/days'
import { historyStore } from '../../stores/history'
import { loadHistoryPage } from './model'

const labels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }
function groups(day: DaySummary | null) { return day?.mealGroups.map((group) => ({ ...group, label: labels[group.mealType] ?? group.mealType })) ?? [] }

Page({
  data: { month: '', selectedDate: '', recordedDates: [] as string[], monthStatus: 'idle', dayStatus: 'idle', day: null as DaySummary | null, groups: [] as ReturnType<typeof groups>, error: '' },
  onShow() { return loadHistoryPage(historyStore, () => this.sync()) },
  sync() { const state = historyStore.state; this.setData({ ...state, recordedDates: [...state.recordedDates], groups: groups(state.day) }) },
  async changeMonth(event: { detail: { value: string } }) { await historyStore.loadMonth(event.detail.value); this.sync() },
  async selectDate(event: { detail: { value: string } }) { await historyStore.selectDate(event.detail.value); this.sync() },
  async retry() { await historyStore.retryDay(); this.sync() },
  openMeal(event: { currentTarget: { dataset: { id?: string } } }) { const id = event.currentTarget.dataset.id; if (id) wx.navigateTo({ url: `/pages/meal-detail/index?id=${encodeURIComponent(id)}` }) },
})
