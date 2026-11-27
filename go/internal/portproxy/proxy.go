package portproxy

import (
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

const (
	Header    = "X-Mx-Target-Port"
	ProxyPort = 18021
	adminPort = 8788
)

var (
	ErrBadPort     = errors.New("端口不正确")
	ErrBlockedPort = errors.New("这个端口不能打开")
	ErrBadPath     = errors.New("路径不正确")
)

func CheckPort(port int) error {
	if port < 1 || port > 65535 {
		return ErrBadPort
	}
	if port == ProxyPort || port == adminPort {
		return ErrBlockedPort
	}
	return nil
}

func CleanPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/", nil
	}
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return "", ErrBadPath
	}
	path := raw
	query := ""
	if cut := strings.Index(raw, "?"); cut >= 0 {
		path = raw[:cut]
		query = raw[cut+1:]
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.Contains(path, "..") {
		return "", ErrBadPath
	}
	if query != "" {
		return path + "?" + query, nil
	}
	return path, nil
}

func PageURL(hostport, rawPath string) (string, error) {
	path, err := CleanPath(rawPath)
	if err != nil {
		return "", err
	}
	return "http://" + hostport + path, nil
}

func Handler() http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			port := req.Header.Get(Header)
			req.Header.Del(Header)
			req.URL.Scheme = "http"
			req.URL.Host = net.JoinHostPort("127.0.0.1", port)
			req.Host = req.URL.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			rewriteLocation(resp)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "打不开这个端口\n", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		port, err := strconv.Atoi(strings.TrimSpace(r.Header.Get(Header)))
		if err != nil {
			http.Error(w, ErrBadPort.Error()+"\n", http.StatusBadRequest)
			return
		}
		if err = CheckPort(port); err != nil {
			http.Error(w, err.Error()+"\n", http.StatusBadRequest)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

func rewriteLocation(resp *http.Response) {
	loc := resp.Header.Get("Location")
	if loc == "" {
		return
	}
	parsed, err := url.Parse(loc)
	if err != nil || parsed.Host == "" {
		return
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return
	}
	parsed.Scheme = ""
	parsed.Host = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	resp.Header.Set("Location", parsed.String())
}
