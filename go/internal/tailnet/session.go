package tailnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/statusdoc"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

type Config struct {
	Dir        string
	Hostname   string
	AuthKey    string
	Port       int
	Generation int
	Login      bool
}

func Run(parent context.Context, cfg Config, board *adminboard.Board) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if board.Changed(cfg.Generation) {
					cancel()
					return
				}
			}
		}
	}()

	server := &tsnet.Server{
		Dir:       cfg.Dir,
		Hostname:  cfg.Hostname,
		AuthKey:   cfg.AuthKey,
		Ephemeral: false,
		Logf:      func(string, ...any) {},
	}
	defer server.Close()
	if err := server.Start(); err != nil {
		return err
	}
	client, err := server.LocalClient()
	if err != nil {
		return err
	}
	listener, err := server.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           statusHandler(client),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = httpServer.Serve(listener) }()
	defer func() {
		shutdown, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		_ = httpServer.Shutdown(shutdown)
		_ = listener.Close()
	}()

	announced := false
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := publish(ctx, client, board, cfg.Login, &announced); err != nil && ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			if board.Changed(cfg.Generation) {
				return nil
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func publish(ctx context.Context, client *local.Client, board *adminboard.Board, login bool, announced *bool) error {
	query, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	status, err := client.Status(query)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	doc := document(status)
	board.SetNode(doc.Hostname, doc.TailscaleIPs, doc.Online)
	if status.AuthURL != "" {
		board.SetLoginURL(status.AuthURL)
		board.SetNotice("用浏览器打开登录链接，使用你的 Tailscale 账号批准这台设备。")
	}
	if doc.Online && !*announced {
		if login {
			board.SetNotice("账号已登录。")
		} else {
			board.SetNotice("auth key 已连接。")
		}
		*announced = true
	}
	return nil
}

func document(status *ipnstate.Status) statusdoc.Document {
	if status == nil || status.Self == nil {
		return statusdoc.Build("", "", nil)
	}
	return statusdoc.Build(status.Self.HostName, status.Self.DNSName, status.Self.TailscaleIPs)
}

func statusHandler(client *local.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		status, err := client.Status(ctx)
		doc := statusdoc.Build("", "", nil)
		if err == nil {
			doc = document(status)
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := json.Marshal(doc)
		_, _ = w.Write(body)
	})
}
