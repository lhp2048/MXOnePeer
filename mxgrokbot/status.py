import ipaddress
import json


def document(hostname, dns_name, ips, backend_state):
    ordered = _order_ips(ips)
    dns = dns_name[0:-1] if dns_name.endswith(".") else dns_name
    return {
        "hostname": hostname,
        "dnsName": dns,
        "tailscaleIPs": ordered,
        "online": backend_state == "Running" and len(ordered) > 0,
        "backendState": backend_state,
    }


def dumps(hostname, dns_name, ips, backend_state):
    return json.dumps(
        document(hostname, dns_name, ips, backend_state),
        ensure_ascii=False,
    ).encode("utf-8")


def _order_ips(ips):
    v4 = []
    v6 = []
    for raw in ips:
        try:
            addr = ipaddress.ip_address(raw)
        except ValueError:
            continue
        if addr.version == 4:
            v4.append(str(addr))
        else:
            v6.append(str(addr))
    return v4 + v6
