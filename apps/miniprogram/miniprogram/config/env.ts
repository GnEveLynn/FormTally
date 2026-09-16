export type MiniProgramEnvironment = 'develop' | 'trial' | 'release'

const defaults: Record<MiniProgramEnvironment, string> = {
  develop: 'https://develop-api.formtally.invalid',
  trial: 'https://api.hzcoder.xyz',
  release: 'https://api.hzcoder.xyz',
}

export function apiBaseUrl(environment: MiniProgramEnvironment, explicit = defaults[environment]): string {
  const match = /^(https?):\/\/([^/?#]+)\/?$/.exec(explicit)
  if (!match) throw new Error('API base URL 无效')
  const [, protocol, authority] = match
  if (authority.includes('@')) {
    throw new Error('API base URL 不能包含凭据、路径、查询或片段')
  }
  const hostname = authority.replace(/:\d+$/, '')
  const localDevelopment = environment === 'develop' && protocol === 'http' && ['127.0.0.1', 'localhost'].includes(hostname)
  if (protocol !== 'https' && !localDevelopment) throw new Error('体验版和正式版 API 必须使用 HTTPS')
  return explicit.replace(/\/$/, '')
}

export function currentApiBaseUrl(): string {
  const environment = wx.getAccountInfoSync().miniProgram.envVersion || 'release'
  const developmentOverride = environment === 'develop' ? wx.getStorageSync('formtally.developApiBaseUrl') : undefined
  return apiBaseUrl(environment, typeof developmentOverride === 'string' && developmentOverride ? developmentOverride : undefined)
}
