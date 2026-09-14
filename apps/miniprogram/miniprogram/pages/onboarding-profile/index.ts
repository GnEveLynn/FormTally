import { sessionStore } from '../../stores/session'
import { onboardingStore } from '../../stores/onboarding'

const sexValues = ['male', 'female'] as const
const activityValues = ['sedentary', 'light', 'moderate', 'high', 'very_high'] as const

Page({
  data: {
    profile: onboardingStore.state.profile,
    errors: onboardingStore.state.profileErrors,
    error: '',
    busy: false,
    sexLabels: ['男性', '女性'],
    sexIndex: 0,
    activityLabels: ['久坐', '轻度', '中度', '高度', '极高'],
    activityIndex: 2,
  },

  onSexChange(event: { detail: { value: string } }) {
    const index = Number(event.detail.value)
    onboardingStore.state.profile.biologicalSex = sexValues[index] ?? 'male'
    this.setData({ profile: onboardingStore.state.profile, sexIndex: index })
  },
  onBirthDateChange(event: { detail: { value: string } }) {
    onboardingStore.state.profile.birthDate = event.detail.value
    this.setData({ profile: onboardingStore.state.profile })
  },
  onHeightInput(event: { detail: { value: string } }) {
    onboardingStore.state.profile.heightCm = Number(event.detail.value)
    this.setData({ profile: onboardingStore.state.profile })
  },
  onWeightInput(event: { detail: { value: string } }) {
    onboardingStore.state.profile.weightKg = Number(event.detail.value)
    this.setData({ profile: onboardingStore.state.profile })
  },
  onActivityChange(event: { detail: { value: string } }) {
    const index = Number(event.detail.value)
    onboardingStore.state.profile.activityLevel = activityValues[index] ?? 'moderate'
    this.setData({ profile: onboardingStore.state.profile, activityIndex: index })
  },
  onHealthChange(event: { detail: { value: string[] } }) {
    const selected = event.detail.value
    onboardingStore.state.profile.healthContext = {
      pregnant: selected.includes('pregnant'),
      breastfeeding: selected.includes('breastfeeding'),
      clinicalDietRequired: selected.includes('clinicalDietRequired'),
    }
    this.setData({ profile: onboardingStore.state.profile })
  },

  async submit() {
    this.setData({ busy: true, error: '' })
    const status = await onboardingStore.saveProfile()
    this.setData({
      errors: onboardingStore.state.profileErrors,
      error: onboardingStore.state.error,
      busy: onboardingStore.state.busy,
    })
    if (!status) return
    if (sessionStore.state.data) sessionStore.state.data.user.onboardingStatus = status
    wx.navigateTo({ url: '/pages/onboarding-goal/index' })
  },
})
