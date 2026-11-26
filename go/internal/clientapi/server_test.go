package clientapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lhp2048/MXOnePeer/go/internal/statusdoc"
)

type fakeSource struct {
	saved string
	login bool
}

func (f *fakeSource) Snapshot() Snapshot {
	return Snapshot{
		Session: Session{
			BackendState:      "Running",
			Online:            true,
			Hostname:          "mac",
			RequestedHostname: "mac",
			TailscaleIPs:      []string{"100.64.0.1"},
			AuthKeyConfigured: true,
			AuthKeyMasked:     "…mple",
		},
		Nodes: []Node{
			{Hostname: "mac", TailscaleIPs: []string{"100.64.0.1"}, Online: true, Self: true},
			{Hostname: "peer", TailscaleIPs: []string{"100.64.0.2"}, Online: false},
		},
		SelfStatus: statusdoc.Document{
			Hostname: "mac", DNSName: "mac.tailnet.ts.net", TailscaleIPs: []string{"100.64.0.1"},
			Online: true, BackendState: "Running",
		},
	}
}

func (f *fakeSource) SaveAuthKey(key string) error {
	if !strings.HasPrefix(strings.TrimSpace(key), "tskey-auth-") {
		return ErrAuthKey
	}
	f.saved = key
	return nil
}

func (f *fakeSource) StartLogin() { f.login = true }

func (f *fakeSource) FetchStatus(ip string) (int, []byte) {
	if ip == "100.64.0.2" {
		return 200, []byte(`{"hostname":"peer","dnsName":"peer.tailnet.ts.net","tailscaleIPs":["100.64.0.2"],"online":false,"backendState":"Offline"}`)
	}
	return 502, []byte(`{"error":"读不到状态"}`)
}

func TestClientAPI(t *testing.T) {
	source := &fakeSource{}
	h := Handler(source)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"mxonepeer-client"`) {
		t.Fatalf("health %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/authkey", strings.NewReader(`{"authKey":"tskey-api-nope"}`)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "tskey-auth-") {
		t.Fatalf("bad key %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/authkey", strings.NewReader(`{"authKey":"tskey-auth-example"}`)))
	if rec.Code != 204 || source.saved != "tskey-auth-example" || strings.Contains(rec.Body.String(), "tskey-auth-example") {
		t.Fatalf("save %d %q", rec.Code, source.saved)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/login", nil))
	if rec.Code != 204 || !source.login {
		t.Fatal("login")
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nodes", nil))
	if !strings.Contains(rec.Body.String(), `"self":true`) || strings.Index(rec.Body.String(), "mac") > strings.Index(rec.Body.String(), "peer") {
		t.Fatalf("nodes %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/node-status?ip=example.com", nil))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "地址不在节点列表里") {
		t.Fatalf("reject %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/node-status?ip=100.64.0.1", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"hostname":"mac"`) {
		t.Fatalf("self %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("404 %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rec.Code != 405 {
		t.Fatalf("405 %d", rec.Code)
	}
}
