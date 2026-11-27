package statusdoc

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
)

func TestOrderAndDNS(t *testing.T) {
	doc := Build("mxonepeer", "mxonepeer.tailnet.ts.net.", []netip.Addr{
		netip.MustParseAddr("fd7a:115c:a1e0::1"),
		netip.MustParseAddr("100.64.0.1"),
	})
	if doc.DNSName != "mxonepeer.tailnet.ts.net" {
		t.Fatalf("dns %q", doc.DNSName)
	}
	if len(doc.TailscaleIPs) != 2 || doc.TailscaleIPs[0] != "100.64.0.1" || doc.TailscaleIPs[1] != "fd7a:115c:a1e0::1" {
		t.Fatalf("ips %#v", doc.TailscaleIPs)
	}
	if !doc.Online || doc.BackendState != "Running" {
		t.Fatalf("state %#v", doc)
	}
	empty := Build("", "", nil)
	if empty.Online || empty.BackendState != "Offline" || strings.Contains(string(JSON("", "", nil)), "null") {
		t.Fatalf("empty %#v %s", empty, JSON("", "", nil))
	}
}

func TestRuntimeOmittedWhenEmpty(t *testing.T) {
	raw := string(JSON("mxonepeer", "", nil))
	if strings.Contains(raw, "runtime") {
		t.Fatalf("empty runtime leaked %s", raw)
	}
	doc := Build("mxonepeer", "", nil)
	doc.Runtime = "go/linux/amd64"
	body, _ := json.Marshal(doc)
	if !strings.Contains(string(body), `"runtime":"go/linux/amd64"`) {
		t.Fatalf("runtime %s", body)
	}
}
