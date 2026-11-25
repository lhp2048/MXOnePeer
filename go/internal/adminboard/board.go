package adminboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html"
	"sync"

	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

type Board struct {
	mu                sync.Mutex
	token             string
	loginURL          string
	loginError        string
	notice            string
	configured        bool
	masked            string
	online            bool
	hostname          string
	requestedHostname string
	ips               []string
	action            string
	generation        int
	adminPort         int
	wake              chan struct{}
}

func New(adminPort int) *Board {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return &Board{
		token:     hex.EncodeToString(buf),
		adminPort: adminPort,
		wake:      make(chan struct{}, 1),
	}
}

func (b *Board) Token() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.token
}

func (b *Board) Refresh(dir string) {
	key := store.ReadAuthKey(dir)
	name := store.ReadHostname(dir)
	b.mu.Lock()
	b.configured = key != ""
	b.masked = store.MaskAuthKey(key)
	b.requestedHostname = name
	b.mu.Unlock()
}

func (b *Board) Poke(action string) {
	b.mu.Lock()
	b.action = action
	b.generation++
	b.mu.Unlock()
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

func (b *Board) ClearAction() {
	b.mu.Lock()
	b.action = ""
	b.mu.Unlock()
}

func (b *Board) Snapshot() (action string, generation int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.action, b.generation
}

func (b *Board) Changed(generation int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.generation != generation
}

func (b *Board) Wake() <-chan struct{} { return b.wake }

func (b *Board) SetNotice(text string) {
	b.mu.Lock()
	b.notice = text
	b.mu.Unlock()
}

func (b *Board) SetLoginError(text string) {
	b.mu.Lock()
	b.loginError = text
	b.mu.Unlock()
}

func (b *Board) ClearLogin() {
	b.mu.Lock()
	b.loginError = ""
	b.loginURL = ""
	b.mu.Unlock()
}

func (b *Board) SetLoginURL(url string) {
	b.mu.Lock()
	if url != "" {
		b.loginURL = url
	}
	b.mu.Unlock()
}

func (b *Board) SetNode(hostname string, ips []string, online bool) {
	b.mu.Lock()
	b.hostname = hostname
	b.ips = append([]string(nil), ips...)
	b.online = online
	if online {
		b.loginURL = ""
		b.loginError = ""
	}
	b.mu.Unlock()
}

func (b *Board) Page() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	pill, state := "off", "未连接"
	if b.online {
		pill, state = "on", "已连接"
	}
	ips := "还没有"
	if len(b.ips) > 0 {
		ips = html.EscapeString(join(b.ips))
	}
	host := "还没有"
	if b.hostname != "" {
		host = html.EscapeString(b.hostname)
	}
	keyState := "未配置"
	if b.configured {
		keyState = "已配置 " + html.EscapeString(b.masked)
	}
	notice, loginError, login := "", "", `<p class="hint">批准之后，这台机器会出现在你的 Tailscale 网络里。</p>`
	if b.notice != "" {
		notice = `<p class="banner">` + html.EscapeString(b.notice) + `</p>`
	}
	if b.loginError != "" {
		loginError = `<p class="banner bad">` + html.EscapeString(b.loginError) + `</p>`
	}
	if b.loginURL != "" {
		url := html.EscapeString(b.loginURL)
		login = `<a class="link" href="` + url + `">打开 Tailscale 登录链接</a><p class="url">` + url + `</p>`
	}
	mcp := html.EscapeString("http://127.0.0.1:" + itoa(b.adminPort) + "/mcp")
	token := html.EscapeString(b.token)
	requested := html.EscapeString(b.requestedHostname)
	return pageHTML(pill, state, host, ips, keyState, notice, loginError, token, requested, login, mcp)
}

func (b *Board) StatusJSON() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ips := b.ips
	if ips == nil {
		ips = []string{}
	}
	body, _ := json.Marshal(map[string]any{
		"online":            b.online,
		"hostname":          b.hostname,
		"requestedHostname": b.requestedHostname,
		"tailscaleIPs":      ips,
		"authKeyConfigured": b.configured,
		"authKeyMasked":     b.masked,
		"loginUrl":          b.loginURL,
		"loginError":        b.loginError,
		"notice":            b.notice,
		"mcpUrl":            "http://127.0.0.1:" + itoa(b.adminPort) + "/mcp",
	})
	return string(body)
}

func join(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [16]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
