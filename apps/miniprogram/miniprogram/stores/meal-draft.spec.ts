import type { Analysis, MealItem } from '@formtally/api-contract/analyses'
import { describe, expect, it, vi } from 'vitest'
import { createMealDraftStore } from './meal-draft'

const image = { path: '/tmp/meal.jpg', size: 300_000, width: 1200, height: 900, mimeType: 'image/jpeg' as const }
const manualItem: MealItem = {
  draftItemId: null, name: '米饭', grams: 100,
  nutrition: { energyKcal: 116, proteinGrams: 2.6, carbGrams: 25.9, fatGrams: 0.3 },
  basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null,
}
const failedAnalysis = {
  id: 'analysis_1', status: 'failed', processingMode: 'ai', occurredAt: '2026-09-14T12:00:00+08:00',
  localDate: '2026-09-14', mealType: 'lunch', image: { url: '', expiresAt: '', width: 1200, height: 900, mimeType: 'image/jpeg' },
  items: [], warnings: [], failure: { code: 'AI_TIMEOUT', message: '分析超时', retryable: true }, mealId: null,
  expiresAt: '', revision: 1, createdAt: '',
} satisfies Analysis

function client(overrides: Record<string, unknown> = {}) {
  return {
    uploadAnalysis: vi.fn(() => ({ promise: Promise.resolve({ analysis: { ...failedAnalysis, status: 'review_required', failure: null } }), cancel: vi.fn() })),
    retryAnalysis: vi.fn(() => Promise.resolve({ analysis: { ...failedAnalysis, status: 'review_required', failure: null, revision: 2 } })),
    saveMeal: vi.fn(() => Promise.resolve({ meal: { id: 'meal_1' }, affectedLocalDates: ['2026-09-14'] })),
    ...overrides,
  }
}

describe('mini-program meal draft store', () => {
  it('keeps one analysis key and the image across transport retries', async () => {
    const uploadAnalysis = vi.fn()
      .mockReturnValueOnce({ promise: Promise.reject(new Error('网络失败')), cancel: vi.fn() })
      .mockReturnValueOnce({ promise: Promise.resolve({ analysis: failedAnalysis }), cancel: vi.fn() })
    const api = client({ uploadAnalysis })
    const keys = ['analysis-key', 'retry-key']
    const store = createMealDraftStore(api as any, async () => keys.shift()!)
    store.selectImage(image)
    store.state.description = '鸡胸肉和米饭，少油'
    await store.analyze()
    await store.analyze()

    expect(uploadAnalysis.mock.calls[0]![0].idempotencyKey).toBe('analysis-key')
    expect(uploadAnalysis.mock.calls[1]![0].idempotencyKey).toBe('analysis-key')
    expect(store.state.image).toEqual(image)
    expect(uploadAnalysis.mock.calls[0]![0].description).toBe('鸡胸肉和米饭，少油')
    expect(store.state.status).toBe('failed')
  })

  it('reuses a retry key for a failed server draft and changes keys after abandon', async () => {
    const retryAnalysis = vi.fn()
      .mockRejectedValueOnce(new Error('lost'))
      .mockResolvedValueOnce({ analysis: { ...failedAnalysis, status: 'review_required', failure: null, revision: 2 } })
    const api = client({
      uploadAnalysis: vi.fn(() => ({ promise: Promise.resolve({ analysis: failedAnalysis }), cancel: vi.fn() })),
      retryAnalysis,
    })
    const keys = ['create-key', 'retry-key', 'next-create-key']
    const store = createMealDraftStore(api as any, async () => keys.shift()!)
    store.selectImage(image)
    await store.analyze()
    await store.analyze()
    await store.analyze()
    expect(retryAnalysis.mock.calls[0]![2]).toBe('retry-key')
    expect(retryAnalysis.mock.calls[1]![2]).toBe('retry-key')

    store.abandon()
    store.selectImage(image)
    await store.analyze()
    expect(api.uploadAnalysis.mock.calls[1]![0].idempotencyKey).toBe('next-create-key')
  })

  it('validates edits with the shared meal editor and keeps the save key across retries', async () => {
    const saveMeal = vi.fn().mockRejectedValueOnce(new Error('lost')).mockResolvedValueOnce({ meal: { id: 'meal_1' }, affectedLocalDates: ['2026-09-14'] })
    const api = client({ saveMeal })
    const store = createMealDraftStore(api as any, async () => 'save-key')
    store.startManual()
    store.state.occurredAt = '2099-01-01T00:00:00+08:00'
    await expect(store.save(new Date('2026-09-14T00:00:00Z'))).resolves.toBeUndefined()
    expect(store.state.errors.occurredAt).toBe('餐食时间不能位于未来')
    expect(saveMeal).not.toHaveBeenCalled()

    store.state.occurredAt = '2026-09-14T12:00:00+08:00'
    store.state.items = [manualItem]
    await store.save()
    const result = await store.save()
    expect(saveMeal.mock.calls[0]![1]).toBe('save-key')
    expect(saveMeal.mock.calls[1]![1]).toBe('save-key')
    expect(result?.affectedLocalDates).toEqual(['2026-09-14'])
  })

  it('coalesces duplicate analyze and save taps while a request is pending', async () => {
    let finishUpload!: (value: unknown) => void
    const uploadAnalysis = vi.fn(() => ({ promise: new Promise((resolve) => { finishUpload = resolve }), cancel: vi.fn() }))
    const api = client({ uploadAnalysis })
    const store = createMealDraftStore(api as any, async () => 'stable-key')
    store.selectImage(image)
    const first = store.analyze()
    const duplicate = store.analyze()
    await Promise.resolve()
    expect(uploadAnalysis).toHaveBeenCalledOnce()
    finishUpload({ analysis: { ...failedAnalysis, status: 'review_required', failure: null } })
    await Promise.all([first, duplicate])

    let finishSave!: (value: any) => void
    api.saveMeal.mockImplementation(() => new Promise((resolve) => { finishSave = resolve }))
    store.state.items = [manualItem]
    const firstSave = store.save()
    const duplicateSave = store.save()
    await Promise.resolve()
    expect(api.saveMeal).toHaveBeenCalledOnce()
    finishSave({ meal: { id: 'meal_1' }, affectedLocalDates: ['2026-09-14'] })
    await Promise.all([firstSave, duplicateSave])
  })

  it('notifies the analyzing page when upload progress changes', async () => {
    let onProgress!: (percent: number) => void
    let finishUpload!: (value: unknown) => void
    const api = client({
      uploadAnalysis: vi.fn((_input, progress) => {
        onProgress = progress
        return { promise: new Promise((resolve) => { finishUpload = resolve }), cancel: vi.fn() }
      }),
    })
    const store = createMealDraftStore(api as any, async () => 'analysis-key')
    const seen: number[] = []
    store.subscribe(() => seen.push(store.state.progress))
    store.selectImage(image)
    const pending = store.analyze()
    await Promise.resolve()
    onProgress(42)
    expect(seen).toContain(42)
    finishUpload({ analysis: { ...failedAnalysis, status: 'review_required', failure: null } })
    await pending
  })
})
