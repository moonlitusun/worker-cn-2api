# worker-cn-2api

将 Qoder、WorkBuddy 和 Trae **中国区**账号接入 OpenAI Chat Completions 客户端（包括 Cora）。

## Docker Compose 部署（推荐）

已发布镜像：[moonlitusun/worker-cn-2api](https://hub.docker.com/r/moonlitusun/worker-cn-2api/tags)，支持 `linux/amd64` 和 `linux/arm64`。无需下载源码或安装 Go。

### 1. 创建配置

新建一个部署目录，在其中保存以下 `docker-compose.yaml`：

```yaml
services:
  worker-cn-2api:
    image: moonlitusun/worker-cn-2api:1.0.0
    container_name: worker-cn-2api
    restart: unless-stopped
    ports:
      - "10081:10081"
    environment:
      TZ: Asia/Shanghai
      WORKER_HOST: "0.0.0.0"
      WORKER_PORT: "10081"
      WORKER_DATA_PATH: /app/data/data.json
      WORKER_ADMIN_PASSWORD: "${WORKER_ADMIN_PASSWORD:?请在 .env 中设置管理密码}"
    volumes:
      - ./data:/app/data
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:10081/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 5s
```

在同一目录创建 `.env`，将示例密码替换为自己的强密码：

```dotenv
WORKER_ADMIN_PASSWORD=请替换为自己的强密码
```

### 2. 启动并配置账号

```sh
docker compose up -d
```

访问 `http://服务器IP:10081/admin`（本机部署使用 `http://localhost:10081/admin`），使用 `.env` 中的密码登录：

1. 在管理页面保存 WorkBuddy 或 Trae 中国区账号 Token。
2. 创建供客户端使用的 API Key。
3. 在客户端填写下方 Base URL 和 API Key，并获取模型列表。

配置了 `WORKER_ADMIN_PASSWORD` 时，管理密码由环境变量管理；修改 `.env` 后执行 `docker compose up -d` 生效，不能在管理页面修改。

账号配置、API Key 和统计保存在部署目录的 `data/` 中，重建容器后仍会保留。请妥善保存 `.env` 和 `data/`，不要公开分享。桌面账号凭证不会自动同步到 Docker；也可通过 `WORKER_WORKBUDDY_TOKEN` / `WORKER_TRAE_TOKEN` 环境变量配置渠道。

### 3. 查看日志与更新

```sh
# 查看日志
docker compose logs -f --tail=100

# 拉取镜像并重建容器
docker compose pull
docker compose up -d

# 停止并删除容器（保留本地 data/）
docker compose down
```

上述配置固定使用 `1.0.0`。发布新版本后，将 `image` 的版本号改为目标版本，再执行更新命令；也可使用 `moonlitusun/worker-cn-2api:latest` 跟随最新发布。

## 源码运行

需要 Go 1.22 或更高版本：

```sh
go build -o worker-cn-2api .
WORKER_WORKBUDDY_TOKEN='你的中国区 accessToken' ./worker-cn-2api
```

也可在源码目录使用仓库自带的 `docker-compose.yaml` 本地构建：

```sh
docker compose up -d --build
```

未设置 `WORKER_ADMIN_PASSWORD` 时，初始管理密码为 `password`，登录后请修改密码、保存渠道 Token，并创建客户端 API Key。本机运行时可自动读取支持的桌面账号凭证。

## Cora 配置

- 协议：OpenAI Chat Completions
- Base URL：`http://你的服务器:10081/v1`
- API Key：管理页面创建的 Key
- 模型：从 `/v1/models` 获取，统一返回 `qoder/<id>`、`workbuddy/<id>` 和 `trae/<id>`，不再重复返回无前缀别名。聊天接口仍兼容旧的 WorkBuddy 原始 ID 配置。

不再提供旧的 `hunyuan-code`、`deepseek-v3` 等硬编码别名，也不会把无效模型悄悄映射到其他模型。聊天请求将原始模型 ID 交给上游验证。

## API

| 接口 | 说明 |
|---|---|
| `POST /v1/chat/completions` | Bearer API Key；流式、非流式聚合、工具调用、推理内容 |
| `GET /v1/models` | Bearer API Key；中国区实时模型目录，成功后缓存一小时 |
| `GET /health` | 无需鉴权；服务状态及 `has_workbuddy` / `has_trae` 凭证配置状态（不代表账号已验证） |
| `/admin` | 中文管理界面：Token、API Key、模型目录、调用统计、服务配置 |

WorkBuddy 上游聊天地址默认为 `https://www.workbuddy.cn/v2/chat/completions`，目录来自同域 `/v2/enterprises/personal/models`。目录请求失败会返回错误，不会展示虚构的内置列表；聊天不依赖目录请求成功。多个渠道同时配置时，目录包含所有已配置渠道；任一渠道目录失败则明确返回错误。

WorkBuddy 上游始终使用 SSE，客户端非流式请求会在服务端聚合。工具定义、工具结果、多段 `tool_calls`、推理、用量及结束原因会保留。指定函数形式的 `tool_choice` 会转换为只保留该工具并设置 `required`，兼容上游只接受字符串的约束。上游接受请求之前不写入 HTTP 200；上游 HTTP 错误返回具体状态与详情，流开始后的错误使用 SSE `error` 对象返回。

## 环境变量

| 变量 | 默认值 / 说明 |
|---|---|
| `WORKER_HOST` | `0.0.0.0` |
| `WORKER_PORT` | `10081` |
| `WORKER_DATA_PATH` | `data.json`；Docker 为 `/app/data/data.json` |
| `WORKER_ADMIN_PASSWORD` | 覆盖管理密码 |
| `WORKER_WORKBUDDY_TOKEN` | 中国区 accessToken；优先级：管理页面保存值、此变量、`CODEBUDDY_AUTH_TOKEN`、本地桌面凭证 |
| `WORKER_WORKBUDDY_ENDPOINT` | 覆盖上游聊天 URL，主要用于测试 |
| `WORKER_WORKBUDDY_CLIENT_VER` | `5.5.4` |
| `WORKER_CHAT_TIMEOUT_SECONDS` | `120`，等待上游响应头的超时 |
| `WORKER_IDLE_TIMEOUT_SECONDS` | `300`，上游流无数据超时 |

可在管理页面调整响应头、空闲超时，对新请求即时生效。每个账号最多两个并发请求，等待队列超过 15 秒返回 429。Token 到期后需更新；当前不自动刷新 OAuth 凭证。

以上并发和手动更新策略适用于 WorkBuddy。Trae 使用下述独立策略。

## Qoder CN

模型目录来自已验证的 Qoder 会话，以 `qoder/` 前缀展示。旧的无前缀 Qoder 显示名按真实目录匹配，WorkBuddy 和 Trae 的显式前缀保持独立路由。原有 PAT 保存在 `pat` 字段，不会被其他渠道配置或统计保存覆盖。

## Trae CN

使用 `solo_work_lite` 兼容聊天通道：

- 鉴权为 `Cloud-IDE-JWT`，聊天路径 `/api/agent/v3/llm_utils_chat`，模型目录 `/api/ide/v1/get_detail_param`。
- 模型目录按账号缓存一小时，凭证变化后失效；只显示可见且启用的配置，过滤已确认属于 raw-agent 通道的配置。模型出现在目录中不保证当前套餐可调用。
- 支持文本、推理内容输出、函数工具调用及结果续轮、流式和非流式聚合、用量返回。流式用量需设置 `stream_options.include_usage=true`。
- 显式指定函数的 `tool_choice` 转成函数名；工具参数 schema 转成上游所需的字符串。错误、截断及缺少终止事件不会伪报成功。
- 当前不支持图片、Remote/workflow Agent、Responses API、`reasoning_effort`、`thinking`、`response_format` 或 `parallel_tool_calls` 控制；这些请求明确返回 400，不暗中换模型或通道。上游模型仍可自然产生推理内容。
- 每个 Bridge 采用一个并发槽位与 15 秒排队上限；这是保守的本地调度策略，不代表实测账号并发上限。流建立后每 3 秒发送心跳。

本机运行时，未配置 Token 会依次尝试读取 `TRAE SOLO CN`、`Trae-CN` 的 `User/globalStorage/storage.json`，仅支持已验证的 `tc` AES 凭证格式。macOS 根目录为 `~/Library/Application Support`，Windows 为 `%APPDATA%`，Linux 为 `~/.config`；不改写桌面客户端文件。

包含 RefreshToken 的会话在到期前一分钟自动刷新，并把轮换会话写入本服务的 `data.json`（文件权限 0600）。过期时间来自会话字段或 JWT `exp`。手动配置只有 Token 时不自动刷新；刷新失败需重新登录或更新凭据。管理页面可选择 Trae Code CN / SOLO CN 产品类型并填入可选刷新凭据。清空渠道后本次运行立即停用；重启后仍会按环境变量和桌面凭据进行自动发现。

| 变量 | 默认值 / 说明 |
|---|---|
| `WORKER_TRAE_TOKEN` | 原始 Cloud-IDE-JWT，不带前缀；优先使用管理页面保存的会话 |
| `WORKER_TRAE_USER_ID` | 可选账号 UserID |
| `WORKER_TRAE_REFRESH_TOKEN` | 可选，需与 Token 属于同一会话 |
| `WORKER_TRAE_EXPIRES_AT` | 可选 ISO 8601 过期时间；省略时尝试 JWT `exp` |
| `WORKER_TRAE_CLIENT_ID` | Code CN：`ono9krqynydwx5`；SOLO CN：`en1oxy7wnw8j9n` |
| `WORKER_TRAE_DEVICE_ID` / `WORKER_TRAE_MACHINE_ID` | 可选，使用账号自己的稳定设备标识 |
| `WORKER_TRAE_BASE_URL` | `https://trae-api-cn.mchost.guru` |
| `WORKER_TRAE_AUTH_URL` | `https://api.trae.com.cn` |
| `WORKER_TRAE_VERSION` / `WORKER_TRAE_VERSION_CODE` | 已验证兼容配置 `0.1.61` / `20260820` |

真实账号检查（只输出结果，不打印凭据；刷新会保存到指定运行配置）：

```sh
go run ./cmd/trae-probe -models-only
go run ./cmd/trae-probe -model glm-5.2 -tools
```

`-model` 使用目录里的原始 ID。第二条会产生少量模型调用和一次无外部副作用的 `ping` 工具续轮。2026-10-26 本机账号已通过自动刷新、目录、`glm-5.2` 流式/非流式及工具续轮验证；未逐个验证目录中的所有模型。

WorkBuddy 本地桌面凭证支持新版 `auth.accessToken` 与旧版 `auth_token`，仅接受中国区账号。文件位置：

- macOS：`~/Library/Application Support/CodeBuddyExtension/Data/Public/auth/workbuddy-desktop.info`
- Windows：`%LOCALAPPDATA%/CodeBuddyExtension/Data/Public/auth/workbuddy-desktop.info`
- Linux：`~/.local/share/CodeBuddyExtension/Data/Public/auth/workbuddy-desktop.info`

旧配置文件中的 Qoder `pat`、API Key、管理密码、WorkBuddy Token 和统计都会保留。Qoder 可在渠道凭证中管理，兼容 `/admin/api/pat`；也可使用 `WORKER_QODER_PAT` 或 `QODER_PAT`。

实现细节和真实账号结果见 [Trae 接入完成记录](docs/research/trae-cn-implementation.md)。

管理后台已补充连接测试、可选短对话测试、到期与续期状态、最近错误、模型搜索和能力信息，以及多个命名账号的手动保存/切换/删除。操作细节见 [后台使用说明](docs/trae-cn-admin-guide.md)。管理页面在单个渠道目录失败时保留其他渠道结果；对外模型 API 仍按严格策略返回错误。

## 验证

```sh
go test -race ./...
go vet ./...
go build ./...
```

测试覆盖上游强制流式、流式透传、非流式工具参数拼接、用量、错误返回和动态目录。真实账号可用模型以接口实际返回为准。
