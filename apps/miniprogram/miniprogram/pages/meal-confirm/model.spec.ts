import type { MealItem } from '@formtally/api-contract/analyses'
import { describe, expect, it } from 'vitest'
import { reviewModel, saveAndRefresh } from './model'

const item = (name: string, confidence: MealItem['confidence'], assumption: string | null): MealItem => ({
  draftItemId: null, name, grams: 100,
  nutrition: { energyKcal: 100, proteinGrams: 2.2, carbGrams: 3.3, fatGrams: 4.4 },
  basisPer100Grams: null, origin: 'ai', confidence, assumption,
})

describe('meal confirmation presentation', () => {
  it('keeps the AI estimate warning, server warnings, and low-confidence assumptions visible', () => {
    const view = reviewModel([item('米饭', 'high', null), item('汤', 'low', '可能含有食用油')], ['图片部分遮挡'])
    expect(view.estimateNotice).toContain('估算')
    expect(view.warnings).toEqual(['图片部分遮挡'])
    expect(view.lowConfidence).toEqual([{ index: 1, name: '汤', assumption: '可能含有食用油' }])
    expect(view.totals).toEqual({ energyKcal: 200, proteinGrams: 4.4, carbGrams: 6.6, fatGrams: 8.8 })
  })

  it('refreshes affected today data before clearing a saved draft', async () => {
    const events: string[] = []
    const store = {
      save: async () => ({ meal: { id: 'meal_1' }, affectedLocalDates: ['2026-09-14'] }),
      clear: () => events.push('clear'),
    }
    const today = { refreshAfterSave: async (dates: string[]) => events.push(`refresh:${dates.join(',')}`) }
    await expect(saveAndRefresh(store as any, today)).resolves.toBe(true)
    expect(events).toEqual(['refresh:2026-09-14', 'clear'])
  })
})
