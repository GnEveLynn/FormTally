import { beforeEach, describe, expect, it, vi } from 'vitest'
import { openPrivacyContract } from './legal'

describe('privacy guide entry', () => {
  beforeEach(() => {
    Object.assign(globalThis, {
      wx: {
        openPrivacyContract: vi.fn(),
        showToast: vi.fn(),
      },
    })
  })

  it('opens the mini-program privacy guide provided by WeChat', () => {
    openPrivacyContract()
    expect(globalThis.wx.openPrivacyContract).toHaveBeenCalledOnce()
  })

  it('shows an error when the privacy guide cannot be opened', () => {
    ;(globalThis.wx.openPrivacyContract as any).mockImplementationOnce(({ fail }: Record<string, any>) => fail({ errMsg: 'openPrivacyContract:fail' }))
    openPrivacyContract()
    expect(globalThis.wx.showToast).toHaveBeenCalledWith(expect.objectContaining({ icon: 'none' }))
  })
})
