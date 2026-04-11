package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthro-env/anthro-env/internal/secure"
)

var anthroVars = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"API_TIMEOUT_MS",
	"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
}

type Manager struct {
	baseDir     string
	profilesDir string
	currentFile string
}

type DoctorReport struct {
	Status  string
	Message string
}

func tokenVarName(vars map[string]string) string {
	baseURL := strings.ToLower(strings.TrimSpace(vars["ANTHROPIC_BASE_URL"]))
	if strings.Contains(baseURL, "minimax") || strings.Contains(baseURL, "minimaxi") {
		return "ANTHROPIC_API_KEY"
	}
	if strings.TrimSpace(vars["ANTHROPIC_API_KEY"]) != "" {
		return "ANTHROPIC_API_KEY"
	}
	return "ANTHROPIC_AUTH_TOKEN"
}

func setStoredToken(vars map[string]string, token string) {
	delete(vars, "ANTHROPIC_API_KEY")
	delete(vars, "ANTHROPIC_AUTH_TOKEN")
	vars[tokenVarName(vars)] = token
}

func clearStoredToken(vars map[string]string) {
	delete(vars, "ANTHROPIC_API_KEY")
	delete(vars, "ANTHROPIC_AUTH_TOKEN")
}

func storedTokenValue(vars map[string]string) string {
	if v := strings.TrimSpace(vars["ANTHROPIC_API_KEY"]); v != "" {
		return v
	}
	return strings.TrimSpace(vars["ANTHROPIC_AUTH_TOKEN"])
}

func correctTokenFieldForBaseURL(baseURL string) string {
	baseURL = strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(baseURL, "minimax") || strings.Contains(baseURL, "minimaxi") {
		return "ANTHROPIC_API_KEY"
	}
	return "ANTHROPIC_AUTH_TOKEN"
}

func maybeMigrateTokenField(vars map[string]string) {
	token := storedTokenValue(vars)
	if token == "" {
		return
	}

	currentField := ""
	if strings.TrimSpace(vars["ANTHROPIC_API_KEY"]) != "" {
		currentField = "ANTHROPIC_API_KEY"
	} else if strings.TrimSpace(vars["ANTHROPIC_AUTH_TOKEN"]) != "" {
		currentField = "ANTHROPIC_AUTH_TOKEN"
	}

	correctField := correctTokenFieldForBaseURL(vars["ANTHROPIC_BASE_URL"])

	if currentField != "" && currentField != correctField {
		vars[correctField] = token
		delete(vars, currentField)
	}
}

func buildExportSnippet(vars map[string]string, token string) string {
	if strings.TrimSpace(token) != "" {
		setStoredToken(vars, token)
	} else {
		maybeMigrateTokenField(vars)
	}
	var b strings.Builder
	for _, k := range anthroVars {
		b.WriteString("unset ")
		b.WriteString(k)
		b.WriteString("\n")
	}
	for _, k := range MapKeysSorted(vars) {
		v := vars[k]
		if strings.TrimSpace(v) == "" {
			continue
		}
		b.WriteString("export ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(shellQuote(v))
		b.WriteString("\n")
	}
	return b.String()
}

func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".config", "anthropic")
	return &Manager{
		baseDir:     base,
		profilesDir: filepath.Join(base, "profiles"),
		currentFile: filepath.Join(base, "current"),
	}
}

func (m *Manager) ProfilesDir() string {
	return m.profilesDir
}

func (m *Manager) EnsureLayout() error {
	if err := os.MkdirAll(m.profilesDir, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(m.baseDir, 0o700)
	_ = os.Chmod(m.profilesDir, 0o700)
	return nil
}

func (m *Manager) SaveProfile(name string, vars map[string]string) error {
	return m.saveProfileWithHeader(name, "# Token is stored in macOS Keychain", vars)
}

func (m *Manager) saveProfileWithHeader(name, comment string, vars map[string]string) error {
	if err := m.EnsureLayout(); err != nil {
		return err
	}
	path := filepath.Join(m.profilesDir, name+".env")

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("# Managed by anthro-env\n")
	b.WriteString(comment + "\n")
	for _, k := range keys {
		v := strings.TrimSpace(vars[k])
		if v == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s=%s\n", k, shellQuote(v)))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

func (m *Manager) UseProfile(name string) error {
	if err := m.EnsureLayout(); err != nil {
		return err
	}
	path := filepath.Join(m.profilesDir, name+".env")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("profile not found: %s", name)
	}
	content := "# Last active profile for anthro-env auto-restore.\n" +
		"# Edit with: anthro-env profile use <name>\n" +
		"ACTIVE_PROFILE=" + name + "\n"
	if err := os.WriteFile(m.currentFile, []byte(content), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(m.currentFile, 0o600)
	return nil
}

func (m *Manager) RemoveProfile(name string) error {
	path := filepath.Join(m.profilesDir, name+".env")
	if err := os.Remove(path); err != nil {
		return err
	}
	_ = m.DeleteToken(name)
	active, _ := m.CurrentProfile()
	if active == name {
		_ = os.Remove(m.currentFile)
	}
	return nil
}

func (m *Manager) ListProfiles() ([]string, error) {
	if err := m.EnsureLayout(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.profilesDir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".env") {
			out = append(out, strings.TrimSuffix(name, ".env"))
		}
	}
	return out, nil
}

func (m *Manager) CurrentProfile() (string, error) {
	data, err := os.ReadFile(m.currentFile)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ACTIVE_PROFILE=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ACTIVE_PROFILE=")), nil
		}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, nil
	}
	return "", fmt.Errorf("active profile not set")
}

func (m *Manager) ProfileModel(name string) (string, error) {
	vars, err := m.ReadProfile(name)
	if err != nil {
		return "", err
	}
	if v := vars["ANTHROPIC_MODEL"]; strings.TrimSpace(v) != "" {
		return v, nil
	}
	if v := vars["ANTHROPIC_SMALL_FAST_MODEL"]; strings.TrimSpace(v) != "" {
		return v, nil
	}
	return "", nil
}

func (m *Manager) ReadProfile(name string) (map[string]string, error) {
	path := filepath.Join(m.profilesDir, name+".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseEnv(string(data)), nil
}

func (m *Manager) SaveToken(name, token string) error {
	if secure.IsSSHSession() {
		fmt.Fprintln(os.Stderr, "Warning: SSH session detected. Cannot write to macOS Keychain.")
		fmt.Fprintln(os.Stderr, "         Token will be stored in plaintext in the profile config file.")
		vars, err := m.ReadProfile(name)
		if err != nil {
			vars = map[string]string{}
		}
		setStoredToken(vars, token)
		return m.saveProfileWithHeader(name,
			"# SSH session: token stored in plaintext (Keychain unavailable)", vars)
	}
	return secure.SaveToken(name, token)
}

// SaveProfileAndToken atomically saves both profile vars and token.
// In SSH mode, token is embedded in vars and written once.
// In non-SSH mode, token goes to Keychain and vars to profile file.
func (m *Manager) SaveProfileAndToken(name string, vars map[string]string, token string) error {
	if secure.IsSSHSession() {
		fmt.Fprintln(os.Stderr, "Warning: SSH session detected. Cannot write to macOS Keychain.")
		fmt.Fprintln(os.Stderr, "         Token will be stored in plaintext in the profile config file.")
		if strings.TrimSpace(token) != "" {
			setStoredToken(vars, token)
		} else {
			clearStoredToken(vars)
		}
		return m.saveProfileWithHeader(name,
			"# SSH session: token stored in plaintext (Keychain unavailable)", vars)
	}
	if strings.TrimSpace(token) != "" {
		if err := secure.SaveToken(name, token); err != nil {
			return err
		}
	}
	return m.SaveProfile(name, vars)
}

func (m *Manager) DeleteToken(name string) error {
	if secure.IsSSHSession() {
		vars, err := m.ReadProfile(name)
		if err != nil {
			return nil
		}
		clearStoredToken(vars)
		return m.SaveProfile(name, vars)
	}
	return secure.DeleteToken(name)
}

func (m *Manager) MigratePlaintextTokens() (int, int, error) {
	if secure.IsSSHSession() {
		return 0, 0, fmt.Errorf(
			"SSH session: token migration to Keychain is not available in SSH environments")
	}
	profiles, err := m.ListProfiles()
	if err != nil {
		return 0, 0, err
	}

	migrated := 0
	skipped := 0
	for _, name := range profiles {
		vars, err := m.ReadProfile(name)
		if err != nil {
			return migrated, skipped, fmt.Errorf("read profile %s: %w", name, err)
		}

		token := storedTokenValue(vars)
		if token == "" {
			skipped++
			continue
		}

		if err := secure.SaveToken(name, token); err != nil {
			return migrated, skipped, fmt.Errorf("save keychain token for %s: %w", name, err)
		}

		clearStoredToken(vars)
		if err := m.SaveProfile(name, vars); err != nil {
			return migrated, skipped, fmt.Errorf("rewrite profile %s: %w", name, err)
		}
		migrated++
	}

	return migrated, skipped, nil
}

func (m *Manager) ExportSnippet() (string, error) {
	active, err := m.CurrentProfile()
	if err != nil {
		return "", err
	}
	vars, err := m.ReadProfile(active)
	if err != nil {
		return "", err
	}
	token, err := secure.ReadToken(active)
	if err == nil && token != "" {
		token = strings.TrimSpace(token)
	}
	// 若 Keychain 读取失败，保留 vars 中来自 .env 的明文 token（SSH 降级路径）
	return buildExportSnippet(vars, token), nil
}

func (m *Manager) Doctor() []DoctorReport {
	reports := make([]DoctorReport, 0)
	if err := m.EnsureLayout(); err != nil {
		reports = append(reports, DoctorReport{Status: "FAIL", Message: "Cannot create ~/.config/anthropic: " + err.Error()})
		return reports
	}
	reports = append(reports, DoctorReport{Status: "OK", Message: "Config directory exists: " + m.baseDir})
	if _, err := os.Stat(m.currentFile); err != nil {
		reports = append(reports, DoctorReport{Status: "WARN", Message: "No active profile set"})
	} else {
		reports = append(reports, DoctorReport{Status: "OK", Message: "Active profile file exists"})
	}
	profiles, err := m.ListProfiles()
	if err != nil || len(profiles) == 0 {
		reports = append(reports, DoctorReport{Status: "WARN", Message: "No profiles found"})
	} else {
		reports = append(reports, DoctorReport{Status: "OK", Message: fmt.Sprintf("Profiles found: %d", len(profiles))})
		plaintextCount := 0
		for _, p := range profiles {
			vars, err := m.ReadProfile(p)
			if err != nil {
				continue
			}
			if storedTokenValue(vars) != "" {
				// 读文件注释，若注释头含 "SSH session" 则不计入明文警告
				rawData, _ := os.ReadFile(filepath.Join(m.profilesDir, p+".env"))
				if strings.Contains(string(rawData), "SSH session") {
					continue
				}
				plaintextCount++
			}
		}
		if plaintextCount > 0 {
			reports = append(reports, DoctorReport{Status: "WARN", Message: fmt.Sprintf("Plaintext token found in %d profile(s); run: anthro-env migrate-tokens", plaintextCount)})
		} else {
			reports = append(reports, DoctorReport{Status: "OK", Message: "No plaintext token found in profile files"})
		}
	}
	shellPath := strings.TrimSpace(os.Getenv("SHELL"))
	shellBase := LoginShellBase(shellPath)
	if shellPath == "" {
		reports = append(reports, DoctorReport{
			Status:  "WARN",
			Message: "$SHELL is unset; anthro-env only supports zsh and bash for hook integration",
		})
	} else if !SupportedHookShell(shellBase) {
		reports = append(reports, DoctorReport{
			Status: "WARN",
			Message: fmt.Sprintf(
				`Login shell %q is not zsh/bash (unsupported). "anthro-env init" assumes zsh when $SHELL is unrecognized and appends to ~/.zshrc — use zsh or bash, or run: eval "$(anthro-env hook bash)"`,
				shellBase),
		})
	}

	shell := DetectShell(shellPath)
	rcFile := RCFile(shell)
	if rcFile != "" {
		if _, err := os.Stat(rcFile); err == nil {
			rcData, _ := os.ReadFile(rcFile)
			hookMarker := fmt.Sprintf("anthro-env hook %s", shell)
			if strings.Contains(string(rcData), hookMarker) {
				reports = append(reports, DoctorReport{Status: "OK", Message: fmt.Sprintf("%s hook installed", shell)})
			} else {
				reports = append(reports, DoctorReport{Status: "WARN", Message: fmt.Sprintf("%s hook not found; run anthro-env init", shell)})
			}
		}
	}
	active, err := m.CurrentProfile()
	if err == nil && active != "" {
		if secure.IsSSHSession() {
			reports = append(reports, DoctorReport{
				Status:  "INFO",
				Message: "SSH session: Keychain not available, token stored in profile file",
			})
		} else if _, err := secure.ReadToken(active); err != nil {
			reports = append(reports, DoctorReport{Status: "WARN", Message: "Keychain token missing or inaccessible for active profile"})
		} else {
			reports = append(reports, DoctorReport{Status: "OK", Message: "Keychain token is accessible"})
		}
	}
	return reports
}
