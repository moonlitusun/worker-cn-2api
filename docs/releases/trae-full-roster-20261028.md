# Trae 当前 IDE 模型目录修复

日期：2026-10-28（Asia/Taipei）。

## 原因与修复

前次企业版接入仅读取旧 CLI `chat` 目录，过滤后仅剩 5 个模型。当前账号的新版 IDE 目录位于 `chat_v3`；实测只调整版本号不能补全旧目录。

现在同时查询 `chat_v3` 和 `chat`，优先使用当前 IDE 目录，按模型去重并保留旧目录独有模型；每个模型携带其所属 function 和上游 model_name，不仅扩大列表。使用与固定参考源码一致、经过真实调用验证的企业 raw-chat 兼容请求配置；个人版兼容请求保持原样。

当前账号目录为 23 个模型，包含 Doubao-Seed-2.1、GLM-5.3、DeepSeek-V4.1、Kimi-K3 等。仅展示可见、有后端键的模型，排除辅助及自定义占位配置。目录可见不代表已逐个验证所有模型权限。

补充忽略 `progress_notice` 的非 JSON 排队提示，仅该信息事件例外；其他事件继续严格校验 JSON 和终止状态。探针错误附带阶段、事件名，不含响应正文。

## 验证

- 完整 Go 功能测试、静态检查通过，Trae/admin 竞态测试通过。
- 新增目录优先级与去重、正确 function/backend 路由、排队信息兼容和错误帧拒绝测试。
- GLM-5.3：真实非流式、流式、工具调用与工具结果续轮通过。
- DeepSeek-V4.1-Flash、Kimi-K3：真实非流式、流式通过。

## 发布

镜像：`worker-cn-2api:trae-full-roster-20261028`。

回滚配置：`/home/roc/service/worker-cn-2api/docker-compose.yaml.pre-full-roster`。保留上一镜像 `worker-cn-2api:clean-copy-20261028`。
