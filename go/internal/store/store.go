package store

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var label = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func Prepare(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.Chmod(dir, 0o750)
}

func NormalizeHostname(value string) (string, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	if !label.MatchString(text) {
		return "", errHostname
	}
	return text, nil
}

func SanitizeHostname(value, fallback string) string {
	var b strings.Builder
	hyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			hyphen = false
			continue
		}
		if !hyphen && b.Len() > 0 {
			b.WriteByte('-')
			hyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 63 {
		out = strings.TrimRight(out[:63], "-")
	}
	if _, err := NormalizeHostname(out); err != nil {
		return fallback
	}
	return out
}

func ReadHostname(dir string) string {
	text, err := os.ReadFile(filepath.Join(dir, "hostname"))
	if err != nil {
		return ""
	}
	name, err := NormalizeHostname(string(text))
	if err != nil {
		return ""
	}
	return name
}

func WriteHostname(dir, value string) (string, error) {
	name, err := NormalizeHostname(value)
	if err != nil {
		return "", err
	}
	if err := writePrivate(dir, "hostname", name+"\n"); err != nil {
		return "", err
	}
	return name, nil
}

func NormalizeAuthKey(value string) (string, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", errAuthKey
	}
	if !strings.HasPrefix(text, "tskey-auth-") {
		return "", errAuthKey
	}
	return text, nil
}

func ReadAuthKey(dir string) string {
	text, err := os.ReadFile(filepath.Join(dir, "authkey"))
	if err != nil {
		return ""
	}
	key, err := NormalizeAuthKey(string(text))
	if err != nil {
		return ""
	}
	return key
}

func WriteAuthKey(dir, value string) error {
	key, err := NormalizeAuthKey(value)
	if err != nil {
		return err
	}
	return writePrivate(dir, "authkey", key+"\n")
}

func MaskAuthKey(value string) string {
	text := strings.TrimSpace(value)
	if len(text) < 8 {
		return ""
	}
	return "…" + text[len(text)-4:]
}

func ReadEnvKey(dir string) string {
	text, err := os.ReadFile(filepath.Join(dir, "env"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(text), "\n") {
		if strings.HasPrefix(line, "TS_AUTHKEY=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "TS_AUTHKEY="))
		}
	}
	return ""
}

func writePrivate(dir, name, content string) error {
	if err := Prepare(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

type storeError string

func (e storeError) Error() string { return string(e) }

const (
	errHostname storeError = "invalid hostname"
	errAuthKey  storeError = "auth key must start with tskey-auth-"
)
