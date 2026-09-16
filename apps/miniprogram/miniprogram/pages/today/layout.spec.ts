import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('today record dialog layout', () => {
  it('opens on the browsable today page instead of login', () => {
    const app = JSON.parse(readFileSync(resolve(import.meta.dirname, '../../app.json'), 'utf8')) as { pages: string[] }
    expect(app.pages[0]).toBe('pages/today/index')
  })

  it('returns to the browsable home after logout', () => {
    const source = readFileSync(resolve(import.meta.dirname, '../me/index.ts'), 'utf8')
    expect(source).toContain("wx.reLaunch({ url: '/pages/today/index' })")
  })

  it('keeps a clearly named confirmation action above the custom tab bar', () => {
    const directory = resolve(import.meta.dirname)
    const template = readFileSync(resolve(directory, 'index.wxml'), 'utf8')
    const styles = readFileSync(resolve(directory, 'index.wxss'), 'utf8')
    expect(template).toContain('确认并生成估算')
    expect(styles).toMatch(/\.record-dialog\s*\{[^}]*padding:[^;}]*180rpx/)
  })
})
