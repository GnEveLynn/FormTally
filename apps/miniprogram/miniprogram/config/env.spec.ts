import { beforeEach, describe, expect, it } from 'vitest'
import { apiBaseUrl, currentApiBaseUrl } from './env'

describe('mini-program API environment', () => {
  beforeEach(() => {
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
        getStorageSync: () => 'http://127.0.0.1:8080',
      },
    })
  })

  it.each(['develop', 'trial', 'release'] as const)('%s has an explicit HTTPS default', (environment) => {
    expect(apiBaseUrl(environment)).toMatch(/^https:\/\//)
  })

  it('uses the public API domain for trial and release builds', () => {
    expect(apiBaseUrl('trial')).toBe('https://api.hzcoder.xyz')
    expect(apiBaseUrl('release')).toBe('https://api.hzcoder.xyz')
  })

  it('allows an explicit localhost URL only in the development simulator', () => {
    expect(apiBaseUrl('develop', 'http://127.0.0.1:8080')).toBe('http://127.0.0.1:8080')
    expect(() => apiBaseUrl('trial', 'http://127.0.0.1:8080')).toThrow('HTTPS')
    expect(() => apiBaseUrl('release', 'http://localhost:8080')).toThrow('HTTPS')
  })

  it('rejects implicit, credentialed, and path-bearing base URLs', () => {
    for (const value of ['', '//api.example.com', 'https://user@example.com', 'https://api.example.com/v1']) {
      expect(() => apiBaseUrl('develop', value)).toThrow()
    }
  })

  it('uses the explicit localhost override from development simulator storage', () => {
    expect(currentApiBaseUrl()).toBe('http://127.0.0.1:8080')
  })
})
