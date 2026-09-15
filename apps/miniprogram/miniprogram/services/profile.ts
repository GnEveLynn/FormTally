import type { ProfileInput, ProfileResponse } from '@formtally/api-contract/profile'
import { request } from './http'

export const getProfile = (): Promise<ProfileResponse> => request('/v1/profile')
export const saveProfile = (input: ProfileInput): Promise<ProfileResponse> => {
  const { biologicalSex, birthDate, heightCm, weightKg, activityLevel, timezone, healthContext, expectedRevision } = input
  return request('/v1/profile', { method: 'PUT', body: { biologicalSex, birthDate, heightCm, weightKg, activityLevel, timezone, healthContext, expectedRevision } })
}
