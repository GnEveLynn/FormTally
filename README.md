# FormTally

移动端饮食记录 H5 与微信原生小程序。功能设计见 [规格](docs/spec-v1.md)，小程序本地测试与导入步骤见 [小程序测试指南](docs/testing-miniprogram.md)。

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

微信小程序服务端能力默认关闭。开发前需在服务端同时配置 `WECHAT_APP_ID` 和 `WECHAT_APP_SECRET`；`WECHAT_API_BASE_URL` 默认使用 `https://api.weixin.qq.com`，仅本地自动化测试可改为假微信服务地址，生产环境会拒绝非微信官方 HTTPS 主机。AppSecret、微信临时 code 和会话令牌不得写入客户端配置或日志。

## 验证

```sh
make verify
```

`verify` 执行 Go 测试、真实 PostgreSQL 集成测试、H5 单元与浏览器旅程测试、TypeScript 检查及生产构建。也可分别使用 `make test-go`、`make test-db`、`make test-web`、`make test-e2e`、`npm run build:web`。

小程序使用微信原生 TypeScript/WXML/WXSS。运行 `make test-miniprogram`、`make build-miniprogram` 和 `make scan-miniprogram` 分别执行单测、类型构建与敏感产物扫描；项目导入目录是 `apps/miniprogram`。仓库不包含真实 AppID、AppSecret、线上域名或个人开发者工具配置。

`make migrate-down` 回滚当前数据库迁移。Playwright 默认复用本机 Chrome 运行 H5 浏览器旅程。
