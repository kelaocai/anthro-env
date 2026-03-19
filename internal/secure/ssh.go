package secure

import "os"

// IsSSHSession 检测是否在 SSH 会话中
func IsSSHSession() bool {
	return os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		os.Getenv("SSH_TTY") != ""
}
