import type { DaySummary } from '@formtally/api-contract/days'
import { historyStore } from '../../stores/history'
import { localDateAt } from '../../stores/today'
import { buildCalendar, dateForMonth, loadHistoryPage, shiftMonth } from './model'

const labels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }
const icons: Record<string, string> = { breakfast: '☀', lunch: '♨', dinner: '☾', snack: '♡' }
function groups(day: DaySummary | null) { return day?.mealGroups.map((group) => ({ ...group, label: labels[group.mealType] ?? group.mealType, icon: icons[group.mealType] ?? '•', meals: group.meals.map((meal) => ({ ...meal, time: new Date(meal.occurredAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) })) })) ?? [] }

Page({
  data: { month: '', monthLabel: '', selectedDate: '', selectedDateLabel: '', recordedDates: [] as string[], calendar: [] as ReturnType<typeof buildCalendar>, monthStatus: 'idle', dayStatus: 'idle', day: null as DaySummary | null, groups: [] as ReturnType<typeof groups>, error: '' },
  onShow() { return loadHistoryPage(historyStore, () => this.sync()) },
  sync() { const state = historyStore.state; const [year, month] = state.month.split('-'); const [, selectedMonth, selectedDay] = state.selectedDate.split('-'); this.setData({ ...state, monthLabel: `${year}年${Number(month)}月`, selectedDateLabel: `${Number(selectedMonth)}月${Number(selectedDay)}日`, recordedDates: [...state.recordedDates], calendar: buildCalendar(state.month, state.recordedDates, state.selectedDate), groups: groups(state.day) }) },
  previousMonth() { return this.loadMonth(shiftMonth(historyStore.state.month, -1)) },
  nextMonth() { return this.loadMonth(shiftMonth(historyStore.state.month, 1)) },
  async loadMonth(month: string) { await historyStore.loadMonth(month); await historyStore.selectDate(dateForMonth(month, historyStore.state.recordedDates)); this.sync() },
  async selectDate(event: { currentTarget: { dataset: { date: string; inMonth: boolean } } }) { const { date, inMonth } = event.currentTarget.dataset; if (!inMonth) return; await historyStore.selectDate(date); this.sync() },
  async goToday() { const today = localDateAt(new Date()); await historyStore.loadMonth(today.slice(0, 7)); await historyStore.selectDate(today); this.sync() },
  async retry() { await historyStore.retryDay(); this.sync() },
  openMeal(event: { currentTarget: { dataset: { id?: string } } }) { const id = event.currentTarget.dataset.id; if (id) wx.navigateTo({ url: `/pages/meal-detail/index?id=${encodeURIComponent(id)}` }) },
})
