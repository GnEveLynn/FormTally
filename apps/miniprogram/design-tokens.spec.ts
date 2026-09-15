import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const repositoryRoot = resolve(import.meta.dirname, '../..')
const webTokens = resolve(repositoryRoot, 'packages/design-tokens/src/tokens.css')
const miniProgramTokens = resolve(repositoryRoot, 'apps/miniprogram/miniprogram/styles/tokens.wxss')

function buildTokens() {
  execFileSync(process.execPath, ['scripts/build-design-tokens.mjs'], { cwd: repositoryRoot })
  return {
    css: readFileSync(webTokens, 'utf8'),
    wxss: readFileSync(miniProgramTokens, 'utf8'),
  }
}

describe('design token build', () => {
  it('preserves the existing H5 variables and emits WeChat values deterministically', () => {
    const first = buildTokens()
    const second = buildTokens()

    expect(second).toEqual(first)
    expect(first.css).toBe(`:root {
  --ft-color-background: #f7faf8;
  --ft-color-text: #173c35;
  --ft-color-text-muted: #71807a;
  --ft-font-size-title: 2rem;
  --ft-font-size-body: 1rem;
  --ft-space-sm: 0.75rem;
  --ft-space-lg: 1.5rem;
}
`)
    expect(first.wxss).toContain('--ft-font-size-title: 64rpx;')
    expect(first.wxss).toContain('--ft-space-lg: 48rpx;')
  })
})
