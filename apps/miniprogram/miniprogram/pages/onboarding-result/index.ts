import { sessionStore } from '../../stores/session'
import { onboardingStore } from '../../stores/onboarding'

Page({
  data: {
    preview: onboardingStore.state.preview,
    busy: false,
    error: '',
  },

  onShow() {
    if (!onboardingStore.state.preview) {
      wx.redirectTo({ url: '/pages/onboarding-goal/index' })
      return
    }
    this.setData({ preview: onboardingStore.state.preview })
  },

  async confirm() {
    this.setData({ busy: true, error: '' })
    const status = await onboardingStore.saveGoal()
    this.setData({ busy: onboardingStore.state.busy, error: onboardingStore.state.error })
    if (!status) return
    if (sessionStore.state.data) sessionStore.state.data.user.onboardingStatus = status
    wx.reLaunch({ url: '/pages/today/index' })
  },
})
