import { describe, expect, it } from 'vitest'

import { validateImage } from './image'

describe('图片预处理', () => {
  it('接受 JPEG、PNG 和 WebP', () => {
    for (const type of ['image/jpeg', 'image/png', 'image/webp']) {
      expect(validateImage(new File(['image'], 'meal', { type }))).toBeNull()
    }
  })

  it('在分析前拒绝不支持或超过 10MB 的图片', () => {
    expect(validateImage(new File(['image'], 'meal.gif', { type: 'image/gif' }))).toContain('JPEG')
    expect(validateImage(new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'meal.jpg', { type: 'image/jpeg' }))).toContain('10 MB')
  })
})
