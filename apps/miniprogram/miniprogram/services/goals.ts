import type { GoalSettingsInput, GoalsResponse, PreviewResponse, SaveGoalResponse } from '@formtally/api-contract/goals'
import { request } from './http'

export const getGoals = (): Promise<GoalsResponse> => request('/v1/goals')
export const previewGoal = (input: GoalSettingsInput): Promise<PreviewResponse> => request('/v1/goal-previews', { method: 'POST', body: { ...input } })
export const saveGoal = (input: GoalSettingsInput): Promise<SaveGoalResponse> => request('/v1/goals', { method: 'PUT', body: { ...input } })
