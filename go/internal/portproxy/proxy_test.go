package portproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestProxyForwardsPathAndHidesHeader(t *testing.T) {
	var gotHeader, gotHost, gotPath string
	var port int
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(Header)
		gotHost = r.Host
		gotPath = r.URL.Path
		http.Redirect(w, r, "http://127.0.0.1:"+strconv.Itoa(port)+"/next", http.StatusFound)
	}))
	defer backend.Close()
	port = backend.Listener.Addr().(*net.TCPAddr).Port

	proxy := httptest.NewServer(Handler())
	defer proxy.Close()
	req, err := http.NewRequest(http.MethodGet, proxy.URL+"/main", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(Header, strconv.Itoa(port))
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if gotHeader != "" || gotPath != "/main" || gotHost != "127.0.0.1:"+strconv.Itoa(port) {
		t.Fatalf("header %q path %q host %q", gotHeader, gotPath, gotHost)
	}
	if resp.Header.Get("Location") != "/next" {
		t.Fatalf("location %q", resp.Header.Get("Location"))
	}
}

func TestProxyRejectsAdminAndMissingPort(t *testing.T) {
	proxy := httptest.NewServer(Handler())
	defer proxy.Close()
	for _, header := range []string{"", "8788", "18021", "0"} {
		req, err := http.NewRequest(http.MethodGet, proxy.URL+"/", nil)
		if err != nil {
			t.Fatal(err)
		}
		if header != "" {
			req.Header.Set(Header, header)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("header %q status %d body %s", header, resp.StatusCode, body)
		}
	}
}

func TestPageURL(t *testing.T) {
	url, err := PageURL("127.0.0.1:9", "main")
	if err != nil || url != "http://127.0.0.1:9/main" {
		t.Fatalf("url %q err %v", url, err)
	}
	url, err = PageURL("127.0.0.1:9", "xxx.xx?q=1")
	if err != nil || url != "http://127.0.0.1:9/xxx.xx?q=1" {
		t.Fatalf("url %q err %v", url, err)
	}
	if _, err = PageURL("127.0.0.1:9", "../secret"); err != ErrBadPath {
		t.Fatal(err)
	}
}

func TestLocalOpenUsesDial(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "page")
	}))
	defer backend.Close()
	remote := httptest.NewServer(Handler())
	defer remote.Close()
	port := backend.Listener.Addr().(*net.TCPAddr).Port
	local := NewLocal(func() Dial {
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", remote.Listener.Addr().String())
		}
	})
	addr, err := local.Open("100.64.0.2", port)
	if err != nil {
		t.Fatal(err)
	}
	again, err := local.Open("100.64.0.2", port)
	if err != nil || again != addr {
		t.Fatalf("reused %q %q %v", addr, again, err)
	}
	page, err := PageURL(addr, "/main")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(page)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "page" {
		t.Fatalf("status %d body %q", resp.StatusCode, body)
	}
}
