package clientapi

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckUpgrade(t *testing.T) {
	nodes := []Node{
		{TailscaleIPs: []string{"100.64.0.1"}, Self: true},
		{TailscaleIPs: []string{"100.64.0.2"}},
	}
	if err := CheckUpgrade("10.0.0.1", "go/linux/amd64", "mxonepeer-linux-amd64", nodes); err != ErrNodeAddress {
		t.Fatalf("address %v", err)
	}
	if err := CheckUpgrade("100.64.0.1", "go/linux/amd64", "mxonepeer-linux-amd64", nodes); err != ErrCannotUpgrade {
		t.Fatalf("self %v", err)
	}
	if err := CheckUpgrade("100.64.0.2", "client/darwin/arm64", "mxonepeer-darwin-arm64", nodes); err != ErrCannotUpgrade {
		t.Fatalf("client %v", err)
	}
	if err := CheckUpgrade("100.64.0.2", "go/linux/amd64", "mxonepeer-darwin-arm64", nodes); err != ErrBadFilename {
		t.Fatalf("name %v", err)
	}
	if err := CheckUpgrade("100.64.0.2", "go/linux/amd64", "mxonepeer-linux-amd64", nodes); err != nil {
		t.Fatalf("ok %v", err)
	}
}

func TestPostUpgrade(t *testing.T) {
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upgrade" || r.Header.Get("X-Mx-Runtime") != "go/linux/amd64" || r.Header.Get("X-Mx-Filename") != "mxonepeer-linux-amd64" {
			http.Error(w, "runtime 不一致\n", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "elf" {
			http.Error(w, "包的内容不正确\n", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer peer.Close()
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != "100.64.0.2:18020" {
			t.Errorf("addr %s", addr)
		}
		return net.Dial(network, peer.Listener.Addr().String())
	}
	code, body, err := PostUpgrade(context.Background(), dial, "100.64.0.2", "go/linux/amd64", "mxonepeer-linux-amd64", []byte("elf"))
	if err != nil || code != 204 || len(body) != 0 {
		t.Fatalf("post %d %q %v", code, body, err)
	}
	code, body, err = PostUpgrade(context.Background(), dial, "100.64.0.2", "go/darwin/arm64", "mxonepeer-darwin-arm64", []byte("elf"))
	if err != nil || code != 400 || !strings.Contains(string(body), "runtime 不一致") {
		t.Fatalf("reject %d %q %v", code, body, err)
	}
}
