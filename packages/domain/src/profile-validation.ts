import type { HealthContext, ProfileInput } from '@formtally/api-contract/profile'

export function automaticGoalEligible(health: HealthContext): boolean {
  return !health.pregnant && !health.breastfeeding && !health.clinicalDietRequired
}

export function validateProfile(profile: ProfileInput): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!profile.birthDate) errors.birthDate = '请选择出生日期'
  if (profile.heightCm < 100 || profile.heightCm > 250) errors.heightCm = '身高需在 100–250 cm'
  if (profile.weightKg < 25 || profile.weightKg > 350) errors.weightKg = '体重需在 25–350 kg'
  if (!profile.timezone) errors.timezone = '无法识别当前时区'
  return errors
}
