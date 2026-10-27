# 发布记录：trae-admin-738bc86d191e

- 发布时间：2026-10-26T14:23:49+08:00（Asia/Taipei）。
- 管理地址：http://172.16.11.67:10081/admin （内网）。
- API Base URL：http://172.16.11.67:10081/v1 。
- 容器：`worker-cn-2api`；镜像：`worker-cn-2api:trae-admin-738bc86d191e`。
- 部署目录：`/home/roc/service/worker-cn-2api`。
- 发布内容：已验证工作区的源码快照和 Linux amd64 二进制；基础 Git 提交 `7004911c3440ef6d4e3fd6571de99ded3c490c51`，包含本次未提交改动，不把它误记为已发布 Git 标签。
- 二进制 SHA-256：`738bc86d191eb393e5736450b4f5ddc6a4e7123326f922003a10331df96c83c4`。
- 源码快照 SHA-256：`e9ccf0ea608783d945380ef5ea8a4ec523378b5f236080937761bbbc2a0bf443`。

## 验证结果

- 容器状态 healthy；从开发机直接访问管理页面返回 HTTP 200。
- 管理员登录、新后台标记、Trae 状态与命名账号接口检查通过。
- 保留原有管理密码、4 个 API Key、WorkBuddy 凭据及持久化数据。
- 原有 WorkBuddy 模型目录正常，返回 64 个接口条目（含带渠道前缀的兼容名称）。
- 服务器 Trae 当前未配置账号；可在渠道凭证页配置。本机个人 Trae 会话未随发布迁移。

## 备份与回退

发布前已备份镜像、Compose、运行数据、二进制和旧源码，验证失败自动回退。备份目录：

`/home/roc/service/worker-cn-2api/backups/pre-trae-admin-738bc86d191e`

服务器保留 `releases/trae-admin-738bc86d191e/manifest.json` 和 `deployment-result.json`。后续人工回退前应另外备份当时的最新运行数据，以保留发布后新增的密钥或账号配置。
