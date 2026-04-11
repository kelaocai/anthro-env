# anthro-env

English | [中文](README.zh.md)

[![Release](https://img.shields.io/github/v/release/kelaocai/anthro-env)](https://github.com/kelaocai/anthro-env/releases)
[![Homebrew Tap](https://img.shields.io/badge/Homebrew-kelaocai%2Fhomebrew--tap-blue)](https://github.com/kelaocai/homebrew-tap)
![Homebrew](https://img.shields.io/badge/homebrew-install-blue)

A CLI-first macOS tool for switching Claude Code / Anthropic environment profiles.
A CLI tool for managing Claude-compatible API environments and switching between AI providers.
Compatible with Claude-compatible APIs.

## Why this exists

Claude Code is a great CLI coding assistant.

But in practice many developers need to switch between:

- Anthropic API
- proxy gateways
- third-party providers
- multiple API keys

This usually means repeatedly editing environment variables like:

- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_API_KEY`
- `ANTHROPIC_AUTH_TOKEN`

Manual shell edits quickly become messy.

`anthro-env` provides a simple profile-based workflow for managing these environments.


<img width="1536" height="1024" alt="banner (1)" src="https://github.com/user-attachments/assets/8cbdc798-96ac-4a27-95a2-13667d942bcf" />

### screenshots

<img width="419" height="219" alt="image" src="https://github.com/user-attachments/assets/ca136501-52aa-4717-868e-4088d3d4e8e2" />


## Install

### Homebrew (recommended)

```bash
brew tap kelaocai/homebrew-tap
brew install anthro-env
```

No local Go toolchain is required for Homebrew users.

### Build from source (optional)

```bash
git clone https://github.com/kelaocai/anthro-env.git
cd anthro-env
go test ./...
go build -o ./bin/anthro-env ./cmd/anthro-env
./bin/anthro-env --help
```

## Profile Directory (How loading works)

`anthro-env` loads profiles from:

```text
~/.config/anthropic/profiles/*.env
```

How it works:
- any `*.env` file in this directory is treated as a profile
- file name (without `.env`) is the profile name
- profiles are auto-discovered by `anthro-env ls` / `menu` / `use`

Example:
- `~/.config/anthropic/profiles/ai-router.env` -> profile name: `ai-router`

## Important Notes

### SSH Environment Limitation

`anthro-env` uses macOS Keychain to store sensitive API tokens. Since Keychain requires user interaction for authorization, **initialization operations (`init`, `add`, `edit`) must be run in a local terminal, not in SSH sessions**.

If you run these commands in an SSH session, you'll get an error:
```
Error: save token to Keychain failed: exit status 36
(security: SecKeychainItemCreateFromContent (<default>): User interaction is not allowed.)
```

**Solution:**
1. Run `anthro-env init` or `anthro-env add` in your local Mac terminal (not via SSH) to complete initialization
2. After initialization, you can use `anthro-env use`, `anthro-env menu`, and other commands in SSH sessions to switch environments

## Quick Start

```bash
anthro-env add kimi
anthro-env use kimi
anthro-env current
```

Or use the menu:

```bash
anthro-env menu
```

If you already have plaintext `ANTHROPIC_AUTH_TOKEN` or `ANTHROPIC_API_KEY` in old profile files:

```bash
anthro-env migrate-tokens
```

### Common commands

These are the most commonly used subcommands:

```bash
anthro-env init            # Initialize anthro-env (first-time setup)
anthro-env menu            # Open the interactive menu
anthro-env add <name>      # Add a new environment; <name> is the env name
anthro-env edit <name>     # Edit an existing environment; <name> is the env name
anthro-env use <name>      # Switch to and activate an environment; <name> is the env name
anthro-env ls              # List all saved environments
anthro-env current         # Show the currently active environment
anthro-env rm <name>       # Remove an environment; <name> is the env name
anthro-env hook <zsh|bash> # Print shell integration snippet (also installed by init)
anthro-env env             # Print eval-able exports for the active profile (alias: export)
anthro-env doctor          # Run diagnostics (check config/dependencies)
anthro-env -v              # Show version
```

### Detailed usage: `edit`

```bash
anthro-env edit <name>
```

During edit:
- `ANTHROPIC_BASE_URL`: press Enter to keep current value
- `ANTHROPIC_MODEL`: press Enter to keep, input `-` to clear (use gateway default)
- `API credential`: press Enter to keep Keychain value, input `-` to delete token from Keychain, input new value to overwrite
- `MiniMax` profiles are exported as `ANTHROPIC_API_KEY`; other providers default to `ANTHROPIC_AUTH_TOKEN`

Example:

```bash
anthro-env edit ai-router
anthro-env use ai-router
anthro-env doctor
```

### Detailed usage: `migrate-tokens`

```bash
anthro-env migrate-tokens
```

What it does:
- reads plaintext `ANTHROPIC_AUTH_TOKEN` or `ANTHROPIC_API_KEY` from profile files
- writes token into macOS Keychain for each profile
- removes plaintext token from profile files
- prints migration summary (`migrated` / `skipped`)

Recommended verification:

```bash
anthro-env doctor
```

## Profile Examples (redacted)

These examples are based on real-world provider setups, with API keys masked.
Keep in mind: in `anthro-env`, token is recommended to be stored in Keychain.

### Example 1: bailian-kimi-k2.5

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

### Example 2: MiniMax-M2.7

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

MiniMax's Anthropic-compatible endpoint currently expects `ANTHROPIC_API_KEY` and the `api.minimaxi.com` base URL.

### Example 3: ai-router (gateway default model routing)

```bash
ANTHROPIC_AUTH_TOKEN=sk-********
ANTHROPIC_BASE_URL=https://ai-router.plugins-world.cn
CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
# No ANTHROPIC_MODEL set here on purpose.
# Model selection is handled by gateway-side default routing rules.
```

## Concepts

- `profile`: a named environment config (base URL, model config, etc.)
- `use`: switch active profile
- `current`: show active profile
- `doctor`: quick health check
- `migrate-tokens`: move plaintext tokens from profile files into Keychain

What makes this project different:

- Keychain-backed token handling
- CLI-first workflow
- Homebrew distribution
- low-friction setup for macOS users

## Storage Layout

- Profiles: `~/.config/anthropic/profiles/*.env`
- Active profile pointer: `~/.config/anthropic/current`
- Token storage: macOS Keychain (`service=anthro-env`)

Profiles are auto-discovered from `~/.config/anthropic/profiles/*.env`.
If you manually add a new `xxx.env` file in that directory, it will be available in:
- `anthro-env ls`
- `anthro-env menu`
- `anthro-env use xxx`

If that file contains plaintext `ANTHROPIC_AUTH_TOKEN` or `ANTHROPIC_API_KEY`, run:

```bash
anthro-env migrate-tokens
```

## Security

- Tokens are stored in macOS Keychain.
- Profile files store metadata/config only.
- API keys are not intended to be stored in plain text profile files.
- Token precedence rule: `Keychain > .env`.
  If both exist, the Keychain token is used.

## Who It Is For

- Developers switching between multiple Claude Code / Anthropic-compatible providers
- Teams that want predictable profile switching on macOS
- Users who want Homebrew install and fast onboarding

## Roadmap

- Linux keyring support
- Windows credential manager support
- Profile export/import
- Team shared profile workflows

## Contributing

Issues and PRs are welcome.
Release mechanism details: [docs/release.md](docs/release.md)

## License

MIT
