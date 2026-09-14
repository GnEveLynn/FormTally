import * as goals from '../../services/goals'
import * as profile from '../../services/profile'
import { accountStore } from '../../stores/account'
import { sessionStore } from '../../stores/session'

Page({
  data: { phone: '', profileSummary: '正在加载…', goalSummary: '正在加载…', error: '' },
  async onShow() { this.setData({ phone: sessionStore.state.data?.user.phoneMasked ?? '' }); try { const [p, g] = await Promise.all([profile.getProfile(), goals.getGoals()]); const target = g.pendingTarget ?? g.activeTarget; this.setData({ profileSummary: p.profile ? `${p.profile.weightKg} kg · ${p.profile.heightCm} cm · ${p.profile.timezone}` : '尚未设置', goalSummary: target ? `${target.target.energyKcal} 千卡 · 蛋白质 ${target.target.proteinGrams}g` : '尚未设置' }) } catch { this.setData({ profileSummary: '暂时无法加载', goalSummary: '暂时无法加载' }) } },
  profile() { wx.navigateTo({ url: '/pages/profile/index' }) }, goals() { wx.navigateTo({ url: '/pages/goals/index' }) }, remove() { wx.navigateTo({ url: '/pages/delete-account/index' }) },
  async logout() { try { await accountStore.logout() } finally { wx.reLaunch({ url: '/pages/login/index' }) } },
})
