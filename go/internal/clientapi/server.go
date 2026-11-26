package clientapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/lhp2048/MXOnePeer/go/internal/statusdoc"
)

var ErrAuthKey = errors.New("auth key 必须以 tskey-auth- 开头。")

type Session struct {
	BackendState      string   `json:"backendState"`
	Online            bool     `json:"online"`
	Hostname          string   `json:"hostname"`
	RequestedHostname string   `json:"requestedHostname"`
	TailscaleIPs      []string `json:"tailscaleIPs"`
	LoginURL          string   `json:"loginUrl"`
	AuthKeyConfigured bool     `json:"authKeyConfigured"`
	AuthKeyMasked     string   `json:"authKeyMasked"`
	LoginError        string   `json:"loginError"`
}

type Snapshot struct {
	Session    Session
	Nodes      []Node
	SelfStatus statusdoc.Document
}

type Source interface {
	Snapshot() Snapshot
	SaveAuthKey(key string) error
	StartLogin()
	FetchStatus(ip string) (int, []byte)
}

func Handler(source Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": "mxonepeer-client", "version": "0.1.0"})
		case "/session":
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			snap := source.Snapshot()
			snap.Session.TailscaleIPs = NonNil(snap.Session.TailscaleIPs)
			writeJSON(w, http.StatusOK, snap.Session)
		case "/authkey":
			if r.Method != http.MethodPost {
				methodNotAllowed(w, http.MethodPost)
				return
			}
			body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
			var req struct {
				AuthKey string `json:"authKey"`
			}
			_ = json.Unmarshal(body, &req)
			if err := source.SaveAuthKey(req.AuthKey); err != nil {
				http.Error(w, "auth key 必须以 tskey-auth- 开头。\n", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "/login":
			if r.Method != http.MethodPost {
				methodNotAllowed(w, http.MethodPost)
				return
			}
			source.StartLogin()
			w.WriteHeader(http.StatusNoContent)
		case "/nodes":
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			snap := source.Snapshot()
			nodes := snap.Nodes
			if nodes == nil {
				nodes = []Node{}
			}
			writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
		case "/node-status":
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			serveNodeStatus(w, r, source)
		default:
			http.NotFound(w, r)
		}
	})
}

func serveNodeStatus(w http.ResponseWriter, r *http.Request, source Source) {
	ip := r.URL.Query().Get("ip")
	snap := source.Snapshot()
	if !Allowed(ip, snap.Nodes) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "地址不在节点列表里"})
		return
	}
	if IsSelf(ip, snap.Nodes) {
		writeJSON(w, http.StatusOK, snap.SelfStatus)
		return
	}
	code, body := source.FetchStatus(ip)
	if code != http.StatusOK || !jsonObject(body) {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "读不到状态"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func Allowed(ip string, nodes []Node) bool {
	return ip != "" && !strings.Contains(ip, " ") && containsIP(ip, nodes)
}

func IsSelf(ip string, nodes []Node) bool {
	for _, node := range nodes {
		if node.Self && containsOne(ip, node.TailscaleIPs) {
			return true
		}
	}
	return false
}

func containsIP(ip string, nodes []Node) bool {
	for _, node := range nodes {
		if containsOne(ip, node.TailscaleIPs) {
			return true
		}
	}
	return false
}

func containsOne(ip string, values []string) bool {
	for _, value := range values {
		if value == ip {
			return true
		}
	}
	return false
}

func jsonObject(body []byte) bool {
	var value map[string]any
	return json.Unmarshal(body, &value) == nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
}
