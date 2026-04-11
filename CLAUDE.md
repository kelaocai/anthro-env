# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## 项目概述

`anthro-env` 是一个 **macOS CLI 工具**，用于管理 Claude Code / Anthropic 环境变量配置（profile），支持在多个 AI 服务提供商之间一键切换。

- **语言**：Go 1.22+
- **依赖**：无外部依赖，纯标准库
- **平台**：仅 macOS（使用 Keychain 存储敏感信息）

---

## 开发命令

### 构建
```bash
make build
# 或
go build -o bin/anthro-env ./cmd/anthro-env
```

### 测试
```bash
make test
# 或
go test ./...
```

### 格式化代码
```bash
make fmt
# 或
gofmt -w ./cmd ./internal
```

### 运行
```bash
./bin/anthro-env <command>
```

---

## 核心架构

### 目录结构
```
cmd/anthro-env/main.go    # CLI 入口，命令解析与分发
internal/
├── core/
│   ├── env.go           # .env 解析、profile 名称校验
│   ├── manager.go       # 核心业务逻辑（profile CRUD、doctor）
│   └── hook.go          # shell hook 安装
├── secure/
│   └── keychain.go      # macOS Keychain 封装
└── ui/
    └── menu.go          # 交互式菜单
```

### 数据存储
| 位置 | 内容 |
|------|------|
| `~/.config/anthropic/profiles/*.env` | 各 profile 的非敏感配置 |
| `~/.config/anthropic/current` | 当前激活的 profile 名称 |
| macOS Keychain | 各 profile 的 `ANTHROPIC_AUTH_TOKEN` |

### 管理的环境变量
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_AUTH_TOKEN`（存储在 Keychain）
- `API_TIMEOUT_MS`
- `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`
- `ANTHROPIC_MODEL`
- `ANTHROPIC_SMALL_FAST_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`

---

## 关键限制

### macOS Keychain SSH 限制
**重要**：Keychain 写入操作（`init`、`add`、`edit`）不能在 SSH 会话中执行，必须在本地终端运行。否则会报 `exit status 36` 错误。

这是 macOS Keychain 的安全机制，无法绕过。

---

## 发布流程

### 1. 更新版本号
在 `cmd/anthro-env/main.go` 中修改 `version` 常量。

### 2. 更新 CHANGELOG.md
记录版本变更。

### 3. 创建发布标签
```bash
./scripts/release.sh v0.x.x
# 或手动
git tag v0.x.x
git push origin v0.x.x
```

这将自动触发：
- GitHub Release 构建并发布 macOS arm64/amd64 二进制
- Homebrew Formula 自动更新（需要 `HOMEBREW_TAP_TOKEN`）

---

## 常见开发任务

### 添加新的 CLI 命令
1. 在 `cmd/anthro-env/main.go` 的 `commands` map 中注册
2. 在 `internal/core/manager.go` 中实现业务逻辑
3. 添加相应测试

### 添加新的环境变量
1. 在 `internal/core/env.go` 的 `ParseEnv()` 中确保支持
2. 在 `docs/` 中更新文档

---

## 文档
- README.md：英文文档
- README.zh.md：中文文档
- docs/ 目录：详细文档和故障排除

---

## 特别说明

- 不要擅自 `npm run dev` 启动服务器（本项目不使用 npm）
- 不自动创建分支，必须等用户确认
- 不自动提交 git，每次都需用户同意
- Git 主分支为 `main`
