import type { AnalysisResponse } from '@formtally/api-contract/analyses'
import { currentApiBaseUrl } from '../config/env'
import { apiError, authHeaders, NetworkError } from './http'

export interface AnalysisUpload {
  imagePath: string
  processingMode: 'ai' | 'manual'
  occurredAt: string
  mealType: string
  aiConsentVersion?: string
  idempotencyKey: string
}

export interface UploadTask {
  promise: Promise<AnalysisResponse>
  cancel(): void
}

export function uploadAnalysis(input: AnalysisUpload, onProgress?: (percent: number) => void): UploadTask {
  let nativeTask: WechatMiniprogram.UploadTask
  const promise = new Promise<AnalysisResponse>((resolve, reject) => {
    nativeTask = wx.uploadFile({
      url: `${currentApiBaseUrl()}/v1/meal-analyses`,
      filePath: input.imagePath,
      name: 'image',
      header: authHeaders({ 'Idempotency-Key': input.idempotencyKey }),
      formData: {
        processingMode: input.processingMode,
        occurredAt: input.occurredAt,
        mealType: input.mealType,
        ...(input.aiConsentVersion ? { aiConsentVersion: input.aiConsentVersion } : {}),
      },
      success(response) {
        let data: unknown = response.data
        try { data = JSON.parse(response.data) as unknown } catch { /* handled below */ }
        if (response.statusCode >= 200 && response.statusCode < 300) {
          resolve(data as AnalysisResponse)
          return
        }
        reject(apiError(response.statusCode, data))
      },
      fail() {
        reject(new NetworkError())
      },
    })
    if (onProgress) nativeTask.onProgressUpdate(({ progress }) => onProgress(progress))
  })
  return { promise, cancel: () => nativeTask.abort() }
}
