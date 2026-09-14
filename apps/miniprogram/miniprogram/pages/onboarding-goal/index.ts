import type { NutritionTarget } from '@formtally/api-contract/goals'
import { automaticGoalEligible } from '../../domain/profile-validation'
import { onboardingStore } from '../../stores/onboarding'

const objectiveValues = ['fat_loss', 'maintain', 'muscle_gain'] as const
const paceValues = ['slow', 'standard', 'fast'] as const

Page({
  data: {
    goal: onboardingStore.state.goal,
    automaticEligible: true,
    objectiveLabels: ['减脂', '保持', '增肌'],
    objectiveIndex: 0,
    paceLabels: ['慢速', '标准', '快速'],
    paceIndex: 1,
    busy: false,
    error: '',
  },

  onLoad() {
    const eligible = automaticGoalEligible(onboardingStore.state.profile.healthContext)
    if (!eligible) onboardingStore.state.goal.mode = 'manual'
    this.setData({ goal: onboardingStore.state.goal, automaticEligible: eligible })
  },
  onModeChange(event: { detail: { value: 'automatic' | 'manual' } }) {
    onboardingStore.state.goal.mode = event.detail.value
    this.setData({ goal: onboardingStore.state.goal })
  },
  onObjectiveChange(event: { detail: { value: string } }) {
    const index = Number(event.detail.value)
    onboardingStore.state.goal.automatic.objective = objectiveValues[index] ?? 'fat_loss'
    this.setData({ goal: onboardingStore.state.goal, objectiveIndex: index })
  },
  onPaceChange(event: { detail: { value: string } }) {
    const index = Number(event.detail.value)
    onboardingStore.state.goal.automatic.pace = paceValues[index] ?? 'standard'
    this.setData({ goal: onboardingStore.state.goal, paceIndex: index })
  },
  onManualInput(event: { currentTarget: { dataset: { field?: string } }; detail: { value: string } }) {
    const field = event.currentTarget.dataset.field as keyof NutritionTarget
    if (!field) return
    onboardingStore.state.goal.manual.target[field] = Number(event.detail.value)
    this.setData({ goal: onboardingStore.state.goal })
  },

  async submit() {
    this.setData({ busy: true, error: '' })
    const next = await onboardingStore.previewGoal()
    this.setData({ busy: onboardingStore.state.busy, error: onboardingStore.state.error })
    if (next) wx.navigateTo({ url: '/pages/onboarding-result/index' })
  },
})
