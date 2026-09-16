# 更新日志

本项目所有值得注意的变更都会记录在此文件。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，标题记录修改日期。

## [2026-09-16]

### 新增

- 新增 `config.GetStringWithEnv()` 辅助函数，统一按 `配置文件 > 环境变量 > 默认值` 的优先级读取配置，与旧版 gf 的 `GetWithEnv` 行为保持一致。
- 新增 `config.NormalizeKey()` 与 `config.GetModelRate()`，确立模型名到配置键的转换约定：**转为大写，并把 `.` 替换为 `_`**。例如模型 `gpt-5.6-sol-wm` 对应的配置键为 `GPT-5_6-SOL-WM`，在 `config.yaml` 与环境变量中均按此书写。
- README 新增「配置说明」章节，完整整理了配置规则：键名变换规则、限流值的格式、`DEFAULT`/`PORT`/`OAIKEY`/`MODERATION` 等特殊键、环境变量在 shell 中的书写限制，以及 `config.yaml` 与环境变量的对照示例。
- README 新增「当前模型列表」小节，列出当前上游接口提供的模型、对应的配置键与默认限流值，并说明「仅保留 UI 可选模型」的筛选依据。
- 新增「模型禁用」约定：把模型键的值写成 `DISABLED`（不区分大小写）即可禁止用户使用该模型，例如 `GPT-5_5-WM: "DISABLED"`，环境变量模式写法相同。请求命中后立即返回 **403** 与 `{"detail":{"code":"model_disabled", ...}}`，便于与 429（额度耗尽）区分。
  - 禁用判断在内容审核与限流之前完成，所以被禁用的模型不消耗额度，也不会触发 moderation 调用。
  - 未单独配置的模型会落到 `DEFAULT`，将 `DEFAULT` 写成 `DISABLED` 即可禁用所有未单独配置的模型。
  - 新增 `config.DisabledValue`、`config.IsDisabled()`、`config.IsModelDisabled()` 与 `api.ErrModelDisabled`，并补充单元测试。
- README 新增「禁用返回格式」章节与「值的格式 - 禁用某个模型」小节；「值的格式」与「完整示例」中的示例模型更新为当前模型清单，并给出 `DISABLED` 的书写示例。

### 变更

- 按当前上游模型清单更新默认配置（`config/config.yaml`、`docker-compose.yml` 与 README 示例），**只保留用户在 UI 中实际能选到的模型**，按档次分组并加注注释：

  | 档次 | 模型数 | 默认限流 |
  | --- | --- | --- |
  | Pro | 3 | `7/24h` |
  | Thinking | 3 | `20/3h` |
  | 标准 | 3 | `60/3h` |
  | 工作模式 / Codex（`-wm`） | 5 | `60/3h` |
  | Codex 内部别名 | 5 | `60/3h` |

  其中含 `.` 的模型键（如 `GPT-5_6-SOL-WM`）按上一节的转换约定书写。
- 清理上游 `models[]` 中**已无法在 UI 中选到**的模型键（`gpt-5-3-mini`、`gpt-5-4-t-mini`、`gpt-5-5`、`gpt-5-5-mini`、`gpt-5-6-mini`）。这些键以**注释**形式保留在 `config/config.yaml`、`docker-compose.yml` 与 README 示例中，需要时去掉行首的 `# ` 即可恢复。
  - 判定依据是上游 `chatmodel.json` 中两个互相印证的信号：`workspace_model_policy_catalog.models` 与 `versions[].slugs`（各版本并集），两者给出的集合完全一致；不在该集合内的 `models[].slug` 即视为 UI 不可选。
  - 「Codex 内部别名」（`eligible_codex_model_slugs`）不受影响，继续保留。注意 `GPT-5-5`（chat 的 `gpt-5-5`，已不可选）与 `GPT-5_5`（Codex 别名 `gpt-5.5`，仍在使用）只差一个字符，是两个不同的键，删除前者不影响后者。
  - 对照例：`gpt-5-6-mini` 与 `gpt-5-6-t-mini` 的上游标题都叫「GPT-5.6 Luna」，但只有 `gpt-5-6-t-mini` 在可选集合中，因此仅后者保留为有效配置。
- 将请求体 `system_hints` 中 `research` / `agent` 覆盖模型名的逻辑提前到内容审核之前，使模型禁用判断、内容审核与限流都基于最终生效的模型名（对外行为无变化）。
- `DEFAULT`、`RESEARCH`、`AGENT`、`AUTO` 的取值保持不变；`docker-compose.yml` 补充了 `AGENT`（原先只有一个），与 `config.yaml` 对齐。
- 修正 README 中对 `AUTO` 的描述：`AUTO` 与 `RESEARCH`/`AGENT` 一样都是**普通模型键**（分别对应模型名 `auto`/`research`/`agent`），不是特殊键。`auto` 对应模型选择器中的「Auto」档，它不出现在上游 `models[].slug` 中（只以 `categories` 中 `category: gpt_5_auto`、`model_lane: "auto"` 的形式存在）。实测：删掉 `AUTO` 后 `model=auto` 会回退到 `DEFAULT`；把 `DEFAULT` 设为 `DISABLED` 时 `auto`/`research`/`agent` 会一并被禁。真正的特殊键只有 `DEFAULT`、`PORT`、`OAIKEY`、`MODERATION`。
- 升级依赖至最新版本：
  - `github.com/gogf/gf/v2` v2.5.7 → v2.10.3
  - `golang.org/x/time` v0.5.0 → v0.16.0
  - `go.opentelemetry.io/otel{,/sdk,/trace}` v1.14.0 → v1.46.0
  - `golang.org/x/net` v0.17.0 → v0.59.0、`golang.org/x/sys` v0.13.0 → v0.48.0、`golang.org/x/text` v0.13.0 → v0.42.0
  - `github.com/olekukonko/tablewriter` v0.0.5 → v1.1.5
  - 其余间接依赖（`fatih/color`、`fsnotify`、`go-logr/logr`、`mattn/go-isatty` 等）一并升级。
- `go.mod` 的 `go` 指令由 `1.18` 提升至 `1.26.0`。**构建本项目现在需要 Go 1.26 及以上版本**，该要求来自 `golang.org/x/{net,sys,text,time}` 的最新版。
- 开发容器镜像由 `mcr.microsoft.com/devcontainers/universal:5-linux` 更新为 `universal:6-linux`，并移除冗余的 `go` feature（镜像已内置新版 Go）。
- 补充 `.devcontainer/devcontainer-lock.json` 锁定 feature 版本。

### 修复

- **修复升级 gogf/gf 后 `config/config.yaml` 全部配置项被静默忽略的问题。**
  gf v2.10 起 `GetWithEnv` 会先用 `utils.FormatCmdKey()` 将键名转成小写再去查配置文件，而本项目 `config.yaml` 使用与 README 环境变量同名的大写键（如 `GPT-4O`、`TEXT-DAVINCI-002-RENDER-SHA`），键名被小写化后无法命中，导致配置被忽略并回退到硬编码兜底值。例如 `TEST: 1/1h` 会被读成 `40/3h`，`MODERATION` 也会退回硬编码地址。
  现已改为按原始键名显式查找（`config/config.go`、`api/limit.go` 中相关调用全部改用 `GetStringWithEnv`）。`GetEffective` / `MustGetEffective` 存在同样的小写化行为，无法用于规避此问题。
- 保持「配置文件优先」的原有优先级语义。`GetStringWithEnv` 严格复刻旧版 gf `GetWithEnv` 的行为：只要 `config.yaml` 中存在该键就直接采用（即使其值为空字符串），仅在该键不存在时才回退到同名环境变量。因此 `config.yaml` 中显式留空的 `OAIKEY: ""` 依然会屏蔽同名环境变量，与升级前一致。
- 避免无配置文件时 panic。`g.Cfg().MustGet` 在找不到配置文件时（例如 Docker 镜像内仅有二进制与 `data/` 目录）会 panic，`GetStringWithEnv` 内部改用 `g.Cfg().Get` 并校验错误码；此时环境变量仍可正常生效。
- **修复模型名中含 `.` 时无法配置限流的问题。** 形如 `gpt-5.6-sol-wm` 的模型名（来自上游接口返回）转换为大写后为 `GPT-5.6-SOL-WM`，而 gf 的配置查找把 `.` 当作路径分隔符，会把该键解析为 `GPT-5` 下的 `6-SOL-WM`，因此无论写在 `config.yaml` 还是环境变量里都读不到值，模型只能落到 `DEFAULT`。反斜杠转义（`GPT-5\.6-SOL-WM`）亦无效，且 `.` 本身不是合法的环境变量名字符，无法在 shell 中直接 `export`。
  现统一按 `NormalizeKey` 的约定在读取前把模型名转换为配置键（`.` → `_`），`config.yaml` 与环境变量两种来源均使用同一套键名；同时保留对原始写法的兼容：若配置中直接写了含 `.` 的键（如 `GPT-5.6-SOL-WM`），会通过配置数据的平铺查找命中，旧配置无需修改。

### 破坏性变更

- 需要 Go 1.26+ 才能构建。
- 依赖 gf 的配置读取行为变化，已通过新增的 `GetStringWithEnv` 适配；若自行改动过 `config.yaml` 的键名大小写，请保持大写形式。
- 配置优先级仍为 `配置文件 > 环境变量`，未发生变化：若 `config.yaml` 中已存在某个键，同名环境变量不会生效（`OAIKEY: ""` 会屏蔽 `OAIKEY` 环境变量）。仅当 `config.yaml` 中不存在该键（例如部署时未挂载配置文件）时，环境变量才会生效。
- 含 `.` 的模型名现在按 `_` 形式书写（如 `GPT-5_6-SOL-WM`）。原先用 `env "GPT-5.6-SOL-WM=..."` 等变通方式设置的环境变量仍可识别，无需改动；仅当环境变量键名与规范化后的键名（`GPT-5_6-SOL-WM`）同时存在时，以后者为准。
- 限流器内部以 `token|规范化模型键` 作为标识（如 `token|GPT-5_6-SOL-WM`），仅影响进程内存中的计数，重启后不残留状态。
- 默认配置中删除了已下线的旧模型键：`TEXT-DAVINCI-002-RENDER-SHA`、`GPT-4`、`GPT-4O`、`GPT-4O-MINI`、`GPT-4O-CANMORE`、`O1-PRO`、`O3-PRO`、`O3`、`O4-MINI`、`O4-MINI-HIGH`。
  升级后如果仍会请求这些旧模型名，它们会落到 `DEFAULT` 的额度；需要保留原有额度的话，请在环境变量中自行补回对应键。
  注意：`docker-compose.yml` 的 `environment` 会覆盖镜像内的默认值，直接拉取新镜像但沿用旧 `docker-compose.yml` 不受影响；反之若使用新版 `docker-compose.yml` 而未同步镜像，也不会报错，只是多出几个当前用不到的键。
