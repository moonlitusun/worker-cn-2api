# Trae CN 接入研究与适配设计

后续状态：兼容聊天通道已完成接入与真实账号验证，见 [实现与验证记录](trae-cn-implementation.md)。下文保留首轮研究时的证据边界。

研究日期：2026-10-26（Asia/Taipei）。范围：对照用户提供的总体设计，核查本机客户端、公开实现与当前 Go 工程，形成开发依据。本轮没有读取账号凭证、刷新 Token 或调用真实聊天接口，因此不代表账号连通性验证通过。

## 1. 结论与原设计修正

Trae CN 有接入线索，但不能按 WorkBuddy 的 OpenAI JSON 透传方式接入。建议先验证单账号、单一聊天通道，再实现独立 `trae` 包。

| 原设计假设 | 本轮核查结果 | 对实现的影响 |
|---|---|---|
| 火山引擎 OpenAPI Token | 本机客户端明确出现 Cloud-IDE-JWT、OAuth ExchangeToken、GetUserToken | 将网页登录凭据、刷新凭据和可能的下游服务票据区分建模；不能套用火山 AK/SK |
| 国内版使用 api.trae.ai | 本机产品配置指定国内 API 与独立 agent 网关 | 登录域名与模型域名分开配置；国际版域名不能作为默认回退 |
| 标准 SSE 可直接转发 | 客户端存在专用 raw chat 与模型配置字段 | SSE 只是传输格式，需要单独研究事件与消息转换 |
| 主要工作是 DeviceID/InstallID 伪装 | 已发现模型绑定字段及请求校验头；没有验证签名算法 | 优先恢复请求契约，设备身份保持稳定；不宣称改设备 ID 就能解决鉴权 |
| Doubao/DeepSeek/GPT 等固定模型表 | 静态安装包不能证明账号当前可用模型 | 模型目录按账号和功能动态发现，不能把设计中的名称当作可用清单 |
| 立即 HTTP 200 能彻底消除 504 | 当前工程先校验上游 HTTP 状态，再开启 SSE | 保留真实 HTTP 错误语义；心跳只能处理流开启后的下游空闲，不能消除所有超时 |

## 2. 本机客户端证据（一级证据）

检查对象为 `/Applications/TRAE SOLO CN.app/Contents/Resources/app`。产品 `appVersion=0.1.66`，底层 `version=1.107.1`，构建时间 `2026-09-15T06:58:12.417Z`，commit `6d9ed14f693545a76d84c8e25a6cdb67fa9a9a4d`。不能把底层 VS Code 版本用作 Trae 产品版本请求头。

可复核的文件哈希及字符串偏移保存在同目录 `trae-cn-local-evidence.json`。偏移仅适用于该安装版本。

### 2.1 产品配置

`product.json` 的配置值：

| 配置项 | 值 |
|---|---|
| `bootConfig.iCubeAgent.normal` | `https://api.trae.com.cn` |
| `bootConfig.agent.trae.normal` | `https://trae-api-cn.mchost.guru` |
| `bootConfig.consoleHost` | `https://www.trae.cn` |
| `iCubeApp.authConfig.TRAE.stable` | `ono9krqynydwx5` |
| `iCubeApp.authConfig.SOLO.stable` | `en1oxy7wnw8j9n` |

这些是安装包配置事实，并不证明当前运行实例一定使用这些地址：运行时配置可能覆盖。尤其不能仅根据域名外观判定 `mchost.guru` 是第三方代理；本机安装包直接包含该地址。

### 2.2 鉴权与刷新

`out/main.js` 包含 `Authorization: Cloud-IDE-JWT ...` 的构造，以及 `POST {host}/api/v2/GetUserToken`、`X-Cloudide-Token`。这提示至少存在不同用途的票据调用，但还不能断言所有聊天接口都必须先换票。

`out/vs/workbench/workbench.desktop.main.solo-lite.js` 包含刷新请求：

```text
POST {resolvedHost}/cloudide/api/v3/trae/oauth/ExchangeToken
JSON fields: ClientID, RefreshToken, ClientSecret: "-", UserID: ""
```

这里只确认了请求代码存在；刷新 host 的运行时解析、有效期和返回字段仍需用合法登录会话验证。不能假定固定 14 天或固定刷新周期。

### 2.3 模型与聊天

`modules/ai-agent/libharness.dylib` 包含：

```text
api/ide/v1/model_list
api/ide/v1/get_model_list
api/ide/v1/get_detail_param
api/ide/v1/batch_get_detail_param
api/ide/v2/llm_raw_chat
api/ide/v1/llm_raw_chat
```

同一二进制包含 `config_name`、`model_name`、`encrypted_model_params`、`raw_chat_function`、`prompt_set`、`ab_versions`、`session_token` 等字段，以及 `X-Request-Pin`、`X-Requested-At` 字符串。

`libai_agent.dylib` 还包含 `llm_raw_chat_custom_model` 与 `pass_back_reasoning` 的日志/字段。它们支持“模型描述符参与请求、推理有独立处理”的研究方向，但字符串存在不能证明完整 JSON 层级、必填关系或签名计算方式。

## 3. 公开实现交叉验证（二级线索）

以下为社区项目维护者的实现与报告，不等于官方协议保证，也没有在本轮真实账号上验证。

- [Trae2api-cn 的 raw_client.py](https://github.com/autumnsentiment/Trae2api-cn/blob/main/src/raw_client.py) 区分新 raw chat 与旧 Work 协议；其代码将模型绑定、功能选择及加密模型参数纳入请求构造。该方向与本机二进制字段相符。具体默认版本号不应照抄到本机版本。
- [trae_client.py](https://github.com/autumnsentiment/Trae2api-cn/blob/main/src/trae_client.py) 同时保留 raw、旧辅助聊天与 Remote 会话路径，说明这些不是可无条件互换的单一协议。
- [sse.py](https://github.com/autumnsentiment/Trae2api-cn/blob/main/src/sse.py) 实现事件转换和独立心跳，工具交还调用方执行。可作为转换边界参考。
- [cpa-multi-plugins 协议记录](https://github.com/mmqz/cpa-multi-plugins/blob/main/docs/PROTOCOL.md) 给出 `llm_utils_chat`、`solo_work_lite`、具名 SSE 事件及工具字段转换线索。不过它对网关归属的描述与本机产品配置存在冲突；以安装包事实记录冲突，不沿用未经核查的归属判断。

研究优先级（经后续日志核查修正）：raw chat 仍是通用聊天桥接的候选，但不能称为已确认的本机直连路径。本机历史 SOLO CN 日志实际出现 workflow/start；llm_utils_chat 关联标题生成。先区分产品版本和调用层，再验证候选通道。详见同目录 `trae-cn-runtime-findings.md`。Remote 和 workflow 涉及服务端 Agent 会话，应作为独立模式研究，避免调用者的工具执行语义被上游 Agent 接管。

## 4. 当前工程接入点与边界

当前代码并未实现总体设计中的 Provider SPI，也没有多账号池：

- `main.go` 只初始化 WorkBuddy Bridge。
- `router.go` 的聊天与目录 handler 直接接收 `*workbuddy.Bridge`，模型校验只接受 WorkBuddy ID。
- `workbuddy/bridge.go` 包含单 Bridge 并发槽位、15 秒排队、3 秒 SSE 心跳和同步聚合。
- `workbuddy/client.go` 的事件读取只向回调交付 data 内容；新 Trae 解析器必须保留 `event`，不能直接复用并丢弃事件类型。
- 现有错误类型和管理配置绑定 WorkBuddy，需要在第二渠道落地时逐步解耦。

本轮只新增研究文件，保留工作区原有未提交改动。用户给出的笔记目录是设计输入；报告落在当前代码工作区，便于与后续适配实现一起审阅。

## 5. 建议实施方案（尚未实现）

### 5.1 先完成一条真实协议闭环

1. 固定客户端版本和产品类型，从一次正常客户端请求取得脱敏样本：目录请求、模型描述符、raw chat 请求和完整 SSE。
2. 使用同一模型描述符复现最短纯文本请求；先排除签名与功能绑定问题。
3. 验证非流式聚合、推理分离、工具请求与工具结果续轮。
4. 这条链路成立后再挂载网关路由和管理页面。不要一次引入所有通道与自动降级。

### 5.2 模块划分

```text
trae/config.go       产品类型、国内 host、版本、超时
trae/auth.go         凭证快照、刷新互斥、过期判断
trae/models.go       目录查询、完整描述符、按账号/功能缓存
trae/client.go       请求、校验头、状态与业务错误
trae/transform.go    消息、工具与模型配置转换
trae/events.go       完整 SSE 帧解析与 Trae 事件归一化
trae/bridge.go       流式输出、同步聚合、取消与资源释放
```

先抽取满足两个渠道的最小 Provider 接口，避免照搬尚未验证的宏大 SPI。新增 `trae/<upstream-id>` 命名空间；现有无前缀 WorkBuddy 模型行为保留。目录聚合需显示渠道级错误，不能用虚构模型掩盖失败。

### 5.3 凭据与模型状态

账号状态至少区分产品类型、ClientID、UserID、AccessToken、RefreshToken、两个过期时间、稳定设备身份和国内 host。每次请求持有一致的凭据快照；刷新用账号级互斥并原子保存新旧字段，避免并发覆盖。

模型内部保存完整上游描述符，不把 `config_name`、展示名和 `model_name` 合并成一个字符串；加密字段作为不透明配置保留，不自行伪造或解密。缓存按账号、产品、功能和协议版本隔离；失效后重新发现。目录失败时能否继续聊天取决于是否仍有有效描述符，不能照搬 WorkBuddy 的裸 ID 直通逻辑。

### 5.4 SSE 与错误语义

用统一内部事件表达文本、推理、工具增量、用量、排队、完成和错误，再分别生成 OpenAI 流式与非流式响应。

- SSE 帧保留 `event`、`data`，支持多行 data、注释、CRLF、分片与大小限制。
- 工具参数根据真实样本区分增量和累计值，不能无条件拼接；工具 ID 和多轮结果关联保持稳定。
- 上游 HTTP 成功后仍可能出现业务错误；正常 EOF 也不能自动当作成功完成。
- 只在确认成功终止时发送正常 finish 和 `[DONE]`；流中断向调用方报告失败。
- 保持单一写入出口，串行化心跳与业务数据；客户端取消后立即关闭上游流并释放槽位。
- 已输出内容后不透明重试；发送前的回退也需要验证是否产生上游计费或任务副作用。
- HTTP 状态提交前可返回明确 4xx/5xx；提交后只能通过 SSE 报错。两种行为不能同时承诺。

## 6. 验证清单与完成条件

| 阶段 | 必须取得的证据 | 完成条件 |
|---|---|---|
| 登录/刷新 | 脱敏字段结构、过期时间、ClientID 与 host 关系 | 刷新后目录与聊天均成功，并发只刷新一次 |
| 模型目录 | 当前账号返回值及完整描述符 | ID、功能、能力可追溯；失败不返回硬编码清单 |
| 最短聊天 | 一次请求和完整事件序列 | 流式与聚合正文一致，终止事件明确 |
| 工具 | 强制工具、并行调用、两轮结果续写 | ID 与参数完整，网关不执行调用者工具 |
| 推理/图片 | 模型能力和对应真实事件 | 仅声明验证通过的能力，不暗中换模型 |
| 错误 | 无效模型、失效凭据、限流、流内错误、截断 | 状态与错误可辨；截断不伪报成功 |
| 韧性 | 长等待、客户端取消、目录更新 | 心跳有效、无槽位泄漏、无凭据/目录串账号 |

当前状态：静态协议研究完成；真实请求、签名规则、模型清单、工具能力、并发上限和账号刷新均待验证。下一开发门槛是取得一套同版本脱敏样本并完成单通道最短请求，不能把本报告标成“Trae 已接入”。
