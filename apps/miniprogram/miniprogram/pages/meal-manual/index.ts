import { mealDraftStore } from '../../stores/meal-draft'

Page({
  onLoad() {
    mealDraftStore.startManual()
    wx.redirectTo({ url: '/pages/meal-confirm/index' })
  },
})
