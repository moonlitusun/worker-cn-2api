# Trae 企业版 CLI PAT 接入

日期：2026-10-28（Asia/Taipei）。

## 行为

在原有 Trae 令牌输入框填写 `trae-lt-…` 即自动进入企业版流程。使用 `https://api.enterprise.trae.cn/cloudide/api/v3/trae/oauth/ExchangeToken`，JSON 为 `RefreshToken`（PAT）及企业 CLI ClientID `app_EMoamEEZ73f0CkXaXp7hrann`，从 `Data` 读取短期访问令牌。

原始 PAT 保留在配置中，短期 JWT 只缓存在内存；到期前一分钟或服务重启后重新兑换。刷新通过现有账号级互斥去重，账号切换清空缓存，迟到的旧兑换不会覆盖新账号。错误不输出上游登录响应内容或凭据。

企业目录走 `batch_get_detail_param`、`functions=[chat]`、`access_type=4`。模型使用目录声明的 `model_detail_list.model_name`，优先 `__dev`，无则首个非空后端键；跳过自定义模型和辅助配置。聊天走企业版 `llm_raw_chat`，每请求独立 conversation/session，并沿用既有文本、工具、SSE 和用量转换。个人版 JWT 保留原有兼容通道。

## 证据和验证

- 官方 CLI 0.211.1 发布清单及程序来自 `lf-cdn.trae.com.cn`，下载 SHA-256 为 `a6f871b43ccbddc4b3e365d7864f8993d62a7c2cf4a5797f70b0d14f938b6f64`。
- 官方 CLI 使用虚构 PAT 对本机模拟服务验证兑换请求；没有在 CLI 中导入用户真实登录。
- 企业 raw_chat 请求参考 Fleey/deepseek-harness-trae 固定源码 `3cd54a4af176ccca29297c09246729c8a8520f08`，并用现有用户令牌真实验证。
- 全部 Go 功能测试与静态检查通过，Trae/admin 竞态检测通过；新增兑换并发去重、过期重兑、切换缓存隔离、错误脱敏及企业目录/聊天协议测试。
- 真实探针：目录 5 个可见模型；`Doubao-Seed-2.0-Code` 非流式、流式、required 工具调用及工具结果续轮通过。目录可见不保证每个模型都已逐个验证。
- 完整网关复核：两次短回复各输入 35 / 输出 23 Token；Trae 渠道统计增量与之相同。后台连接测试通过；合并 `/v1/models` 返回 48 个模型；原始 PAT 保留，续期状态与有效期可见，容器 healthy。

## 发布

镜像：`worker-cn-2api:enterprise-pat-20261028`。

回滚配置：`/home/roc/service/worker-cn-2api/docker-compose.yaml.pre-enterprise-pat`。回滚镜像：`worker-cn-2api:usage-tabs-20261028`。
