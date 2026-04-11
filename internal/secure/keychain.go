package secure

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const serviceName = "anthro-env"
const securityBinary = "/usr/bin/security"

func SaveToken(profile, token string) error {
	account := accountName(profile)
	cmd := exec.Command(securityBinary, "add-generic-password", "-U", "-a", account, "-s", serviceName, "-w", token)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("save token to Keychain failed: %v (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func ReadToken(profile string) (string, error) {
	account := accountName(profile)
	cmd := exec.Command(securityBinary, "find-generic-password", "-a", account, "-s", serviceName, "-w")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("read token from Keychain failed: %v (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func DeleteToken(profile string) error {
	account := accountName(profile)
	cmd := exec.Command(securityBinary, "delete-generic-password", "-a", account, "-s", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "could not be found") {
			return nil
		}
		return fmt.Errorf("delete token from Keychain failed: %v (%s)", err, msg)
	}
	return nil
}

func accountName(profile string) string {
	return "profile:" + profile + ":anthropic_auth_token"
}
