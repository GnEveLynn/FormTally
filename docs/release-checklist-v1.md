# FormTally V1 发布清单

候选结论：**BLOCKED，不可公开发布**。本地实现可以形成发布包，但以下外部质量、供应商、凭据和设备证据尚未提供。

## 本地自动门禁

- [x] 21 个 API method/path 均由生产 Handler 注册并符合统一 JSON 错误协议。
- [x] 未认证、非法 Origin、所有权隔离、过期/冲突/幂等回归有自动测试。
- [x] 日志脱敏扫描验证码、token、Cookie、完整手机号、图片内容和完整身体资料。
- [x] PostgreSQL 18 跨模块 journey 使用真实迁移数据库；数据库包串行运行。
- [x] 普通 Go 套件不注入 `TEST_DATABASE_URL`。
- [x] H5 单元测试与生产构建通过。
- [x] Chromium 移动项目通过（20/20）。
- [x] WebKit 移动项目通过（20/20）。
- [x] 72/72 AC 已映射，0 条未映射。
- [x] AI evaluator 与 CLI 单元测试通过；示例 manifest 返回 blocked。
- [ ] ≥50 餐真实模型评测达到结构成功率 ≥95%、MdAPE ≤30%、p90 APE ≤60%。
- [x] 多阶段容器以非 root 用户运行，包含 HTTP 健康检查与 `-check-config`。
- [x] 容器 build/run、非 root、配置校验、健康检查与 `--read-only --tmpfs /tmp` 演练完成。
- [x] 迁移空库 up/down 自动测试存在；生产数据迁移不自动回滚。
- [x] H5 与 Go 生产产物扫描不包含环境文件、私钥、源图片和已知秘密模式。

## 外部阻塞

- [ ] 中国大陆短信供应商、签名、模板、生产适配器和测试手机号。
- [ ] 生产 PostgreSQL、私有 S3 最小权限凭据及备份恢复演练。
- [ ] OpenAI API 凭据、最终 `OPENAI_MODEL` 和完整真实评测报告。
- [ ] 不少于 50 餐、具合法测试用途的人工称重/营养标注图片集。
- [ ] 用户协议、隐私政策和 AI 图片处理说明正式文本审阅。
- [ ] Android Chrome、iOS Safari（当前及前两个主要版本）与微信内置浏览器人工检查。
- [ ] 域名、HTTPS、部署地区及中国大陆备案/合规确认。

任何一项未完成时保持 BLOCKED；不得通过删除失败样本、使用 fake 凭据、把 Playwright fixture 当真实服务或把示例 manifest 当真实模型结果来勾选。
