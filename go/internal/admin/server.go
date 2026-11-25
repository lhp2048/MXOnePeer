package admin

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/mcpx"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

func Handler(board *adminboard.Board, stateDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" {
			mcpx.Handle(w, r, board, stateDir)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, board.Page())
		case r.Method == http.MethodPost && (r.URL.Path == "/hostname" || r.URL.Path == "/authkey" || r.URL.Path == "/login"):
			post(w, r, board, stateDir)
		default:
			http.NotFound(w, r)
		}
	})
}

func Serve(host string, port int, board *adminboard.Board, stateDir string) (*http.Server, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, itoa(port)))
	if err != nil {
		return nil, err
	}
	server := &http.Server{
		Handler:           Handler(board, stateDir),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.Serve(ln) }()
	return server, nil
}

func post(w http.ResponseWriter, r *http.Request, board *adminboard.Board, stateDir string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	values, err := url.ParseQuery(string(body))
	if err != nil || values.Get("token") != board.Token() {
		http.Error(w, "forbidden\n", http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case "/hostname":
		previous := store.ReadHostname(stateDir)
		saved, err := store.WriteHostname(stateDir, values.Get("hostname"))
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `<!DOCTYPE html><html lang="zh-CN"><meta charset="utf-8"><title>MXOnePeer</title><p>节点名称要用小写字母、数字和连字符，长度 1 到 63，并且不能用连字符开头或结尾。</p>`)
			return
		}
		if saved == previous {
			board.SetNotice("节点名称没有变化。")
		} else {
			board.SetNotice("节点名称已保存，正在连接。")
			board.Poke("name")
		}
	case "/authkey":
		if err := store.WriteAuthKey(stateDir, values.Get("authkey")); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `<!DOCTYPE html><html lang="zh-CN"><meta charset="utf-8"><title>MXOnePeer</title><p>auth key 必须以 tskey-auth- 开头。</p>`)
			return
		}
		board.ClearLogin()
		board.SetNotice("auth key 已保存，正在连接。")
		board.Poke("key")
	case "/login":
		board.SetLoginError("")
		board.SetNotice("正在向 Tailscale 申请登录链接。")
		board.Poke("login")
	}
	board.Refresh(stateDir)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func Loopback(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [16]byte
	i := len(digits)
	n := value
	if n < 0 {
		n = -n
	}
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
