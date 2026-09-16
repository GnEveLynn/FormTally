import { weChatLogin } from '../../platform/auth'
import { openPrivacyContract } from '../../platform/legal'
import { sessionStore } from '../../stores/session'
import { agreementVersion, nextLoginAction, routeForSession } from './model'

Page({
  data: {
    agreementVersion,
    acceptedAgreements: false,
    busy: false,
    error: '',
  },

  async onLoad() {
    await this.restore()
  },

  async restore() {
    this.setData({ busy: true, error: '' })
    if (sessionStore.state.status === 'failed') await sessionStore.retry()
    else await sessionStore.restore()
    if (sessionStore.state.status === 'authenticated' && sessionStore.state.data) {
      wx.reLaunch({ url: routeForSession(sessionStore.state.data.user.onboardingStatus) })
      return
    }
    if (sessionStore.state.status === 'failed') {
      this.setData({ busy: false, error: sessionStore.state.error })
      return
    }
    this.setData({ busy: false })
  },

  async startWeChatSession() {
    if (!this.data.acceptedAgreements) { this.setData({ error: '请先阅读并同意服务协议与隐私政策' }); return }
    this.setData({ busy: true, error: '' })
    try {
      const result = await weChatLogin({ termsVersion: agreementVersion, privacyVersion: agreementVersion })
      sessionStore.setSession(result)
      const action = nextLoginAction(result)
      if ('route' in action) wx.reLaunch({ url: action.route })
    } catch (error) {
      this.setData({ error: error instanceof Error ? error.message : '登录失败，请重试' })
    } finally {
      this.setData({ busy: false })
    }
  },

  onAgreementChange(event: { detail: { value: string[] } }) {
    this.setData({ acceptedAgreements: event.detail.value.includes('accepted'), error: '' })
  },

  openPrivacyContract,

})
