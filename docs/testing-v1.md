# FormTally V1 测试与验收追溯

更新时间：2026-09-13。本文区分“已实现自动测试”“本机已通过”和“外部条件阻塞”，不把 fixture、fake 或示例 manifest 描述成真实外部系统验证。

## 自动化门禁

| 层级 | 命令 | 当前证据 |
| --- | --- | --- |
| Go 单元/组件 | `cd server && env -u TEST_DATABASE_URL go test ./...` | 本机通过；普通套件不获得数据库 URL |
| PostgreSQL 集成 | `make test-db` | PostgreSQL 18，`-p 1` 串行；包含跨模块 full journey |
| H5 单元 | `npm --prefix apps/web run test:unit -- --run` | 19 files / 53 tests 通过 |
| H5 生产构建 | `npm --prefix apps/web run build` | 通过 |
| Chromium / WebKit | `npm --prefix apps/web run test:e2e -- --project=<name>` | Pixel 7 Chromium 20/20、iPhone 15 WebKit 20/20 通过 |
| AI 评测器 | `make ai-eval` | 评测器/CLI 自动测试通过；示例 manifest 明确为 blocked |
| 发布产物 | `make scan-artifacts` | 扫描环境文件、密钥、私钥、源图片和已知测试秘密模式 |

跨模块数据库测试 `TestPostgresFullJourneyAcrossModules` 使用真实迁移数据库，穿过验证码登录、资料、目标、AI 草稿、正式餐食、编辑、历史与删除；短信、模型和对象存储只在外部边界替换为测试实现。

## 72 条 AC 追溯矩阵

状态说明：`自动` 表示由已运行的 Go/Vitest 测试保护；`浏览器` 表示 Playwright 已在 Chromium 与 WebKit 移动项目运行通过；`人工` 表示必须在真实设备或外部服务上执行。

| AC | 状态 | 自动测试或人工证据 |
| --- | --- | --- |
| AC-AUTH-01 | 浏览器 | `auth.spec.ts` 未登录欢迎页 |
| AC-AUTH-02 | 自动 | `TestValidMainlandPhone`；phone domain tests |
| AC-AUTH-03 | 自动 | login component tests；`auth.spec.ts` 协议阻断 |
| AC-AUTH-04 | 自动 | `TestAuthHTTPJourney`、`TestVerificationAndSessionLifecycle` |
| AC-AUTH-05 | 浏览器 | `auth.spec.ts` 会话恢复 |
| AC-AUTH-06 | 自动 | `TestAuthHTTPJourney`；`auth.spec.ts` 退出守卫 |
| AC-ONB-01 | 浏览器 | `onboarding.spec.ts` 多步骤资料 |
| AC-ONB-02 | 自动 | onboarding store/component tests |
| AC-ONB-03 | 自动 | profile-validation tests、`TestProfileValidationAndAutomaticEligibility` |
| AC-ONB-04 | 自动 | `TestCalculateFixedExample`；`onboarding.spec.ts` 解释展示 |
| AC-ONB-05 | 自动 | `TestEligibilityRejectsSpecialHealthContexts`、manual-goal tests |
| AC-ONB-06 | 浏览器 | `onboarding.spec.ts` 保存后进入今日 |
| AC-GOAL-01 | 自动 | `TestCalculateFixedExample`、`TestCalculateVariants` |
| AC-GOAL-02 | 自动 | `TestCalculateVariants` |
| AC-GOAL-03 | 自动 | `TestRoundingUsesHalfAwayFromZero` |
| AC-GOAL-04 | 自动 | `TestManualGoalWorksForSpecialHealthContextAndWarnsOnMacroMismatch` |
| AC-GOAL-05 | 自动 | `TestManualGoalWorksForSpecialHealthContextAndWarnsOnMacroMismatch` |
| AC-GOAL-06 | 自动 | `TestGoalUpdateStartsTomorrowAndKeepsActiveSnapshot` |
| AC-GOAL-07 | 自动 | `TestPastTargetIsCopiedOnceAndRemainsStable`、day tests |
| AC-GOAL-08 | 自动 | `TestPastTargetIsCopiedOnceAndRemainsStable` |
| AC-GOAL-09 | 自动 | `TestCalculateFixedExample`、GoalCalculation component tests |
| AC-GOAL-10 | 自动 | `TestCalculateFixedExample`、`TestReplaySavedV1Input` |
| AC-GOAL-11 | 自动 | `TestReplaySavedV1Input` |
| AC-GOAL-12 | 自动 | GoalCalculation component tests；前端无目标公式实现 |
| AC-TODAY-01 | 自动 | `TestDaySummaryReturnsRealEmptyState`、today store tests |
| AC-TODAY-02 | 自动 | `TestDaySummaryCountsOnlyPersistedMealsAndKeepsRealOverage` |
| AC-TODAY-03 | 自动 | `TestDaySummaryCountsOnlyPersistedMealsAndKeepsRealOverage`、NutritionProgress tests |
| AC-TODAY-04 | 浏览器 | `full-journey.spec.ts` 保存后今日列表 |
| AC-TODAY-05 | 自动 | today store tests；`auth.spec.ts` 失败态重试 |
| AC-TODAY-06 | 浏览器 | `full-journey.spec.ts` 今日进入详情 |
| AC-IMG-01 | 人工 | Android Chrome / iOS Safari / 微信相机调用清单 |
| AC-IMG-02 | 自动 | ImagePicker/image domain tests；`TestProcessImageAcceptsJPEGAndPNGAndStripsMetadata`、WebP test |
| AC-IMG-03 | 自动 | ImagePicker/image domain tests；`TestProcessImageRejectsUnsupportedOversizeAndTooManyPixels` |
| AC-IMG-04 | 自动 | ImagePicker component tests |
| AC-AI-01 | 浏览器 | `full-journey.spec.ts` 第三方 AI 独立确认 |
| AC-AI-02 | 自动 | capture component/manual flow tests |
| AC-AI-03 | 浏览器 | meal-flow/full-journey Playwright 状态序列 |
| AC-AI-04 | 自动 | `TestCreateAnalysisPersistsRecoverableFailedResource` 的成功分支；meal editor tests |
| AC-AI-05 | 自动 | `TestParseResultAcceptsPartialAndLowConfidenceItems`、AI notice tests |
| AC-AI-06 | 自动 | `TestParseResultAcceptsPartialAndLowConfidenceItems` |
| AC-AI-07 | 自动 | `TestCreateAnalysisPersistsRecoverableFailedResource`；`meal-flow.spec.ts` |
| AC-AI-08 | 自动 | `TestPostgresFullJourneyAcrossModules` 保存前/后汇总边界；day tests |
| AC-AI-09 | 自动 | analysis/meals expired tests |
| AC-EDIT-01 | 自动 | meal-editor weight recalculation tests |
| AC-EDIT-02 | 自动 | meal-editor totals tests、`TestTotalsUseFinalEditedValues` |
| AC-EDIT-03 | 自动 | meal-editor add/remove tests |
| AC-EDIT-04 | 自动 | meal-editor validation、`TestValidateCreateRejectsInvalidItemsMealTypeAndFutureTime` |
| AC-EDIT-05 | 自动 | meal-editor field validation、server validation tests |
| AC-SAVE-01 | 浏览器 | confirm component/`full-journey.spec.ts` 保存中禁用 |
| AC-SAVE-02 | 自动 | `TestSaveMealAtomicallyConsumesAnalysisAndIsIdempotent`、meal-flow retry Playwright |
| AC-SAVE-03 | 自动 | `TestMealHandlerCreatesFromFinalEditedValues`、`TestTotalsUseFinalEditedValues` |
| AC-SAVE-04 | 自动 | meal-draft store tests、meal-flow retry Playwright |
| AC-SAVE-05 | 自动 | `TestValidateCreateRejectsInvalidItemsMealTypeAndFutureTime` |
| AC-SAVE-06 | 自动 | `TestPostgresFullJourneyAcrossModules`、day summary tests |
| AC-HIS-01 | 自动 | history store tests |
| AC-HIS-02 | 浏览器 | `history.spec.ts` 月历正式记录标记 |
| AC-HIS-03 | 自动 | `TestHistoryIncludesUserTimezone`、history Playwright |
| AC-HIS-04 | 自动 | `TestPastEmptyDayDoesNotCreateTargetSnapshot`、history empty-state tests |
| AC-HIS-05 | 自动 | `TestPostgresFullJourneyAcrossModules`、history edit Playwright |
| AC-HIS-06 | 自动 | meals/day cross-date service tests、history edit Playwright |
| AC-HIS-07 | 自动 | `TestPostgresFullJourneyAcrossModules`、history delete Playwright |
| AC-HIS-08 | 浏览器 | `history.spec.ts` 取消删除确认 |
| AC-HIS-09 | 自动 | storage/meal remove-image tests、history Playwright |
| AC-HIS-10 | 自动 | `TestSaveMealAtomicallyConsumesAnalysisAndIsIdempotent`、`TestPostgresDraftRoundTripIsolationAndDiscard` |
| AC-PRIV-01 | 浏览器 | `full-journey.spec.ts` 第三方处理文案 |
| AC-PRIV-02 | 自动 | AIEstimateNotice tests；详情/编辑 Playwright |
| AC-PRIV-03 | 自动 | profile/analysis/meal PostgreSQL ownership tests |
| AC-PRIV-04 | 自动 | meal deletion/store worker tests |
| AC-PRIV-05 | 自动 | `TestFilesystemStoreUsesOpaqueKeysAndExpiringPrivateURLs`、S3 store tests |
| AC-PRIV-06 | 自动 | account service/handler/PostgreSQL tests |
| AC-PRIV-07 | 自动 | `TestDeleteConsumesOwnedCodeRevokesDataAndQueuesImages`、account Playwright |
| AC-PRIV-08 | 自动 | `TestAccessLogsRedactAuthenticationImageAndProfileData`、`TestMiddlewareRecoversAndLogsOnlySafeFields` |

矩阵计数：72/72 已映射，0 条未映射。

## 非功能证据

- 触控与可访问性：组件 CSS 为主要操作提供至少 44px 高度；`full-journey.spec.ts` 已在 Chromium 与 WebKit 移动视口检查主要按钮高度、200% 根字号、字段 label、文字化超标状态和破坏性二次确认。
- 性能体验：同一 Playwright 文件把 `/v1/days` 延迟 1.1 秒，检查 loading 仍可见并在 3 秒内出现数据；把分析延迟 12.5 秒，检查持续处理反馈。服务端 OpenAI 超时与可恢复 failed 资源由 analysis tests 覆盖。这是确定性客户端预算测试，不替代真实移动网络测量。
- 兼容性人工清单：当前及前两个主要版本的 Android Chrome、iOS Safari，以及微信内置浏览器，逐项检查浏览、短信登录、相册/相机退化、AI 确认、保存、历史、编辑、删除与系统字号放大。没有设备实测前均为 blocked。
