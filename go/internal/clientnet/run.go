package clientnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/lhp2048/MXOnePeer/go/internal/clientapi"
	"github.com/lhp2048/MXOnePeer/go/internal/statusdoc"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
	"github.com/lhp2048/MXOnePeer/go/internal/upgrade"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

var errRestart = errors.New("restart")

type Options struct {
	StateDir string
	Hostname string
	APIHost  string
	APIPort  int
	Port     int
}

type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type Live struct {
	dir     string
	mu      sync.Mutex
	snap    clientapi.Snapshot
	login   bool
	restart chan struct{}
	dial    dialFunc
}

func Run(ctx context.Context, opts Options) error {
	if opts.StateDir == "" {
		return errors.New("state dir is required")
	}
	live := &Live{dir: opts.StateDir, restart: make(chan struct{}, 1)}
	listener, err := net.Listen("tcp", net.JoinHostPort(opts.APIHost, strconv.Itoa(opts.APIPort)))
	if err != nil {
		return err
	}
	api := &http.Server{Handler: clientapi.Handler(live), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = api.Serve(listener) }()
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = api.Shutdown(stop)
	}()
	for {
		if ctx.Err() != nil {
			return nil
		}
		err = live.once(ctx, opts)
		if errors.Is(err, errRestart) {
			continue
		}
		if err != nil {
			live.setError(clientapi.Redact(err.Error()))
			log.Print(clientapi.Redact(err.Error()))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func (l *Live) once(ctx context.Context, opts Options) error {
	name := store.ReadHostname(opts.StateDir)
	if name == "" {
		name = "mxonepeer-mac"
	}
	key := store.ReadAuthKey(opts.StateDir)
	l.mu.Lock()
	if l.login {
		key = ""
	}
	l.mu.Unlock()
	server := &tsnet.Server{
		Dir: opts.StateDir, Hostname: name, AuthKey: key, Ephemeral: false,
		Logf: func(format string, args ...any) {
			log.Print(clientapi.Redact(fmt.Sprintf(format, args...)))
		},
	}
	defer server.Close()
	if err := server.Start(); err != nil {
		return err
	}
	client, err := server.LocalClient()
	if err != nil {
		return err
	}
	statusLn, err := server.Listen("tcp", fmt.Sprintf(":%d", opts.Port))
	if err != nil {
		return err
	}
	statusSrv := &http.Server{Handler: tailStatus(client), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = statusSrv.Serve(statusLn) }()
	defer statusSrv.Close()
	l.setDial(server.Dial)
	defer l.setDial(nil)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		l.publish(ctx, client, name)
		select {
		case <-ctx.Done():
			return nil
		case <-l.restart:
			return errRestart
		case <-ticker.C:
		}
	}
}

func (l *Live) Snapshot() clientapi.Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	snap := l.snap
	snap.Session.TailscaleIPs = clientapi.NonNil(snap.Session.TailscaleIPs)
	if snap.Nodes == nil {
		snap.Nodes = []clientapi.Node{}
	}
	return snap
}

func (l *Live) SaveAuthKey(key string) error {
	if err := store.WriteAuthKey(l.dir, key); err != nil {
		return clientapi.ErrAuthKey
	}
	l.mu.Lock()
	l.login = false
	l.mu.Unlock()
	l.poke()
	return nil
}

func (l *Live) StartLogin() {
	l.mu.Lock()
	l.login = true
	l.mu.Unlock()
	l.poke()
}

func (l *Live) poke() {
	select {
	case l.restart <- struct{}{}:
	default:
	}
}

func (l *Live) setDial(dial dialFunc) {
	l.mu.Lock()
	l.dial = dial
	l.mu.Unlock()
}

func (l *Live) setError(text string) {
	l.mu.Lock()
	l.snap.Session.LoginError = text
	l.snap.Session.Online = false
	l.mu.Unlock()
}

func (l *Live) FetchStatus(ip string) (int, []byte) {
	l.mu.Lock()
	dial := l.dial
	l.mu.Unlock()
	if dial == nil {
		return http.StatusBadGateway, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DialContext: dial}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientapi.StatusURL(ip), nil)
	if err != nil {
		return http.StatusBadGateway, nil
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return http.StatusBadGateway, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if parsed, ok := clientapi.ReadStatus(resp.StatusCode, body); ok {
		return http.StatusOK, parsed
	}
	return http.StatusBadGateway, nil
}

func (l *Live) publish(ctx context.Context, client *local.Client, requested string) {
	query, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	status, err := client.Status(query)
	if err != nil || status == nil {
		return
	}
	doc := statusdoc.Build("", "", nil)
	doc.Runtime = upgrade.Current("client")
	self := clientapi.Node{Hostname: requested, Self: true}
	if status.Self != nil {
		doc = statusdoc.Build(status.Self.HostName, status.Self.DNSName, status.Self.TailscaleIPs)
		doc.Runtime = upgrade.Current("client")
		self = clientapi.Node{Hostname: doc.Hostname, DNSName: doc.DNSName, TailscaleIPs: doc.TailscaleIPs, Self: true}
	}
	ips := clientapi.NonNil(doc.TailscaleIPs)
	online := clientapi.SessionOnline(status.BackendState, ips)
	self.Online = online
	var peers []clientapi.Node
	for _, peer := range status.Peer {
		if peer == nil {
			continue
		}
		peerDoc := statusdoc.Build(peer.HostName, peer.DNSName, peer.TailscaleIPs)
		peers = append(peers, clientapi.Node{
			Hostname: peerDoc.Hostname, DNSName: peerDoc.DNSName,
			TailscaleIPs: peerDoc.TailscaleIPs, Online: peer.Online,
		})
	}
	stored := store.ReadAuthKey(l.dir)
	l.mu.Lock()
	l.snap = clientapi.Snapshot{
		Session: clientapi.Session{
			BackendState: status.BackendState, Online: online, Hostname: self.Hostname,
			RequestedHostname: requested, TailscaleIPs: ips, LoginURL: status.AuthURL,
			AuthKeyConfigured: stored != "", AuthKeyMasked: store.MaskAuthKey(stored),
		},
		Nodes: clientapi.OrderNodes(self, peers), SelfStatus: doc,
	}
	l.mu.Unlock()
}

func tailStatus(client *local.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		status, err := client.Status(ctx)
		doc := statusdoc.Build("", "", nil)
		doc.Runtime = upgrade.Current("client")
		if err == nil && status != nil && status.Self != nil {
			doc = statusdoc.Build(status.Self.HostName, status.Self.DNSName, status.Self.TailscaleIPs)
			doc.Runtime = upgrade.Current("client")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
}
