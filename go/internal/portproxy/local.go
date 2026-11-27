package portproxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync"
	"time"
)

type Dial func(ctx context.Context, network, addr string) (net.Conn, error)

type Local struct {
	mu    sync.Mutex
	sites map[string]string
	dial  func() Dial
}

func NewLocal(dial func() Dial) *Local {
	return &Local{sites: map[string]string{}, dial: dial}
}

func (l *Local) Open(ip string, port int) (string, error) {
	key := ip + "\x00" + strconv.Itoa(port)
	l.mu.Lock()
	defer l.mu.Unlock()
	if addr, ok := l.sites[key]; ok {
		return addr, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	server := &http.Server{
		Handler:           forward(ip, port, l.dial),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.Serve(ln) }()
	addr := ln.Addr().String()
	l.sites[key] = addr
	return addr, nil
}

func forward(ip string, port int, dial func() Dial) http.Handler {
	target := net.JoinHostPort(ip, strconv.Itoa(ProxyPort))
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = target
			req.Header.Set(Header, strconv.Itoa(port))
		},
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := dial()
				if dialer == nil {
					return nil, ErrBadPort
				}
				return dialer(ctx, "tcp", target)
			},
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "打不开这个端口\n", http.StatusBadGateway)
		},
	}
}
