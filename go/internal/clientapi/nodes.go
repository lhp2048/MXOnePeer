package clientapi

import "strings"

type Node struct {
	Hostname     string   `json:"hostname"`
	DNSName      string   `json:"dnsName"`
	TailscaleIPs []string `json:"tailscaleIPs"`
	Online       bool     `json:"online"`
	Self         bool     `json:"self"`
}

func StripDNS(name string) string {
	if strings.HasSuffix(name, ".") {
		return strings.TrimSuffix(name, ".")
	}
	return name
}

func NonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func OrderNodes(self Node, peers []Node) []Node {
	self.Self = true
	self.TailscaleIPs = NonNil(self.TailscaleIPs)
	ordered := append([]Node{}, peers...)
	for i := 1; i < len(ordered); i++ {
		item := ordered[i]
		j := i
		for j > 0 && ordered[j-1].Hostname > item.Hostname {
			ordered[j] = ordered[j-1]
			j--
		}
		ordered[j] = item
	}
	for i := range ordered {
		ordered[i].Self = false
		ordered[i].TailscaleIPs = NonNil(ordered[i].TailscaleIPs)
		ordered[i].DNSName = StripDNS(ordered[i].DNSName)
	}
	self.DNSName = StripDNS(self.DNSName)
	return append([]Node{self}, ordered...)
}

func SessionOnline(backendState string, ips []string) bool {
	return backendState == "Running" && len(ips) > 0
}
