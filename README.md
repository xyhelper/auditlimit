# AuditLimit

用于内容审核的限流器,本代码更多的是为了演示如果使用限流器,大家可以根据自己的需求进行修改。

## 部署方法

创建`docker-compose.yml`文件

```yml
version: '3'
services:
  auditlimit:
    image: xyhelper/auditlimit
    restart: always
    ports:
      - 9611:8080
    environment:
      OAIKEY: "" # OpenAI API key 用于内容审核
      DEFAULT: "20/3h" # 默认限流,没有单独配置的模型用这一项
      RESEARCH: "2/24h" # 研究模型限流
      AGENT: "2/24h" # 代理模型限流
      AUTO: "200/3h" # Auto 自动选择档, 对应模型名 auto
      GPT-5-6-PRO: "7/24h" # 模型名称: "次数/时间" 时间单位: h(小时) m(分钟) s(秒)  模型名称要改成大写
      GPT-5-6: "60/3h"
      GPT-5_6-SOL-WM: "60/3h" # 模型名中的 "." 写作 "_", 例如 gpt-5.6-sol-wm
      GPT-5-6-T-MINI: "20/3h" # 仅上游 UI 中可选的模型才需要配置
      # 完整模型列表见下方「配置说明 - 当前模型列表」

```

然后执行

```bash
docker-compose up -d
```

限流器接口地址为: `http://ip:9611/audit_limit`

> 从源码构建需要 **Go 1.26 及以上版本**（该要求来自 `golang.org/x/{net,sys,text,time}` 的最新版，见 `go.mod` 中的 `go` 指令）。使用上面的 Docker 镜像部署无需关心此项。

## 配置说明

配置有三个来源，优先级从高到低为：

**`config/config.yaml` > 环境变量 > 内置默认值**

只要 `config.yaml` 中存在某个键就直接采用它 —— **即使该键的值是空字符串**（例如 `OAIKEY: ""` 会屏蔽同名环境变量）。只有某个键在 `config.yaml` 中完全不存在时（例如 Docker 部署未挂载配置文件），才会去读环境变量。

### 一、键名变换规则（模型名 → 配置键）

模型名由上游接口返回，大小写不统一，部分模型名还含有 `.`，因此需要一套固定的映射规则：

| 步骤 | 规则 | 说明 |
| --- | --- | --- |
| 1 | 转为**全大写** | 环境变量名大小写敏感，必须全大写 |
| 2 | 把 `.` 替换为 `_` | 原因见下方 |
| 3 | `-`、`_` 与数字原样保留 | 不需要任何处理 |

| 模型名（上游返回） | 配置键（`config.yaml` / 环境变量） |
| --- | --- |
| `gpt-5-6` | `GPT-5-6` |
| `gpt-5-6-t-mini` | `GPT-5-6-T-MINI` |
| `gpt-5.6-sol-wm` | `GPT-5_6-SOL-WM` |
| `gpt-6-astra-wm` | `GPT-6-ASTRA-WM` |
| `gpt-5.6.1-x` | `GPT-5_6_1-X` |
| `research` / `agent` | `RESEARCH` / `AGENT` |

**为什么必须把 `.` 写成 `_`：**

1. 配置框架把 `.` 当作键的路径分隔符。写 `GPT-5.6-SOL-WM` 时，它会被解析为「`GPT-5` 下面的 `6-SOL-WM`」，永远读不到值；反斜杠转义 `GPT-5\.6-SOL-WM` 也无效。
2. `.` 不是合法的环境变量名字符，shell 里无法直接写成 `GPT-5.6-SOL-WM=10/1h ./auditlimit`。

所以模型名里每一个 `.` 都要改写成 `_`，这是**唯一**需要人工转换的字符，其余字符照抄即可。

### 二、值的格式

模型限流项的值统一为 `次数/时间`：

```yaml
GPT-5-6: "60/3h"         # 每 3 小时最多 60 次
GPT-5-6-PRO: "7/24h"     # 每 24 小时最多 7 次
RESEARCH: "2/24h"        # 每 24 小时最多 2 次
GPT-5_6-SOL-WM: "10/1h" # 每 1 小时最多 10 次
```

- **次数**：正整数。
- **时间**：沿用 Go duration 写法，单位有 `s`（秒）、`m`（分钟）、`h`（小时），也可以组合，如 `90m`、`1h30m`。
- 值必须**恰好包含一段 `/`**。格式不合法时（缺少 `/`、时间单位写错等）会回退到内置兜底值 `40/3h`。
- YAML 中建议始终加引号，避免被解析成别的类型。

限流的统计维度是 **调用方 token + 模型**，其中 token 取自请求头 `Authorization`（去掉前缀 `Bearer `）。

#### 禁用某个模型

把某个模型的值写成 `DISABLED`（不区分大小写，也忽略首尾空白）即可禁止用户使用该模型：

```yaml
# config/config.yaml
GPT-5_5-WM: "DISABLED"
```

```yaml
# docker-compose.yml
environment:
  GPT-5_5-WM: "DISABLED"
```

- 命中的请求返回 **403**，响应体形如 `{"detail":{"code":"model_disabled","message":"..."}}`，详见下方「禁用返回格式」。
- 判断在内容审核与限流之前完成，所以被禁用的模型**不消耗额度**，也不会触发 moderation 调用。
- 模型名与配置键的对应关系与限流值完全一致（见「键名变换规则」），因此写 `GPT-5_6-SOL-WM: "DISABLED"` 就是禁用 `gpt-5.6-sol-wm`。
- 未单独配置的模型会落到 `DEFAULT`，由 `DEFAULT` 决定是否禁用；把 `DEFAULT` 写成 `DISABLED` 即可禁用所有未单独配置的模型。
- 请求体 `system_hints` 中的 `research` / `agent` 会覆盖模型名，进而命中 `RESEARCH` / `AGENT`，所以一般**不要**把这两项设为 `DISABLED`，否则研究 / 代理模式会整体不可用。

两个容易踩的边界：

- **`DEFAULT: "DISABLED"` 会连带禁用 `AUTO`、`RESEARCH`、`AGENT`。** 这三个都是普通模型键，删掉后对应模型会回退到 `DEFAULT`。若只想禁用“没名字的“未知模型，需要把 `AUTO`、`RESEARCH`、`AGENT` 显式写成合法限流值（如 `AUTO: "200/3h"`）。同理，任何需要放行的模型都必须显式配置。
- **请求体中 `model` 字段缺失或为空时会直接返回 403**，而不是 400。此时响应消息里的模型名位置是空白，形如 `The model  is disabled.`。若 `DEFAULT` 不是 `DISABLED`，空模型名会回退到 `DEFAULT` 的限流值而正常放行，不会报错。

### 三、特殊键

除模型键外，还有几个固定用途的键：

| 键 | 含义 | 内置默认值 |
| --- | --- | --- |
| `DEFAULT` | 未单独配置的模型统一走这一项 | 代码兜底 `40/3h` |
| `PORT` | 监听端口 | `8080` |
| `OAIKEY` | 内容审核所用 API key，留空则不启用审核 | `""` |
| `MODERATION` | moderations 接口地址 | 内置网关地址 |

说明：

- `DEFAULT` 本身也是一个模型键，只是充当所有未命中模型的兜底项。若 `DEFAULT` 缺失或其值格式不合法，则使用代码内置的 `40/3h`。
- `RESEARCH`、`AGENT`、`AUTO` 都是普通模型键（分别对应 `research`、`agent`、`auto` 三个模型名），不是特殊键，详见「当前模型列表」。也就是说，**删掉它们并不会让对应模型不受限流，而是会回退到 `DEFAULT`**。
- `RESEARCH` 与 `AGENT` 的特别之处在于：当请求体的 `system_hints` 中出现 `research` / `agent` 时，模型名会被强制替换为 `research` / `agent`，从而命中这两个键。
- 把任意模型键的值写成 `DISABLED` 即可禁止用户使用该模型，详见「值的格式 - 禁用某个模型」。
- `OAIKEY`、`MODERATION`、`PORT` 只在进程启动时读取一次，修改后需重启生效。

### 四、环境变量的书写

- 变量名**大小写敏感**，必须使用全大写形式（写成 `gpt-4o` 不会生效）。
- 变量名含 `-`，bash 里不能直接写 `GPT-4O=60/3h ./auditlimit`，需要用 `env` 命令：

```bash
env "GPT-4O=60/3h" "GPT-5_6-SOL-WM=10/1h" ./auditlimit
```

- 同理，在 shell 里引用这类变量也不可靠：`$GPT-4O` 会被解析成 `${GPT}` 拼上 `-4O`，`${GPT-4O}` 则会被当成参数展开语法。建议只把它们写在 `env` 命令、`docker-compose.yml` 或 `.env` 文件里，不要在 shell 脚本里引用。
- `docker-compose.yml` / Kubernetes 的 `environment` 是键值映射，不经过 shell 解析，直接写 `GPT-4O: "60/3h"` 即可，不需要 `env`。

### 五、完整示例

```yaml
# config/config.yaml，放在工作目录（或二进制所在目录）下的 config/ 目录中
PORT: 9612
OAIKEY: ""
MODERATION: "https://api.openai.com/v1/moderations"
# 被注释掉的模型键表示该模型在上游 UI 中已不可选, 仅保留备用, 需要时去掉行首的 "# "
# 兜底值: 未单独配置的模型使用这一项
DEFAULT: "20/3h"
# Auto 档 (模型选择器中的「Auto」, 请求带 model=auto; 该模型名不在上游 models 清单中)
AUTO: "200/3h"
# Pro
GPT-5-5-PRO: "7/24h"
GPT-5-6-PRO: "7/24h"
GPT-6-PRO: "7/24h"
# Thinking
GPT-5-5-THINKING: "20/3h"
GPT-5-6-THINKING: "20/3h"
# GPT-5-4-T-MINI: "20/3h" # 上游 UI 已不可选
GPT-5-6-T-MINI: "20/3h"
# 标准
# GPT-5-5: "60/3h" # 上游 UI 已不可选
GPT-5-6: "60/3h"
GPT-5-5-INSTANT: "60/3h"
GPT-5-6-INSTANT: "60/3h"
# 工作模式 / Codex
GPT-5_5-WM: "60/3h"
GPT-5_6-SOL-WM: "60/3h"
GPT-5_6-TERRA-WM: "60/3h"
GPT-5_6-LUNA-WM: "60/3h"
GPT-6-ASTRA-WM: "60/3h"
# Codex 内部别名
GPT-5_5: "60/3h"
GPT-5_6-SOL: "60/3h"
GPT-5_6-TERRA: "60/3h"
GPT-5_6-LUNA: "60/3h"
GPT-6-ASTRA: "60/3h"
# Mini (上游 UI 均不可选, 保留备用)
# GPT-5-3-MINI: "200/3h"
# GPT-5-5-MINI: "200/3h"
# GPT-5-6-MINI: "200/3h"
# 研究 / 代理
RESEARCH: "2/24h"
AGENT: "2/24h"
# 禁用示例: 值写成 DISABLED 表示禁止用户使用该模型(返回 403)
# GPT-5_5-WM: "DISABLED"
```

等价的 docker-compose 写法（仅列出部分键）：

```yaml
environment:
  DEFAULT: "20/3h"
  GPT-5-6-PRO: "7/24h"
  GPT-5_6-SOL-WM: "60/3h"
  GPT-5-6-T-MINI: "20/3h"
  RESEARCH: "2/24h"
  # 禁用示例: 值写成 DISABLED 表示禁止用户使用该模型(返回 403)
  # GPT-5_5-WM: "DISABLED"
```

### 六、当前模型列表

下表是当前上游接口提供、且**用户在 UI 中能够选到**的模型，以及它们对应的配置键与默认限流值。配置键按上方「键名变换规则」生成；未列出的模型（含已不可选的）统一使用 `DEFAULT`。

| 模型名 | 配置键 | 默认限流 |
| --- | --- | --- |
| **Pro** | | |
| `gpt-5-5-pro` | `GPT-5-5-PRO` | `7/24h` |
| `gpt-5-6-pro` | `GPT-5-6-PRO` | `7/24h` |
| `gpt-6-pro` | `GPT-6-PRO` | `7/24h` |
| **Thinking** | | |
| `gpt-5-5-thinking` | `GPT-5-5-THINKING` | `20/3h` |
| `gpt-5-6-thinking` | `GPT-5-6-THINKING` | `20/3h` |
| `gpt-5-6-t-mini` | `GPT-5-6-T-MINI` | `20/3h` |
| **标准** | | |
| `gpt-5-6` | `GPT-5-6` | `60/3h` |
| `gpt-5-5-instant` | `GPT-5-5-INSTANT` | `60/3h` |
| `gpt-5-6-instant` | `GPT-5-6-INSTANT` | `60/3h` |
| **工作模式 / Codex** | | |
| `gpt-5.5-wm` | `GPT-5_5-WM` | `60/3h` |
| `gpt-5.6-sol-wm` | `GPT-5_6-SOL-WM` | `60/3h` |
| `gpt-5.6-terra-wm` | `GPT-5_6-TERRA-WM` | `60/3h` |
| `gpt-5.6-luna-wm` | `GPT-5_6-LUNA-WM` | `60/3h` |
| `gpt-6-astra-wm` | `GPT-6-ASTRA-WM` | `60/3h` |
| **Codex 内部别名** | | |
| `gpt-5.5` | `GPT-5_5` | `60/3h` |
| `gpt-5.6-sol` | `GPT-5_6-SOL` | `60/3h` |
| `gpt-5.6-terra` | `GPT-5_6-TERRA` | `60/3h` |
| `gpt-5.6-luna` | `GPT-5_6-LUNA` | `60/3h` |
| `gpt-6-astra` | `GPT-6-ASTRA` | `60/3h` |
| **研究 / 代理** | | |
| `research` | `RESEARCH` | `2/24h` |
| `agent`（由 `system_hints` 触发） | `AGENT` | `2/24h` |
| **Auto（自动选择）** | | |
| `auto`（上游模型清单中未列出） | `AUTO` | `200/3h` |
| **兜底** | | |
| 未列出的模型 | `DEFAULT` | `20/3h` |

说明：

- 上表只列出**当前版本中用户实际能在 UI 里选到的模型**。判定依据是上游 `chatmodel.json` 中的两个独立信号 —— `workspace_model_policy_catalog.models` 与 `versions[].slugs`（各版本并集），两者给出的集合完全一致。
- 上游模型清单会随版本变动，模型名可在 ChatGPT 的 `chatmodel.json` / Codex 的 `codexmodel.json` 中查看。
- `models[]` 中还存在若干个**已不可选**的 slug：`gpt-5-3-mini`、`gpt-5-4-t-mini`、`gpt-5-5`、`gpt-5-5-mini`、`gpt-5-6-mini`。它们不在上表中，在 `config/config.yaml` 与 `docker-compose.yml` 里以**注释**形式保留备用，需要时去掉行首的 `# ` 即可恢复。
- 「Codex 内部别名」指 `eligible_codex_model_slugs` 中不带 `-wm` 的名字，加上它们可以让这些名字也命中对应限流，而不是落到 `DEFAULT`。
- 注意 `GPT-5-5`（chat 的 `gpt-5-5`）与 `GPT-5_5`（Codex 别名 `gpt-5.5`）**只差一个字符但是两个不同的键**，前者已不可选、后者仍在使用，书写时请勿混淆。
- 多个模型共用同一个标题（如 `gpt-5-6-mini` 与 `gpt-5-6-t-mini` 都叫 GPT-5.6 Luna）是上游的命名，配置键仍以模型名（`slug`）为准。
- **`auto` 不出现在上面两个 `model` 清单的 `models[]` 里**，它对应的是模型选择器中的「Auto」档（在上游数据里体现为 `categories` 中 `category: gpt_5_auto`、`human_category_name: "Auto"`、`model_lane: "auto"` 的那一项）。它与其它模型键完全等价：**若删除 `AUTO`，`model=auto` 会回退到 `DEFAULT`**。

### 七、兼容性

- **含 `.` 的旧写法仍可识别。** 直接在 `config.yaml` 里写 `GPT-5.6-SOL-WM: "10/1h"`，或用 `env "GPT-5.6-SOL-WM=10/1h"` 设置环境变量，都能正常生效：程序先按 `_` 形式查找，查不到时再回退到原始形式。不过**推荐统一使用 `_` 形式**。
- 若规范化后的键与原始键同时存在，以规范化后的键（`_` 形式）为准。
- `config.yaml` 中的键名不做大小写兼容，请保持全大写。

## 超速返回格式

状态码: 429

```json
{
  "detail": {
    "clears_in": 252,
    "code": "model_cap_exceeded",
    "message": "You have triggered the usage frequency limit of gpt-5.6, the current limit is 60 times/3h0m0s, please wait 252 seconds before trying again.\n您已经触发 gpt-5.6 使用频率限制,当前限制为 60 次/3h0m0s,请等待 252 秒后再试."
  }
}
```

- `clears_in` 为预计需要等待的秒数；内部预留失败、无法给出具体时长时为 `0`。
- `message` 中带有实际命中的模型名、当前生效的限流值与等待秒数，便于排障。
- `detail` 是**对象而不是字符串**，客户端请读 `detail.code` / `detail.clears_in`，不要按字符串处理 `detail`。

## 禁用返回格式

模型的值被配置为 `DISABLED` 时（见「配置说明 - 值的格式 - 禁用某个模型」），状态码: 403

```json
{
  "detail": {
    "code": "model_disabled",
    "message": "The model gpt-5.6-sol-wm is disabled.\n模型 gpt-5.6-sol-wm 已被禁用,当前不可使用,请更换其他模型后重试."
  }
}
```

与 429（额度耗尽）区分开：本响应的 `code` 为 `model_disabled`，429 为 `model_cap_exceeded`，内容审核未通过为 `flagged_by_moderation`。客户端据此区分「提示用户更换模型」与「提示稍后重试」。

## 通用提示

状态码: 400

```json
{
  "detail": "别闹了"
}
```

命中违禁词（`data/keywords.txt`）时返回上述结构；请求体不是合法 JSON 时同样返回 400，此时 `detail` 为解析错误的原因。

## 正常返回

状态码: 200


## 内容审核

配置环境变量

OAIKEY: "sk-xxxxxx"  # api.openai.com可用的key或sess

将启用moderations接口内容审核