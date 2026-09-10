# FormTally V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (when the user explicitly authorizes subagents) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付一个可上线验证的 FormTally H5：用户通过手机号登录，设置身体资料与营养目标，上传一张餐食图片获得可编辑的 AI 估算，确认后形成饮食记录，并在今日、历史和详情页面查看与维护摄入进度。

**Architecture:** 单仓库、前后端分离。第一期只实现 `apps/web` 的 Vue 3 H5；未来客户端各自实现页面与平台适配，并通过 `packages/` 共享 API 契约类型、纯业务规则和设计变量。服务端使用 Go 标准库 `net/http` 的模块化单体，是日期、目标公式、汇总和权限判断的唯一事实来源。

**Tech Stack:** Vue 3、Vite、TypeScript、Vue Router、Pinia、npm workspaces、Vitest、Playwright；Go 1.27、`net/http`、`log/slog`、pgx/v5、goose；PostgreSQL 18；S3 兼容对象存储；OpenAI Go SDK / Responses API。

**Spec:** [功能规格](../../spec-v1.md)、[API 契约](../../api-v1.md)、[技术架构](../../architecture-v1.md)、[用户流程](../../user-flow-v1.md)。

## 0. 执行约束

- 本文只拆任务，不实现产品代码。
- 严格按任务编号执行；每个任务先写失败测试，再写最小实现，再运行该任务的验证命令。
- 不在前端重写每日目标公式；前端只展示 API 返回的结果和计算说明。
- `mifflin_st_jeor_v1`、`daily_nutrition_target_v1` 等固定规则必须在 Go 源码邻近位置写明来源、单位、适用范围、取整方式和升级约束。
- 所有写接口按 `docs/api-v1.md` 落实 Origin 校验、所有权校验、错误格式；指定的三个创建接口落实幂等键。
- 每个任务只引入当前任务必需的抽象和依赖，不提前建设训练模块、通用事件平台、Redis、消息队列或管理后台。
- 不使用内存数据库替代 PostgreSQL 集成测试；外部 AI、短信、对象存储在自动测试中使用接口级 fake。
- 未经用户明确授权，不执行 `git commit`、`git push`、创建 PR 或部署。
- 每个里程碑结束时做一次规格回归；测试通过不等于用户流程已通过，必须保留实际 H5 流程验证。

## 1. 里程碑与依赖

| 里程碑 | 任务 | 可观察结果 | 前置 |
| --- | --- | --- | --- |
| M0 开发基座 | 1–2 | API 与 H5 可启动，数据库迁移和测试命令稳定 | 无 |
| M1 登录与目标 | 3–7 | 新用户能登录、填写资料、看到可解释目标并进入空今日页 | M0 |
| M2 识别与记录 | 8–14 | 用户能选图、识别、编辑、保存，今日汇总立即变化 | M1 |
| M3 历史与隐私 | 15–16 | 用户能按日期查看、修改、删餐、删图和删除账户 | M2 |
| M4 发布候选 | 17–18 | 契约、安全、兼容性、AI 质量和部署门槛有证据 | M3 + 外部资源 |

建议顺序是串行完成里程碑；同一里程碑内只有在用户明确允许并行执行后，才能把互不写同一文件的测试或适配器工作交给子任务。

## 2. 上线前外部前置条件

这些条件不阻碍本地开发，但阻碍“可公开上线”的结论：

- 确认中国大陆短信供应商、短信签名、模板和测试手机号；开发环境只允许使用显式的测试发送器，生产环境不得回显验证码。
- 提供生产 PostgreSQL、私有 S3 兼容对象存储及其最小权限凭据。
- 提供 OpenAI API 凭据并确认生产模型配置；模型名只能来自环境变量。
- 提供或审阅用户协议、隐私政策、AI 图片处理说明的正式版本文本。
- 准备不少于 50 餐的人工标注评测集，并确认其中图片有合法测试用途。
- 确认域名、HTTPS、部署地区以及面向中国大陆用户时所需的备案与合规流程。

---

## Task 1：初始化单仓库开发基座

**规模：** M
**依赖：** 无
**目标：** 建立可重复启动、可统一验证的最小前后端工程，不加入业务功能。

**Files:**

- Create: `.gitignore`
- Create: `.env.example`
- Create: `Makefile`
- Create: `compose.yaml`
- Create: `package.json`
- Create: `server/go.mod`
- Create: `server/cmd/api/main.go`
- Create: `server/internal/app/config.go`
- Create: `server/internal/httpapi/router.go`
- Create: `server/internal/httpapi/health_test.go`
- Create: `apps/web/package.json` and the standard Vite Vue 3 TypeScript scaffold
- Create: `apps/web/src/pages/index/index.vue`
- Create: `apps/web/src/pages/index/index.spec.ts`
- Create: `apps/web/src/router/index.ts`
- Create: `apps/web/vitest.config.ts`
- Create: `packages/design-tokens/package.json`
- Create: `packages/design-tokens/src/tokens.css`

**Steps:**

- [ ] 用官方 Vite Vue 3 TypeScript 模板初始化 `apps/web/`，根 `package.json` 使用 npm workspaces；只保留启动页、Vue Router 和必要构建配置。
- [ ] 把当前页面实际使用的颜色、字号和间距放入 `packages/design-tokens`；不创建空的小程序或 App 工程。
- [ ] 在 `server/` 初始化 Go module；配置只从环境变量读取并在启动时校验，不在仓库写秘密。
- [ ] 先写 `GET /healthz` handler 测试，断言 `200`、JSON Content-Type、固定健康结构。
- [ ] 运行 Go 测试，确认因为路由尚未实现而失败。
- [ ] 实现最小 `ServeMux` 和可注入的 HTTP server，令测试通过。
- [ ] 先写 H5 启动页 smoke test，再补最小页面实现。
- [ ] 在 `Makefile` 固定 `dev-api`、`dev-h5`、`test-go`、`test-db`、`test-web`、`test-e2e`、`verify` 命令名。
- [ ] 验证 Ctrl+C 能让 API 优雅停止，配置错误会在监听端口前失败。

**Verify:**

- `cd server && go test ./...`
- `npm --prefix apps/web run test:unit -- --run`
- `npm --prefix apps/web run build`

**Exit criteria:** API 和 H5 都能单独启动；空工程没有业务占位入口；锁文件已生成。

---

## Task 2：建立 HTTP 公共协议、PostgreSQL 与集成测试底座

**规模：** M
**依赖：** Task 1
**目标：** 让后续所有模块共享契约规定的错误、请求标识、日志、事务和真实数据库测试方式。

**Files:**

- Create: `server/internal/httpapi/json.go`
- Create: `server/internal/httpapi/errors.go`
- Create: `server/internal/httpapi/middleware.go`
- Create: `server/internal/httpapi/middleware_test.go`
- Create: `server/internal/postgres/pool.go`
- Create: `server/internal/postgres/testdb_test.go`
- Create: `server/migrations/00001_extensions.sql`
- Modify: `server/internal/httpapi/router.go`
- Modify: `compose.yaml`
- Modify: `Makefile`

**Steps:**

- [ ] 先写公共协议测试：成功与错误响应 Content-Type、`Cache-Control: no-store`、`X-Request-ID` 与错误体一致、未知路由格式统一。
- [ ] 先写变更请求 Origin 白名单测试；覆盖允许、缺失、不允许三种情况。
- [ ] 实现最小 JSON/error helper 与 `request-id → recover → access-log → origin` 中间件链。
- [ ] `slog` 只记录白名单字段；测试日志不出现 Cookie、Authorization、验证码、图片字节和请求体。
- [ ] 在 Compose 中加入 PostgreSQL 18；迁移只通过显式命令执行，不在 API 启动时自动迁移。
- [ ] 建立使用 `TEST_DATABASE_URL` 的真实 PostgreSQL 测试 helper；每个测试获得隔离 schema 或事务清理。
- [ ] 写迁移 up/down smoke test，确认空库能升级到最新并回滚当前迁移。

**Verify:**

- `make db-up`
- `make migrate`
- `make test-go`
- `make test-db`

**Exit criteria:** 后续 handler 不需自行定义错误格式或数据库启动逻辑；测试明确连接 PostgreSQL 而非内存替身。

---

## Task 3：实现手机号验证码与会话后端

**规模：** L
**依赖：** Task 2；正式上线另依赖短信供应商选择
**目标：** 完成 4 个认证接口、会话 Cookie、频率限制、协议版本记录和用户启动状态。

**Files:**

- Create: `server/migrations/00002_auth.sql`
- Create: `server/internal/auth/types.go`
- Create: `server/internal/auth/service.go`
- Create: `server/internal/auth/service_test.go`
- Create: `server/internal/auth/postgres.go`
- Create: `server/internal/auth/postgres_test.go`
- Create: `server/internal/auth/handler.go`
- Create: `server/internal/auth/handler_test.go`
- Create: `server/internal/auth/middleware.go`
- Create: `server/internal/sms/sender.go`
- Create: `server/internal/sms/test_sender.go`
- Modify: `server/internal/httpapi/router.go`
- Modify: `server/internal/app/config.go`

**Steps:**

- [ ] 迁移创建 `users`、`login_codes`、`sessions`、`user_consents`；验证码和会话只保存哈希，带有效期、用途、尝试次数和撤销时间。
- [ ] 先写验证码领域测试：`+86` 格式、5 分钟过期、60 秒重发、用途隔离、尝试次数、一次性消费、探测防护。
- [ ] 用 `crypto/rand` 生成验证码与不透明 session token；比较哈希时使用恒时比较。
- [ ] 定义最小 `sms.Sender` 边界和测试发送器；生产配置使用测试发送器时必须拒绝启动。
- [ ] 先写 `POST /v1/auth/codes`、`POST /v1/auth/sessions`、`GET /v1/auth/session`、`DELETE /v1/auth/session` 的 handler 测试，再实现 handler。
- [ ] 测试 Cookie 包含 `HttpOnly; Secure; SameSite=Lax`，退出后 Cookie 清除且数据库会话撤销。
- [ ] 测试首次用户为 `profile_required`，已有资料或目标的用户返回正确 onboarding 状态。
- [ ] 测试验证码、登录分别限流，并返回 `429` 与 `Retry-After`。

**Verify:**

- `cd server && go test ./internal/auth/... ./internal/sms/...`
- `make test-db`

**Acceptance mapping:** `AC-AUTH-02`–`AC-AUTH-06`、`AC-PRIV-08`。

**Exit criteria:** 认证契约全部通过 `httptest` 和 PostgreSQL 集成测试；生产环境不会暴露验证码。

---

## Task 4：实现 H5 登录、启动恢复与路由守卫

**规模：** M
**依赖：** Task 3
**目标：** 用户打开 H5 后能按会话状态进入登录、资料、目标或今日页。

**Files:**

- Create: `apps/web/src/api/http.ts`
- Create: `apps/web/src/api/auth.ts`
- Create: `apps/web/src/stores/session.ts`
- Create: `apps/web/src/pages/welcome/index.vue`
- Create: `apps/web/src/pages/login/index.vue`
- Create: `packages/api-contract/package.json`
- Create: `packages/api-contract/src/auth.ts`
- Create: `packages/api-contract/src/errors.ts`
- Create: `packages/domain/package.json`
- Create: `packages/domain/src/phone.ts`
- Create: `packages/domain/src/phone.spec.ts`
- Create: `apps/web/src/router/boot.ts`
- Create: `apps/web/src/router/boot.spec.ts`
- Create: `apps/web/e2e/auth.spec.ts`
- Modify: `apps/web/src/router/index.ts`

**Steps:**

- [ ] API client 默认发送 `credentials: include`、`Accept-Language: zh-CN` 和 `X-Request-ID`，不读取 Cookie。
- [ ] 先写手机号校验、验证码倒计时、协议未同意阻断、API 错误映射的单元测试。
- [ ] 实现欢迎与登录页面；表单错误与字段关联，按钮触控区域至少 44×44 CSS 像素。
- [ ] 先写启动路由决策测试，再实现对 `GET /v1/auth/session` 的唯一启动探测。
- [ ] Playwright 覆盖未登录、有效会话恢复、退出后三条路径；网络层使用确定性 API fixture。
- [ ] 确认失败状态不会被误判为未登录或空数据，并提供重试。

**Verify:**

- `npm --prefix apps/web run test:unit -- --run`
- `npm --prefix apps/web run test:e2e -- auth.spec.ts`

**Acceptance mapping:** `AC-AUTH-01`、`AC-AUTH-03`–`AC-AUTH-06`。

**Exit criteria:** 重新打开 H5 不重复登录，受保护页面不可绕过，失败态与未登录态可区分。

---

## Task 5：实现可版本化的每日营养目标计算内核

**规模：** M
**依赖：** Task 1
**目标：** 在 Go 中形成公式唯一实现，能复现输入、中间值、取整和历史版本。

**Files:**

- Create: `server/internal/goals/calculator.go`
- Create: `server/internal/goals/calculator_test.go`
- Create: `server/internal/goals/rounding.go`
- Create: `server/internal/goals/types.go`
- Create: `server/internal/goals/eligibility.go`
- Create: `server/internal/goals/eligibility_test.go`

**Steps:**

- [ ] 先写固定样例：男性 31 岁、178 cm、72.5 kg、中度活动、标准减脂，最终为 2220 kcal、蛋白质 139 g、碳水 278 g、脂肪 62 g。
- [ ] 增加女性、保持目标、三档增减速度、5 档活动系数、半数远离零取整的表驱动测试。
- [ ] 增加生日临界点和 IANA 时区测试，年龄必须按目标本地日期计算。
- [ ] 增加 18–100 岁、身高 100–250 cm、体重 25–350 kg 边界测试。
- [ ] 增加孕期、哺乳期、疾病膳食管理关闭自动目标的测试。
- [ ] 实现 `daily_nutrition_target_v1`，同时输出 target、warnings 和完整 `GoalCalculation`；不要给前端提供另一套计算参数让其自行运算。
- [ ] 在公式代码邻近注释中写明研究来源、公式版本、单位、适用范围、取整与“规则变化必须升版本”；测试断言版本 ID 和解释步骤。
- [ ] 加入历史 payload 回放测试：保存的 V1 输入和步骤能够复现保存结果，未来计算器版本不能改写 V1 fixture。

**Verify:**

- `cd server && go test ./internal/goals -run 'Test(Calculate|Eligibility|Replay)' -count=1`

**Acceptance mapping:** `AC-GOAL-01`–`AC-GOAL-03`、`AC-GOAL-09`–`AC-GOAL-12`、`AC-ONB-04`–`AC-ONB-05`。

**Exit criteria:** 固定公式结果、解释结果和历史回放三者一致；公式版本与来源可在代码中直接审查。

---

## Task 6：实现身体资料、目标设置与每日目标快照 API

**规模：** L
**依赖：** Task 3、Task 5
**目标：** 完成资料与目标的 5 个接口，并正确处理首次生效、次日生效和历史快照不变。

**Files:**

- Create: `server/migrations/00003_profiles_goals.sql`
- Create: `server/internal/profile/types.go`
- Create: `server/internal/profile/service.go`
- Create: `server/internal/profile/service_test.go`
- Create: `server/internal/profile/postgres.go`
- Create: `server/internal/profile/handler.go`
- Create: `server/internal/profile/handler_test.go`
- Create: `server/internal/goals/service.go`
- Create: `server/internal/goals/service_test.go`
- Create: `server/internal/goals/postgres.go`
- Create: `server/internal/goals/postgres_test.go`
- Create: `server/internal/goals/handler.go`
- Create: `server/internal/goals/handler_test.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

- [ ] 迁移创建 `profiles`、`goal_settings`、`daily_targets`，包含 revision、有效日期、目标数值、计算版本、原始输入、中间步骤和 warnings。
- [ ] 先写资料完整校验、首次 PUT、更新 revision、其他用户隔离测试。
- [ ] 先写目标预览无副作用测试：不得创建快照或改变设置。
- [ ] 先写首次目标今天生效、已有目标明天生效、修改资料只重算待生效目标的事务测试。
- [ ] 先写手动目标测试：允许宏量与热量不一致但必须返回 warning；特殊健康状态只能手动目标。
- [ ] 先写过去日期首次补录复制当前目标、随后修改目标不改变历史快照的测试。
- [ ] 实现 `GET/PUT /v1/profile`、`GET/PUT /v1/goals`、`POST /v1/goal-previews`。
- [ ] 覆盖 `expectedRevision` 缺失或冲突；冲突返回当前资源摘要，不静默覆盖。

**Verify:**

- `cd server && go test ./internal/profile ./internal/goals`
- `make test-db`

**Acceptance mapping:** `AC-ONB-01`、`AC-ONB-03`–`AC-ONB-06`、`AC-GOAL-04`–`AC-GOAL-12`。

**Exit criteria:** API 返回完整公式解释；今天与历史目标一旦形成快照就不会被新设置改写。

---

## Task 7：实现 H5 首次设置、目标预览和“如何计算”

**规模：** L
**依赖：** Task 6
**目标：** 完成新用户 onboarding，并把公式解释以可理解、默认折叠的方式展示出来。

**Files:**

- Create: `apps/web/src/api/profile.ts`
- Create: `apps/web/src/api/goals.ts`
- Create: `apps/web/src/stores/onboarding.ts`
- Create: `packages/api-contract/src/profile.ts`
- Create: `packages/api-contract/src/goals.ts`
- Create: `packages/domain/src/profile-validation.ts`
- Create: `packages/domain/src/profile-validation.spec.ts`
- Create: `apps/web/src/components/GoalCalculation.vue`
- Create: `apps/web/src/components/GoalCalculation.spec.ts`
- Create: `apps/web/src/pages/onboarding/profile.vue`
- Create: `apps/web/src/pages/onboarding/goal.vue`
- Create: `apps/web/src/pages/onboarding/result.vue`
- Create: `apps/web/e2e/onboarding.spec.ts`
- Modify: `apps/web/src/router/index.ts`

**Steps:**

- [ ] 先写多步骤草稿返回不丢失、字段边界、特殊健康状态切换手动模式的单元测试。
- [ ] 实现资料、目标、结果三步页面；只有最终确认才保存目标。
- [ ] `GoalCalculation` 只渲染后端返回的名称、版本、输入、步骤、取整、来源和免责声明；组件内不得出现公式常数或重新计算函数。
- [ ] 测试自动与手动目标文案、宏量不一致 warning、次日生效提示。
- [ ] Playwright 完成“新用户 → 填资料 → 预览 → 展开如何计算 → 保存 → 今日页”流程。
- [ ] 在测试中核对示例计算步骤与最终 2220 kcal 一致，但断言对象来自 API fixture。

**Verify:**

- `npm --prefix apps/web run test:unit -- --run`
- `npm --prefix apps/web run test:e2e -- onboarding.spec.ts`

**Acceptance mapping:** `AC-ONB-01`–`AC-ONB-06`、`AC-GOAL-03`–`AC-GOAL-05`、`AC-GOAL-09`–`AC-GOAL-12`。

**Exit criteria:** 用户能看懂且追溯目标算法；前端源码搜索不到 Mifflin 公式实现。

---

## Task 8：实现 H5 图片预处理与服务端私有图片存储

**规模：** L
**依赖：** Task 2、Task 4
**目标：** 支持一餐一图的拍摄/选择、方向正确预览、客户端压缩、服务端验证重编码和私有存取。

**Files:**

- Create: `apps/web/src/domain/image.ts`
- Create: `apps/web/src/domain/image.spec.ts`
- Create: `apps/web/src/components/ImagePicker.vue`
- Create: `apps/web/src/components/ImagePicker.spec.ts`
- Create: `apps/web/src/pages/meal/capture.vue`
- Create: `server/internal/storage/store.go`
- Create: `server/internal/storage/image.go`
- Create: `server/internal/storage/image_test.go`
- Create: `server/internal/storage/filesystem.go`
- Create: `server/internal/storage/s3.go`
- Create: `server/internal/storage/s3_test.go`
- Create: `server/internal/storage/private_url.go`
- Create: `server/internal/storage/private_url_test.go`
- Modify: `apps/web/src/router/index.ts`
- Modify: `server/internal/app/config.go`

**Steps:**

- [ ] 建立 JPEG、PNG、WebP、带 EXIF 方向和超限图片 fixtures；fixtures 不包含真实用户数据。
- [ ] 先写 H5 类型/大小检查与方向预览测试；日期、餐别必须在重新选择图片后保留。
- [ ] 使用浏览器能力缩放并重编码；不支持相机时自动退化为文件选择。
- [ ] 先写服务端 MIME sniff、像素上限、解码失败、WebP 解码、重编码去元数据测试。
- [ ] 定义最小 Store：put、短时读取 URL、delete；提供本地开发实现与 S3 兼容生产实现。
- [ ] 对象 key 必须使用用户不可推断的随机标识；数据库和日志不得保存公开 URL。
- [ ] 测试私有 URL 到期失效，删除对象后旧地址不能长期访问。

**Verify:**

- `cd server && go test ./internal/storage -count=1`
- `npm --prefix apps/web run test:unit -- --run ImagePicker image`

**Acceptance mapping:** `AC-IMG-01`–`AC-IMG-04`、`AC-PRIV-03`–`AC-PRIV-05`。

**Exit criteria:** 不支持图片在调用 AI 前失败；存储层没有公共 bucket 假设；原始元数据不被保留。

---

## Task 9：实现 AI 结构化分析适配器与结果校验

**规模：** L
**依赖：** Task 8；联调另依赖 OpenAI 凭据
**目标：** 把图片分析封装为可替换、可测试的服务端边界，并把不可信模型输出转换为严格领域结果。

**Files:**

- Create: `server/internal/analysis/analyzer.go`
- Create: `server/internal/analysis/schema.go`
- Create: `server/internal/analysis/schema_test.go`
- Create: `server/internal/analysis/normalize.go`
- Create: `server/internal/analysis/normalize_test.go`
- Create: `server/internal/analysis/openai.go`
- Create: `server/internal/analysis/openai_test.go`
- Create: `server/internal/analysis/prompt.go`
- Create: `server/internal/analysis/testdata/*.json`
- Modify: `server/internal/app/config.go`

**Steps:**

- [ ] 先写有效、部分识别、低置信度、空项目、负值、超范围、缺字段、非法 JSON fixtures。
- [ ] 定义一期最小结果字段：食物名、重量、四项营养、单位重量基准、置信度、估算依据和“不完整”提示。
- [ ] 先写 schema 与二次领域校验测试；结构合法但业务越界同样判定失败。
- [ ] 用 OpenAI Responses API 和结构化输出实现 adapter；模型和超时从配置读取，保存模型、提示版本、响应状态和耗时。
- [ ] 不在日志或数据库保存完整原始模型输出；只保存标准化结果和必要诊断码。
- [ ] 单元测试通过 fake transport 验证请求包含图片、schema 与提示版本，不访问真实网络。
- [ ] 增加 opt-in 的真实联调测试，缺少显式开关时自动跳过且不伪装为通过线上 AI。

**Verify:**

- `cd server && go test ./internal/analysis -count=1`
- `cd server && FORMTALLY_OPENAI_INTEGRATION=1 go test ./internal/analysis -run TestOpenAILive -count=1`（仅有测试凭据时）

**Acceptance mapping:** `AC-AI-04`–`AC-AI-07`、`AC-PRIV-01`–`AC-PRIV-02`。

**Exit criteria:** 任意 AI 输出都必须先经过结构与范围校验；网络/超时/无效结果被转换为可恢复的失败类别。

---

## Task 10：实现分析草稿生命周期与 4 个 API

**规模：** L
**依赖：** Task 3、Task 8、Task 9
**目标：** 创建、查询、重试和放弃 AI/手工草稿；失败时保留图片和上下文，草稿不进入汇总。

**Files:**

- Create: `server/migrations/00004_meal_analyses.sql`
- Create: `server/migrations/00005_idempotency.sql`
- Create: `server/internal/idempotency/store.go`
- Create: `server/internal/idempotency/postgres.go`
- Create: `server/internal/idempotency/postgres_test.go`
- Create: `server/internal/analysis/service.go`
- Create: `server/internal/analysis/service_test.go`
- Create: `server/internal/analysis/postgres.go`
- Create: `server/internal/analysis/postgres_test.go`
- Create: `server/internal/analysis/handler.go`
- Create: `server/internal/analysis/handler_test.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

- [ ] 迁移创建 `meal_analyses` 与 `idempotency_records`；草稿包含用户、图片 key、日期、餐别、状态、结果、失败码、模型信息和 24 小时有效期。
- [ ] 先写状态机测试：`processing → ready|failed`，过期后不可保存，删除只允许未保存草稿。
- [ ] 先写 AI 同意版本测试：首次调用缺少当前确认返回 `AI_CONSENT_REQUIRED`，拒绝时不调用 analyzer。
- [ ] 先写创建、相同键重放、同键异内容冲突、处理中重试、24 小时后键过期测试。
- [ ] 实现 `POST/GET/DELETE /v1/meal-analyses...` 与 retry；AI 失败返回 `201` 的 failed resource，而不是丢失上下文的通用 5xx。
- [ ] 测试草稿无论 ready 或 failed 都不改变任何日期汇总。
- [ ] 测试用户隔离与删除草稿后的失去引用图片清理请求。

**Verify:**

- `cd server && go test ./internal/idempotency ./internal/analysis`
- `make test-db`

**Acceptance mapping:** `AC-AI-01`–`AC-AI-09`、`AC-PRIV-01`、`AC-PRIV-03`–`AC-PRIV-04`。

**Exit criteria:** 上传成功后即使 AI 失败也有可查询、可重试、可手工接续的草稿；重复请求不重复分析。

---

## Task 11：实现餐食营养编辑领域逻辑

**规模：** M
**依赖：** Task 9
**目标：** 定义用户最终确认值的校验、重量联动计算与整餐汇总，供后端和 H5 一致使用。

**Files:**

- Create: `server/internal/meals/types.go`
- Create: `server/internal/meals/nutrition.go`
- Create: `server/internal/meals/nutrition_test.go`
- Create: `server/internal/meals/validation.go`
- Create: `server/internal/meals/validation_test.go`
- Create: `packages/domain/src/meal-editor.ts`
- Create: `packages/domain/src/meal-editor.spec.ts`

**Steps:**

- [ ] 先写重量变化按单位重量基准重算、直接营养覆盖、添加、删除和整餐求和测试。
- [ ] 固定数值规范：热量整数 kcal，重量与营养素最多一位小数；不要依赖 JS 二进制浮点的偶然结果。
- [ ] 先写空项目、负数、NaN、超范围、未来时间、错误日期和非法餐别测试。
- [ ] 后端实现权威校验与汇总；前端实现即时预览，但保存结果以服务端返回为准。
- [ ] 用同一组 JSON fixtures 分别驱动 Go 与 TypeScript 测试，防止规则漂移。

**Verify:**

- `cd server && go test ./internal/meals -run 'Test(Nutrition|Validate)' -count=1`
- `npm --prefix apps/web run test:unit -- --run meal-editor`

**Acceptance mapping:** `AC-EDIT-01`–`AC-EDIT-05`、`AC-SAVE-03`、`AC-SAVE-05`–`AC-SAVE-06`。

**Exit criteria:** 相同 fixture 在前端即时预览和后端权威计算中得到相同值，非法输入不能进入持久层。

---

## Task 12：实现正式饮食记录的保存、详情、修改与删除 API

**规模：** XL
**依赖：** Task 6、Task 10、Task 11
**目标：** 完成 6 个餐食接口及事务、幂等、revision、跨日期更新和图片删除流程。

**Files:**

- Create: `server/migrations/00006_meals.sql`
- Create: `server/migrations/00007_object_deletions.sql`
- Create: `server/internal/meals/service.go`
- Create: `server/internal/meals/service_test.go`
- Create: `server/internal/meals/postgres.go`
- Create: `server/internal/meals/postgres_test.go`
- Create: `server/internal/meals/handler.go`
- Create: `server/internal/meals/handler_test.go`
- Create: `server/internal/storage/deletion_worker.go`
- Create: `server/internal/storage/deletion_worker_test.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

- [ ] 迁移创建 `meals`、`meal_items`、`object_deletions`；meal 带 revision，item 保存用户最终值和可选 AI provenance。
- [ ] 先写“合法草稿确认后原子创建 meal/items 并消费草稿”的事务测试。
- [ ] 先写 POST meal 幂等测试：重复提交只生成一餐；同键异内容冲突；失败事务不留下半餐。
- [ ] 先写手工草稿保存与 AI 草稿保存测试，正式记录必须使用请求中的最终编辑值。
- [ ] 先写 GET/PATCH/DELETE 所有权测试，跨用户统一返回 404。
- [ ] 先写 revision 冲突和 `affectedLocalDates` 测试；跨日期移动同时返回旧、新日期。
- [ ] 先写只删图片不改营养、删餐重算汇总、取消删除由客户端不发请求的边界测试。
- [ ] 对象删除通过数据库中的耐久删除任务执行；短暂失败可重试，日志不包含对象内容或签名 URL。
- [ ] 实现 `POST /v1/meals`、`GET/PATCH/DELETE /v1/meals/{id}`、`DELETE /v1/meals/{id}/image`。

**Verify:**

- `cd server && go test ./internal/meals ./internal/storage`
- `make test-db`

**Acceptance mapping:** `AC-SAVE-01`–`AC-SAVE-06`、`AC-HIS-05`–`AC-HIS-10`、`AC-PRIV-03`–`AC-PRIV-05`。

**Exit criteria:** 任何成功响应均对应完整事务；重复、并发或跨用户请求不能破坏或泄露数据。

---

## Task 13：实现某日汇总和月度历史 API

**规模：** L
**依赖：** Task 6、Task 12
**目标：** 服务端按用户时区返回目标快照、四项真实累计、进度状态、餐次与月历标记。

**Files:**

- Create: `server/internal/days/types.go`
- Create: `server/internal/days/service.go`
- Create: `server/internal/days/service_test.go`
- Create: `server/internal/days/postgres.go`
- Create: `server/internal/days/postgres_test.go`
- Create: `server/internal/days/handler.go`
- Create: `server/internal/days/handler_test.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

- [ ] 先写空日测试：返回零摄入和空餐次，不创建 meal；仅在业务需要时惰性形成该日目标快照。
- [ ] 先写多餐四项求和、超过目标不截断、90%/110% 边界和超出量测试。
- [ ] 先写过去日期沿用旧快照、过去无快照首次补录复制当前目标测试。
- [ ] 先写用户时区跨 UTC 日期、夏令时边界、未来日期拒绝测试。
- [ ] 先写月历史仅正式 meal 标记、有草稿无标记、默认月份和非法月份测试。
- [ ] 实现 `GET /v1/days/{localDate}` 与 `GET /v1/history?month=YYYY-MM`，避免 N+1 查询。
- [ ] 将查询计划纳入集成测试或审查记录，索引覆盖 `user_id + local_date`。

**Verify:**

- `cd server && go test ./internal/days -count=1`
- `make test-db`

**Acceptance mapping:** `AC-TODAY-01`–`AC-TODAY-06`、`AC-GOAL-06`–`AC-GOAL-08`、`AC-HIS-01`–`AC-HIS-04`。

**Exit criteria:** 任意日期汇总可由正式 meal_items 重算得到；草稿与失败分析永不进入查询结果。

---

## Task 14：实现 H5 今日、选图、AI 状态、确认与保存闭环

**规模：** XL
**依赖：** Task 7、Task 8、Task 10–13
**目标：** 打通一期最高价值流程：“今日 → 拍照/上传 → AI/手工编辑 → 保存 → 今日立即更新”。

**Files:**

- Create: `apps/web/src/api/analyses.ts`
- Create: `apps/web/src/api/meals.ts`
- Create: `apps/web/src/api/days.ts`
- Create: `packages/api-contract/src/analyses.ts`
- Create: `packages/api-contract/src/meals.ts`
- Create: `packages/api-contract/src/days.ts`
- Create: `apps/web/src/stores/meal-draft.ts`
- Create: `apps/web/src/stores/today.ts`
- Create: `apps/web/src/components/NutritionProgress.vue`
- Create: `apps/web/src/components/MealItemEditor.vue`
- Create: `apps/web/src/components/AIEstimateNotice.vue`
- Create: `apps/web/src/pages/today/index.vue`
- Create: `apps/web/src/pages/meal/analyzing.vue`
- Create: `apps/web/src/pages/meal/confirm.vue`
- Create: `apps/web/src/pages/meal/manual.vue`
- Create: `apps/web/e2e/meal-flow.spec.ts`
- Modify: `apps/web/src/pages/meal/capture.vue`
- Modify: `apps/web/src/router/index.ts`

**Steps:**

- [ ] 先写 today store 的 loading/empty/error/ready 四态测试，错误不得被显示为零摄入。
- [ ] 实现今日四项进度和餐别列表；超标展示真实累计及超出量，不能只靠颜色表达。
- [ ] 先写草稿 store 测试：选图、重选、AI 失败、保存失败、页面返回均保留日期、餐别和编辑值。
- [ ] 第一次 AI 分析前展示单独同意；拒绝时不调用分析 API 并转手工录入。
- [ ] 实现上传中、分析中、超过 12 秒仍在处理、ready、部分结果、低置信度、failed、expired 全状态。
- [ ] 实现编辑器即时合计、字段定位错误、至少一项校验和“估算值”标识。
- [ ] 保存按钮在请求中禁用；每次用户意图生成并复用一个 Idempotency-Key，网络重试不能换键。
- [ ] 保存成功直接使用 API 返回/重新拉取的目标日期汇总，清除草稿并进入今日或目标日期页面。
- [ ] Playwright 用固定图片和 fake analyzer 覆盖成功、失败后手工、重复保存三条路径。

**Verify:**

- `npm --prefix apps/web run test:unit -- --run`
- `npm --prefix apps/web run test:e2e -- meal-flow.spec.ts`

**Acceptance mapping:** `AC-TODAY-01`–`AC-TODAY-06`、`AC-IMG-01`–`AC-IMG-04`、`AC-AI-01`–`AC-AI-09`、`AC-EDIT-01`–`AC-EDIT-05`、`AC-SAVE-01`–`AC-SAVE-06`、`AC-PRIV-01`–`AC-PRIV-02`。

**Exit criteria:** 在浏览器中实际完成一餐闭环；任一步失败均可恢复且不生成重复记录。

---

## Task 15：实现 H5 历史、详情、修改、移日、删图与删餐

**规模：** L
**依赖：** Task 13–14
**目标：** 用户能按日期回看并维护已确认记录，所有受影响日期同步刷新。

**Files:**

- Create: `apps/web/src/api/history.ts`
- Create: `packages/api-contract/src/history.ts`
- Create: `apps/web/src/stores/history.ts`
- Create: `apps/web/src/components/MonthCalendar.vue`
- Create: `apps/web/src/components/ConfirmDialog.vue`
- Create: `apps/web/src/pages/history/index.vue`
- Create: `apps/web/src/pages/meal/detail.vue`
- Create: `apps/web/src/pages/meal/edit.vue`
- Create: `apps/web/e2e/history.spec.ts`
- Modify: `apps/web/src/router/index.ts`

**Steps:**

- [ ] 先写月历标记、用户时区今天、月份切换与空日不创建数据的 store 测试。
- [ ] 实现历史页：日历只标记正式记录；选中日期复用 `DaySummary` 展示目标、汇总和餐次。
- [ ] 实现详情和完整编辑；PATCH 必须携带当前 expectedRevision。
- [ ] revision 冲突时提示数据已变化并重新载入，不能静默覆盖用户值。
- [ ] 使用 `affectedLocalDates` 精确刷新移日前后日期，不在客户端猜测汇总。
- [ ] 删除整餐和移除图片都使用明确动词与二次确认；取消时不得发送请求。
- [ ] Playwright 覆盖修改营养、移到另一日期、只删图片、取消删除、确认删餐。

**Verify:**

- `npm --prefix apps/web run test:unit -- --run history`
- `npm --prefix apps/web run test:e2e -- history.spec.ts`

**Acceptance mapping:** `AC-HIS-01`–`AC-HIS-09`、`AC-GOAL-07`–`AC-GOAL-11`、`AC-PRIV-02`、`AC-PRIV-05`。

**Exit criteria:** 历史、详情和汇总在修改后无长期矛盾；删除确认和并发冲突行为可观察。

---

## Task 16：实现“我的”、资料/目标维护与账户删除

**规模：** L
**依赖：** Task 3、Task 6、Task 12、Task 15
**目标：** 完成设置入口、次日生效提示、退出登录和短信二次验证删号。

**Files:**

- Create: `server/internal/account/service.go`
- Create: `server/internal/account/service_test.go`
- Create: `server/internal/account/handler.go`
- Create: `server/internal/account/handler_test.go`
- Create: `server/internal/account/postgres_test.go`
- Create: `apps/web/src/api/account.ts`
- Create: `packages/api-contract/src/account.ts`
- Create: `apps/web/src/pages/me/index.vue`
- Create: `apps/web/src/pages/me/profile.vue`
- Create: `apps/web/src/pages/me/goals.vue`
- Create: `apps/web/src/pages/me/delete-account.vue`
- Create: `apps/web/e2e/account.spec.ts`
- Modify: `server/internal/httpapi/router.go`
- Modify: `apps/web/src/router/index.ts`

**Steps:**

- [ ] 先写删号验证码必须属于当前用户、purpose 正确、未过期且未消费的测试。
- [ ] 先写账户删除事务测试：业务数据不可再访问、所有 session 立即撤销、图片加入删除任务。
- [ ] 实现 `POST /v1/account-deletions`；重复请求不泄露旧账户状态。
- [ ] 实现“我的”页面及资料、目标编辑；目标结果继续复用 `GoalCalculation`。
- [ ] 清楚展示修改从下一个本地日期生效，今天与历史保持原目标。
- [ ] 删号页面要求验证码与二次确认；成功后清空客户端 store 并回欢迎页。
- [ ] Playwright 覆盖退出、目标修改次日生效、删除账户后三条受保护页面均不可访问。

**Verify:**

- `cd server && go test ./internal/account ./internal/auth`
- `npm --prefix apps/web run test:e2e -- account.spec.ts`

**Acceptance mapping:** `AC-GOAL-06`–`AC-GOAL-12`、`AC-PRIV-03`–`AC-PRIV-08`。

**Exit criteria:** 删除账户后原会话与原数据立即不可用；外部图片删除失败也有耐久重试记录。

---

## Task 17：完成 21 个 API 契约、安全与非功能回归

**规模：** L
**依赖：** Task 3–16
**目标：** 用统一证据证明实现与 `api-v1.md` 一致，并覆盖跨模块风险。

**Files:**

- Create: `server/internal/httpapi/contract_test.go`
- Create: `server/internal/httpapi/security_test.go`
- Create: `server/internal/httpapi/redaction_test.go`
- Create: `server/internal/httpapi/testdata/*.json`
- Create: `apps/web/e2e/full-journey.spec.ts`
- Create: `docs/testing-v1.md`
- Modify: `Makefile`

**Steps:**

- [ ] 为 21 个接口建立 method/path/status/请求响应 fixture 表；校验 camelCase、时间、数值精度和统一错误体。
- [ ] 覆盖未认证、其他用户资源 404、非法 Origin、过期会话、过期草稿、revision 冲突、幂等冲突。
- [ ] 做日志捕获测试，扫描验证码、token、Cookie、完整手机号、图片内容和完整身体资料泄漏。
- [ ] 用真实 PostgreSQL 运行“首次登录 → 目标 → 分析 → 保存 → 编辑 → 历史 → 删除”集成测试；AI、短信、存储只替换外部边界。
- [ ] 用 Playwright 的 Chromium、WebKit 和移动视口执行 full journey；补微信内置浏览器人工检查清单。
- [ ] 检查 44×44 触控区、字号放大、字段错误关联、非颜色状态和破坏性确认。
- [ ] 记录今日页 1 秒 loading 与 3 秒数据出现、AI 超过 12 秒反馈、服务端超时恢复证据。
- [ ] 在 `docs/testing-v1.md` 建立 AC 编号到测试名/人工证据的可追溯矩阵，所有 72 条 AC 必须有归属。

**Verify:**

- `make verify`
- `npm --prefix apps/web run test:e2e -- --project=chromium`
- `npm --prefix apps/web run test:e2e -- --project=webkit`

**Acceptance mapping:** `docs/spec-v1.md` 第 11、12、13 节全部项目。

**Exit criteria:** 21 个接口和 72 条验收标准无未映射项；测试报告能区分自动通过、人工通过、因外部条件阻塞。

---

## Task 18：AI 质量门槛、容器化与发布候选验证

**规模：** XL
**依赖：** Task 17 + 本文第 2 节外部前置条件
**目标：** 形成可以作出“发布/不发布”决定的质量报告和可重复部署包。

**Files:**

- Create: `server/cmd/ai-eval/main.go`
- Create: `server/internal/aieval/evaluator.go`
- Create: `server/internal/aieval/evaluator_test.go`
- Create: `testdata/ai-quality/manifest.example.json`
- Create: `testdata/ai-quality/README.md`
- Create: `Containerfile`
- Create: `.dockerignore`
- Create: `docs/deployment-v1.md`
- Create: `docs/release-checklist-v1.md`
- Modify: `.gitignore`
- Modify: `Makefile`

**Steps:**

- [ ] 定义评测 manifest：图片引用、人工称重、参考营养、菜品类别、隐含用油标签；真实图片默认不纳入 Git。
- [ ] 先写评测器测试：结构化成功率、整餐热量绝对百分比误差、中位数、p90、耗时和成本汇总。
- [ ] 对不少于 50 餐运行真实模型评测；门槛为结构成功率 ≥95%、热量 MdAPE ≤30%、p90 APE ≤60%。
- [ ] 评测失败时停止发布，不通过隐藏项目或只展示总数绕过；调整模型/提示/图片引导后重新执行完整集。
- [ ] 构建多阶段容器；以非 root 用户运行 Go API，加入健康检查和只读根文件系统兼容性。
- [ ] 部署文档写清静态 H5、API、迁移、PostgreSQL、S3、短信、OpenAI、HTTPS、Origin 与回滚顺序。
- [ ] 在空数据库演练迁移、启动、核心 journey、回滚应用版本；数据库回滚仅在迁移明确支持时执行。
- [ ] 运行生产构建并扫描产物，不得包含 `.env`、测试验证码、源图片、评测密钥或 source map 中的秘密。
- [ ] 完成发布清单；只有所有自动测试、人工兼容检查、AI 门槛和外部配置均有证据时标记候选版本。

**Verify:**

- `make verify`
- `make ai-eval`
- `docker build -t formtally:v1-rc .`
- `docker run --rm formtally:v1-rc /app/formtally-api -check-config`

**Acceptance mapping:** `docs/spec-v1.md` 第 12.1–12.5 节及第 13 节发布场景。

**Exit criteria:** 有明确、可重复的发布证据；任一外部依赖或质量门槛缺失时，清单明确显示 blocked，而不是把本地测试通过等同于已可上线。

---

## 3. 全局完成定义

FormTally V1 只有同时满足以下条件才算完成：

- [ ] `docs/api-v1.md` 的 21 个接口全部实现，无额外未评审业务接口。
- [ ] `docs/spec-v1.md` 的 72 条验收标准全部映射到自动测试或明确的人工证据。
- [ ] 三条核心 Playwright 路径在 Chromium 和 WebKit 通过，并完成 Android Chrome、iOS Safari、微信内置浏览器人工检查。
- [ ] 目标公式只在 Go 后端实现；UI 能展示公式名称、版本、输入、步骤、取整、来源和免责声明。
- [ ] 任何日期汇总均可由正式 meal_items 重算；草稿、上传失败、AI 失败不改变汇总。
- [ ] 重复保存不生成重复记录，并发修改不静默覆盖，跨用户访问不泄露资源存在性。
- [ ] 图片私有、签名地址短时有效、删图/删餐/删号后的失去引用对象有删除证据或耐久重试任务。
- [ ] 日志不包含认证秘密、图片内容或完整身体资料。
- [ ] AI 评测达到内部发布门槛，报告记录模型、提示版本、日期、样本量、误差、耗时和成本。
- [ ] 生产构建、迁移、配置校验、健康检查、备份和回滚流程完成演练。
- [ ] 用户明确决定是否进行 Git 提交、推送和部署。

## 4. 建议执行节奏

- 每完成一个 Task，更新本文件复选框和 `docs/testing-v1.md` 的证据，不等到最后补记。
- 每完成一个里程碑，运行截至该阶段的完整测试，并在真实 H5 中走一遍该里程碑的用户流程。
- 默认先执行 Task 1；如果要用子任务并行开发，先由用户明确授权，再按“不同任务不写同一文件”的边界分配。
