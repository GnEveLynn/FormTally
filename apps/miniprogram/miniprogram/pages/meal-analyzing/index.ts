import { mealDraftStore } from '../../stores/meal-draft'

let slowTimer: ReturnType<typeof setTimeout> | undefined
let unsubscribe: (() => void) | undefined

Page({
  data: { imagePath: '', status: 'idle', progress: 0, slow: false, error: '', busy: false },

  onLoad() {
    if (!mealDraftStore.state.image) {
      wx.redirectTo({ url: '/pages/meal-capture/index' })
      return
    }
    unsubscribe = mealDraftStore.subscribe(() => this.sync())
    this.setData({ imagePath: mealDraftStore.state.image.path })
    void this.run()
  },

  onUnload() {
    if (slowTimer) clearTimeout(slowTimer)
    unsubscribe?.()
    unsubscribe = undefined
  },

  sync() {
    this.setData({
      status: mealDraftStore.state.status,
      progress: mealDraftStore.state.progress,
      error: mealDraftStore.state.error,
    })
  },

  async run() {
    if (this.data.busy) return
    this.setData({ busy: true, slow: false, error: '' })
    slowTimer = setTimeout(() => this.setData({ slow: true }), 12_000)
    await mealDraftStore.analyze()
    if (slowTimer) clearTimeout(slowTimer)
    this.setData({ busy: false })
    this.sync()
    if (mealDraftStore.state.status === 'ready') wx.redirectTo({ url: '/pages/meal-confirm/index' })
  },

  retry() { void this.run() },

  manual() {
    if (this.data.busy) return
    mealDraftStore.startManual()
    wx.redirectTo({ url: '/pages/meal-manual/index' })
  },
})
