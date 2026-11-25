package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lhp2048/MXOnePeer/go/internal/admin"
	"github.com/lhp2048/MXOnePeer/go/internal/runtime"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

var cleared = []string{
	"TS_AUTHKEY",
	"TS_AUTH_KEY",
	"TS_CLIENT_SECRET",
	"TS_CLIENT_ID",
	"TS_ID_TOKEN",
	"TS_AUDIENCE",
	"TS_CONTROL_URL",
	"TS_HOSTNAME",
}

func main() {
	hostname := flag.String("hostname", "mxonepeer", "default node name when none is saved")
	stateDir := flag.String("state-dir", "/var/lib/mxonepeer", "directory for the node state")
	port := flag.Int("port", 18020, "status port on the Tailscale address")
	adminHost := flag.String("admin-host", "127.0.0.1", "local admin address")
	adminPort := flag.Int("admin-port", 8788, "local admin port")
	flag.Parse()
	if !admin.Loopback(*adminHost) {
		fmt.Fprintln(os.Stderr, "admin host must be 127.0.0.1 or ::1")
		os.Exit(1)
	}
	envKey := os.Getenv("TS_AUTHKEY")
	for _, name := range cleared {
		_ = os.Unsetenv(name)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	err := runtime.Run(ctx, runtime.Options{
		Hostname:   *hostname,
		StateDir:   *stateDir,
		Port:       *port,
		AdminHost:  *adminHost,
		AdminPort:  *adminPort,
		EnvKey:     envKey,
		EnvFileKey: store.ReadEnvKey(*stateDir),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, redact(err.Error(), envKey))
		os.Exit(1)
	}
}

func redact(message, authKey string) string {
	if authKey == "" {
		return message
	}
	return runtimeRedact(message, authKey)
}

func runtimeRedact(message, authKey string) string {
	out := message
	for authKey != "" && len(out) >= len(authKey) {
		next := indexOf(out, authKey)
		if next < 0 {
			break
		}
		out = out[:next] + "tskey-auth-***" + out[next+len(authKey):]
	}
	return out
}

func indexOf(text, part string) int {
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
