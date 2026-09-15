import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setSessionToken } from '../platform/storage'
import { ApiError, NetworkError } from './http'
import { uploadAnalysis } from './upload'

describe('meal analysis upload', () => {
  const values = new Map<string, unknown>()
  let uploadOptions: Record<string, any>
  const abort = vi.fn()
  let progress: ((event: { progress: number }) => void) | undefined

  beforeEach(() => {
    values.clear()
    uploadOptions = {}
    abort.mockClear()
    progress = undefined
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
        getStorageSync: vi.fn((key: string) => values.get(key)),
        setStorageSync: vi.fn((key: string, value: unknown) => values.set(key, value)),
        removeStorageSync: vi.fn((key: string) => values.delete(key)),
        uploadFile: vi.fn((options: Record<string, any>) => {
          uploadOptions = options
          return { abort, onProgressUpdate: (handler: typeof progress) => { progress = handler } }
        }),
      },
    })
  })

  it('matches the existing multipart contract and forwards progress', async () => {
    setSessionToken('session-token')
    const onProgress = vi.fn()
    const task = uploadAnalysis({
      imagePath: '/tmp/meal.jpg',
      processingMode: 'ai',
      occurredAt: '2026-09-14T12:00:00+08:00',
      mealType: 'lunch',
      description: '鸡胸肉和米饭，少油',
      aiConsentVersion: '2026-09-10',
      idempotencyKey: 'analysis-key',
    }, onProgress)

    expect(uploadOptions).toMatchObject({
      url: 'https://develop-api.formtally.invalid/v1/meal-analyses',
      filePath: '/tmp/meal.jpg',
      name: 'image',
      formData: {
        processingMode: 'ai',
        occurredAt: '2026-09-14T12:00:00+08:00',
        mealType: 'lunch',
        description: '鸡胸肉和米饭，少油',
        aiConsentVersion: '2026-09-10',
      },
    })
    expect(uploadOptions.header.Authorization).toBe('Bearer session-token')
    expect(uploadOptions.header['Idempotency-Key']).toBe('analysis-key')
    progress?.({ progress: 48 })
    expect(onProgress).toHaveBeenCalledWith(48)

    uploadOptions.success({ statusCode: 201, data: JSON.stringify({ analysis: { id: 'analysis-1' } }) })
    await expect(task.promise).resolves.toEqual({ analysis: { id: 'analysis-1' } })
  })

  it('keeps the supplied idempotency key stable and exposes cancellation', () => {
    const input = { imagePath: '/tmp/meal.jpg', processingMode: 'manual' as const, occurredAt: '2026-09-14T12:00:00+08:00', mealType: 'lunch', idempotencyKey: 'stable-key' }
    const first = uploadAnalysis(input)
    expect(uploadOptions.header['Idempotency-Key']).toBe('stable-key')
    const second = uploadAnalysis(input)
    expect(uploadOptions.header['Idempotency-Key']).toBe('stable-key')
    second.cancel()
    expect(abort).toHaveBeenCalledOnce()
    first.cancel()
  })

  it('maps API and transport failures', async () => {
    const httpTask = uploadAnalysis({ imagePath: '/tmp/meal.jpg', processingMode: 'ai', occurredAt: '2026-09-14T12:00:00+08:00', mealType: 'lunch', idempotencyKey: 'key-1' })
    uploadOptions.success({ statusCode: 413, data: JSON.stringify({ error: { code: 'IMAGE_TOO_LARGE', message: '图片过大', requestId: 'req-3' } }) })
    await expect(httpTask.promise).rejects.toBeInstanceOf(ApiError)

    const networkTask = uploadAnalysis({ imagePath: '/tmp/meal.jpg', processingMode: 'ai', occurredAt: '2026-09-14T12:00:00+08:00', mealType: 'lunch', idempotencyKey: 'key-2' })
    uploadOptions.fail({ errMsg: 'uploadFile:fail timeout' })
    await expect(networkTask.promise).rejects.toBeInstanceOf(NetworkError)
  })
})
