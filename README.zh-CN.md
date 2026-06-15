<p align="center">
  <img src="docs/logo-animated.svg" alt="Reasonix-Hermes" width="540"/>
</p>

<p align="center">
  <a href="./README.md">English</a>
  &nbsp;·&nbsp;
  <strong>简体中文</strong>
  &nbsp;·&nbsp;
  <a href="./docs/GUIDE.zh-CN.md">指南</a>
  &nbsp;·&nbsp;
  <a href="./docs/SPEC.md">规格</a>
  &nbsp;·&nbsp;
  <a href="./AGENTS.md">项目</a>
</p>

> **Reasonix Hermes** 是 [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)
> 的扩展 fork（已同步至 v1.8.0）—— 一个 DeepSeek 原生的 AI coding agent。
> 我们在上游的配置驱动、插件驱动 Go 核心之上，增加了 Discord bot、
> MCP bridge server、Hindsight 记忆服务、社区 skill 仓库、
> 原生 hook runner 和 portable mode —— 让 agent 生态具备连接、记忆和协作能力。

<br/>

<h3 align="center">面向终端的 DeepSeek 原生 AI coding agent —— 社区扩展版。</h3>
<p align="center">单一 Go 二进制。配置驱动。插件驱动。围绕 DeepSeek 前缀缓存调优，长会话也能压低 token 成本。</p>

<br/>

## Hermes 新增功能

Hermes 保留上游核心 —— agent 循环、provider、工具、权限、插件系统、桌面应用 ——
并在其上叠加跨 agent 连接和持久记忆：

| 新增功能 | 说明 |
|----------|------|
| **Discord bot** | `/goal <目标>` 自主循环、`/model flash\|pro\|mimo` 按会话切换模型、审批/拒绝/问答斜杠命令、多平台 gateway |
| **MCP bridge server** | 6 个工具（`reasonix_run`、`doctor`、`plan`、`orchestrate`、`get_skill`、`get_skills`）—— 将 Reasonix 通过 MCP 暴露给 Claude Code、Codex 等 |
| **Hindsight 记忆** | 3 个工具（`retain`、`recall`、`reflect`）—— 跨 session 持久记忆，SQLite + 文件后端，TTL/重要性衰减，TF-IDF 向量搜索 |
| **Skills hub** | 17 个社区策划 skill（调试、安全审计、代码审查、重构、前端开发、迁移、对抗性审查……）—— frontmatter playbook，支持 `runAs` 和 `allowedTools` |
| **Native hook runner** | 零依赖 Go 二进制，用于 PreToolUse/Stop 钩子 —— 替代 shell 脚本，向记忆服务 POST retain/reflect |
| **Portable mode** | `REASONIX_PORTABLE=1` 将所有数据重定向到 `<exe_dir>/.reasonix/` —— 从 U 盘或离线环境运行 |

<br/>

## 上游基础

Reasonix 本身是一个**配置与插件驱动**的 coding agent —— 单一静态 Go 二进制。
没有硬编码的模型。所有 provider、工具和插件都在 `reasonix.toml` 中声明。
内置工具编译期自注册；外部 MCP server 运行时通过 stdio 或 HTTP 接入。

- **多模型**：DeepSeek V4 Flash/Pro 和 MiMo v2.5 Pro 作为预设内置。任何
  OpenAI 兼容端点都只是一条配置。可选让 planner + executor 在两个独立、
  缓存稳定的 session 中协同工作。
- **权限控制**：每次工具调用进行 allow/ask/deny 判断 —— `Bash(go test:*)`、
  `Edit(docs/**)`、glob 匹配。在 chat、desktop 和 Discord 中均支持交互式审批。
- **桌面应用**：Wails v2 + React 19 + TypeScript 前端 —— 可切换主题的工作区、
  文件树、checkpoint/rewind、bot gateway。
- **零摩擦**：`CGO_ENABLED=0` 单二进制；一条命令交叉编译到六个目标平台。

详见 [指南](./docs/GUIDE.zh-CN.md)、[规格](./docs/SPEC.md) 和 [Hermes 指南](./docs/HERMES-GUIDE.md)。

<br/>

## 安装

### 一行命令（npm）

```sh
npm i -g reasonix-hermes
```

自动拉取对应平台的预编译二进制，安装后 `reasonix-hermes`（及 `reasonix`）可用。

### 从源码构建

```sh
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes

# 核心 CLI
go build -o bin/reasonix ./cmd/reasonix

# Hermes 服务
go build -o bin/reasonix-mcpbridge  ./cmd/reasonix-mcpbridge   # MCP bridge（6 个工具）
go build -o bin/reasonix-memoryserver     ./cmd/reasonix-memoryserver # Hindsight 记忆
go build -o bin/reasonix-bot        ./bot                       # Discord、Telegram、LINE bot
go build -o bin/reasonix-hooks      ./cmd/reasonix-hooks        # Hook runner
go build -o bin/reasonix-pr-review     ./cmd/reasonix-pr-review    # PR review CLI

# 桌面应用（Wails + React 19）
cd desktop && wails build -o ../bin/reasonix-desktop
```

### 安装 17 个社区 skill

```sh
./bin/reasonix install-source install \
  --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills
```

<br/>

## 快速开始

```sh
./bin/reasonix setup                      # 配置向导 → ./reasonix.toml
export DEEPSEEK_API_KEY=sk-...            # 或写入 .env
./bin/reasonix chat                       # 启动会话

# 一次性任务
./bin/reasonix run "给 auth 模块补单元测试"

# 启动 MCP bridge（向其他 agent 暴露 Reasonix）
./bin/reasonix-mcpbridge --http --port 9090

# 启动记忆服务
./bin/reasonix-memoryserver --backend sqlite --http --port 8080

# 运行 Discord/Telegram bot
export DISCORD_BOT_TOKEN="你的token"
./bin/reasonix-bot
```

<br/>

## 文档

| 文档 | 内容 |
|------|------|
| **[指南](./docs/GUIDE.zh-CN.md)** | 配置、权限与沙盒、插件(MCP)、斜杠命令、双模型协同 |
| **[规格](./docs/SPEC.md)** | 工程契约 —— 架构、registry、数据类型与设计原则 |
| **[项目](./AGENTS.md)** | Hermes fork 架构、命令、定制说明与贡献者指南 |
| **[从 0.x 迁移](./docs/MIGRATING.md)** | 从 legacy TypeScript 版本迁到 1.0 Go 重写版 |
| **[Checkpoints & rewind](./docs/CHECKPOINTS.md)** | 基于快照的编辑安全网（Esc-Esc / `/rewind`） |
| **[更新日志](./docs/CHANGELOG-HERMES.md)** | Hermes 分支里程碑、扩展包、机器人平台 |
| **[生态参考](./reasonix-deepseek-ecosystem-2026.md)** | 全景调研：MCP bridge、skills、desktop、IDE、fork、协议 |

<br/>

## 与上游的关系

Hermes 跟踪 [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)
（`main-v2` 分支）作为上游。我们将上游 release 合并到 `main` 分支，并在此基础上叠加 Hermes 的扩展功能。

```sh
git fetch upstream
git merge upstream/main-v2
```

自定义代码位于 `cmd/reasonix-*`、`internal/bot/`、`internal/learn/`、
`internal/mesh/`、`internal/collab/`、`internal/compress/`、
`internal/scheduler/`、`internal/publish/`、`pkg/`、`bot/`、`deploy/`、
`skills-hub/`，以及 `desktop/hermes_dashboard.go` 和 React hermes 组件。
除 `internal/bot/` 共用 gateway 以及 `internal/control/` 的 getter 外，
不修改上游引擎。

上游完整功能集（桌面应用、bot gateway（飞书/微信/QQ）、ACP session、PDF 提取、
可切换主题工作区）详见 [上游仓库](https://github.com/esengine/deepseek-reasonix)。

<br/>

## Star 趋势

<a href="https://www.star-history.com/?repos=esengine%2FDeepSeek-Reasonix&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=esengine/DeepSeek-Reasonix&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=esengine/DeepSeek-Reasonix&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=esengine/DeepSeek-Reasonix&type=date&legend=top-left" />
 </picture>
</a>

<br/>

## 致谢

Hermes 构建于 [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) 之上 ——
感谢[数十位贡献者](https://github.com/esengine/DeepSeek-Reasonix/graphs/contributors)的社区努力。
核心引擎、provider 抽象、工具系统、权限层、插件框架、桌面应用和原始构想的所有功劳归于上游团队。

<p align="center">
  <a href="https://github.com/esengine/DeepSeek-Reasonix/graphs/contributors">
    <img src="https://contrib.rocks/image?repo=esengine/DeepSeek-Reasonix&max=100&columns=12" alt="esengine/DeepSeek-Reasonix 贡献者" width="860"/>
  </a>
</p>

<br/>

---
<p align="center">
  <sub>MIT —— 见 <a href="./LICENSE">LICENSE</a></sub>
  <br/>
  <sub>Hermes 扩展由 <a href="https://github.com/aliatx2017">aliatx2017</a> 构建 · 上游：<a href="https://github.com/esengine/DeepSeek-Reasonix">esengine/DeepSeek-Reasonix</a></sub>
</p>
