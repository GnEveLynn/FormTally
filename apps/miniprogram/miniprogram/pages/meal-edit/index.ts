import type { MealItem } from '@formtally/api-contract/analyses'
import type { Meal } from '@formtally/api-contract/meals'
import { validateMeal } from '../../domain/meal-editor'
import { ApiError } from '../../services/http'
import * as meals from '../../services/meals'
import { historyStore } from '../../stores/history'
import { addMealItem, changeLocalDate, changeLocalTime, fromRFC3339, removeMealItem, toRFC3339 } from './model'

const mealTypes = ['breakfast', 'lunch', 'dinner', 'snack'] as const

Page({
  data: { id: '', meal: null as Meal | null, occurredAt: '', occurredDate: '', occurredTime: '', mealType: 'lunch', mealTypeLabels: ['早餐', '午餐', '晚餐', '加餐'], mealTypeIndex: 1, items: [] as MealItem[], error: '', busy: false },
  onLoad(query: Record<string, string | undefined>) { this.setData({ id: query.id ?? '' }); void this.load() },
  async load() { try { const meal = await meals.getMeal(this.data.id); const occurredAt = fromRFC3339(meal.occurredAt); this.setData({ meal, occurredAt, occurredDate: occurredAt.slice(0, 10), occurredTime: occurredAt.slice(11, 16), mealType: meal.mealType, mealTypeIndex: Math.max(0, mealTypes.findIndex((value) => value === meal.mealType)), items: meal.items.map((item) => ({ ...item, nutrition: { ...item.nutrition } })), error: '' }) } catch (error) { this.setData({ error: error instanceof Error ? error.message : '加载失败' }) } },
  input(event: { currentTarget: { dataset: { index?: number; field?: string } }; detail: { value: string } }) { const index = Number(event.currentTarget.dataset.index); const field = event.currentTarget.dataset.field; const items = this.data.items; if (field === 'name') items[index]!.name = event.detail.value; else if (field) (items[index]!.nutrition as unknown as Record<string, number>)[field] = Number(event.detail.value); this.setData({ items }) },
  grams(event: { currentTarget: { dataset: { index?: number } }; detail: { value: string } }) { const items = this.data.items; items[Number(event.currentTarget.dataset.index)]!.grams = Number(event.detail.value); this.setData({ items }) },
  addItem() { this.setData({ items: addMealItem(this.data.items) }) },
  removeItem(event: { currentTarget: { dataset: { index?: number } } }) { this.setData({ items: removeMealItem(this.data.items, Number(event.currentTarget.dataset.index)) }) },
  occurred(event: { detail: { value: string } }) { this.setData({ occurredDate: event.detail.value, occurredAt: changeLocalDate(this.data.occurredAt, event.detail.value) }) },
  occurredTime(event: { detail: { value: string } }) { this.setData({ occurredTime: event.detail.value, occurredAt: changeLocalTime(this.data.occurredAt, event.detail.value) }) },
  type(event: { detail: { value: string } }) { const mealTypeIndex = Number(event.detail.value); this.setData({ mealType: mealTypes[mealTypeIndex] ?? 'lunch', mealTypeIndex }) },
  async save() { const meal = this.data.meal; if (!meal) return; const occurredAt = toRFC3339(this.data.occurredAt); const errors = validateMeal({ occurredAt, mealType: this.data.mealType, items: this.data.items }); if (Object.keys(errors).length) { this.setData({ error: Object.values(errors)[0] }); return } this.setData({ busy: true, error: '' }); try { const result = await meals.updateMeal(meal.id, { expectedRevision: meal.revision, occurredAt, mealType: this.data.mealType, items: this.data.items }); await historyStore.refreshAfterMutation(result.affectedLocalDates); wx.switchTab({ url: '/pages/history/index' }) } catch (error) { if (error instanceof ApiError && error.code === 'REVISION_CONFLICT') { this.setData({ error: '数据已变化，已重新载入最新内容' }); await this.load() } else this.setData({ error: error instanceof Error ? error.message : '保存失败' }) } finally { this.setData({ busy: false }) } },
})
