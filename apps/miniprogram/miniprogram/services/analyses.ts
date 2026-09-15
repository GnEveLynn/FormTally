import type { AnalysisResponse } from '@formtally/api-contract/analyses'
import { request } from './http'

export function retryAnalysis(id: string, input: { aiConsentVersion: string; expectedRevision: number; description?: string }, idempotencyKey: string): Promise<AnalysisResponse> {
  return request(`/v1/meal-analyses/${id}/retry`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: input,
  })
}

export function discardAnalysis(id: string, expectedRevision: number): Promise<void> {
  return request(`/v1/meal-analyses/${id}?expectedRevision=${expectedRevision}`, { method: 'DELETE' })
}
