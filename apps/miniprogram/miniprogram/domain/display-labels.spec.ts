import { describe, expect, it } from 'vitest'
import { displayLabel } from './display-labels'

describe('user-facing enum labels', () => {
  it.each([
    ['male', '男性'],
    ['moderate', '中度'],
    ['automatic', '自动估算'],
    ['fat_loss', '减脂'],
    ['standard', '标准'],
    ['lunch', '午餐'],
    ['Asia/Shanghai', '中国标准时间（上海）'],
    ['future_value', '未知'],
  ])('displays %s as %s', (value, label) => {
    expect(displayLabel(value)).toBe(label)
  })
})
