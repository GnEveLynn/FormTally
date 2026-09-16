import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('privacy entry points', () => {
  let page: Record<string, any>

  beforeEach(() => {
    vi.resetModules()
    Object.assign(globalThis, {
      Page: vi.fn((definition: Record<string, any>) => { page = definition }),
      wx: { openPrivacyContract: vi.fn(), showToast: vi.fn() },
    })
  })

  it('opens the privacy guide from the login page', async () => {
    await import('../login/index')
    page.openPrivacyContract()
    expect(globalThis.wx.openPrivacyContract).toHaveBeenCalledOnce()
  })

  it('opens the privacy guide from the account page', async () => {
    await import('../me/index')
    page.openPrivacyContract()
    expect(globalThis.wx.openPrivacyContract).toHaveBeenCalledOnce()
  })
})
