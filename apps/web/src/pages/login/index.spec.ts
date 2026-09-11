import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import * as authApi from '../../api/auth'
import { ApiError } from '../../api/http'
import LoginPage from './index.vue'

vi.mock('../../api/auth', () => ({
  requestLoginCode: vi.fn(),
  login: vi.fn(),
}))

describe('登录页', () => {
  afterEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('手机号无效时指出字段问题且不请求验证码', async () => {
    const wrapper = mount(LoginPage)
    await wrapper.get('input[name="phone"]').setValue('12812345678')
    await wrapper.get('[data-action="send-code"]').trigger('click')

    expect(wrapper.get('#phone-error').text()).toContain('手机号')
    expect(authApi.requestLoginCode).not.toHaveBeenCalled()
  })

  it('发送成功后按服务端秒数倒计时', async () => {
    vi.useFakeTimers()
    vi.mocked(authApi.requestLoginCode).mockResolvedValue({
      verification: {
        requestId: 'verify_123',
        expiresInSeconds: 300,
        retryAfterSeconds: 60,
      },
    })
    const wrapper = mount(LoginPage)
    await wrapper.get('input[name="phone"]').setValue('13812345678')
    await wrapper.get('[data-action="send-code"]').trigger('click')
    await Promise.resolve()

    expect(wrapper.get('[data-action="send-code"]').text()).toContain('60 秒')
    vi.advanceTimersByTime(1000)
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-action="send-code"]').text()).toContain('59 秒')
  })

  it('把手机号 API 错误关联回手机号字段', async () => {
    vi.mocked(authApi.requestLoginCode).mockRejectedValue(
      new ApiError(422, 'PHONE_UNSUPPORTED', '请输入有效的中国大陆手机号', 'req_1'),
    )
    const wrapper = mount(LoginPage)
    await wrapper.get('input[name="phone"]').setValue('13812345678')
    await wrapper.get('[data-action="send-code"]').trigger('click')
    await Promise.resolve()

    expect(wrapper.get('#phone-error').text()).toBe('请输入有效的中国大陆手机号')
  })

  it('未同意协议时阻止登录并关联协议错误', async () => {
    const wrapper = mount(LoginPage)
    await wrapper.get('input[name="phone"]').setValue('13812345678')
    await wrapper.get('input[name="code"]').setValue('123456')
    await wrapper.get('form').trigger('submit')

    expect(wrapper.get('#agreements-error').text()).toContain('协议')
    expect(wrapper.get('input[name="agreements"]').attributes('aria-describedby')).toBe('agreements-error')
    expect(authApi.login).not.toHaveBeenCalled()
  })
})
