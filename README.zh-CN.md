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
> 的扩展 fork（已同步至 v1.10.x）—— 一个 DeepSeek 原生的 AI coding agent。
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
| **MCP bridge server** | 6 个工具（`reasonix_run`、`reasonix_doctor`、`plan_task`、`orchestrate_task`、`get_skill`、`get_skills`）—— 将 Reasonix 通过 MCP 暴露给 Claude Code、Codex 等 |
| **Hindsight 记忆** | 3 个工具（`hindsight_retain`、`hindsight_recall`、`hindsight_reflect`）—— 跨 session 持久记忆，SQLite + 文件后端，TTL/重要性衰减，TF-IDF 向量搜索 |
| **Skills hub** | 17 个社区策划 skill（调试、安全审计、代码审查、重构、前端开发、迁移、对抗性审查……）—— frontmatter playbook，支持 `runAs` 和 `allowedTools` |
| **Native hook runner** | 零依赖 Go 二进制，用于 PreToolUse/Stop 钩子 —— 替代 shell 脚本，向记忆服务 POST retain/reflect |
| **Portable mode** | `REASONIX_PORTABLE=1` 将所有数据重定向到 `<exe_dir>/.reasonix/` —— 从 U 盘或离线环境运行 |

<br/>

## 上游基础

- **配置驱动**：provider、agent、启用的工具、插件全部在 `reasonix.toml` 中声明，
  内核无硬编码模型。
- **多模型 · 可组合**：DeepSeek 作为预设内置；任何 OpenAI 兼容
  端点都只是一条配置。可选让两个模型协同（执行器 + 规划器），各自独立、缓存稳定的 session。
- **插件驱动**：外部工具以子进程形式运行，通过 stdio JSON-RPC 通信（MCP 兼容）；
  内置工具在编译期自注册。
- **零摩擦分发**：`CGO_ENABLED=0` 单二进制；一条命令交叉编译到六个目标平台。
  唯一依赖是一个 TOML 解析库。

详见 [指南](./docs/GUIDE.zh-CN.md)、[规格](./docs/SPEC.md) 和 [Hermes 指南](./docs/HERMES-GUIDE.md)。

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
go build -o bin/reasonix-bot        ./bot                       # Discord、Telegram、LINE、Slack bot
go build -o bin/reasonix-hooks      ./cmd/reasonix-hooks        # Hook runner
go build -o bin/reasonix-pr-review     ./cmd/reasonix-pr-review    # PR review CLI
go build -o bin/reasonix-e2ebench     ./cmd/e2ebench             # E2E benchmark tool
go build -o bin/reasonix-learner-live-test ./cmd/learner-live-test # Learner live e2e validation

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
reasonix setup                      # 配置向导 → ./reasonix.toml
export DEEPSEEK_API_KEY=sk-...      # 也可以让 setup 保存到 Reasonix 全局 .env
reasonix chat                       # 启动会话

# 一次性任务
reasonix run "给 auth 模块补单元测试"

# 启动 MCP bridge（向其他 agent 暴露 Reasonix）
reasonix-mcpbridge --http --port 9090

# 启动记忆服务
reasonix-memoryserver --backend sqlite --http --port 8080

# 运行 Discord/Telegram/LINE/Slack bot
export DISCORD_BOT_TOKEN="你的token"
reasonix-bot
reasonix                            # 然后在会话里运行 /init 生成 AGENTS.md（项目记忆）
reasonix run "把 main.go 里的 TODO 实现掉"
reasonix run --model deepseek-pro "给这个函数补单元测试"
echo "解释这段代码" | reasonix run
```

> 如果从源码构建，请将上方的 `reasonix` 替换为 `./bin/reasonix`。

<br/>
一个最小的 `reasonix.toml`——一个 provider 加一个默认模型——就够跑起来:

```toml
default_model = "deepseek-flash"

[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

优先级为 **flag > `./reasonix.toml` > 用户配置文件 > 内置默认值**；从
**Reasonix v1.8.1** 开始，用户配置位于 macOS/Linux 的 `~/.reasonix/config.toml`，
Windows 为 `%AppData%\reasonix\config.toml`。迁移细节见
**[配置路径](./docs/CONFIG_PATHS.zh-CN.md)**，其中也说明了全局 `config.toml`
和 `.env` 的完整结构。Provider 通过 `api_key_env` 命名密钥，真实密钥值保存在
CLI 与桌面端共用的 Reasonix 全局 `<Reasonix home>/.env`；项目 `.env` 不再作为
provider key 的运行时 fallback，但仍会作为当前 workspace 范围内的 MCP/plugin 非 provider `${VAR}` 展开来源，不导入 Reasonix 控制变量。权限、沙盒、插件(MCP)、
斜杠命令、`@` 引用与双模型设置,全部在 **[指南](./docs/GUIDE.zh-CN.md)** 里。

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
| **[Bot 指南](./docs/BOT_GUIDE.zh-CN.md)** | 连接 Discord、Telegram、LINE、Slack 等多平台机器人 |
| **[推理语言](./docs/REASONING_LANGUAGE.zh-CN.md)** | 强制模型以指定语言进行推理 |
| **[Goal 执行](./docs/GOAL_ENFORCEMENT.zh-CN.md)** | OMO 风格目标执行功能 |
| **[配置路径](./docs/CONFIG_PATHS.zh-CN.md)** | 配置文件与凭据的查找和迁移详情 |

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
