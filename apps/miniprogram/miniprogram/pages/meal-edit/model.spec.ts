import type { MealItem } from '@formtally/api-contract/analyses'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { addMealItem, changeLocalDate, changeLocalTime, fromRFC3339, removeMealItem, toRFC3339 } from './model'

const item = (name: string): MealItem => ({ draftItemId: null, name, grams: 100, nutrition: { energyKcal: 1, proteinGrams: 1, carbGrams: 1, fatGrams: 1 }, basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null })

describe('meal edit model', () => {
  afterEach(() => vi.restoreAllMocks())
  it('adds and removes editable food items without mutating the source', () => {
    const original = [item('米饭')]
    const added = addMealItem(original)
    expect(added).toHaveLength(2)
    expect(added[1]).toMatchObject({ name: '', grams: 100, origin: 'manual' })
    expect(original).toHaveLength(1)
    expect(removeMealItem(added, 0).map(({ name }) => name)).toEqual([''])
  })

  it('converts RFC3339 instants to device-local wall time', () => {
    expect(fromRFC3339('2026-09-14T04:30:00Z', 8 * 60)).toBe('2026-09-14T12:30')
    expect(fromRFC3339('2026-09-14T12:30:00+08:00', 8 * 60)).toBe('2026-09-14T12:30')
  })

  it('converts device-local wall time to RFC3339 with its offset', () => {
    expect(toRFC3339('2026-09-14T12:30', 8 * 60)).toBe('2026-09-14T12:30:00+08:00')
    expect(toRFC3339('2026-09-14T12:30', 0)).toBe('2026-09-14T12:30:00+00:00')
  })

  it('changes date and time independently', () => {
    expect(changeLocalDate('2026-09-14T12:30', '2026-09-16')).toBe('2026-09-16T12:30')
    expect(changeLocalTime('2026-09-14T12:30', '18:45')).toBe('2026-09-14T18:45')
  })

  it('derives default offsets from the target date instead of the current date', () => {
    vi.spyOn(Date.prototype, 'getTimezoneOffset').mockImplementation(function () { return this.getMonth() === 0 ? 300 : 240 })
    expect(fromRFC3339('2026-01-15T17:30:00Z')).toBe('2026-01-15T12:30')
    expect(toRFC3339('2026-01-15T12:30')).toBe('2026-01-15T12:30:00-05:00')
  })
})
