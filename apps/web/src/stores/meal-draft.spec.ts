import { describe, expect, it, vi } from 'vitest'
import { createMealDraftStore } from './meal-draft'
import type { MealItem } from '@formtally/api-contract/analyses'

const item: MealItem = { name: '米饭', grams: 100, nutrition: { energyKcal: 116, proteinGrams: 2.6, carbGrams: 25.9, fatGrams: .3 }, basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null, draftItemId: null }

describe('meal draft store', () => {
  it('reselect and failed requests preserve date meal type image and edits', async () => {
    const api = { createAnalysis: vi.fn().mockRejectedValue(new Error('AI 超时')), saveMeal: vi.fn().mockRejectedValue(new Error('保存失败')) }
    const store = createMealDraftStore(api)
    store.state.occurredAt = '2026-09-11T12:00:00+08:00'
    store.state.mealType = 'lunch'
    store.state.description = '鸡胸肉和米饭，少油'
    store.selectImage(new File(['first'], 'first.jpg', { type: 'image/jpeg' }))
    store.selectImage(new File(['second'], 'second.jpg', { type: 'image/jpeg' }))
    await store.analyze()
    expect(api.createAnalysis.mock.calls[0]![0].description).toBe('鸡胸肉和米饭，少油')
    expect(store.state.status).toBe('failed')
    expect(store.state.occurredAt).toContain('2026-09-11')
    expect(store.state.mealType).toBe('lunch')
    store.state.items = [item]
    await store.save()
    expect(store.state.items[0]?.name).toBe('米饭')
    expect(store.state.status).toBe('save_failed')
  })

  it('reuses one idempotency key across save retries', async () => {
    const saveMeal = vi.fn().mockRejectedValueOnce(new Error('lost')).mockResolvedValueOnce({ meal: { id: 'meal_1' }, affectedLocalDates: ['2026-09-11'] })
    const store = createMealDraftStore({ createAnalysis: vi.fn(), saveMeal })
    store.state.items = [item]
    await store.save()
    await store.save()
    expect(saveMeal.mock.calls[0]![1]).toBe(saveMeal.mock.calls[1]![1])
  })

  it('reuses the create key after a transport failure and uses retry for a failed resource', async () => {
    const failedAnalysis = { id: 'analysis_1', status: 'failed', revision: 1, failure: { code: 'AI_TIMEOUT', message: '超时', retryable: true }, items: [] }
    const createAnalysis = vi.fn().mockRejectedValueOnce(new Error('lost')).mockResolvedValueOnce({ analysis: failedAnalysis })
    const retryAnalysis = vi.fn().mockResolvedValue({ analysis: { ...failedAnalysis, status: 'review_required', revision: 2, failure: null } })
    const store = createMealDraftStore({ createAnalysis, retryAnalysis, saveMeal: vi.fn() } as any)
    store.selectImage(new File(['image'], 'meal.jpg', { type: 'image/jpeg' }))
    await store.analyze()
    await store.analyze()
    expect(createAnalysis.mock.calls[0]![1]).toBe(createAnalysis.mock.calls[1]![1])
    await store.analyze()
    expect(retryAnalysis).toHaveBeenCalledWith('analysis_1', { aiConsentVersion: '', expectedRevision: 1 }, expect.any(String))
  })
})
