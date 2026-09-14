import type { WeChatPhoneBindingResponse, WeChatSessionResult } from '@formtally/api-contract/auth'
import { request, NetworkError } from '../services/http'
import { setSessionToken } from './storage'

export interface Agreements {
  termsVersion: string
  privacyVersion: string
}

function loginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    wx.login({
      success: ({ code }) => code ? resolve(code) : reject(new NetworkError()),
      fail: () => reject(new NetworkError()),
    })
  })
}

export async function weChatLogin(): Promise<WeChatSessionResult> {
  const result = await request<WeChatSessionResult>('/v1/auth/wechat/sessions', {
    method: 'POST',
    body: { loginCode: await loginCode() },
  })
  if (!result.bindingRequired) setSessionToken(result.token)
  return result
}

export async function bindWeChatPhone(bindingTicket: string, phoneCode: string, agreements: Agreements): Promise<WeChatPhoneBindingResponse> {
  const result = await request<WeChatPhoneBindingResponse>('/v1/auth/wechat/phone-bindings', {
    method: 'POST',
    body: { bindingTicket, phoneCode, agreements },
  })
  setSessionToken(result.token)
  return result
}
