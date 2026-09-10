# FormTally

移动端饮食记录 H5。功能设计见 [规格](docs/spec-v1.md)，开发顺序见 [实施计划](docs/superpowers/plans/2026-09-10-formtally-v1.md)。

## 本地开发

需要 Go 1.27.x、Node.js 22.22.2+ 或 24.15.0+（建议 Node.js 24 LTS）和 npm。前端使用官方 Vite Vue 3 TypeScript 模板，依赖由根目录 `package-lock.json` 锁定。

```sh
npm ci
make db-up
make migrate
make dev-api
```

另开终端启动 H5：

```sh
make dev-h5
```

H5：<http://127.0.0.1:5173>；API：<http://127.0.0.1:8080/healthz>，返回 `{"status":"ok"}`。两者独立启动，Ctrl+C 停止；API 会等待正在处理的请求结束，最多 5 秒。

开发命令默认连接 `postgres://formtally:formtally@127.0.0.1:5432/formtally`，并只允许 `http://127.0.0.1:5173` 作为写请求来源。可用环境变量覆盖：

```sh
cp .env.example .env
set -a
. ./.env
set +a
make dev-api
```

`.env` 不自动加载、不提交；空值、非法地址或端口会导致 API 在监听前退出。不要在前端 `VITE_*` 变量中放秘密。

## 验证

```sh
make verify
```

`verify` 执行 Go 测试、真实 PostgreSQL 集成测试、H5 单元测试、TypeScript 检查及生产构建。也可分别使用 `make test-go`、`make test-db`、`make test-web`、`npm run build:web`。

`make migrate-down` 回滚当前数据库迁移。`make test-e2e` 将在 Task 4 接入 H5 业务流程测试，目前会返回非零状态并说明原因。
