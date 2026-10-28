# Trae 配置简化与线上诊断

日期：2026-10-27（Asia/Taipei）。

- 常规入口只需访问令牌，用户 ID、续期配置、运行详情和命名账号默认折叠。
- 保存后自动执行新鲜模型目录测试，不把保存成功当作连接成功。
- 支持粘贴带 Cloud-IDE-JWT / Bearer 前缀的令牌，持久化原始值。
- 更新令牌时不带入旧过期时间，默认从新 JWT 推导。
- 认证、权限和超时显示中文提示，原始诊断保留在运行详情。

## 验证

- 服务端 `go test ./...`、`go vet ./...` 通过。
- 本机 `go test -race -ldflags=-linkmode=external ./...` 通过。Go 1.22 在当前 macOS 上需外部链接，存在系统链接器警告。
- JavaScript 语法、元素 ID 唯一性、保存后自动测试、失败提示及按钮恢复检查通过。
- 真实线上连接测试返回上游 HTTP 401、code 1001，认证失败。线上令牌非可解析 JWT，未保存刷新令牌；未改动用户凭据。目录验证失败，因此未继续模型聊天测试。
- 更新后健康检查 HTTP 200，容器 healthy，新界面内容及提示已通过 HTTP 确认。浏览器视觉检查工具因工作目录路径兼容问题无法启动，未完成视觉验收。

## 部署

经用户明确授权上传源码并更新现有服务器。

- 镜像：`worker-cn-2api:trae-simplify-20261027`。
- 二进制 SHA-256：`4d060fca1efec7d901e9dad96ca9ab4c7fecac91f434464f1c4a4b06c33d0f5c`。
- 回滚镜像：`worker-cn-2api:rollback-trae-simplify-20261027`。
- 原部署配置：`/home/roc/service/worker-cn-2api/backups/pre-trae-simplify-20261027/docker-compose.yaml`。
