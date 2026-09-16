import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('legal document page', () => {
  let page: Record<string, any>

  beforeEach(async () => {
    vi.resetModules()
    Object.assign(globalThis, { Page: vi.fn((definition: Record<string, any>) => { page = definition }) })
    await import('./index')
  })

  it('shows the user agreement from a terms link', () => {
    const setData = vi.fn()
    page.onLoad.call({ setData }, { kind: 'terms' })
    expect(setData).toHaveBeenCalledWith(expect.objectContaining({ title: '用户协议' }))
  })

  it('shows the AI image notice from an AI link', () => {
    const setData = vi.fn()
    page.onLoad.call({ setData }, { kind: 'ai' })
    expect(setData).toHaveBeenCalledWith(expect.objectContaining({ title: 'AI 图片处理说明' }))
  })
})
