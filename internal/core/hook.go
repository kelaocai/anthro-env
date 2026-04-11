package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	hookStart = "# >>> anthro-env >>>"
	hookEnd   = "# <<< anthro-env <<<"
)

// LoginShellBase returns the basename of a login shell path (e.g. /bin/zsh -> zsh).
// Empty path yields "".
func LoginShellBase(shellPath string) string {
	s := strings.TrimSpace(shellPath)
	if s == "" {
		return ""
	}
	return filepath.Base(s)
}

// SupportedHookShell reports whether anthro-env provides a hook for this shell name.
func SupportedHookShell(base string) bool {
	switch base {
	case "zsh", "bash":
		return true
	default:
		return false
	}
}

func DetectShell(shell string) string {
	base := LoginShellBase(shell)
	switch base {
	case "zsh", "bash":
		return base
	default:
		return "zsh"
	}
}

func RCFile(shell string) string {
	home, _ := os.UserHomeDir()
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile
		}
		return filepath.Join(home, ".bashrc")
	default:
		return ""
	}
}

func InstallHook(rcFile, shell string) error {
	if rcFile == "" {
		return fmt.Errorf("rc file is empty")
	}
	data, _ := os.ReadFile(rcFile)
	if strings.Contains(string(data), hookStart) {
		return nil
	}
	block := hookStart + "\n" +
		"if command -v anthro-env >/dev/null 2>&1; then\n" +
		"  eval \"$(anthro-env hook " + shell + ")\"\n" +
		"fi\n" +
		hookEnd + "\n"
	f, err := os.OpenFile(rcFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + block)
	return err
}

func HookScript(shell string) string {
	switch shell {
	case "bash":
		return bashHookScript()
	default:
		return zshHookScript()
	}
}

func zshHookScript() string {
	return `
_anthro_env_sync() {
  eval "$(command anthro-env env 2>/dev/null || true)"
}

anthro_env_cmd() {
  command anthro-env "$@"
  local rc=$?
  if [ $rc -eq 0 ]; then
    case "$1:$2" in
      ":"|"menu:"|"init:"|"add:"|"use:"|"rm:"|"edit:"|"profile:use"|"profile:rm") _anthro_env_sync ;;
    esac
  fi
  return $rc
}

alias anthro-env='anthro_env_cmd'

_anthro_env_sync
`
}

func bashHookScript() string {
	return `
_anthro_env_sync() {
  eval "$(command anthro-env env 2>/dev/null || true)"
}

anthro_env_cmd() {
  command anthro-env "$@"
  local rc=$?
  if [ $rc -eq 0 ]; then
    case "$1:$2" in
      ":"|"menu:"|"init:"|"add:"|"use:"|"rm:"|"edit:"|"profile:use"|"profile:rm") _anthro_env_sync ;;
    esac
  fi
  return $rc
}

alias anthro-env='anthro_env_cmd'

_anthro_env_sync
`
}
