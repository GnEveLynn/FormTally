import { describe, expect, it } from 'vitest'
import { automaticGoalEligible, validateProfile } from './profile-validation'

const profile = { biologicalSex: 'male', birthDate: '1995-06-18', heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', timezone: 'Asia/Shanghai', healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false } } as const

describe('身体资料校验', () => {
  it('接受完整且在边界内的资料', () => expect(validateProfile(profile)).toEqual({}))
  it('拒绝超出允许范围的字段', () => expect(validateProfile({ ...profile, heightCm: 99 })).toHaveProperty('heightCm'))
  it('特殊健康状态切换为手动目标', () => expect(automaticGoalEligible({ ...profile.healthContext, pregnant: true })).toBe(false))
})
