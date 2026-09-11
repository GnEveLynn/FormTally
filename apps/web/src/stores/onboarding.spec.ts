import { expect, it } from 'vitest'
import { createOnboardingStore } from './onboarding'

it('多步骤往返时保留资料和目标草稿', () => {
  const store = createOnboardingStore()
  store.profile.weightKg = 72.5
  store.goal.automatic.objective = 'muscle_gain'
  expect(store.profile.weightKg).toBe(72.5)
  expect(store.goal.automatic.objective).toBe('muscle_gain')
})
