package clientapi

import (
	"bytes"
	"mime/multipart"
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
			{Hostname: "down", TailscaleIPs: []string{"100.64.0.3"}, Online: false},
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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/node-status?ip=100.64.0.2", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "peer") {
		t.Fatalf("peer %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/node-status?ip=100.64.0.3", nil))
	if rec.Code != 502 || !strings.Contains(rec.Body.String(), "读不到状态") {
		t.Fatalf("peer fail %d %s", rec.Code, rec.Body.String())
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

type uploadFake struct {
	fakeSource
	called bool
}

func (u *uploadFake) Upload(ip, runtimeID, filename string, payload []byte) (int, []byte) {
	u.called = true
	if ip != "100.64.0.2" || runtimeID != "go/linux/amd64" || filename != "mxonepeer-linux-amd64" || string(payload) != "elf" {
		return 400, []byte(`{"error":"文件名不正确"}`)
	}
	return 204, nil
}

func TestUpgradeAPI(t *testing.T) {
	source := &uploadFake{}
	h := Handler(source)
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("ip", "100.64.0.2")
	_ = form.WriteField("runtime", "go/linux/amd64")
	part, _ := form.CreateFormFile("package", "mxonepeer-linux-amd64")
	_, _ = part.Write([]byte("elf"))
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, "/upgrade", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || !source.called {
		t.Fatalf("send %d %s", rec.Code, rec.Body.String())
	}

	source.called = false
	body.Reset()
	form = multipart.NewWriter(body)
	_ = form.WriteField("ip", "100.64.0.2")
	_ = form.WriteField("runtime", "go/linux/amd64")
	part, _ = form.CreateFormFile("package", "mxonepeer-darwin-arm64")
	_, _ = part.Write([]byte("elf"))
	_ = form.Close()
	req = httptest.NewRequest(http.MethodPost, "/upgrade", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 400 || source.called || !strings.Contains(rec.Body.String(), "文件名不正确") {
		t.Fatalf("name %d called %v %s", rec.Code, source.called, rec.Body.String())
	}
}
