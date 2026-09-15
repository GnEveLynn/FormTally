import { chooseAndProcessMealImage } from '../../platform/media'
import { mealDraftStore } from '../../stores/meal-draft'
import { mediaFailureView } from './model'

const mealTypes = ['breakfast', 'lunch', 'dinner', 'snack'] as const

Page({
  data: {
    mealLabels: ['早餐', '午餐', '晚餐', '加餐'],
    mealTypeIndex: 1,
    imagePath: '',
    busy: false,
    error: '',
    offerAlbum: false,
    description: '',
  },

  onShow() {
    const mealTypeIndex = Math.max(0, mealTypes.indexOf(mealDraftStore.state.mealType as typeof mealTypes[number]))
    this.setData({ imagePath: mealDraftStore.state.image?.path ?? '', mealTypeIndex, description: mealDraftStore.state.description })
  },

  onDescriptionInput(event: { detail: { value: string } }) {
    mealDraftStore.state.description = event.detail.value
    this.setData({ description: event.detail.value })
  },

  onMealTypeChange(event: { detail: { value: string } }) {
    const mealTypeIndex = Number(event.detail.value)
    mealDraftStore.state.mealType = mealTypes[mealTypeIndex] ?? 'lunch'
    this.setData({ mealTypeIndex })
  },

  chooseCamera() { void this.choose('camera') },
  chooseAlbum() { void this.choose('album') },

  async choose(source: 'camera' | 'album') {
    if (this.data.busy) return
    this.setData({ busy: true, error: '', offerAlbum: false })
    try {
      const image = await chooseAndProcessMealImage(source)
      mealDraftStore.selectImage(image)
      this.setData({ imagePath: image.path })
    } catch (error) {
      const view = mediaFailureView(error)
      this.setData({ error: view.message, offerAlbum: view.offerAlbum })
    } finally {
      this.setData({ busy: false })
    }
  },

  startAI() {
    if (this.data.busy || !mealDraftStore.state.image) return
    mealDraftStore.state.mode = 'ai'
    mealDraftStore.state.consentVersion = '2026-09-10'
    wx.navigateTo({ url: '/pages/meal-analyzing/index' })
  },

  startManual() {
    if (this.data.busy) return
    mealDraftStore.startManual()
    wx.navigateTo({ url: '/pages/meal-manual/index' })
  },

  abandon() {
    mealDraftStore.abandon()
    this.setData({ imagePath: '', error: '', offerAlbum: false })
  },
})
