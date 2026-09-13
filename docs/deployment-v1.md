# FormTally V1 部署与回滚

## 组件边界

- H5：`apps/web/dist` 是静态文件，部署到 HTTPS 静态站点/CDN；API 基址由同源反向代理或部署配置提供。
- API：根目录 `Containerfile` 构建 `/app/formtally-api`，运行用户为 `formtally`（非 root），不依赖写入根文件系统。
- 数据：PostgreSQL 18；迁移由 `server/cmd/migrate` 显式执行，API 启动不自动迁移。
- 外部边界：私有 S3 兼容对象存储、短信供应商、OpenAI。对象桶不得公开，模型名只能来自 `OPENAI_MODEL`。

## 配置

生产环境必须通过秘密管理器注入：`DATABASE_URL`、`S3_ACCESS_KEY`、`S3_SECRET_KEY`、`OPENAI_API_KEY`、短信供应商凭据和强随机 `IMAGE_URL_SECRET`。非秘密配置包括：

- `HTTP_ADDR=0.0.0.0:8080`
- `ALLOWED_ORIGINS=https://实际-H5-域名`（逗号分隔精确 origin，不带路径）
- `APP_ENV=production`
- `SMS_DRIVER=<已实现并审查的生产驱动>`
- `STORAGE_DRIVER=s3`
- `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET`
- `OPENAI_MODEL`、`OPENAI_BASE_URL`（默认 `https://api.openai.com/v1`，不含接口路径）、`OPENAI_API_STYLE`（`responses` 或 `chat_completions`，Qwen 使用后者）、`OPENAI_TIMEOUT`（正时长）

当前代码只包含显式测试短信发送器；生产短信适配器、签名、模板和测试手机号未提供前，`APP_ENV=production` 会拒绝测试发送器启动，这是发布阻塞而不是可绕过配置。

仅校验配置而不连接数据库：

```sh
/app/formtally-api -check-config
```

## 构建与发布顺序

1. 运行 `make verify-rc`；确认 AI 报告不是使用示例 manifest。
2. 备份 PostgreSQL，并记录备份 ID、恢复演练时间和当前 migration version。
3. 在空的预发布数据库运行 `DATABASE_URL=... go run ./server/cmd/migrate up`，再运行 `status`。
4. 构建 H5 与 API 镜像：`make build-release container-build`；按摘要而不是可变 tag 推送。
5. 先部署兼容旧应用版本的数据库迁移，再部署 API；检查 `/healthz`，然后部署静态 H5。
6. 以允许的 Origin 走“登录 → 目标 → 记录 → 历史”smoke journey，确认日志只有 request ID、method、path、status、duration。
7. 观察错误率、p95 延迟、数据库连接、对象删除重试和 AI failed 状态，再扩大流量。

容器运行示例（值仅示意，凭据必须来自秘密文件或平台注入）：

```sh
docker run --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  -p 127.0.0.1:8080:8080 \
  --env-file /secure/formtally-production.env \
  formtally:v1-rc
```

Docker 健康检查访问 `http://127.0.0.1:8080/healthz`。外部负载均衡也应使用该路径；它只表示 API 进程可响应，核心 smoke journey 才验证依赖可用性。

## 回滚

1. 停止继续放量，保留失败版本日志和 request ID。
2. 若迁移向后兼容，先把 API/H5 回滚到上一镜像摘要；不要自动回滚数据库。
3. 只有对应 goose migration 明确提供安全的 Down、已备份且在预发布演练过，才执行 `go run ./server/cmd/migrate down`。涉及数据删除或不可逆变换时，从备份恢复到独立实例并切换流量。
4. 回滚后重新检查配置、`/healthz`、登录与核心 journey，确认对象删除 worker 没有重复删除仍被引用对象。

## HTTPS、数据与合规

公网入口必须只开放 HTTPS，Cookie 保持 `Secure; HttpOnly; SameSite=Lax`。限制数据库与 S3 网络入口和最小权限，启用备份、生命周期、审计与密钥轮换。面向中国大陆用户前必须确认域名、部署地区、备案/合规流程，以及经审阅的用户协议、隐私政策和 AI 图片处理说明。
