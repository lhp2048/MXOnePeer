package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostname(t *testing.T) {
	dir := t.TempDir()
	saved, err := WriteHostname(dir, " My-Node ")
	if err != nil {
		t.Fatal(err)
	}
	if saved != "my-node" || ReadHostname(dir) != "my-node" {
		t.Fatalf("saved %q", saved)
	}
	info, err := os.Stat(filepath.Join(dir, "hostname"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if _, err := WriteHostname(dir, "-mxonepeer"); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestAuthKey(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAuthKey(dir, "tskey-auth-example"); err != nil {
		t.Fatal(err)
	}
	if got := MaskAuthKey(ReadAuthKey(dir)); got != "…mple" {
		t.Fatalf("mask %q", got)
	}
	if err := WriteAuthKey(dir, "tskey-api-nope"); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestSanitizeHostname(t *testing.T) {
	if got := SanitizeHostname("  MacBook Pro ", "mxonepeer-mac"); got != "macbook-pro" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeHostname("Muxin's Mac", "mxonepeer-mac"); got != "muxin-s-mac" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeHostname("---", "mxonepeer-mac"); got != "mxonepeer-mac" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 70)
	if got := SanitizeHostname(long, "mxonepeer-mac"); got != strings.Repeat("a", 63) {
		t.Fatalf("len %d", len(got))
	}
}
