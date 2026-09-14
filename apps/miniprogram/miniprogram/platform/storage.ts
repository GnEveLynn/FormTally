const sessionTokenKey = 'formtally.sessionToken'

export function getSessionToken(): string | null {
  const value = wx.getStorageSync(sessionTokenKey)
  return typeof value === 'string' && value.length > 0 ? value : null
}

export function setSessionToken(token: string): void {
  if (!token) throw new Error('session token 不能为空')
  wx.setStorageSync(sessionTokenKey, token)
}

export function clearSessionToken(): void {
  wx.removeStorageSync(sessionTokenKey)
}
