import type { DaySummary } from '@formtally/api-contract/days'
import { chooseAndProcessMealImage } from '../../platform/media'
import { mealDraftStore } from '../../stores/meal-draft'
import { dateTabsAt, todayStore } from '../../stores/today'
import { mediaFailureView } from '../meal-capture/model'

const mealLabels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }
const mealTypes = ['breakfast', 'lunch', 'dinner', 'snack'] as const

function displayGroups(day: DaySummary | null) {
  return mealTypes.map((mealType) => {
    const group = day?.mealGroups.find((candidate) => candidate.mealType === mealType)
    return { mealType, label: mealLabels[mealType], meals: group?.meals.map((meal) => ({ ...meal, time: new Date(meal.occurredAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) })) ?? [] }
  })
}

Page({
  data: {
    status: 'idle',
    localDate: '',
    day: null as DaySummary | null,
    groups: [] as ReturnType<typeof displayGroups>,
    error: '',
    dateTabs: [] as ReturnType<typeof dateTabsAt>,
    recordOpen: false,
    mealLabels: ['早餐', '午餐', '晚餐', '加餐'],
    mealTypeIndex: 1,
    imagePath: '',
    description: '',
    busy: false,
    offerAlbum: false,
  },

  async onShow() {
    const pending = todayStore.load()
    this.sync()
    await pending
    this.sync()
  },

  sync() {
    this.setData({
      status: todayStore.state.status,
      localDate: todayStore.state.localDate,
      day: todayStore.state.day,
      groups: displayGroups(todayStore.state.day),
      error: todayStore.state.error,
      dateTabs: dateTabsAt(todayStore.state.todayDate),
    })
  },

  async retry() {
    const pending = todayStore.retry()
    this.sync()
    await pending
    this.sync()
  },

  async selectDate(event: WechatMiniprogram.TouchEvent) {
    const date = String(event.currentTarget.dataset.date)
    const pending = todayStore.load(date)
    this.sync()
    await pending
    this.sync()
  },

  recordMeal() {
    this.openRecord()
  },

  openRecord(event?: WechatMiniprogram.TouchEvent) {
    const type = String(event?.currentTarget.dataset.type ?? mealDraftStore.state.mealType)
    const index = Math.max(0, mealTypes.indexOf(type as typeof mealTypes[number]))
    mealDraftStore.state.mealType = mealTypes[index] ?? 'lunch'
    this.setData({ recordOpen: true, mealTypeIndex: index, imagePath: mealDraftStore.state.image?.path ?? '', description: mealDraftStore.state.description })
  },
  closeRecord() { this.setData({ recordOpen: false, error: '', offerAlbum: false }) },
  stopPropagation() {},
  onMealTypeChange(event: { detail: { value: string } }) { const index = Number(event.detail.value); mealDraftStore.state.mealType = mealTypes[index] ?? 'lunch'; this.setData({ mealTypeIndex: index }) },
  onDescriptionInput(event: { detail: { value: string } }) { mealDraftStore.state.description = event.detail.value; this.setData({ description: event.detail.value }) },
  chooseCamera() { void this.choose('camera') },
  chooseAlbum() { void this.choose('album') },
  async choose(source: 'camera' | 'album') {
    if (this.data.busy) return
    this.setData({ busy: true, error: '', offerAlbum: false })
    try { const image = await chooseAndProcessMealImage(source); mealDraftStore.selectImage(image); this.setData({ imagePath: image.path }) }
    catch (error) { const view = mediaFailureView(error); this.setData({ error: view.message, offerAlbum: view.offerAlbum }) }
    finally { this.setData({ busy: false }) }
  },
  startAI() {
    if (!mealDraftStore.state.image) { this.setData({ error: '请先拍照或选择食物照片' }); return }
    mealDraftStore.state.mode = 'ai'; mealDraftStore.state.consentVersion = '2026-09-10'
    wx.navigateTo({ url: '/pages/meal-analyzing/index' })
  },
  startManual() { mealDraftStore.startManual(); wx.navigateTo({ url: '/pages/meal-manual/index' }) },
})
