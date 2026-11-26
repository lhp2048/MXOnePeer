package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/lhp2048/MXOnePeer/go/internal/admin"
	"github.com/lhp2048/MXOnePeer/go/internal/clientapi"
	"github.com/lhp2048/MXOnePeer/go/internal/clientnet"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

func main() {
	home, _ := os.UserHomeDir()
	stateDefault := filepath.Join(home, "Library", "Application Support", "mxonepeer-client")
	stateDir := flag.String("state-dir", stateDefault, "client state directory")
	apiHost := flag.String("api-host", "127.0.0.1", "local api address")
	apiPort := flag.Int("api-port", 19020, "local api port")
	port := flag.Int("port", 18020, "tailscale status port")
	hostname := flag.String("hostname", "mxonepeer-mac", "computer name")
	flag.Parse()
	if !admin.Loopback(*apiHost) {
		fmt.Fprintln(os.Stderr, "api host must be 127.0.0.1 or ::1")
		os.Exit(1)
	}
	if err := store.Prepare(*stateDir); err != nil {
		fmt.Fprintln(os.Stderr, clientapi.Redact(err.Error()))
		os.Exit(1)
	}
	if store.ReadHostname(*stateDir) == "" {
		_, _ = store.WriteHostname(*stateDir, store.SanitizeHostname(*hostname, "mxonepeer-mac"))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	err := clientnet.Run(ctx, clientnet.Options{
		StateDir: *stateDir,
		Hostname: store.ReadHostname(*stateDir),
		APIHost:  *apiHost,
		APIPort:  *apiPort,
		Port:     *port,
	})
	if err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, clientapi.Redact(err.Error()))
		os.Exit(1)
	}
}
