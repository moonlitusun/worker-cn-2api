# Qoder 恢复记录（2026-10-26）

版本：`qoder-restore-b2a41deb4c9d`。地址：http://172.16.11.67:10081/admin 。

## 原因与修复

此前发布的版本没有保留 Qoder 入口，配置结构也没有保留原来的 `pat` 字段。此次从服务器原备份恢复 PAT，重新接入 Qoder；WorkBuddy 和 Trae 继续保留。

- Qoder 模型以 `qoder/` 前缀展示；原有无前缀 Qoder 显示名继续匹配真实目录。
- 恢复管理页 Qoder PAT 卡片以及受登录保护的 `/admin/api/qoder`、兼容 `/admin/api/pat`。
- 增加原 PAT 在其他渠道配置、API Key 和统计保存时仍被保留的回归覆盖。
- 移除渠道页重复说明文案，对话测试的额度提示放在按钮上。

## 实际验证

- 线上目录为 41 个唯一条目：Qoder 9 个、WorkBuddy 32 个。
- Qoder 前缀非流式、流式对话，以及旧模型名称对话全部通过。
- 原有 4 个 API Key、管理密码、WorkBuddy 和 Trae 配置保留。
- 容器 healthy，管理页面包含 Qoder 卡片，用户指出的重复文案已移除。
- Qoder 原备份未被修改，凭据无需重新填写。

二进制 SHA-256：`b2a41deb4c9d5db58483e7bc66b8c03377e672eeb402c689ed037d0f3c9a2d84`。
发布前备份：`/home/roc/service/worker-cn-2api/backups/pre-qoder-restore-b2a41deb4c9d`。
