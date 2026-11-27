package upgrade

import (
	"runtime"
	"testing"
)

func TestCurrent(t *testing.T) {
	got := Current("go")
	want := "go/" + runtime.GOOS + "/" + runtime.GOARCH
	if got != want {
		t.Fatalf("current %q", got)
	}
}

func TestFilenameOK(t *testing.T) {
	ok := []struct{ runtimeID, name string }{
		{"go/linux/amd64", "mxonepeer-linux-amd64"},
		{"go/darwin/arm64", "mxonepeer-darwin-arm64"},
		{"go/windows/amd64", "mxonepeer-windows-amd64.exe"},
		{"python/linux/amd64", "mxonepeer-python-linux-amd64.zip"},
		{"python/windows/amd64", "mxonepeer-python-windows-amd64.zip"},
	}
	for _, item := range ok {
		if !FilenameOK(item.runtimeID, item.name) {
			t.Fatalf("want ok %s %s", item.runtimeID, item.name)
		}
	}
	bad := []struct{ runtimeID, name string }{
		{"go/linux/amd64", "mxonepeer-linux-arm64"},
		{"go/linux/amd64", "mxonepeer-linux-amd64.exe"},
		{"go/windows/amd64", "mxonepeer-windows-amd64"},
		{"go/linux/unknown", "mxonepeer-linux-unknown"},
		{"python/linux/amd64", "mxonepeer-linux-amd64"},
		{"client/darwin/arm64", "mxonepeer-darwin-arm64"},
		{"go/linux/amd64", "dir/mxonepeer-linux-amd64"},
	}
	for _, item := range bad {
		if FilenameOK(item.runtimeID, item.name) {
			t.Fatalf("want reject %s %s", item.runtimeID, item.name)
		}
	}
}
