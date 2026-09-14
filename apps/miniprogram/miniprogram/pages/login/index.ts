import { bindWeChatPhone, weChatLogin } from '../../platform/auth'
import { sessionStore } from '../../stores/session'
import { agreementVersion, nextLoginAction, phoneAuthorization, routeForSession } from './model'

Page({
  data: {
    agreementVersion,
    acceptedAgreements: false,
    bindingTicket: '',
    needsPhone: false,
    busy: false,
    error: '',
  },

  async onLoad() {
    await this.restoreOrLogin()
  },

  async restoreOrLogin() {
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
    await this.startWeChatSession()
  },

  async startWeChatSession() {
    this.setData({ busy: true, error: '', needsPhone: false, bindingTicket: '' })
    try {
      const result = await weChatLogin()
      if (result.bindingRequired) {
        const action = nextLoginAction(result)
        if ('bindingTicket' in action) this.setData({ needsPhone: true, bindingTicket: action.bindingTicket })
        return
      }
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

  async onGetPhoneNumber(event: { detail: { code?: string; errMsg?: string } }) {
    const authorization = phoneAuthorization({
      acceptedAgreements: this.data.acceptedAgreements,
      code: event.detail.code,
      errMsg: event.detail.errMsg,
    })
    if ('error' in authorization) {
      this.setData({ error: authorization.error })
      return
    }

    this.setData({ busy: true, error: '' })
    try {
      const result = await bindWeChatPhone(this.data.bindingTicket, authorization.code, {
        termsVersion: agreementVersion,
        privacyVersion: agreementVersion,
      })
      sessionStore.setSession(result)
      wx.reLaunch({ url: routeForSession(result.user.onboardingStatus) })
    } catch (error) {
      this.setData({ error: error instanceof Error ? error.message : '手机号验证失败，请重试' })
    } finally {
      this.setData({ busy: false })
    }
  },
})
