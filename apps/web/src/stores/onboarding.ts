import type { EffectiveTarget, GoalSettingsInput } from '@formtally/api-contract/goals'
import type { ProfileInput } from '@formtally/api-contract/profile'
import { reactive, ref } from 'vue'

export function createOnboardingStore() {
  const profile = reactive<ProfileInput>({ biologicalSex: 'male', birthDate: '', heightCm: 170, weightKg: 65, activityLevel: 'moderate', timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai', healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false } })
  const goal = reactive({ mode: 'automatic' as 'automatic' | 'manual', automatic: { objective: 'fat_loss' as 'fat_loss' | 'maintain' | 'muscle_gain', pace: 'standard' as 'slow' | 'standard' | 'fast' }, manual: { target: { energyKcal: 2000, proteinGrams: 125, carbGrams: 250, fatGrams: 56 } } })
  const preview = ref<EffectiveTarget | null>(null)
  const goalInput = (): GoalSettingsInput => goal.mode === 'automatic'
    ? { mode: 'automatic', automatic: { objective: goal.automatic.objective, ...(goal.automatic.objective === 'maintain' ? {} : { pace: goal.automatic.pace }) }, manual: null }
    : { mode: 'manual', automatic: null, manual: goal.manual }
  return { profile, goal, preview, goalInput }
}
export const onboardingStore = createOnboardingStore()
