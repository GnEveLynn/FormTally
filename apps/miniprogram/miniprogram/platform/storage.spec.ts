import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSessionToken, getSessionToken, setSessionToken } from './storage'

describe('session token storage', () => {
  const values = new Map<string, unknown>()

  beforeEach(() => {
    values.clear()
    Object.assign(globalThis, {
      wx: {
        getStorageSync: vi.fn((key: string) => values.get(key)),
        setStorageSync: vi.fn((key: string, value: unknown) => values.set(key, value)),
        removeStorageSync: vi.fn((key: string) => values.delete(key)),
      },
    })
  })

  it('round-trips only non-empty session tokens', () => {
    expect(getSessionToken()).toBeNull()
    setSessionToken('session-token')
    expect(getSessionToken()).toBe('session-token')
    clearSessionToken()
    expect(getSessionToken()).toBeNull()
  })

  it('ignores malformed persisted values', () => {
    values.set('formtally.sessionToken', { token: 'not-a-string-value' })
    expect(getSessionToken()).toBeNull()
  })
})
