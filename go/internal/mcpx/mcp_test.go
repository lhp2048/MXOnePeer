package mcpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

func TestToolsAndKeyMask(t *testing.T) {
	dir := t.TempDir()
	board := adminboard.New(8788)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	rec := post(board, dir, body, "2025-03-26")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status"`) {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	for _, name := range []string{`"status"`, `"set_hostname"`, `"set_auth_key"`, `"start_login"`} {
		if !strings.Contains(rec.Body.String(), name) {
			t.Fatalf("missing %s", name)
		}
	}
	rec = post(board, dir, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"set_auth_key","arguments":{"auth_key":"tskey-auth-example"}}}`, "2025-03-26")
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "tskey-auth-example") || !strings.Contains(rec.Body.String(), "…mple") {
		t.Fatalf("call %s", rec.Body.String())
	}
	if store.ReadAuthKey(dir) != "tskey-auth-example" {
		t.Fatal("key was not saved")
	}
}

func post(board *adminboard.Board, dir, body, version string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", version)
	rec := httptest.NewRecorder()
	Handle(rec, req, board, dir)
	return rec
}
