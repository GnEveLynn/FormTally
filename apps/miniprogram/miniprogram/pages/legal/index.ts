import { openPrivacyContract } from '../../platform/legal'

const documents = {
  terms: {
    title: '用户协议',
    sections: [
      { heading: '服务内容', body: '餐见style（FormTally）提供饮食记录、营养估算、每日目标和历史查看等功能。使用微信身份登录后，您可以设置身体资料并管理自己的记录。' },
      { heading: '估算结果', body: '食物识别、热量及营养数据均为估算，可能因图片、份量和食材差异而不准确。请在保存前核对和修改；本服务不提供医疗诊断或治疗建议。' },
      { heading: '使用规则', body: '请仅上传您有权使用的餐食图片，不要上传他人的敏感信息或违法内容。请妥善保管微信账户，并对通过该账户提交的内容负责。' },
      { heading: '数据与退出', body: '您可以在小程序内编辑或删除餐食记录、仅移除餐食图片，并在“我的－数据管理”申请删除账户与个人数据。个人信息处理方式请查看本小程序的隐私保护指引；AI 图片处理另有说明。' },
      { heading: '运营者与联系', body: '小程序运营主体及联系途径请以微信平台展示的本小程序主体信息和隐私保护指引为准。服务规则发生重要变化时，我们会在小程序中更新并按需要重新征求同意。' },
    ],
  },
  ai: {
    title: 'AI 图片处理说明',
    sections: [
      { heading: '何时处理', body: '只有您选择餐食图片并点击“同意并开始分析”后，图片和您填写的补充描述才会用于本次 AI 分析。您也可以不上传图片，改用手动录入。' },
      { heading: '处理方式', body: '小程序会将所选图片压缩为 JPEG 后上传至本服务端，并将图片和补充描述发送给第三方 AI 服务识别食物、份量与营养信息。' },
      { heading: '结果使用', body: 'AI 返回的识别结果和营养数值仅是估算，可能有遗漏或错误；请在保存餐食前核对、修改。它们不构成医疗或营养诊断。' },
      { heading: '管理图片', body: '您可以放弃尚未保存的草稿；保存餐食后，可在餐食详情中仅移除图片或删除整餐。有关个人信息处理和联系方式，请查看本小程序的隐私保护指引。' },
    ],
  },
}

Page({
  data: { title: '', version: '2026-09-10', sections: [] as Array<{ heading: string; body: string }> },
  onLoad(options: { kind?: string }) {
    this.setData(documents[options.kind === 'ai' ? 'ai' : 'terms'])
  },
  openPrivacyContract,
})
