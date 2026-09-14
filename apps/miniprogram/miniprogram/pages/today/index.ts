import type { DaySummary } from '@formtally/api-contract/days'
import { todayStore } from '../../stores/today'

const mealLabels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }

function displayGroups(day: DaySummary | null) {
  return day?.mealGroups.map((group) => ({
    ...group,
    label: mealLabels[group.mealType] ?? group.mealType,
    meals: group.meals.map((meal) => ({ ...meal, time: new Date(meal.occurredAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) })),
  })) ?? []
}

Page({
  data: {
    status: 'idle',
    localDate: '',
    day: null as DaySummary | null,
    groups: [] as ReturnType<typeof displayGroups>,
    error: '',
  },

  async onShow() {
    const pending = todayStore.load()
    this.sync()
    await pending
    this.sync()
  },

  sync() {
    this.setData({
      status: todayStore.state.status,
      localDate: todayStore.state.localDate,
      day: todayStore.state.day,
      groups: displayGroups(todayStore.state.day),
      error: todayStore.state.error,
    })
  },

  async retry() {
    const pending = todayStore.retry()
    this.sync()
    await pending
    this.sync()
  },

  recordMeal() {
    wx.switchTab({ url: '/pages/meal-capture/index' })
  },
})
