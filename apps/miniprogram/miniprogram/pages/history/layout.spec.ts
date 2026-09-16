import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('history page layout', () => {
  it('presents calendar, nutrition summary, and meal records as coordinated cards', () => {
    const directory = resolve(import.meta.dirname)
    const template = readFileSync(resolve(directory, 'index.wxml'), 'utf8')

    expect(template).toContain('历史记录')
    expect(template).toContain('回到今天')
    expect(template).toContain('营养概览')
    expect(template).toContain('当日餐次')
    expect(template).toContain('calendar-grid')
  })
})
