package statusdoc

import (
	"encoding/json"
	"net/netip"
	"strings"
)

type Document struct {
	Hostname     string   `json:"hostname"`
	DNSName      string   `json:"dnsName"`
	TailscaleIPs []string `json:"tailscaleIPs"`
	Online       bool     `json:"online"`
	BackendState string   `json:"backendState"`
	Runtime      string   `json:"runtime,omitempty"`
}

func Build(hostname, dnsName string, ips []netip.Addr) Document {
	ordered := order(ips)
	if strings.HasSuffix(dnsName, ".") {
		dnsName = strings.TrimSuffix(dnsName, ".")
	}
	state := "Offline"
	if len(ordered) > 0 {
		state = "Running"
	}
	if ordered == nil {
		ordered = []string{}
	}
	return Document{
		Hostname:     hostname,
		DNSName:      dnsName,
		TailscaleIPs: ordered,
		Online:       state == "Running",
		BackendState: state,
	}
}

func JSON(hostname, dnsName string, ips []netip.Addr) []byte {
	body, _ := json.Marshal(Build(hostname, dnsName, ips))
	return body
}

func order(ips []netip.Addr) []string {
	var v4, v6 []string
	for _, addr := range ips {
		if !addr.IsValid() {
			continue
		}
		if addr.Is4() {
			v4 = append(v4, addr.String())
		} else {
			v6 = append(v6, addr.String())
		}
	}
	return append(v4, v6...)
}
