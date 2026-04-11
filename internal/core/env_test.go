package core

import (
	"sort"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	input := `
# comment
export ANTHROPIC_BASE_URL='https://example.com'
ANTHROPIC_MODEL="kimi-k2.5"
INVALID_LINE
`
	m := ParseEnv(input)
	if m["ANTHROPIC_BASE_URL"] != "https://example.com" {
		t.Fatalf("unexpected base url: %q", m["ANTHROPIC_BASE_URL"])
	}
	if m["ANTHROPIC_MODEL"] != "kimi-k2.5" {
		t.Fatalf("unexpected model: %q", m["ANTHROPIC_MODEL"])
	}
}

func TestValidProfileName(t *testing.T) {
	if !ValidProfileName("bailian-kimi-k2_5") {
		t.Fatal("expected valid profile name")
	}
	if ValidProfileName("bad name") {
		t.Fatal("expected invalid profile name")
	}
}

func TestTokenVarName(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{
			name: "minimax uses api key",
			vars: map[string]string{"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic"},
			want: "ANTHROPIC_API_KEY",
		},
		{
			name: "explicit api key stays api key",
			vars: map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
			want: "ANTHROPIC_API_KEY",
		},
		{
			name: "default uses auth token",
			vars: map[string]string{"ANTHROPIC_BASE_URL": "https://example.com"},
			want: "ANTHROPIC_AUTH_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokenVarName(tt.vars); got != tt.want {
				t.Fatalf("tokenVarName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildExportSnippetUsesMiniMaxAPIKey(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
		"ANTHROPIC_MODEL":    "MiniMax-M2.7",
	}

	got := buildExportSnippet(vars, "sk-test")

	if !containsLine(got, "export ANTHROPIC_API_KEY='sk-test'") {
		t.Fatalf("expected export snippet to set ANTHROPIC_API_KEY, got:\n%s", got)
	}
	if containsLine(got, "export ANTHROPIC_AUTH_TOKEN='sk-test'") {
		t.Fatalf("expected export snippet not to set ANTHROPIC_AUTH_TOKEN for MiniMax, got:\n%s", got)
	}
	if !containsLine(got, "unset ANTHROPIC_API_KEY") {
		t.Fatalf("expected export snippet to unset ANTHROPIC_API_KEY, got:\n%s", got)
	}
}

func TestStoredTokenValuePrefersAPIKey(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_API_KEY":    "sk-api",
		"ANTHROPIC_AUTH_TOKEN": "sk-auth",
	}

	if got := storedTokenValue(vars); got != "sk-api" {
		t.Fatalf("storedTokenValue() = %q, want %q", got, "sk-api")
	}
}

func TestSetStoredToken(t *testing.T) {
	vars := map[string]string{}

	setStoredToken(vars, "sk-test")
	if vars["ANTHROPIC_AUTH_TOKEN"] != "sk-test" {
		t.Fatalf("expected ANTHROPIC_AUTH_TOKEN, got %q", vars["ANTHROPIC_AUTH_TOKEN"])
	}

	vars = map[string]string{"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic"}
	setStoredToken(vars, "sk-minimax")
	if vars["ANTHROPIC_API_KEY"] != "sk-minimax" {
		t.Fatalf("expected ANTHROPIC_API_KEY for MiniMax, got %q", vars["ANTHROPIC_API_KEY"])
	}
	if _, ok := vars["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Fatal("ANTHROPIC_AUTH_TOKEN should not exist for MiniMax")
	}
}

func TestClearStoredToken(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_API_KEY":    "sk-old",
		"ANTHROPIC_AUTH_TOKEN": "sk-other",
	}
	clearStoredToken(vars)
	if _, ok := vars["ANTHROPIC_API_KEY"]; ok {
		t.Fatal("ANTHROPIC_API_KEY should be cleared")
	}
	if _, ok := vars["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Fatal("ANTHROPIC_AUTH_TOKEN should be cleared")
	}
}

func TestBuildExportSnippetPreservesTokenWhenKeychainEmpty(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
		"ANTHROPIC_API_KEY":  "sk-plaintext-minimax",
	}

	got := buildExportSnippet(vars, "")

	if !containsLine(got, "export ANTHROPIC_API_KEY='sk-plaintext-minimax'") {
		t.Fatalf("expected ANTHROPIC_API_KEY to be preserved when Keychain returns empty, got:\n%s", got)
	}
}

func TestBuildExportSnippetMigratesTokenOnProviderSwitch(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_BASE_URL":   "https://api.anthropic.com",
		"ANTHROPIC_AUTH_TOKEN": "sk-anthropic-token",
	}

	got := buildExportSnippet(vars, "")

	if containsLine(got, "export ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("expected ANTHROPIC_AUTH_TOKEN to not be exported (wrong provider), got:\n%s", got)
	}
	if !containsLine(got, "export ANTHROPIC_AUTH_TOKEN='sk-anthropic-token'") {
		t.Fatalf("expected ANTHROPIC_AUTH_TOKEN to be preserved when switching to non-MiniMax, got:\n%s", got)
	}
}

func TestMaybeMigrateTokenField(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		wantKey string
		wantVal string
	}{
		{
			name:    "MiniMax API key stays",
			vars:    map[string]string{"ANTHROPIC_BASE_URL": "https://api.minimaxi.com", "ANTHROPIC_API_KEY": "sk-minimax"},
			wantKey: "ANTHROPIC_API_KEY",
			wantVal: "sk-minimax",
		},
		{
			name:    "Anthropic auth token stays",
			vars:    map[string]string{"ANTHROPIC_BASE_URL": "https://api.anthropic.com", "ANTHROPIC_AUTH_TOKEN": "sk-anthropic"},
			wantKey: "ANTHROPIC_AUTH_TOKEN",
			wantVal: "sk-anthropic",
		},
		{
			name:    "MiniMax token migrates to API key",
			vars:    map[string]string{"ANTHROPIC_BASE_URL": "https://api.minimaxi.com", "ANTHROPIC_AUTH_TOKEN": "sk-should-migrate"},
			wantKey: "ANTHROPIC_API_KEY",
			wantVal: "sk-should-migrate",
		},
		{
			name:    "Anthropic token migrates to auth token",
			vars:    map[string]string{"ANTHROPIC_BASE_URL": "https://api.anthropic.com", "ANTHROPIC_API_KEY": "sk-should-migrate"},
			wantKey: "ANTHROPIC_AUTH_TOKEN",
			wantVal: "sk-should-migrate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maybeMigrateTokenField(tt.vars)
			if tt.wantKey == "" {
				if v := storedTokenValue(tt.vars); v != "" {
					t.Errorf("expected token to be cleared, got %q", v)
				}
				return
			}
			if tt.vars[tt.wantKey] != tt.wantVal {
				t.Errorf("vars[%q] = %q, want %q", tt.wantKey, tt.vars[tt.wantKey], tt.wantVal)
			}
		})
	}
}

func TestBuildExportSnippetExportKeysSorted(t *testing.T) {
	vars := map[string]string{
		"ZZZ_LAST":           "z",
		"ANTHROPIC_MODEL":    "m",
		"ANTHROPIC_BASE_URL": "https://example.com",
	}
	got := buildExportSnippet(vars, "")
	var exportKeys []string
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "export ") {
			rest := strings.TrimPrefix(line, "export ")
			if i := strings.Index(rest, "="); i > 0 {
				exportKeys = append(exportKeys, rest[:i])
			}
		}
	}
	if !sort.StringsAreSorted(exportKeys) {
		t.Fatalf("export lines not sorted by key: %v", exportKeys)
	}
}

func TestBuildExportSnippetClearsAuthTokenOnMiniMaxSwitch(t *testing.T) {
	vars := map[string]string{
		"ANTHROPIC_BASE_URL":   "https://api.minimaxi.com",
		"ANTHROPIC_AUTH_TOKEN": "sk-old-anthropic",
	}

	got := buildExportSnippet(vars, "sk-minimax")

	if containsLine(got, "export ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("expected ANTHROPIC_AUTH_TOKEN to be unset when switching providers, got:\n%s", got)
	}
	if !containsLine(got, "export ANTHROPIC_API_KEY='sk-minimax'") {
		t.Fatalf("expected ANTHROPIC_API_KEY for MiniMax, got:\n%s", got)
	}
}

func TestCorrectTokenFieldForBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://api.minimaxi.com/anthropic", "ANTHROPIC_API_KEY"},
		{"https://api.minimax.com", "ANTHROPIC_API_KEY"},
		{"https://api.anthropic.com", "ANTHROPIC_AUTH_TOKEN"},
		{"https://api.openai.com", "ANTHROPIC_AUTH_TOKEN"},
		{"", "ANTHROPIC_AUTH_TOKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.baseURL, func(t *testing.T) {
			if got := correctTokenFieldForBaseURL(tt.baseURL); got != tt.want {
				t.Errorf("correctTokenFieldForBaseURL(%q) = %q, want %q", tt.baseURL, got, tt.want)
			}
		})
	}
}

func TestLoginShellBase(t *testing.T) {
	if got := LoginShellBase(""); got != "" {
		t.Errorf("LoginShellBase(\"\") = %q, want \"\"", got)
	}
	if got := LoginShellBase("  /opt/homebrew/bin/fish  "); got != "fish" {
		t.Errorf("LoginShellBase(trim fish) = %q, want fish", got)
	}
	if got := LoginShellBase("/bin/zsh"); got != "zsh" {
		t.Errorf("LoginShellBase(/bin/zsh) = %q, want zsh", got)
	}
}

func TestSupportedHookShell(t *testing.T) {
	if !SupportedHookShell("zsh") || !SupportedHookShell("bash") {
		t.Fatal("zsh and bash should be supported")
	}
	if SupportedHookShell("fish") || SupportedHookShell("") || SupportedHookShell("sh") {
		t.Fatal("fish, empty, sh should not be supported")
	}
}

func TestDetectShell(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/bin/zsh", "zsh"},
		{"/bin/bash", "bash"},
		{"/usr/local/bin/fish", "zsh"},
		{"/bin/sh", "zsh"},
	}
	for _, tt := range tests {
		if got := DetectShell(tt.input); got != tt.want {
			t.Errorf("DetectShell(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRCFile(t *testing.T) {
	if got := RCFile("zsh"); got != "" && !strings.HasSuffix(got, ".zshrc") {
		t.Errorf("RCFile(zsh) = %q, want path ending with .zshrc", got)
	}
	if got := RCFile("bash"); got != "" && !strings.HasSuffix(got, ".bash_profile") && !strings.HasSuffix(got, ".bashrc") {
		t.Errorf("RCFile(bash) = %q, want path ending with .bash_profile or .bashrc", got)
	}
}

func TestHookScriptContainsEditTrigger(t *testing.T) {
	zshScript := HookScript("zsh")
	if !strings.Contains(zshScript, "edit:") {
		t.Fatal("zsh hook script should contain 'edit:' trigger")
	}

	bashScript := HookScript("bash")
	if !strings.Contains(bashScript, "edit:") {
		t.Fatal("bash hook script should contain 'edit:' trigger")
	}
}

func TestHookScriptContainsAnthroEnvCmd(t *testing.T) {
	script := HookScript("zsh")
	if !strings.Contains(script, "command anthro-env env") {
		t.Error("hook script should call anthro-env env")
	}
	if !strings.Contains(script, "alias anthro-env=") {
		t.Error("hook script should define anthro-env alias")
	}
	if !strings.Contains(script, "_anthro_env_sync") {
		t.Error("hook script should define _anthro_env_sync function")
	}
}

func TestOrDefault(t *testing.T) {
	if OrDefault("", "default") != "default" {
		t.Error("OrDefault with empty string should return default")
	}
	if OrDefault("value", "default") != "value" {
		t.Error("OrDefault with non-empty string should return original")
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{"'world'", "world"},
		{"noquotes", "noquotes"},
		{`"quoted with spaces"`, "quoted with spaces"},
	}
	for _, tt := range tests {
		if got := unquote(tt.input); got != tt.want {
			t.Errorf("unquote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func containsLine(s, line string) bool {
	for _, part := range strings.Split(s, "\n") {
		if part == line {
			return true
		}
	}
	return false
}
