import { expect, test, type Page, type Route } from '@playwright/test'

const authenticatedSession = {
  session: { expiresAt: '2026-10-10T10:00:00+08:00' },
  user: { id: 'user_123', phoneMasked: '+86 138****5678', onboardingStatus: 'completed' },
  consents: {
    termsVersion: '2026-09-10',
    privacyVersion: '2026-09-10',
    aiImageProcessingVersion: null,
    currentAiImageProcessingVersion: '2026-09-10',
  },
}

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
}

async function mockSession(page: Page, handler: (route: Route) => Promise<void>) {
  await page.route('**/v1/auth/session', handler)
  await page.route('**/v1/days/*', route => json(route, 503, { error: { code: 'DEPENDENCY_UNAVAILABLE', message: '测试中未加载日期数据', requestId: 'req_day' } }))
}

test('未登录用户打开 H5 后看到产品说明与登录入口', async ({ page }) => {
  await mockSession(page, (route) =>
    json(route, 401, { error: { code: 'UNAUTHENTICATED', message: '请先登录', requestId: 'req_1' } }),
  )

  await page.goto('/')

  await expect(page).toHaveURL(/\/welcome$/)
  await expect(page.getByRole('heading', { name: '看见每一餐，了解每一天' })).toBeVisible()
  await expect(page.getByRole('link', { name: '开始使用' })).toBeVisible()
})

test('有效会话重新打开 H5 后直接恢复到今日页', async ({ page }) => {
  await mockSession(page, (route) => json(route, 200, authenticatedSession))

  await page.goto('/')

  await expect(page).toHaveURL(/\/today$/)
  await expect(page.getByRole('heading', { name: '今天' })).toBeVisible()
})

test('退出后不能再次进入受保护页面', async ({ page }) => {
  let loggedIn = true
  await mockSession(page, async (route) => {
    if (route.request().method() === 'DELETE') {
      loggedIn = false
      await route.fulfill({ status: 204 })
      return
    }
    await (loggedIn
      ? json(route, 200, authenticatedSession)
      : json(route, 401, { error: { code: 'UNAUTHENTICATED', message: '请先登录', requestId: 'req_2' } }))
  })

  await page.goto('/')
  await page.getByRole('button', { name: '退出登录' }).click()
  await expect(page).toHaveURL(/\/welcome$/)

  await page.goto('/today')
  await expect(page).toHaveURL(/\/welcome$/)
})

test('会话服务失败时显示重试，而不是跳到未登录页', async ({ page }) => {
  let attempts = 0
  await mockSession(page, (route) => {
    attempts += 1
    return attempts === 1
      ? json(route, 503, { error: { code: 'DEPENDENCY_UNAVAILABLE', message: '服务暂时不可用', requestId: 'req_3' } })
      : json(route, 200, authenticatedSession)
  })

  await page.goto('/')
  await expect(page.getByRole('heading', { name: '暂时无法连接' })).toBeVisible()
  await expect(page).toHaveURL(/\/$/)
  await page.getByRole('button', { name: '重试' }).click()
  await expect(page).toHaveURL(/\/today$/)
  expect(attempts).toBe(2)
})
