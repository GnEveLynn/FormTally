import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('today record dialog layout', () => {
  it('keeps a clearly named confirmation action above the custom tab bar', () => {
    const directory = resolve(import.meta.dirname)
    const template = readFileSync(resolve(directory, 'index.wxml'), 'utf8')
    const styles = readFileSync(resolve(directory, 'index.wxss'), 'utf8')
    expect(template).toContain('确认并生成估算')
    expect(styles).toMatch(/\.record-dialog\s*\{[^}]*padding:[^;}]*180rpx/)
  })
})
