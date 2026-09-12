import type { GoalSettingsInput, GoalsResponse, PreviewResponse, SaveGoalResponse } from '@formtally/api-contract/goals'
import { http } from './http'
export const previewGoal = (input: GoalSettingsInput) => http<PreviewResponse>('/v1/goal-previews', { method: 'POST', body: JSON.stringify(input) })
export const saveGoal = (input: GoalSettingsInput) => http<SaveGoalResponse>('/v1/goals', { method: 'PUT', body: JSON.stringify(input) })
export const getGoals = () => http<GoalsResponse>('/v1/goals')
