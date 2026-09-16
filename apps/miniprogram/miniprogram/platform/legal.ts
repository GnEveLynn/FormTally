export function openPrivacyContract(): void {
  if (typeof wx.openPrivacyContract !== 'function') {
    wx.showToast({ title: '请更新微信后查看隐私政策', icon: 'none' })
    return
  }
  wx.openPrivacyContract({
    fail: () => wx.showToast({ title: '隐私政策暂时无法打开', icon: 'none' }),
  })
}
