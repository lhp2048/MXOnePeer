package runtime

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/lhp2048/MXOnePeer/go/internal/admin"
	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
	"github.com/lhp2048/MXOnePeer/go/internal/tailnet"
)

type Options struct {
	Hostname   string
	StateDir   string
	Port       int
	AdminHost  string
	AdminPort  int
	EnvKey     string
	EnvFileKey string
}

func Run(ctx context.Context, opts Options) error {
	if err := store.Prepare(opts.StateDir); err != nil {
		return err
	}
	envKey := strings.TrimSpace(opts.EnvKey)
	if envKey == "" {
		envKey = opts.EnvFileKey
	}
	if envKey != "" {
		if err := store.WriteAuthKey(opts.StateDir, envKey); err != nil {
			log.Print(err.Error())
		}
	}
	if store.ReadHostname(opts.StateDir) == "" {
		if _, err := store.WriteHostname(opts.StateDir, opts.Hostname); err != nil {
			_, _ = store.WriteHostname(opts.StateDir, "mxonepeer")
		}
	}
	board := adminboard.New(opts.AdminPort)
	board.Refresh(opts.StateDir)
	server, err := admin.Serve(opts.AdminHost, opts.AdminPort, board, opts.StateDir)
	if err != nil {
		return err
	}
	defer server.Close()
	log.Printf("local admin http://%s:%d/", opts.AdminHost, opts.AdminPort)
	for {
		if ctx.Err() != nil {
			return nil
		}
		action, generation := board.Snapshot()
		board.ClearAction()
		name := store.ReadHostname(opts.StateDir)
		if name == "" {
			name = "mxonepeer"
		}
		key := store.ReadAuthKey(opts.StateDir)
		login := action == "login"
		if login {
			key = ""
		} else if key == "" {
			wait(ctx, board.Wake())
			continue
		}
		err := tailnet.Run(ctx, tailnet.Config{
			Dir:        opts.StateDir,
			Hostname:   name,
			AuthKey:    key,
			Port:       opts.Port,
			Generation: generation,
			Login:      login,
		}, board)
		if ctx.Err() != nil {
			return nil
		}
		if board.Changed(generation) {
			continue
		}
		if err != nil {
			board.SetNode("", nil, false)
			board.SetLoginError(redact(err.Error()))
			log.Printf("tailscale: %s", redact(err.Error()))
			wait(ctx, board.Wake(), 3*time.Second)
		}
	}
}

func wait(ctx context.Context, wake <-chan struct{}, timeout ...time.Duration) {
	var timer <-chan time.Time
	if len(timeout) > 0 {
		timer = time.After(timeout[0])
	}
	select {
	case <-ctx.Done():
	case <-wake:
	case <-timer:
	}
}

func redact(text string) string {
	const marker = "tskey-auth-"
	for {
		start := strings.Index(text, marker)
		if start < 0 {
			return text
		}
		end := start + len(marker)
		for end < len(text) && text[end] != ' ' && text[end] != '\n' && text[end] != '\r' && text[end] != '\t' {
			end++
		}
		text = text[:start] + "tskey-auth-***" + text[end:]
	}
}
