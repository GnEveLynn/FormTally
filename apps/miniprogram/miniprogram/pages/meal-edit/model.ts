import type { MealItem } from '@formtally/api-contract/analyses'
import { emptyMealItem } from '../../stores/meal-draft'

export const addMealItem = (items: MealItem[]): MealItem[] => [...items, emptyMealItem()]
export const removeMealItem = (items: MealItem[], index: number): MealItem[] => items.filter((_, itemIndex) => itemIndex !== index)
export function fromRFC3339(value: string, offsetMinutes?: number): string { const target = new Date(value); const offset = offsetMinutes ?? -target.getTimezoneOffset(); return new Date(target.getTime() + offset * 60_000).toISOString().slice(0, 16) }
export function toRFC3339(value: string, offsetMinutes?: number): string { const offset = offsetMinutes ?? -new Date(value).getTimezoneOffset(); const sign = offset >= 0 ? '+' : '-'; const absolute = Math.abs(offset); return `${value}:00${sign}${String(Math.floor(absolute / 60)).padStart(2, '0')}:${String(absolute % 60).padStart(2, '0')}` }
export const changeLocalDate = (value: string, date: string): string => `${date}T${value.slice(11, 16)}`
export const changeLocalTime = (value: string, time: string): string => `${value.slice(0, 10)}T${time}`
