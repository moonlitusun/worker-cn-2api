# 发布记录：trae-admin-d7f2f87356cc

日期：2026-10-26。地址：http://172.16.11.67:10081/admin （内网）。

本次修正模型发现：WorkBuddy 仅返回 `workbuddy/<id>`，Trae 使用 `trae/<id>`；移除无前缀重复发现条目。旧 WorkBuddy 无前缀聊天请求继续兼容。

线上目录验证为 32 个唯一条目，全部具有渠道前缀。此前的 64 个条目是 32 个 WorkBuddy 配置分别以原始名称和前缀名称各展示一次，不代表额外接入了其他上游服务。

模型来自 WorkBuddy 实时账号目录，实际请求也走 WorkBuddy；客户端保存的“Qoder CN”名称不由服务端模型目录控制。客户端可重新 Discover 获取最新目录；保留的旧缓存可自行移除。

管理员登录、容器 healthy、原有 4 个 API Key 和数据保留验证通过。服务器 Trae 仍未配置账号。

- 镜像：`worker-cn-2api:trae-admin-d7f2f87356cc`。
- 二进制 SHA-256：`d7f2f87356ccc58b9e50d04ac21dc922720cb53d044d6276a9393cda1b90c3a3`。
- 备份：`/home/roc/service/worker-cn-2api/backups/pre-trae-admin-d7f2f87356cc`。
- 源码为通过回归测试的工作区快照，未提交为 Git 标签。
