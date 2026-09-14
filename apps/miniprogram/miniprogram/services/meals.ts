import type { DeleteMealResponse, Meal, SaveMealInput, SaveMealResponse, UpdateMealInput } from '@formtally/api-contract/meals'
import { request } from './http'

export function saveMeal(input: SaveMealInput, idempotencyKey: string): Promise<SaveMealResponse> {
  return request('/v1/meals', { method: 'POST', headers: { 'Idempotency-Key': idempotencyKey }, body: input })
}

export const getMeal = async (id: string): Promise<Meal> =>
  (await request<{ meal: Meal }>(`/v1/meals/${encodeURIComponent(id)}`)).meal

export const updateMeal = (id: string, input: UpdateMealInput): Promise<SaveMealResponse> =>
  request(`/v1/meals/${encodeURIComponent(id)}`, { method: 'PUT', body: input })

export const removeMealImage = async (id: string, expectedRevision: number): Promise<Meal> =>
  (await request<{ meal: Meal }>(`/v1/meals/${encodeURIComponent(id)}/image?expectedRevision=${expectedRevision}`, { method: 'DELETE' })).meal

export const deleteMeal = (id: string, expectedRevision: number): Promise<DeleteMealResponse> =>
  request(`/v1/meals/${encodeURIComponent(id)}?expectedRevision=${expectedRevision}`, { method: 'DELETE' })
