package clientapi

import "testing"

func TestOrderAndOnline(t *testing.T) {
	if StripDNS("mxonepeer.tailnet.ts.net.") != "mxonepeer.tailnet.ts.net" {
		t.Fatal("dns")
	}
	self := Node{Hostname: "self", TailscaleIPs: nil, Online: true, Self: true}
	peers := []Node{{Hostname: "b"}, {Hostname: "a", Online: true}}
	got := OrderNodes(self, peers)
	if len(got) != 3 || !got[0].Self || got[1].Hostname != "a" || got[2].Hostname != "b" {
		t.Fatalf("%#v", got)
	}
	if got[0].TailscaleIPs == nil {
		t.Fatal("nil ips")
	}
	if !SessionOnline("Running", []string{"100.64.0.1"}) || SessionOnline("NeedsLogin", []string{"100.64.0.1"}) || SessionOnline("Running", nil) {
		t.Fatal("online")
	}
}
