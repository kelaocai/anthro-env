# Changelog

## 0.1.7

- Deterministic key order in `anthro-env env` / `export` output
- Document `env`, `export`, and `hook` in `--help` and READMEs
- Interactive prompts propagate stdin read errors (no silent I/O failures)
- `doctor` warns when `$SHELL` is unset or not zsh/bash
- Keychain: invoke `/usr/bin/security` explicitly

## 0.1.6

- Color highlight current profile in `ls` and `menu` commands (respects NO_COLOR)
- SSH session graceful degradation: tokens stored in plaintext when Keychain unavailable
- Improved `doctor` output to distinguish SSH environments
- `migrate-tokens` returns error in SSH environments

## 0.1.5

- Auto-initialize on first run when no profiles exist
- Improved user experience for first-time users

## 0.1.0-alpha

- Initial CLI release.
- Added `init`, `menu`, `profile`, `hook`, and `doctor` commands.
- Added Keychain-backed token management.
