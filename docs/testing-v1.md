# FormTally 验证记录

## Task 1：开发基座

工作分支：`codex/task1-dev-foundation`。范围为单仓库、H5 启动页和 Go API 生命周期。

| 验证项 | 命令 / 方式 | 结果 |
| --- | --- | --- |
| Go 测试先失败 | 原有 `TestHealth` 调用尚未实现的 `NewRouter` | 已确认 `undefined: NewRouter` |
| 配置和生命周期测试先失败 | `config_test.go`、`server_test.go` | 已确认缺少 `LoadConfig`、`NewServer`、`Serve` |
| Go 本机工具链预检 | Go 1.24.11，临时模块文件；不修改仓库 Go 版本 | 通过全部 Go 测试及 API 构建 |
| 在途请求优雅停止 | `TestServeDrainsInFlightRequest`，真实 TCP 连接 | 通过，取消上下文后等待响应完成 |
| API 实际进程 | Go 1.27.0 编译产物，请求 `/healthz`，分别发 SIGINT/SIGTERM | 通过，响应固定 JSON，进程退出 0，端口释放 |
| API 开发命令 | `make dev-api` 后请求 `/healthz` 并在终端 Ctrl+C | 通过，健康响应 200，记录 `api stopped` 后退出 |
| 统一验证 | `make verify`（Go 1.27.0） | Go 测试、H5 单元测试、类型检查、生产构建全部通过 |
| 非法配置 | 空地址、端口 0、65536、非数字端口 | 通过，非零退出且没有监听日志 |
| Compose 配置 | `docker compose config` | 通过；Task 2 接入 PostgreSQL |
| 预留测试命令 | `make test-db`、`make test-e2e` | 明确返回非零状态并说明接入任务 |
| Go 1.27 完整验证 | `cd server && go test ./...` | 通过（Go 1.27.0） |
| H5 测试先失败 | 缺少 `index.vue` 时运行原有 smoke test | 已确认导入失败，再补页面实现 |
| H5 单元测试 | `npm --prefix apps/web run test:unit -- --run` | 通过（1 项） |
| H5 生产构建 | `npm --prefix apps/web run build` | TypeScript 检查与 Vite 构建通过 |
| H5 浏览器检查 | 390×844、1280×800、刷新、控制台 | 通过；无横向溢出，无 warn/error |
| 本地 H5 清理 | 停止 `make dev-h5` 并关闭临时浏览器页 | 已完成 |

### 环境说明

- 前端基于官方 `create-vite@9.2.0` 的 `vue-ts` 模板，并保留仓库原有首页测试。
- 本机预检使用 Go 1.24.11；最终已按仓库 `go.mod` 的 Go 1.27.0 要求通过测试。工具链放入标准 Go module 缓存，可自动选用。
- npm 使用国内 `registry.npmmirror.com`；锁文件包含 166 个 registry 包的固定版本、下载地址和完整性校验。空目录执行 `npm ci --offline --ignore-scripts` 成功，使用该干净依赖再次通过单元测试与构建。
- 本机 Node.js 为 24.14.0。部分测试工具的传递依赖声明最低 24.15.0，启动文档与根 `engines` 已列明支持范围。
- Task 1 尚无数据库和业务端到端测试；这些项目不计作通过。
