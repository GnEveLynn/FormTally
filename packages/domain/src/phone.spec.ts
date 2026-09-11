import { describe, expect, it } from 'vitest'

import { normalizeChinaPhone } from './phone'

describe('中国大陆手机号', () => {
  it.each([
    ['13812345678', '+8613812345678'],
    ['+8613812345678', '+8613812345678'],
    [' 138 1234 5678 ', '+8613812345678'],
  ])('将 %s 规范化为 E.164', (input, expected) => {
    expect(normalizeChinaPhone(input)).toBe(expected)
  })

  it.each(['', '12812345678', '1381234567', '+86138123456789'])('拒绝不受支持的号码 %s', (input) => {
    expect(normalizeChinaPhone(input)).toBeNull()
  })
})
