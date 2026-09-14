import type { OnboardingStatus } from '@formtally/api-contract/auth'
import type { EffectiveTarget, GoalSettingsInput, NutritionTarget, PreviewResponse, SaveGoalResponse } from '@formtally/api-contract/goals'
import type { ProfileInput, ProfileResponse } from '@formtally/api-contract/profile'
import { validateProfile } from '../domain/profile-validation'
import * as goalsApi from '../services/goals'
import * as profileApi from '../services/profile'

interface OnboardingClient {
  saveProfile(input: ProfileInput): Promise<ProfileResponse>
  previewGoal(input: GoalSettingsInput): Promise<PreviewResponse>
  saveGoal(input: GoalSettingsInput): Promise<SaveGoalResponse>
}

interface GoalDraft {
  mode: 'automatic' | 'manual'
  automatic: { objective: 'fat_loss' | 'maintain' | 'muscle_gain'; pace: 'slow' | 'standard' | 'fast' }
  manual: { target: NutritionTarget }
}

export interface OnboardingState {
  profile: ProfileInput
  goal: GoalDraft
  preview: EffectiveTarget | null
  profileErrors: Record<string, string>
  error: string
  busy: boolean
}

export function createOnboardingStore(timezone = 'Asia/Shanghai', client: OnboardingClient = { ...profileApi, ...goalsApi }) {
  const state: OnboardingState = {
    profile: {
      biologicalSex: 'male',
      birthDate: '',
      heightCm: 170,
      weightKg: 65,
      activityLevel: 'moderate',
      timezone,
      healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false },
    },
    goal: {
      mode: 'automatic',
      automatic: { objective: 'fat_loss', pace: 'standard' },
      manual: { target: { energyKcal: 2000, proteinGrams: 125, carbGrams: 250, fatGrams: 56 } },
    },
    preview: null,
    profileErrors: {},
    error: '',
    busy: false,
  }

  function goalInput(): GoalSettingsInput {
    return state.goal.mode === 'automatic'
      ? { mode: 'automatic', automatic: { objective: state.goal.automatic.objective, ...(state.goal.automatic.objective === 'maintain' ? {} : { pace: state.goal.automatic.pace }) }, manual: null }
      : { mode: 'manual', automatic: null, manual: { target: { ...state.goal.manual.target } } }
  }

  async function run<T>(operation: () => Promise<T>): Promise<T | null> {
    state.busy = true
    state.error = ''
    try { return await operation() } catch (error) {
      state.error = error instanceof Error ? error.message : '请求失败，请重试'
      return null
    } finally { state.busy = false }
  }

  return {
    state,
    goalInput,
    async saveProfile(): Promise<OnboardingStatus | null> {
      state.profileErrors = validateProfile(state.profile)
      if (Object.keys(state.profileErrors).length) return null
      const result = await run(() => client.saveProfile(state.profile))
      return result ? 'goal_required' : null
    },
    async previewGoal(): Promise<'result' | null> {
      const result = await run(() => client.previewGoal(goalInput()))
      if (!result) return null
      state.preview = result.preview
      return 'result'
    },
    async saveGoal(): Promise<OnboardingStatus | null> {
      const result = await run(() => client.saveGoal(goalInput()))
      return result ? 'completed' : null
    },
  }
}

export const onboardingStore = createOnboardingStore()
