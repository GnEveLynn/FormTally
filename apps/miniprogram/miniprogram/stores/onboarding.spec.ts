import type { PreviewResponse, SaveGoalResponse } from '@formtally/api-contract/goals'
import type { ProfileResponse } from '@formtally/api-contract/profile'
import { describe, expect, it } from 'vitest'
import { createOnboardingStore } from './onboarding'

const preview: PreviewResponse = {
  preview: {
    effectiveFrom: '2026-09-15',
    target: { energyKcal: 2200, proteinGrams: 138, carbGrams: 275, fatGrams: 61 },
    calculation: null,
    warnings: [],
  },
}

function client() {
  return {
    saveProfile: async (): Promise<ProfileResponse> => ({ profile: null }),
    previewGoal: async (): Promise<PreviewResponse> => preview,
    saveGoal: async (): Promise<SaveGoalResponse> => ({
      settings: { mode: 'automatic', automatic: { objective: 'maintain' }, manual: null, revision: 1, updatedAt: '2026-09-14T00:00:00Z' },
      effectiveTarget: preview.preview,
    }),
  }
}

describe('onboarding store', () => {
  it('keeps profile and goal drafts while moving between pages', () => {
    const store = createOnboardingStore('Asia/Shanghai', client())
    store.state.profile.weightKg = 72.5
    store.state.goal.automatic.objective = 'muscle_gain'
    expect(store.state.profile.weightKg).toBe(72.5)
    expect(store.state.goal.automatic.objective).toBe('muscle_gain')
  })

  it('uses the shared profile validator and exposes field errors', async () => {
    const store = createOnboardingStore('Asia/Shanghai', client())
    store.state.profile.birthDate = ''
    store.state.profile.heightCm = 99
    await expect(store.saveProfile()).resolves.toBeNull()
    expect(store.state.profileErrors).toEqual({ birthDate: '请选择出生日期', heightCm: '身高需在 100–250 cm' })
  })

  it('stores the server preview and returns completed after saving the goal', async () => {
    const store = createOnboardingStore('Asia/Shanghai', client())
    store.state.goal.automatic.objective = 'maintain'
    await expect(store.previewGoal()).resolves.toBe('result')
    expect(store.state.preview).toEqual(preview.preview)
    await expect(store.saveGoal()).resolves.toBe('completed')
  })
})
