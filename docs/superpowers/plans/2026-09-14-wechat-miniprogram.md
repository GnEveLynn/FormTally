# FormTally 微信原生小程序 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` only when the user explicitly authorizes subagents; otherwise use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不改变现有 H5 用户流程和 Cookie 安全边界的前提下，交付一个使用微信身份与手机号快捷验证登录、覆盖 FormTally 核心饮食闭环的微信原生小程序。

**Architecture:** 新增独立的 `apps/miniprogram` 原生 TypeScript/WXML/WXSS 客户端，通过 npm workspace 复用 API 契约、纯业务规则和设计值。Go API 新增微信服务端适配器、微信身份绑定和 Bearer 会话，同时保留 H5 的 Cookie、Origin 校验和现有接口响应。

**Tech Stack:** 微信原生小程序、TypeScript、WXML、WXSS、微信开发者工具、Vitest、npm workspaces；Go 1.27、`net/http`、pgx/v5、PostgreSQL 18。

**Spec:** [微信小程序技术设计](../specs/2026-09-14-wechat-miniprogram-design.md)

## Global Constraints

- 小程序只面向微信，不引入跨端框架或第三方 UI 框架。
- H5 继续使用 HttpOnly Cookie；小程序只使用不透明 Bearer token；不使用 JWT。
- H5 的登录接口、响应结构、Cookie 属性和写请求 Origin 校验保持兼容。
- AppSecret、微信临时 code、session_key、完整手机号和会话令牌不得进入日志或客户端构建产物。
- 微信生产请求只允许 HTTPS 官方主机；可配置 API base URL 仅用于自动化测试中的本地假服务。
- 每个非平凡行为先写失败测试，再实现最小代码；每个任务完成后运行其聚焦测试及 H5 回归。
- 未经用户明确授权，不执行 `git commit`、`git push`、创建 PR、部署或上传小程序代码。
- 不增加支付、订阅消息、运动记录、离线同步、后台管理或跨进程图片草稿。
- 自动化测试不等于微信真实流程通过；登录、手机号、相机、相册和上传必须在 iOS、Android 真机验收。

---

## 文件结构锁定

### 服务端新增

- `server/internal/wechat/client.go`：微信登录凭证与手机号 API 客户端。
- `server/internal/wechat/client_test.go`：使用 `httptest.Server` 验证请求、响应、超时和脱敏。
- `server/internal/wechat/types.go`：微信侧最小请求响应类型和稳定错误。
- `server/internal/wechat/fake_test.go`：服务端测试共用的假微信客户端。
- `server/migrations/00008_wechat_identities.sql`：身份、绑定凭证及 session 客户端类型迁移；当前最新迁移为 `00007_object_deletions.sql`，禁止覆盖已有文件。
- `server/internal/auth/wechat.go`：微信会话与首次手机号绑定服务逻辑。
- `server/internal/auth/wechat_handler.go`：两个微信认证 HTTP 接口。

### 服务端修改

- `server/internal/app/config.go`：微信配置读取和生产约束。
- `server/internal/auth/types.go`：微信请求、绑定响应和客户端类型。
- `server/internal/auth/service.go`：共享会话创建与协议校验。
- `server/internal/auth/postgres.go`：身份查询、绑定凭证消费和事务归并。
- `server/internal/auth/middleware.go`：Cookie/Bearer 凭证解析。
- `server/internal/auth/handler.go`：现有会话查询和退出使用统一凭证。
- `server/internal/httpapi/middleware.go`：按凭证类型执行 CSRF Origin 校验。
- `server/cmd/api/main.go`：组装微信客户端和统一认证器。
- `.env.example`、`Makefile`、`README.md`：开发配置和验证命令。

### 共享包修改

- `packages/api-contract/src/auth.ts`：微信登录与绑定契约。
- `packages/api-contract/src/index.ts` 不创建；继续使用当前子路径 exports。
- `packages/design-tokens/src/tokens.json`：设计值的单一数据源。
- `scripts/build-design-tokens.mjs`：从 JSON 生成 H5 CSS 和小程序 WXSS。

### 小程序新增

- `apps/miniprogram/package.json`、`tsconfig.json`、`project.config.json`、`project.private.config.json.example`、`sitemap.json`。
- `apps/miniprogram/miniprogram/app.ts`、`app.json`、`app.wxss`、`styles/tokens.wxss`。
- `apps/miniprogram/miniprogram/config/env.ts`：开发/体验/生产 API 基地址。
- `apps/miniprogram/miniprogram/services/http.ts`、`upload.ts` 及测试。
- `apps/miniprogram/miniprogram/platform/auth.ts`、`media.ts`、`storage.ts` 及测试。
- `apps/miniprogram/miniprogram/stores/session.ts`、`onboarding.ts`、`today.ts`、`history.ts`、`meal-draft.ts` 及测试。
- `apps/miniprogram/miniprogram/pages/**`：登录、首次设置、今日、餐食、历史和账户页面。

---

### Task 1：锁定 H5 基线并扩展微信配置

**Files:**

- Modify: `server/internal/app/config.go`
- Modify: `server/internal/app/config_test.go`
- Modify: `.env.example`
- Modify: `Makefile`
- Modify: `README.md`

**Interfaces:**

```go
type Config struct {
    // existing fields remain unchanged
    WeChatAppID     string
    WeChatAppSecret string
    WeChatAPIBaseURL string
    WeChatAPITimeout time.Duration
}
```

- [ ] **Step 1:** 运行 `git status --short`，记录并保护用户已有改动；运行 `make test-go && make test-web && make test-e2e`，保存当前 H5 基线结果。
- [ ] **Step 2:** 在 `config_test.go` 写失败测试：开发环境允许微信凭据为空；任一凭据单独出现时报错；生产启用小程序时必须同时存在 AppID/Secret；生产 base URL 非 HTTPS 或非 `api.weixin.qq.com` 时拒绝；超时默认 `5s` 且必须为正数。
- [ ] **Step 3:** 运行 `cd server && go test ./internal/app -run WeChat -count=1`，确认失败原因是字段或校验尚不存在。
- [ ] **Step 4:** 在 `LoadConfig` 中加入 `WECHAT_APP_ID`、`WECHAT_APP_SECRET`、`WECHAT_API_BASE_URL`、`WECHAT_API_TIMEOUT` 的最小解析，不改变其他配置默认值。
- [ ] **Step 5:** 更新 `.env.example` 和 README 的小程序开发前置说明；Makefile 只增加后续任务实际使用的 `test-miniprogram`、`build:miniprogram` 入口，不增加发布命令。
- [ ] **Step 6:** 运行 `cd server && go test ./internal/app -count=1 && make test-web`。

**Exit criteria:** 不提供微信配置时，现有开发和 H5 测试行为与基线一致；错误微信配置在监听端口前失败。

---

### Task 2：实现微信服务端 API 客户端

**Files:**

- Create: `server/internal/wechat/types.go`
- Create: `server/internal/wechat/client.go`
- Create: `server/internal/wechat/client_test.go`

**Interfaces:**

```go
type LoginIdentity struct { OpenID, UnionID, SessionKey string }
type Phone struct { Number string }
type Client interface {
    ExchangeLoginCode(context.Context, string) (LoginIdentity, error)
    ExchangePhoneCode(context.Context, string) (Phone, error)
}
func NewClient(appID, appSecret, baseURL string, timeout time.Duration) Client
```

- [ ] **Step 1:** 用 `httptest.Server` 写失败测试，断言登录 code 被 URL 编码并发往登录凭证接口；成功响应必须包含 `openid` 和 `session_key`，`unionid` 可空。
- [ ] **Step 2:** 写手机号接口失败测试，断言使用服务端 access token 调用手机号接口；access token 只在内存按微信返回有效期缓存，并在微信明确报告 token 无效后刷新一次。
- [ ] **Step 3:** 写错误测试覆盖无效 code、微信业务错误、非 2xx、非 JSON、缺字段、超时；对外错误只暴露稳定分类 `invalid_credential`、`rate_limited`、`unavailable`、`invalid_response`。
- [ ] **Step 4:** 写日志捕获测试，确认 AppSecret、loginCode、phoneCode、session_key、access token 和手机号不出现在错误文本。
- [ ] **Step 5:** 运行 `cd server && go test ./internal/wechat -count=1`，确认因为客户端未实现而失败。
- [ ] **Step 6:** 使用 `net/http.Client`、`url.Values`、`encoding/json` 和互斥锁实现最小客户端，不增加 SDK 依赖。
- [ ] **Step 7:** 再运行 `cd server && go test ./internal/wechat -count=1`。

**Exit criteria:** 微信边界可由本地假服务确定性测试，生产秘密不会进入错误或日志。

---

### Task 3：增加微信身份、绑定凭证与会话客户端类型

**Files:**

- Create: `server/migrations/00008_wechat_identities.sql`
- Modify: `server/internal/auth/types.go`
- Modify: `server/internal/auth/postgres.go`
- Modify: `server/internal/auth/postgres_test.go`

**Interfaces:**

```go
type ClientType string
const (
    ClientWeb ClientType = "web"
    ClientWeChatMiniProgram ClientType = "wechat_miniprogram"
)
```

- [ ] **Step 1:** 执行 `find server/migrations -maxdepth 1 -type f | sort`，确认 `00008` 仍是下一个未占用编号；若执行前仓库已新增迁移，则暂停并把本计划中的迁移编号统一顺延后再继续。
- [ ] **Step 2:** 写 PostgreSQL 失败测试：`(provider, provider_subject)` 唯一；删除用户级联删除身份和绑定凭证；`sessions.client_type` 只接受 `web`、`wechat_miniprogram`，历史行回填 `web`。
- [ ] **Step 3:** 写绑定凭证测试：数据库只保存 SHA-256 哈希；5 分钟过期；只能消费一次；并发消费只有一个事务成功。
- [ ] **Step 4:** 写账户归并测试：新身份绑定已有手机号用户；新手机号创建用户；同一微信身份重复绑定幂等；微信身份与手机号分别属于不同用户时返回 `IDENTITY_CONFLICT` 且不改变数据。
- [ ] **Step 5:** 运行 `make test-db`，确认迁移或方法缺失导致预期失败。
- [ ] **Step 6:** 新增 `user_identities`、`wechat_binding_tickets` 和 `sessions.client_type` 迁移；绑定凭证表包含 `token_hash`、`openid`、可空 `unionid`、`expires_at`、`consumed_at`。服务端校验完 loginCode 后立即丢弃 `session_key`，不落库也不返回客户端。
- [ ] **Step 7:** 用单个数据库事务实现手机号账户归并、协议记录、身份绑定、ticket 消费和会话创建。
- [ ] **Step 8:** 运行 `make test-db`，再运行 `make migrate-down && make migrate` 验证该迁移可回滚并重新应用。

**Exit criteria:** 数据库约束阻止重复身份、重复消费和静默账户冲突；历史 H5 session 被识别为 `web`。

---

### Task 4：实现两段式微信登录接口

**Files:**

- Create: `server/internal/auth/wechat.go`
- Create: `server/internal/auth/wechat_handler.go`
- Create: `server/internal/auth/wechat_handler_test.go`
- Modify: `server/internal/auth/service.go`
- Modify: `server/internal/auth/handler.go`
- Modify: `server/internal/auth/types.go`
- Modify: `packages/api-contract/src/auth.ts`
- Modify: `server/cmd/api/main.go`

**Interfaces:**

```ts
export interface WeChatSessionRequest { loginCode: string }
export type WeChatSessionResult =
  | ({ bindingRequired: false; token: string } & SessionResponse)
  | { bindingRequired: true; bindingTicket: string; expiresInSeconds: 300 }
export interface WeChatPhoneBindingRequest {
  bindingTicket: string
  phoneCode: string
  agreements: { termsVersion: string; privacyVersion: string }
}
export type WeChatPhoneBindingResponse = { token: string } & SessionResponse
```

- [ ] **Step 1:** 写 `POST /v1/auth/wechat/sessions` 失败测试：无效 body 为 400；微信 code 无效为 422；已有身份返回 session 与 token；首次身份只返回 5 分钟 binding ticket，不创建用户或 session。
- [ ] **Step 2:** 写 `POST /v1/auth/wechat/phone-bindings` 失败测试：缺失/过期/已消费 ticket、无效 phoneCode、过期协议版本和身份冲突均返回稳定错误码。
- [ ] **Step 3:** 写成功测试：已有手机号关联、全新用户创建、协议记录、`wechat_miniprogram` session 创建；响应不设置 Cookie，token 只出现一次。
- [ ] **Step 4:** 运行 `cd server && go test ./internal/auth -run WeChat -count=1`，确认失败。
- [ ] **Step 5:** 抽取现有 session 结果组装和 token 生成作为 auth 包内共享函数；不建立新的认证框架或工厂。
- [ ] **Step 6:** 实现两个 handler 和服务方法，将微信错误映射为 `WECHAT_CODE_INVALID`、`WECHAT_PHONE_UNAVAILABLE`、`WECHAT_RATE_LIMITED`、`WECHAT_UNAVAILABLE`、`IDENTITY_CONFLICT`。
- [ ] **Step 7:** 在 `main.go` 仅当微信配置完整时注册真实客户端；测试通过构造函数注入 fake。
- [ ] **Step 8:** 运行 `cd server && go test ./internal/auth ./internal/wechat -count=1 && make test-db`。

**Exit criteria:** 老用户只需静默微信登录；首次用户才需手机号授权；所有归并在服务端事务内完成。

---

### Task 5：扩展 Bearer 认证且保持 H5 Cookie 与 Origin 行为

**Files:**

- Modify: `server/internal/auth/middleware.go`
- Create: `server/internal/auth/middleware_test.go`
- Modify: `server/internal/auth/handler.go`
- Modify: `server/internal/account/handler.go`
- Modify: `server/internal/httpapi/middleware.go`
- Modify: `server/internal/httpapi/middleware_test.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**

```go
type CredentialKind string
const (CredentialCookie CredentialKind = "cookie"; CredentialBearer CredentialKind = "bearer")
type Credential struct { Token string; Kind CredentialKind }
func SessionCredential(*http.Request) (Credential, error)
```

- [ ] **Step 1:** 写凭证解析失败测试：合法 Cookie、合法 Bearer、错误 scheme、空 token、重复 Authorization，以及 Cookie 与 Bearer 同时存在时的歧义拒绝。
- [ ] **Step 2:** 写 Origin 回归失败测试：Cookie 写请求缺少或不匹配 Origin 仍为 403；Bearer 写请求不依赖 Origin；微信公开认证接口只按精确路由豁免 Origin；其他匿名写请求仍为 403。
- [ ] **Step 3:** 写现有 H5 handler 回归测试，确认登录仍设置原 Cookie 属性，查询和退出仍接受 Cookie，退出清除 Cookie。
- [ ] **Step 4:** 运行 `cd server && go test ./internal/auth ./internal/httpapi ./internal/account -count=1`，确认新增用例失败。
- [ ] **Step 5:** 实现统一凭证解析，把凭证种类放入 request context 供 Origin 中间件使用；不要信任客户端自报的平台 header。
- [ ] **Step 6:** 让会话查询、退出和业务 authenticate 函数使用统一凭证；Bearer 退出只撤销服务端 session，不输出 Set-Cookie。
- [ ] **Step 7:** 运行 `make test-go && make test-db && make test-web && make test-e2e`。

**Exit criteria:** 新增 Bearer 后，现有 H5 登录、保护路由、写操作和退出旅程全部通过；CSRF 防护没有弱化。

---

### Task 6：建立原生小程序工程、设计值和测试基座

**Files:**

- Create: `apps/miniprogram/package.json`
- Create: `apps/miniprogram/tsconfig.json`
- Create: `apps/miniprogram/project.config.json`
- Create: `apps/miniprogram/project.private.config.json.example`
- Create: `apps/miniprogram/sitemap.json`
- Create: `apps/miniprogram/miniprogram/app.ts`
- Create: `apps/miniprogram/miniprogram/app.json`
- Create: `apps/miniprogram/miniprogram/app.wxss`
- Create: `apps/miniprogram/miniprogram/config/env.ts`
- Create: `apps/miniprogram/vitest.config.ts`
- Create: `packages/design-tokens/src/tokens.json`
- Create: `scripts/build-design-tokens.mjs`
- Modify: `packages/design-tokens/src/tokens.css`
- Modify: `package.json`
- Modify: `package-lock.json`
- Modify: `.gitignore`
- Modify: `Makefile`

**Interfaces:**

```ts
export type MiniProgramEnvironment = 'develop' | 'trial' | 'release'
export function apiBaseUrl(env: MiniProgramEnvironment): string
```

- [ ] **Step 1:** 写设计值生成测试，断言 JSON 能确定性生成现有 H5 CSS 和 `apps/miniprogram/miniprogram/styles/tokens.wxss`，生成两次无 diff。
- [ ] **Step 2:** 写环境映射测试，断言三种微信版本只映射到显式 HTTPS 地址；开发模拟器允许显式本机地址，体验/生产拒绝 HTTP。
- [ ] **Step 3:** 运行对应 Vitest，确认文件尚不存在而失败。
- [ ] **Step 4:** 建立微信原生 TypeScript 工程和最小空页面；`project.config.json` 不写个人路径或秘密，个人设置使用被忽略的 private 文件。
- [ ] **Step 5:** 用一个无依赖 Node 脚本从 `tokens.json` 生成 CSS/WXSS；把原 CSS 当前值迁入 JSON，运行 H5 页面测试确认视觉变量名不变。
- [ ] **Step 6:** 只安装官方 TypeScript 类型和已有 Vitest 所需依赖，执行 `npm install --package-lock-only` 更新根锁文件。
- [ ] **Step 7:** 运行 `npm run build:miniprogram && npm run test:miniprogram && npm run build:web && make test-web`。
- [ ] **Step 8:** 用微信开发者工具导入 `apps/miniprogram`，确认编译后出现最小启动页；记录工具版本和基础库版本。

**Exit criteria:** 小程序能在开发者工具编译；共享设计值构建不会改变 H5 变量和值。

---

### Task 7：实现小程序请求、上传、存储与会话恢复

**Files:**

- Create: `apps/miniprogram/miniprogram/services/http.ts`
- Create: `apps/miniprogram/miniprogram/services/http.spec.ts`
- Create: `apps/miniprogram/miniprogram/services/upload.ts`
- Create: `apps/miniprogram/miniprogram/services/upload.spec.ts`
- Create: `apps/miniprogram/miniprogram/platform/auth.ts`
- Create: `apps/miniprogram/miniprogram/platform/auth.spec.ts`
- Create: `apps/miniprogram/miniprogram/platform/storage.ts`
- Create: `apps/miniprogram/miniprogram/platform/storage.spec.ts`
- Create: `apps/miniprogram/miniprogram/stores/session.ts`
- Create: `apps/miniprogram/miniprogram/stores/session.spec.ts`

**Interfaces:**

```ts
export function request<T>(path: string, options?: RequestOptions): Promise<T>
export function uploadAnalysis(input: AnalysisUpload, onProgress?: (percent: number) => void): UploadTask
export function weChatLogin(): Promise<WeChatSessionResult>
export function bindWeChatPhone(phoneCode: string, agreements: Agreements): Promise<WeChatPhoneBindingResponse>
export const sessionStore: { restore(): Promise<void>; logout(): Promise<void> }
```

- [ ] **Step 1:** mock 最小 `wx` 接口并写 request 失败测试：API 基地址、Bearer、中文、请求 ID、JSON、统一错误、网络失败和 401 清理。
- [ ] **Step 2:** 写 upload 失败测试：multipart 字段与现有分析接口一致、幂等键稳定、进度回调、取消和失败错误映射。
- [ ] **Step 3:** 写登录平台测试：每次发起 session 都重新调用 `wx.login`；只有 `bindingRequired` 时进入手机号授权；token 不写日志。
- [ ] **Step 4:** 写存储与恢复测试：有效 token 调用 `GET /v1/auth/session`；401 清除；网络失败保留 token 并展示可重试失败态，不误判匿名。
- [ ] **Step 5:** 运行 `npm run test:miniprogram -- services platform stores/session`，确认失败。
- [ ] **Step 6:** 实现最小适配层和 session store；页面只能调用这些边界，不直接调用 `wx.request`、`wx.uploadFile` 或 token storage。
- [ ] **Step 7:** 运行 `npm run test:miniprogram && npm run build:miniprogram`。

**Exit criteria:** API、上传、微信登录和本地会话都有可替换边界，业务页面不包含协议实现。

---

### Task 8：完成登录和首次资料设置流程

**Files:**

- Create: `apps/miniprogram/miniprogram/pages/login/**`
- Create: `apps/miniprogram/miniprogram/pages/onboarding-profile/**`
- Create: `apps/miniprogram/miniprogram/pages/onboarding-goal/**`
- Create: `apps/miniprogram/miniprogram/pages/onboarding-result/**`
- Create: `apps/miniprogram/miniprogram/stores/onboarding.ts`
- Create: `apps/miniprogram/miniprogram/stores/onboarding.spec.ts`
- Create: `apps/miniprogram/miniprogram/services/profile.ts`
- Create: `apps/miniprogram/miniprogram/services/goals.ts`
- Modify: `apps/miniprogram/miniprogram/app.json`

**Interfaces:**

```ts
export function routeForSession(status: OnboardingStatus): '/pages/today/index' | '/pages/onboarding-profile/index' | '/pages/onboarding-goal/index'
```

- [ ] **Step 1:** 写路由决策、协议勾选、手机号授权拒绝和重试测试。
- [ ] **Step 2:** 写 onboarding store 测试，直接复用 `@formtally/domain/profile-validation`，覆盖资料草稿、字段错误、目标保存和返回状态。
- [ ] **Step 3:** 运行聚焦测试确认失败。
- [ ] **Step 4:** 实现登录页：先静默 session；首次身份显示手机号快捷验证按钮；明确展示协议版本和同意状态；拒绝授权后停留并允许重试。
- [ ] **Step 5:** 实现资料、目标和结果页，调用现有 `/v1/profile`、`/v1/goals`；不在客户端复制营养目标公式。
- [ ] **Step 6:** 在开发者工具模拟器覆盖页面和校验；使用真实 AppID 在真机验证微信登录与手机号授权，若账号权限或后台配置不足，记录为外部阻塞而不伪造通过。
- [ ] **Step 7:** 运行 `npm run test:miniprogram && npm run build:miniprogram && make test-e2e`。

**Exit criteria:** 新用户能完成微信登录、手机号绑定和首次设置；老用户静默登录后按 onboarding 状态跳转。

---

### Task 9：完成饮食记录核心闭环

**Files:**

- Create: `apps/miniprogram/miniprogram/platform/media.ts`
- Create: `apps/miniprogram/miniprogram/platform/media.spec.ts`
- Create: `apps/miniprogram/miniprogram/stores/meal-draft.ts`
- Create: `apps/miniprogram/miniprogram/stores/meal-draft.spec.ts`
- Create: `apps/miniprogram/miniprogram/stores/today.ts`
- Create: `apps/miniprogram/miniprogram/stores/today.spec.ts`
- Create: `apps/miniprogram/miniprogram/services/analyses.ts`
- Create: `apps/miniprogram/miniprogram/services/meals.ts`
- Create: `apps/miniprogram/miniprogram/services/days.ts`
- Create: `apps/miniprogram/miniprogram/pages/today/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-capture/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-analyzing/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-confirm/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-manual/**`
- Modify: `apps/miniprogram/miniprogram/app.json`

**Interfaces:**

```ts
export interface ProcessedImage { path: string; size: number; width: number; height: number; mimeType: 'image/jpeg' }
export function chooseAndProcessMealImage(source: 'camera' | 'album'): Promise<ProcessedImage>
```

- [ ] **Step 1:** 写 media 失败测试：只选一张；拒绝不可解码和超过 10 MB；长边缩至不超过 1600；输出 JPEG；拒绝权限时返回可展示分类。
- [ ] **Step 2:** 写 meal draft 测试：分析/保存幂等键在重试中不变，成功或放弃后更新；分析失败保留图片；编辑使用共享 `meal-editor` 规则。
- [ ] **Step 3:** 写 today store 测试：加载态、空态、失败重试、保存后刷新和用户时区日期。
- [ ] **Step 4:** 运行聚焦测试确认失败。
- [ ] **Step 5:** 使用 `wx.chooseMedia`、图片信息和微信原生压缩/Canvas 能力实现最小图片管线；服务端安全校验保持不变。
- [ ] **Step 6:** 实现今日、拍摄、分析中、确认和手动录入页面，保持 AI 估算提示、低置信度信息和重复点击保护。
- [ ] **Step 7:** 在开发者工具验证相册与失败恢复；真机分别验证相机、相册、压缩、上传、AI 超时重试和保存后今日汇总。
- [ ] **Step 8:** 运行 `npm run test:miniprogram && npm run build:miniprogram && make test-go && make test-e2e`。

**Exit criteria:** 真机完成“拍照/相册 → 分析 → 编辑 → 保存 → 今日更新”，失败无需重新选图。

---

### Task 10：完成历史、账户、隐私和发布候选验收

**Files:**

- Create: `apps/miniprogram/miniprogram/stores/history.ts`
- Create: `apps/miniprogram/miniprogram/stores/history.spec.ts`
- Create: `apps/miniprogram/miniprogram/services/history.ts`
- Create: `apps/miniprogram/miniprogram/services/account.ts`
- Create: `apps/miniprogram/miniprogram/pages/history/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-detail/**`
- Create: `apps/miniprogram/miniprogram/pages/meal-edit/**`
- Create: `apps/miniprogram/miniprogram/pages/me/**`
- Create: `apps/miniprogram/miniprogram/pages/profile/**`
- Create: `apps/miniprogram/miniprogram/pages/goals/**`
- Create: `apps/miniprogram/miniprogram/pages/delete-account/**`
- Create: `docs/testing-miniprogram.md`
- Modify: `apps/miniprogram/miniprogram/app.json`
- Modify: `docs/release-checklist-v1.md`
- Modify: `README.md`

**Interfaces:** 只消费现有 history、meal、profile、goal、account API 契约，不新增服务端业务接口。

- [ ] **Step 1:** 写 history store 失败测试：月份切换、日期选择、加载失败、编辑/删除后刷新；使用确定性本地日期样例。
- [ ] **Step 2:** 写账户操作失败测试：资料/目标保存错误回显；退出撤销 Bearer session；删除账户要求当前已绑定手机号的验证流程，成功后清除 token 和本地草稿。
- [ ] **Step 3:** 运行聚焦测试确认失败。
- [ ] **Step 4:** 实现历史、详情、编辑、我的、资料、目标和删除账户页面；删除继续使用现有二次确认及验证码安全流程，微信身份不降低删除验证强度。
- [ ] **Step 5:** 在 `docs/testing-miniprogram.md` 写出开发者工具导入路径、AppID/private 配置、本地 API、真机 HTTPS 环境、合法域名、体验成员和故障排查步骤。
- [ ] **Step 6:** 更新发布清单：隐私保护指引、相机/相册/手机号用途、request/uploadFile 域名、体验版、iOS/Android 真机、敏感产物扫描。
- [ ] **Step 7:** 运行 `make verify`、`npm run test:miniprogram`、`npm run build:miniprogram` 和 `git diff --check`；确认所有临时服务停止。
- [ ] **Step 8:** 用微信开发者工具上传体验版前先请求用户明确授权；获授权后，在 iOS 和 Android 各执行设计稿第 11.2 节完整旅程，并保存版本号、机型、微信版本和结果。
- [ ] **Step 9:** 扫描小程序构建产物，确认不存在 `WECHAT_APP_SECRET`、真实 AppSecret、session token、手机号、测试图片或 `.env` 文件。

**Exit criteria:** H5 全回归通过；小程序自动测试和构建通过；两类真机核心旅程通过；只有此时才能声明微信小程序实现完成。

---

## 执行顺序与检查点

| 检查点 | 任务 | 用户可观察结果 | 必须保留的证据 |
| --- | --- | --- | --- |
| M1 | 1–5 | H5 不变，服务端支持微信身份和 Bearer | Go/PostgreSQL/H5 全回归 |
| M2 | 6–8 | 小程序可打开并完成微信登录与首次设置 | 开发者工具构建 + 真机登录 |
| M3 | 9 | 小程序完成饮食核心闭环 | 真机拍照、分析、保存 |
| M4 | 10 | 历史、账户和发布检查完成 | iOS/Android 旅程与产物扫描 |

执行时默认在当前会话内按任务串行推进，每个检查点汇报一次。微信账号权限、AppID、合法域名、体验成员和正式隐私文本属于外部前置条件；缺失时继续完成可验证的本地部分，并明确标记“本地已验证，真实微信流程待验收”。
