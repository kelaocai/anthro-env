package core

import (
	"regexp"
	"sort"
	"strings"

	"github.com/anthro-env/anthro-env/internal/secure"
)

var profileNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ParseEnv(content string) map[string]string {
	out := map[string]string{}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		out[k] = unquote(v)
	}
	return out
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func ValidProfileName(name string) bool {
	return profileNameRegex.MatchString(name)
}

func OrDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func MapKeysSorted(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// IsSSHSession 报告当前是否在 SSH 会话中
func IsSSHSession() bool {
	return secure.IsSSHSession()
}
