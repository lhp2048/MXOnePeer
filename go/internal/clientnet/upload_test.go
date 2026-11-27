package clientnet

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lhp2048/MXOnePeer/go/internal/clientapi"
)

func TestUploadWithoutDial(t *testing.T) {
	live := &Live{}
	live.snap.Nodes = []clientapi.Node{{TailscaleIPs: []string{"100.64.0.2"}}}
	code, body := live.Upload("100.64.0.2", "go/linux/amd64", "mxonepeer-linux-amd64", []byte("elf"))
	if code != 502 || !strings.Contains(string(body), "传不上去") {
		t.Fatalf("offline %d %s", code, body)
	}
}

func TestUploadDialsPeer(t *testing.T) {
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer peer.Close()
	live := &Live{}
	live.snap.Nodes = []clientapi.Node{{TailscaleIPs: []string{"100.64.0.2"}}}
	live.setDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.Dial(network, peer.Listener.Addr().String())
	})
	code, body := live.Upload("100.64.0.2", "go/linux/amd64", "mxonepeer-linux-amd64", []byte("elf"))
	if code != 204 || len(body) != 0 {
		t.Fatalf("dial %d %s", code, body)
	}
}
