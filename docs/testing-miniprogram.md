# 微信小程序测试指南

## 本地自动化

环境要求与根 README 一致。首次检出运行 `npm ci`，然后执行：

```sh
make test-miniprogram
make build-miniprogram
make scan-miniprogram
```

完整回归使用 `make verify`，它同时覆盖 Go、真实 PostgreSQL 集成、H5 单测与 E2E、H5 构建，以及小程序单测、类型构建和敏感产物扫描。

## 微信开发者工具导入

1. 在微信开发者工具中导入仓库的 `apps/miniprogram` 目录，不要导入仓库根目录。
2. 复制 `apps/miniprogram/project.private.config.json.example` 为 `project.private.config.json`，仅在本机填写获授权的真实 AppID 和个人工具设置；该文件已忽略，禁止提交。
3. 执行 `npm run build:miniprogram`。开发者工具开启“使用 npm 模块”所需能力，并选择项目支持的稳定基础库。
4. 开发模拟器可使用 `http://127.0.0.1:8080`；启动 PostgreSQL、迁移和 API：`make db-up && make migrate && make dev-api`。开发者工具仅限本地调试时可关闭域名校验。
5. 不在小程序或 private 配置中填写 `WECHAT_APP_SECRET`。AppSecret 仅通过服务端环境变量提供。

## 体验版与真机环境

真机不能访问电脑的 `127.0.0.1`。体验/生产环境必须使用可从手机访问的 HTTPS API，并在微信公众平台同时配置相同主机的 `request` 与 `uploadFile` 合法域名。配置 AppID、项目成员、体验成员、隐私保护指引和所用隐私接口后，先取得用户对上传体验版的明确授权。

在 iOS 和 Android 微信各记录一次：体验版版本号、机型、系统版本、微信版本、开发者工具版本、基础库版本、执行时间与结果。完整旅程为：勾选协议并微信一键登录 → 资料与目标 → 拍照/相册 → AI 分析 → 修改保存 → 今日更新 → 历史查看编辑 → 退出重登 → 输入 DELETE 并以新的微信登录 code 复验删号。另测未勾选协议、弱网重试、拒绝相机、会话过期、上传超限、AI 超时，以及用另一微信身份尝试删号被拒绝。

## 故障排查

- 登录不可用：确认服务端同时设置 `WECHAT_APP_ID` 与 `WECHAT_APP_SECRET`，AppID 与当前小程序一致；不要记录临时 code。
- 首次登录不会请求手机号权限；个人主体可使用基础 `wx.login`/OpenID 登录。协议未勾选时客户端不发起建号请求。
- 真机网络失败：确认 API 为有效 HTTPS、证书链完整、手机可达，并分别检查 `request`/`uploadFile` 合法域名。
- 401：重新登录；客户端会清除失效 Bearer token。网络错误不会误清有效 token。
- 图片失败：检查相机/相册授权、10 MB 上限、图片可解码性和上传域名；分析失败应保留本次图片供重试。
- 更新餐食失败：小程序使用微信支持的 `PUT /v1/meals/{id}`，H5 继续兼容 `PATCH`；`REVISION_CONFLICT` 会重新加载最新内容。
- 开发者工具与真机差异：记录工具、基础库、设备和微信版本，优先以对应平台真机结果为准。

真实 AppID、合法域名、体验成员、正式隐私文本、体验版上传和双平台真机结果均属于外部验收；缺失时只能结论为“本地已验证，真实微信流程待验收”。
