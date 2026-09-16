import { sessionStore } from '../stores/session'

const items = [
  { path: '/pages/today/index', text: '今日', icon: '⌂' },
  { path: '/pages/meal-capture/index', text: '记录饮食', icon: '▣' },
  { path: '/pages/history/index', text: '历史', icon: '▤' },
  { path: '/pages/me/index', text: '我的', icon: '♙' },
]

Component({
  data: { items, selected: 0 },
  attached() {
    const pages = getCurrentPages()
    const route = pages[pages.length - 1]?.route
    this.setData({ selected: route === 'pages/history/index' ? 2 : route === 'pages/me/index' ? 3 : 0 })
  },
  methods: {
    async switchTab(event: WechatMiniprogram.TouchEvent) {
      const index = Number(event.currentTarget.dataset.index)
      if (index !== 0) {
        await sessionStore.restore()
        if (sessionStore.state.status !== 'authenticated') {
          wx.navigateTo({ url: '/pages/login/index' })
          return
        }
      }
      if (index !== 1) { wx.switchTab({ url: String(event.currentTarget.dataset.path) }); return }
      const open = () => {
        const pages = getCurrentPages()
        const page = pages[pages.length - 1] as (WechatMiniprogram.Page.Instance<Record<string, unknown>, Record<string, unknown>> & { openRecord?: () => void }) | undefined
        page?.openRecord?.()
      }
      const pages = getCurrentPages()
      if (pages[pages.length - 1]?.route === 'pages/today/index') open()
      else wx.switchTab({ url: '/pages/today/index', success: () => setTimeout(open, 0) })
    },
  },
})
