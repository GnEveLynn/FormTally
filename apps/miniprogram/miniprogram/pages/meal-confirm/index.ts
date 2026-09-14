import type { MealItem, Nutrition } from '@formtally/api-contract/analyses'
import { mealDraftStore } from '../../stores/meal-draft'
import { todayStore } from '../../stores/today'
import { reviewModel, saveAndRefresh } from './model'

Page({
  data: {
    mode: 'ai',
    items: [] as MealItem[],
    totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 } as Nutrition,
    warnings: [] as string[],
    lowConfidence: [] as ReturnType<typeof reviewModel>['lowConfidence'],
    estimateNotice: '',
    error: '',
    validationError: '',
    busy: false,
  },

  onShow() {
    if (mealDraftStore.state.mode === 'ai' && !mealDraftStore.state.analysis) {
      wx.redirectTo({ url: '/pages/meal-capture/index' })
      return
    }
    this.sync()
  },

  sync() {
    const review = reviewModel(mealDraftStore.state.items, mealDraftStore.state.warnings)
    this.setData({
      mode: mealDraftStore.state.mode,
      items: mealDraftStore.state.items,
      totals: review.totals,
      warnings: review.warnings,
      lowConfidence: review.lowConfidence,
      estimateNotice: review.estimateNotice,
      error: mealDraftStore.state.error,
      validationError: Object.values(mealDraftStore.state.errors)[0] ?? '',
      busy: mealDraftStore.state.status === 'saving',
    })
  },

  onItemInput(event: { currentTarget: { dataset: { index?: number; field?: string } }; detail: { value: string } }) {
    const index = Number(event.currentTarget.dataset.index)
    const field = event.currentTarget.dataset.field
    const current = mealDraftStore.state.items[index]
    if (!current || !field) return
    const item = { ...current, nutrition: { ...current.nutrition }, origin: current.origin === 'ai' ? 'ai_modified' as const : current.origin }
    if (field === 'name') item.name = event.detail.value
    else if (field === 'grams') item.grams = Number(event.detail.value)
    else if (field in item.nutrition) item.nutrition[field as keyof Nutrition] = Number(event.detail.value)
    mealDraftStore.state.items = mealDraftStore.state.items.map((value, itemIndex) => itemIndex === index ? item : value)
    mealDraftStore.state.errors = {}
    this.sync()
  },

  addItem() {
    mealDraftStore.state.items = [...mealDraftStore.state.items, {
      draftItemId: null, name: '', grams: 100,
      nutrition: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 },
      basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null,
    }]
    this.sync()
  },

  removeItem(event: { currentTarget: { dataset: { index?: number } } }) {
    if (mealDraftStore.state.items.length <= 1) return
    const index = Number(event.currentTarget.dataset.index)
    mealDraftStore.state.items = mealDraftStore.state.items.filter((_, itemIndex) => itemIndex !== index)
    this.sync()
  },

  async save() {
    if (this.data.busy) return
    this.setData({ busy: true, error: '', validationError: '' })
    const saved = await saveAndRefresh(mealDraftStore, todayStore)
    if (saved) {
      wx.switchTab({ url: '/pages/today/index' })
      return
    }
    this.sync()
  },
})
