import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('me page layout', () => {
  it('groups profile, health settings, and account services into coordinated cards', () => {
    const template = readFileSync(resolve(import.meta.dirname, 'index.wxml'), 'utf8')

    expect(template).toContain('管理资料、目标与账户')
    expect(template).toContain('健康设置')
    expect(template).toContain('账户与服务')
    expect(template).toContain('数据管理')
    expect(template).toContain('退出登录')
  })
})
