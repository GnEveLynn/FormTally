import type { WeChatPhoneBindingResponse, WeChatSessionResult } from '@formtally/api-contract/auth'
import { request, NetworkError } from '../services/http'
import { setSessionToken } from './storage'

export interface Agreements {
  termsVersion: string
  privacyVersion: string
}

export function freshWeChatLoginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    wx.login({
      success: ({ code }) => code ? resolve(code) : reject(new NetworkError('wx.login 失败：未返回登录凭证')),
      fail: ({ errMsg }) => reject(new NetworkError(`wx.login 失败：${errMsg || '未知错误'}`)),
    })
  })
}

export async function weChatLogin(agreements: Agreements): Promise<WeChatSessionResult> {
  const loginCode = await freshWeChatLoginCode()
  let result: WeChatSessionResult
  try {
    result = await request<WeChatSessionResult>('/v1/auth/wechat/sessions', {
      method: 'POST',
      body: { loginCode, agreements },
    })
  } catch (error) {
    if (error instanceof NetworkError) throw new NetworkError(`wx.request 失败：${error.nativeMessage || '未知错误'}`)
    throw error
  }
  setSessionToken(result.token)
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
