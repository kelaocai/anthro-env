# anthro-env

[English](README.md) | 中文

[![Release](https://img.shields.io/github/v/release/kelaocai/anthro-env)](https://github.com/kelaocai/anthro-env/releases)
[![Homebrew Tap](https://img.shields.io/badge/Homebrew-kelaocai%2Fhomebrew--tap-blue)](https://github.com/kelaocai/homebrew-tap)

`anthro-env` 是一个面向 macOS 的 Claude Code / Anthropic 环境变量配置切换 CLI（命令行工具）。
它可以让你在多个兼容 Anthropic 协议的网关与模型之间一键切换，并把 token 存在 macOS Keychain，避免泄露。

## 快速链接

- English README: [README.md](README.md)
- 中文 README: [README.zh.md](README.zh.md)
- 快速上手（3分钟）: [docs/快速上手.md](docs/快速上手.md)
- 完整中文手册: [docs/项目中文手册.md](docs/项目中文手册.md)

## Homebrew 安装（推荐）

```bash
brew tap kelaocai/homebrew-tap
brew install anthro-env
```

普通用户安装不需要本地 Go 环境。

## Profile 目录（加载机制）

`anthro-env` 会从这个目录加载 profile：

```text
~/.config/anthropic/profiles/*.env
```

规则：
- 目录下任意 `*.env` 都会被识别为一个 profile
- 文件名去掉 `.env` 就是 profile 名称
- `anthro-env ls` / `menu` / `use` 会自动发现这些文件

示例：
- `~/.config/anthropic/profiles/ai-router.env` -> profile 名：`ai-router`

## 重要提示

### SSH 环境限制

`anthro-env` 使用 macOS Keychain 存储敏感的 API Token。由于 Keychain 需要用户交互授权，**初始化操作（`init`、`add`、`edit`）必须在本地终端执行，不能在 SSH 会话中运行**。

如果在 SSH 会话中运行会报错：
```
Error: save token to Keychain failed: exit status 36
(security: SecKeychainItemCreateFromContent (<default>): User interaction is not allowed.)
```

**解决方案：**
1. 在本地 Mac 终端（非 SSH）执行 `anthro-env init` 或 `anthro-env add` 完成初始化
2. 初始化完成后，在 SSH 会话中可以正常使用 `anthro-env use`、`anthro-env menu` 等命令切换环境

## 开箱即用

```bash
anthro-env init
source ~/.zshrc
anthro-env menu
```

## 常用命令

```bash

  anthro-env init            # 初始化 anthro-env 环境（首次使用）
  anthro-env menu            # 打开交互式菜单（可视化选择操作）
  anthro-env add <name>      # 新增一个环境配置；<name> 为环境名称
  anthro-env edit <name>     # 编辑指定环境配置；<name> 为环境名称
  anthro-env use <name>      # 切换并启用指定环境；<name> 为环境名称
  anthro-env ls              # 列出所有已保存的环境
  anthro-env current         # 显示当前正在使用的环境
  anthro-env rm <name>       # 删除指定环境；<name> 为环境名称
  anthro-env hook <zsh|bash> # 输出 shell 集成片段（init 时也会写入 rc）
  anthro-env env             # 输出当前 profile 的可 eval 的 export（别名：export）
  anthro-env doctor          # 执行诊断检查（排查配置/依赖问题）
  anthro-env -v              # 显示版本号

  <name> 表示你自定义的环境名，例如：minimax、qwen、kimi。
```

如果你之前的 profile 文件里有明文 `ANTHROPIC_AUTH_TOKEN` 或 `ANTHROPIC_API_KEY`：

```bash
anthro-env migrate-tokens
```

### 修改已经设定过的配置：`edit`

```bash
anthro-env edit <name>
```

交互规则：
- `ANTHROPIC_BASE_URL`：直接回车 = 保留原值
- `ANTHROPIC_MODEL`：直接回车 = 保留；输入 `-` = 清空（走网关默认模型）
- `API credential`：直接回车 = 保留 Keychain 当前值；输入 `-` = 从 Keychain 删除；输入新值 = 覆盖
- `MiniMax` profile 会导出成 `ANTHROPIC_API_KEY`；其他 provider 默认导出成 `ANTHROPIC_AUTH_TOKEN`

示例：

```bash
anthro-env edit ai-router
anthro-env use ai-router
anthro-env doctor
```

### 把明文自动存入Keychain用法：`migrate-tokens`

```bash
anthro-env migrate-tokens
```

这个命令会：
- 读取 profile 文件中的明文 `ANTHROPIC_AUTH_TOKEN` 或 `ANTHROPIC_API_KEY`
- 写入对应 profile 的 macOS Keychain
- 从 profile 文件删除明文 token
- 输出迁移统计（`migrated` / `skipped`）

建议迁移后执行doctor 检查是否合规：

```bash
anthro-env doctor
```

## 配置示例

以下示例基于真实可用配置，API Key 已脱敏。  
注意：在 `anthro-env` 中，推荐把 token 存到 Keychain。

### 示例 1：bailian-kimi-k2.5

```bash
ANTHROPIC_AUTH_TOKEN=sk-********
ANTHROPIC_BASE_URL=https://coding.dashscope.aliyuncs.com/apps/anthropic
CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
ANTHROPIC_MODEL=kimi-k2.5
ANTHROPIC_SMALL_FAST_MODEL=kimi-k2.5
ANTHROPIC_DEFAULT_SONNET_MODEL=kimi-k2.5
ANTHROPIC_DEFAULT_OPUS_MODEL=kimi-k2.5
ANTHROPIC_DEFAULT_HAIKU_MODEL=kimi-k2.5
```

### 示例 2：MiniMax-M2.7

```bash
ANTHROPIC_API_KEY=sk-********
ANTHROPIC_BASE_URL=https://api.minimaxi.com/anthropic
CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
ANTHROPIC_MODEL=MiniMax-M2.7
ANTHROPIC_SMALL_FAST_MODEL=MiniMax-M2.7
ANTHROPIC_DEFAULT_SONNET_MODEL=MiniMax-M2.7
ANTHROPIC_DEFAULT_OPUS_MODEL=MiniMax-M2.7
ANTHROPIC_DEFAULT_HAIKU_MODEL=MiniMax-M2.7
```

MiniMax 当前的 Anthropic 兼容入口应使用 `ANTHROPIC_API_KEY` 和 `api.minimaxi.com`。

### 示例 3：ai-router（走网关默认模型路由）

```bash
ANTHROPIC_AUTH_TOKEN=sk-********
ANTHROPIC_BASE_URL=https://ai-router.plugins-world.cn
CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
# 这里故意不设置 ANTHROPIC_MODEL
# 具体模型由网关侧默认路由策略决定
```

## 安全与优先级规则

- Token 推荐存放在 macOS Keychain。
- profile 文件建议只放非敏感配置。
- Token 生效优先级：`Keychain > .env`。  
  如果两边都存在，以 Keychain 为准（通过edit修改）。

## 手动新增 profile 文件（自动发现）

`anthro-env` 会自动扫描 `~/.config/anthropic/profiles/*.env`。  
如果你手动放入新的 `xxx.env`，会自动出现在：

- `anthro-env ls`
- `anthro-env menu`
- `anthro-env use xxx`

如果该文件里有明文 `ANTHROPIC_AUTH_TOKEN` 或 `ANTHROPIC_API_KEY`，建议执行：

```bash
anthro-env migrate-tokens
```

## 源码编译

如果你希望本地编译或参与开发：

```bash
git clone https://github.com/kelaocai/anthro-env.git
cd anthro-env
go test ./...
go build -o ./bin/anthro-env ./cmd/anthro-env
./bin/anthro-env --help
```

## License

MIT

发布机制与流水线说明：见 [docs/release.md](docs/release.md)
