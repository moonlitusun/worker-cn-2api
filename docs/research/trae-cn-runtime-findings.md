# Trae CN 第二轮：本机历史运行日志核查

后续状态：已通过额外协议样本完成兼容聊天接入并真实验证，见 [实现与验证记录](trae-cn-implementation.md)。下文为当时仅依据历史日志的结论。

研究日期：2026-10-26。输入为本机 `~/Library/Application Support/TRAE SOLO CN/logs` 下的明文 `.log`，只读扫描；未调用聊天、刷新、签到接口。输出只保存协议元数据，不复制对话、Token 或设备标识。

## 1. 本轮的重要修正

第一轮在安装包中发现了 raw chat 字符串，但这不足以确认本机客户端直接请求该接口。本轮实际请求记录显示，历史 SOLO CN 使用了模型批量配置与工作流接口；辅助聊天接口有明确的标题生成调用上下文。

还发现了版本差异：历史日志请求头出现 `x-ide-version=0.1.41`、`x-ide-version-code=20260728`，第一轮检查的已安装产品为 `0.1.66`。因此本轮结果属于旧版本运行证据，不能描述成当前安装版本的抓包结果。

## 2. 直接可核查的请求记录

扫描 303 个明文日志文件。下面的数量是日志记录数，不保证去重，也不能当作成功率统计。

| 路径 | 发送日志 | 流响应状态日志 | 能确认的事实 |
|---|---:|---|---|
| `POST /api/ide/v1/batch_get_detail_param` | 8 | 未提取到 | 实际模型配置查询使用批量详情接口 |
| `POST /api/agent/v3/llm_utils_chat` | 1 | HTTP 200 × 1 | 三次路径提及均有 `generate_session_title_and_icon` 上下文 |
| `POST /api/agent/v3/workflow/start` | 3 | HTTP 200 × 3 | 历史客户端确实发起工作流请求 |
| `/api/ide/v2/llm_raw_chat` | 0 | 无 | 在扫描范围内没有发现该 URL 的直接发送记录 |

上述请求 host 均为 `trae-api-cn.mchost.guru`。日志还提及 `api5-normal.mchost.guru` 上的批量模型配置 URL，但没有对应的 Fetch 发送记录，不把它算作已确认成功的备用路由。

HTTP 200 只能说明传输层响应成功；没有完整终止事件样本时，不能宣称每次模型生成成功。

## 3. 模型配置请求的结构

历史日志打印的 Rust 请求类型为 `BatchDetailParamRequest`，可确认字段名：

```text
functions
agent_type
current_config_info { config_name, is_custom_model }
mode_type
access_type
ab_force_vids
ab_autotest_advanced_mode
```

观察到的 functions：`assistant`、`solo_agent_lite`、`solo_coder`、`solo_agent_remote`、`solo_work_lite`、`solo_work_remote`、`solo_design_lite`、`solo_design_remote`、`builder`。

日志中枚举值为 `Manual`、`SoloLite`。这是 Rust Debug 表示，不是已确认的 JSON wire 值，不能直接据此写请求序列化实现。

另有一条 `CustomModel` 实例打印了 32 个字段名，包括 `config_name`、`config_source`、`model_name`、`encrypted_model_params`、`raw_chat_function`、`prompt_set`、`session_token`、`reasoning_effort` 等。这证明真实客户端使用模型配置对象，进一步说明裸模型 ID 不足以描述所有调用状态。字段存在不代表该次调用有非空值，也不代表所有字段都是必填。

## 4. 请求头能确认到哪一层

`[HTTPClient] add_header` 日志包含产品、版本、设备、追踪头名称。脱敏证据中保留头名称与格式受限的产品版本，不保留头值。

这些日志不等于完整最终请求头：例如配置请求记录的 `headers_count=20`，而中间 `add_header` 映射没有完整展示全部 20 项。因此不能因为中间日志没出现 Authorization 或校验头，就推断请求不需要鉴权或签名。

## 5. 不能误读的服务端耗时记录

共发现 12 条 `TimingCostOriginEvent`，其中 JSON 的 `name` 出现 `llm_raw_chat_v2`。其调用栈位于 `do_create_cloud_agent_task` / `call_server_generate_plan_item`。

合理推断：工作流内部可能调用 raw chat 服务。证据只到服务端耗时标签这一层，不能据此证明客户端直接发往 `/api/ide/v2/llm_raw_chat`，更不能从中恢复完整 raw 请求体。

日志还出现等待服务端 `ModelConfig` 事件覆盖本地配置的警告。这提示模型配置可能在会话过程中更新，但警告本身不能证明已收到该事件，或它的完整 SSE 结构。

## 6. 下一步的开发约束

当前还不能编写可用的真实 Trae Provider：缺少一套同版本的完整请求、模型描述符和 SSE 帧。现在直接将社区请求模板挂到生产路由，会把未经验证的通道和版本组合混在一起。

优先验证顺序调整为：

1. 明确目标是 Trae Code CN 还是 SOLO/Work CN，以及本机目标产品的版本；两者各自记录凭据来源和调用契约。
2. 取同版本的模型配置响应与聊天请求；若目标是 OpenAI 模型桥接，验证 raw 通道可否使用该账号正常调用。
3. 若只能走 workflow，则单独评估其 Agent 生命周期与工具执行行为，不能直接伪装成无状态模型调用。
4. 从完整 SSE 样本实现事件转换，再开放 `trae/` 路由。未知终止或工具字段应明确报错，不猜测补全。

本轮数据库文件无法通过标准 SQLite 只读连接识别，未进一步尝试解密。部分新日志使用 `.alaudalog`，本轮未解析；“未发现”仅适用于已扫描明文日志。

## 7. 可复核材料

- `trae-cn-log-evidence.json`：脱敏汇总、源日志哈希、行号、固定白名单协议字段。
- `../../scripts/inspect_trae_logs.py`：可重复运行的离线提取工具，不发起网络请求。

在代码仓库根目录执行：

```sh
python3 scripts/inspect_trae_logs.py \
  "$HOME/Library/Application Support/TRAE SOLO CN/logs"
```

已用合成日志验证 Token、设备 ID、Prompt 和加密参数不会进入输出，且能正确提取字段名、请求方法与响应状态。此检查验证研究工具的过滤与提取逻辑，不代表 Trae 协议或账号已通过联调。
