# 更新日志

本项目所有值得注意的变更都会记录在此文件。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号沿用发布脚本生成的时间戳（见 `release.sh`）。

## [Unreleased]

### 新增

- 新增 `config.GetStringWithEnv()` 辅助函数，统一按 `环境变量 > 配置文件 > 默认值` 的优先级读取配置。

### 变更

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
- 避免无配置文件时 panic。`g.Cfg().MustGet` 在找不到配置文件时（例如 Docker 镜像内仅有二进制与 `data/` 目录）会 panic，`GetStringWithEnv` 内部改用 `g.Cfg().Get` 并校验错误码。
- 修正配置优先级语义：`OAIKEY` 等值在 `config.yaml` 中为空字符串时，现在可被同名环境变量正常覆盖，与 README 中「用环境变量配置」的说明保持一致。

### 破坏性变更

- 需要 Go 1.26+ 才能构建。
- 依赖 gf 的配置读取行为变化，已通过新增的 `GetStringWithEnv` 适配；若自行改动过 `config.yaml` 的键名大小写，请保持大写形式。
