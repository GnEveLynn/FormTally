import { accountStore } from '../../stores/account'
import { sessionStore } from '../../stores/session'

Page({
  data: { hasPhone: false, phone: '', code: '', confirmation: '', sent: false, busy: false, error: '' },
  onLoad() { this.setData({ hasPhone: sessionStore.state.data?.user.phoneMasked != null }) },
  field(event: { currentTarget: { dataset: { field?: string } }; detail: { value: string } }) { const field = event.currentTarget.dataset.field; if (field) this.setData({ [field]: event.detail.value }) },
  async send() { this.setData({ busy: true, error: '' }); const ok = await accountStore.requestDeleteCode(this.data.phone); this.setData({ sent: ok, busy: false, error: accountStore.state.error }) },
  remove() { wx.showModal({ title: '永久删除账户？', content: '账户、资料、目标、饮食记录和图片将进入删除流程，所有会话立即失效。', confirmColor: '#a22b2b', success: async ({ confirm }) => { if (!confirm) return; this.setData({ busy: true, error: '' }); const ok = this.data.hasPhone ? await accountStore.deleteAccount(this.data.code, this.data.confirmation) : await accountStore.deleteWeChatAccount(this.data.confirmation); this.setData({ busy: false, error: accountStore.state.error }); if (ok) wx.reLaunch({ url: '/pages/login/index' }) } }) },
})
