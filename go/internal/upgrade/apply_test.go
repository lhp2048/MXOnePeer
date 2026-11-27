package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckBinary(t *testing.T) {
	elf := []byte{0x7f, 'E', 'L', 'F', 0}
	mach := []byte{0xfe, 0xed, 0xfa, 0xcf, 0}
	pe := []byte{'M', 'Z', 0}
	if CheckBinary("go/linux/amd64", elf) != nil {
		t.Fatal("elf")
	}
	if CheckBinary("go/darwin/arm64", mach) != nil {
		t.Fatal("macho")
	}
	if CheckBinary("go/windows/amd64", pe) != nil {
		t.Fatal("pe")
	}
	if err := CheckBinary("go/linux/amd64", pe); err == nil || err.Error() != "包的内容不正确" {
		t.Fatalf("mismatch %v", err)
	}
	if CheckBinary("go/linux/amd64", nil) == nil {
		t.Fatal("empty")
	}
}

func TestApplyFileKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "mxonepeer")
	if err := os.WriteFile(program, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ApplyFile(program, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(program)
	if err != nil || string(got) != "new" {
		t.Fatalf("current %q %v", got, err)
	}
	prev, err := os.ReadFile(program + ".previous")
	if err != nil || string(prev) != "old" {
		t.Fatalf("previous %q %v", prev, err)
	}
	info, err := os.Stat(program)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v", info.Mode())
	}
}

func TestApplyFileRollsBackWhenFinalRenameFails(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "mxonepeer")
	if err := os.WriteFile(program, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := 0
	original := renameFn
	renameFn = func(oldPath, newPath string) error {
		calls++
		if calls == 2 {
			return os.ErrPermission
		}
		return original(oldPath, newPath)
	}
	defer func() { renameFn = original }()
	if err := ApplyFile(program, []byte("new")); err == nil {
		t.Fatal("expected write failure")
	}
	got, err := os.ReadFile(program)
	if err != nil || string(got) != "old" {
		t.Fatalf("restored %q %v", got, err)
	}
}
