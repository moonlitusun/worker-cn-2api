# Worker-CN-2API

> 将国内 AI 服务（QoderWork、腾讯 WorkBuddy）桥接为 OpenAI 兼容 API。

本项目在 [Qoder-2API-Go](https://github.com/EchoPing07/Qoder-2API-Go) 基础上演进，支持聚合接入 Qoder 和 WorkBuddy，将上游专有协议转换为标准的 OpenAI 接口格式（`/v1/chat/completions` 和 `/v1/models`），便于接入 NextChat、Chatbox、Cherry Studio、Cline、Cursor 等客户端。

## 功能特性

- **OpenAI API 兼容** —— 支持 `/v1/chat/completions` 与 `/v1/models` 端点，支持流式与非流式调用、多轮对话、图片输入和 Tool Calls
- **多渠道自动路由** —— 根据请求的 `model` 自动分发至 WorkBuddy 或 Qoder 上游网关，亦可使用 `workbuddy/` 前缀显式指定
- **思考过程提取** —— 针对 DeepSeek-R1 推理模型，自动解析 `<think>` 标签并转为标准的 `reasoning_content` 流式增量
- **硬件指纹推导** —— WorkBuddy 适配器内置确定性硬件指纹派生算法（根据 Token 自动计算 `derive_id`、MAC 及 Machine ID），降低触发平台风控的概率
- **长连接与保活** —— 内置 SSE 心跳机制与首包即刻下发，配合本地并发队列与超时熔断，避免长连接假死与网关 504 超时
- **真实用量统计** —— 还原上游 token 真实消耗（含 prompt / completion / cached / reasoning tokens），支持 Qoder 的 credits 扣费与折扣统计
- **Web 管理面板** —— 中文管理界面（支持深浅模式），可在线管理 API 密钥、配置 PAT 与 WorkBuddy Token、查看模型列表及近 24 小时调用趋势
- **单文件部署** —— 纯 Go 编写（仅使用 `uuid` 外部依赖），内存占用低，零外部运行时依赖

## 支持的模型

### WorkBuddy 渠道

| 模型名称 | 上游模型 Key | 说明 |
| :--- | :--- | :--- |
| `deepseek-v3` | `deepseek-v3` | 通用编程与文本生成模型 |
| `deepseek-r1` | `deepseek-r1` | 推理模型，支持提取 `reasoning_content` 思考过程 |
| `hunyuan-code` | `hunyuan-standard-code` | 腾讯混元代码专精模型 |
| `kimi-k2` | `kimi-k2` | Kimi 长上下文模型 |
| `glm-4` | `glm-4-plus` | 智谱 GLM-4 模型 |

> 所有 WorkBuddy 模型均同时支持 `workbuddy/<model_id>` 形式的别名调用（如 `workbuddy/deepseek-r1`）。

### Qoder 渠道

| 显示名称 | 内部 Key |
| :--- | :--- |
| Qwen3.8-Max | `qmodel_38max` |
| Qwen3.7-Max | `qmodel_latest` |
| Qwen3.7-Plus | `qmodel` |
| Qwen3.6-Flash | `q36fmodel` |
| DeepSeek-V4-Pro | `dmodel` |
| DeepSeek-V4-Flash | `dfmodel` |
| GLM-5.3 | `gmodel` |
| GLM-5.2 | `gm51model` |
| Kimi-K2.7-Code | `kmodel` |
| MiniMax-M2.7 | `mmodel` |

> 以上为内置默认列表，服务启动后会自动从 Qoder 网关拉取最新模型目录。部分模型的图片输入由网关辅助识别提供支持。

## 快速开始

### 方式一：Docker Compose（推荐）

使用 `docker-compose.yaml`：

```yaml
services:
  worker-cn-2api:
    image: worker-cn-2api:latest
    build: .
    container_name: worker-cn-2api
    restart: unless-stopped
    ports:
      - "10081:10081"
    environment:
      WORKER_HOST: "0.0.0.0"
      WORKER_PORT: "10081"
      WORKER_DATA_PATH: /app/data/data.json
      TZ: "Asia/Shanghai"
      WORKER_MAX_CONCURRENCY: "2"
      # 也可在启动后直接在 Web 管理面板中配置 Token
      # WORKER_WORKBUDDY_TOKEN: "your-workbuddy-auth-token"
    volumes:
      - ./data:/app/data
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:10081/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 5s
```

启动命令：

```bash
docker-compose up -d
```

### 方式二：直接运行

```bash
# 编译
go build -o worker-cn-2api .

# 启动（默认监听 0.0.0.0:10081）
./worker-cn-2api
```

收到 `SIGINT`（Ctrl+C）或 `SIGTERM` 时，程序会等待在途请求处理完毕并落盘统计数据后安全退出。

## 配置

### 环境变量

系统支持 `WORKER_*` 环境变量，同时完全兼容原有的 `QODER_*` 变量：

| 变量名 | 兼容旧变量 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `WORKER_HOST` | `QODER_HOST` | `0.0.0.0` | 监听地址 |
| `WORKER_PORT` | `QODER_PORT` | `10081` | 监听端口 |
| `WORKER_DATA_PATH` | `QODER_DATA_PATH` | `data.json` | 数据文件持久化路径 |
| `WORKER_ADMIN_PASSWORD` | `QODER_ADMIN_PASSWORD` | `password` | Web 管理面板密码（优先于 data.json） |
| `WORKER_WORKBUDDY_TOKEN` | `CODEBUDDY_AUTH_TOKEN` | 空 | WorkBuddy 鉴权 Token |
| `WORKER_MAX_CONCURRENCY` | `QODER_MAX_CONCURRENCY` | `1` | 单账号上游并发数上限，超出部分在本地排队 |
| `WORKER_QUEUE_TIMEOUT_SECONDS` | - | `120` | 本地排队最大等待超时（秒），超时返回 429 |
| `WORKER_CHAT_TIMEOUT_SECONDS` | `QODER_CHAT_TIMEOUT_SECONDS` | `120` | Chat 响应头等待超时（1-3600 秒） |
| `WORKER_IDLE_TIMEOUT_SECONDS` | `QODER_IDLE_TIMEOUT_SECONDS` | `300` | 流式传输空闲超时（秒） |

> 超时时间与渠道 Token 也可在 Web 管理面板中修改并即时生效，无需重启服务。

### 配置文件 (data.json)

首次运行时会自动创建，支持在管理界面在线修改：

```json
{
  "host": "0.0.0.0",
  "port": 10081,
  "pat": "pt-xxxxxxxx",
  "workbuddy_token": "your-workbuddy-auth-token",
  "password": "password",
  "chat_timeout_seconds": 120,
  "idle_timeout_seconds": 300,
  "api_keys": [
    {
      "id": "c1a2b3d4",
      "key": "sk-xxxxxxxxxxxxxxxx",
      "note": "NextChat",
      "created_at": 1740000000
    }
  ]
}
```

### 健康检查

`GET /health` 端点无需鉴权，可用于容器探针或反向代理心跳检测：

```json
{
  "status": "ok",
  "has_pat": true,
  "has_workbuddy": true,
  "chat_timeout_seconds": 120,
  "idle_timeout_seconds": 300
}
```

## 使用说明

### 1. 配置上游凭证

服务启动后，访问 `http://localhost:10081/admin`（默认密码 `password`，首次登录后请在「设置」中修改）。

在「令牌」页面配置所需渠道：
- **Qoder PAT**：填入 `pt-` 开头的 PAT 令牌。
- **WorkBuddy Token**：填入 WorkBuddy 的 `auth_token`。若在装有 WorkBuddy / CodeBuddy 客户端的本机上运行，程序启动时会自动检测本地凭证文件并载入。

### 2. 创建 API Key

在管理界面的「API 密钥」标签页创建密钥，调用 API 时通过 HTTP Header 传入：

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxx
```

### 3. API 调用示例

#### 流式调用 DeepSeek-R1（带思考过程）

```bash
curl http://localhost:10081/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-r1",
    "messages": [
      {"role": "user", "content": "9.11 和 9.9 哪个大？"}
    ],
    "stream": true
  }'
```

#### 调用 Qoder 渠道模型

```bash
curl http://localhost:10081/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen3.7-Max",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'
```

#### 查询可用模型列表

```bash
curl http://localhost:10081/v1/models \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxx"
```

## API 端点

| 端点 | 方法 | 鉴权要求 | 说明 |
| :--- | :--- | :--- | :--- |
| `/v1/chat/completions` | POST | Bearer Key | 对话补全（兼容 OpenAI 格式） |
| `/v1/models` | GET | Bearer Key | 获取聚合模型列表 |
| `/health` | GET/HEAD | 无 | 服务健康状态探针 |
| `/admin` | GET | 无 | Web 管理面板 |
| `/admin/api/login` | POST | 无 | 管理面板登录（带 IP 退避限流） |
| `/admin/api/logout` | POST | 无 | 退出登录 |
| `/admin/api/auth` | GET | 无 | 会话有效性检查 |
| `/admin/api/keys` | GET/POST/DELETE | Admin 会话 | API 密钥管理 |
| `/admin/api/pat` | GET/POST | Admin 会话 | Qoder PAT 令牌配置 |
| `/admin/api/workbuddy` | GET/POST | Admin 会话 | WorkBuddy Token 配置 |
| `/admin/api/models` | GET | Admin 会话 | 获取当前可用模型列表 |
| `/admin/api/stats` | GET | Admin 会话 | 请求量与 token 统计数据 |
| `/admin/api/config` | GET/POST | Admin 会话 | 超时及监听配置 |
| `/admin/api/password` | POST | Admin 会话 | 修改管理密码 |

## 安全机制

- **HTTP 超时防护**：设置 `ReadHeaderTimeout=10s` 与 `IdleTimeout=120s`，防御 slowloris 慢速连接；SSE 流式响应空闲超时自动断开，防止网关连接卡死
- **请求体上限**：`/v1/chat/completions` 限制 10 MiB，管理 API 限制 64 KiB，防止大 Payload 占用过多内存
- **登录限流**：管理面板登录按客户端 IP 计数，密码错误后指数退避锁定（1s 到 30s），锁定期间返回 `429 Too Many Requests`
- **常量时间比较**：API Key 鉴权与管理密码校验均使用 `crypto/subtle.ConstantTimeCompare`，避免时序侧信道泄露
- **Cookie 安全策略**：会话 Cookie 设置 `HttpOnly` 与 `SameSite=Lax`，并在 HTTPS 或带 `X-Forwarded-Proto: https` 请求下自动启用 `Secure`
- **XSS 防护**：Web 面板使用原生 DOM 渲染和事件监听，避免内联脚本注入

## 项目结构

```
worker-cn-2api/
├── main.go               # 程序入口与服务生命周期管理
├── router.go             # 多渠道请求分发与聚合模型处理
├── admin/                # Web 管理面板与管理 API
├── auth/                 # Qoder 协议加解密与鉴权
├── bridge/               # Qoder 网关桥接与并发队列
├── models/               # Qoder 动态模型目录
├── stats/                # 请求统计与持久化落盘
├── store/                # JSON 文件持久化存储
├── transform/            # Qoder 协议与 OpenAI 协议互转
├── workbuddy/            # WorkBuddy 协议适配器（指纹派生、思维链提取）
├── Dockerfile           # Docker 构建文件
└── docker-compose.yaml  # Docker Compose 配置
```

## 技术栈

- **语言**：Go 1.22+
- **依赖**：仅使用 `github.com/google/uuid`，其余全部为标准库
- **HTTP**：标准库 `net/http`，SSE 流式输出，支持平滑优雅退出
- **存储**：单文件 JSON 持久化，线程安全读写

## 开发构建与镜像发布

### 本地编译

```bash
# 运行测试
go test ./...

# 编译
go build -o worker-cn-2api .

# 交叉编译（以 Linux arm64 为例）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o worker-cn-2api-linux-arm64 .
```

### Docker 镜像构建与发布

本项目提供了完整的 GitHub Actions 自动化 CI 流水线（支持多架构 `linux/amd64` 与 `linux/arm64`）：

#### 1. 自动发布（通过 GitHub Actions 推送至 Docker Hub & GHCR）

1. **Docker Hub 访问令牌配置**：
   - 登录 [Docker Hub](https://hub.docker.com/) $\rightarrow$ `Account Settings` $\rightarrow$ `Security` $\rightarrow$ `New Access Token`（权限选择 Read & Write）。
   - 在 GitHub 仓库 `Settings` $\rightarrow$ `Secrets and variables` $\rightarrow$ `Actions` 中添加：
     - `DOCKERHUB_USERNAME`: Docker Hub 用户名
     - `DOCKERHUB_TOKEN`: 刚才获取的 Access Token
2. **打 Tag 自动发布**：
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
   推送后会自动打包多架构镜像并发布到：
   - `docker.io/<username>/worker-cn-2api:v0.1.0` 与 `:latest`
   - `ghcr.io/moonlitusun/worker-cn-2api:v0.1.0` 与 `:latest`

#### 2. 本地/服务器手动构建与推送

```bash
# 登录 Docker Hub
docker login -u <username>

# 构建并推送到 Docker Hub
docker build -t <username>/worker-cn-2api:latest .
docker push <username>/worker-cn-2api:latest
```

## 许可证

本项目基于 [MIT License](./LICENSE) 开源。

## 免责声明

本项目仅供个人学习与研究目的使用。

1. **本项目不隶属于 Qoder、腾讯或其关联公司**，Qoder 与 WorkBuddy 是其各自所有者的商标。本项目不对任何官方服务提供保证或支持。
2. **仅供学习交流使用**，不得用于任何商业用途或非法用途。使用者应遵守所在地区的法律法规。
3. **使用者需自行承担使用风险**，项目维护者不对因使用本项目导致的账号限制、服务中断或数据异常承担直接或连带责任。
4. **请勿将本项目用于高频爬取、滥用或可能损害服务提供方利益的行为**。请尊重官方平台的服务条款。


## 推理开关与参数兼容性

Qoder 与 WorkBuddy 通道共同识别以下客户端思考开关：

- `thinking: {"type": "enabled" | "disabled"}`（Cora 的 DeepSeek 格式）。
- `reasoning_effort: "none" | "off"` 关闭；`minimal/low/medium/high/xhigh/max` 开启。
- `enable_thinking: true | false`，或 `chat_template_kwargs.enable_thinking`。

多种写法同时出现时必须一致，否则在访问上游之前返回 HTTP 400；无效类型和枚举同样返回 400。不传开关时保留原有开启行为。

两个上游目前仅映射布尔开关，不保证保留推理强度档位。Qoder 的 `temperature`、`top_p`、`max_tokens`、`max_completion_tokens`、`stop`、`frequency_penalty`、`presence_penalty`、`seed` 没有已验证的字段映射，会列入 `X-Worker-Unsupported-Parameters` 响应头；开启思考的 `reasoning_effort` 也列出，表示仅应用开关、未应用强度。WorkBuddy 保留原有的 `temperature` 和 `max_tokens` options 映射。

这层映射保证代理向上游发送的开关值，实际模型是否支持关闭仍需针对账号和模型验证。代理不会通过删除模型返回的思考内容来假装关闭思考。
