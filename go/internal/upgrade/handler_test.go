package upgrade

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandlerRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "mxonepeer")
	if err := os.WriteFile(program, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	exited := 0
	h := NewHandler("go/linux/amd64", program, func(code int) { exited = code })
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader("MZ"))
	req.Header.Set("X-Mx-Runtime", "go/darwin/arm64")
	req.Header.Set("X-Mx-Filename", "mxonepeer-darwin-arm64")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "runtime 不一致") || exited != 0 {
		t.Fatalf("runtime %d %s %d", rec.Code, rec.Body.String(), exited)
	}
	got, _ := os.ReadFile(program)
	if string(got) != "old" {
		t.Fatalf("changed %q", got)
	}
}

func TestHandlerReplacesAndExits(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "mxonepeer")
	if err := os.WriteFile(program, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	exited := 0
	h := NewHandler("go/linux/amd64", program, func(code int) { exited = code })
	body := []byte{0x7f, 'E', 'L', 'F', 'n', 'e', 'w'}
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(string(body)))
	req.Header.Set("X-Mx-Runtime", "go/linux/amd64")
	req.Header.Set("X-Mx-Filename", "mxonepeer-linux-amd64")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || exited != 75 {
		t.Fatalf("accept %d %d", rec.Code, exited)
	}
	got, _ := os.ReadFile(program)
	if string(got) != string(body) {
		t.Fatalf("new %q", got)
	}
	prev, _ := os.ReadFile(program + ".previous")
	if string(prev) != "old" {
		t.Fatalf("prev %q", prev)
	}
}

func TestRoutesLeaveStatusInPlace(t *testing.T) {
	status := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"hostname":"n"}`))
	})
	h := Routes(status, "go/linux/amd64", filepath.Join(t.TempDir(), "mxonepeer"), func(int) {})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "hostname") {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}
