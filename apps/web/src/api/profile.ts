import type { ProfileInput, ProfileResponse } from '@formtally/api-contract/profile'
import { http } from './http'
export const getProfile = () => http<ProfileResponse>('/v1/profile')
export const saveProfile = (input: ProfileInput) => http<ProfileResponse>('/v1/profile', { method: 'PUT', body: JSON.stringify(input) })
