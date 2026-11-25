package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

func TestLoopback(t *testing.T) {
	if !Loopback("127.0.0.1") || !Loopback("::1") || Loopback("0.0.0.0") || Loopback("localhost") {
		t.Fatal("loopback check")
	}
}

func TestPageAndHostname(t *testing.T) {
	dir := t.TempDir()
	board := adminboard.New(8788)
	board.Refresh(dir)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Handler(board, dir).ServeHTTP(rec, req)
	page := rec.Body.String()
	for _, text := range []string{"保存节点名称", "保存 auth key", "用 Tailscale 账号登录", "未配置", "本地 MCP", "http://127.0.0.1:8788/mcp"} {
		if !strings.Contains(page, text) {
			t.Fatalf("page missing %s", text)
		}
	}
	form := url.Values{"token": {board.Token()}, "hostname": {"Desk-1"}}
	req = httptest.NewRequest(http.MethodPost, "/hostname", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	Handler(board, dir).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || store.ReadHostname(dir) != "desk-1" {
		t.Fatalf("save %d %q", rec.Code, store.ReadHostname(dir))
	}
	form.Set("hostname", "-no")
	req = httptest.NewRequest(http.MethodPost, "/hostname", strings.NewReader(form.Encode()))
	rec = httptest.NewRecorder()
	Handler(board, dir).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || store.ReadHostname(dir) != "desk-1" {
		t.Fatalf("reject %d %q", rec.Code, store.ReadHostname(dir))
	}
}
