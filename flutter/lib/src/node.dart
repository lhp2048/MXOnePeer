class TailNode {
  const TailNode({
    required this.hostname,
    this.dnsName = '',
    this.tailscaleIPs = const [],
    this.online = false,
    this.self = false,
  });

  final String hostname;
  final String dnsName;
  final List<String> tailscaleIPs;
  final bool online;
  final bool self;

  factory TailNode.fromJson(Map<String, dynamic> json) {
    return TailNode(
      hostname: json['hostname'] as String? ?? '',
      dnsName: json['dnsName'] as String? ?? '',
      tailscaleIPs: (json['tailscaleIPs'] as List<dynamic>? ?? const []).cast<String>(),
      online: json['online'] as bool? ?? false,
      self: json['self'] as bool? ?? false,
    );
  }
}

class NodeStatus {
  const NodeStatus({
    required this.hostname,
    this.dnsName = '',
    this.tailscaleIPs = const [],
    this.online = false,
    this.backendState = '',
  });

  final String hostname;
  final String dnsName;
  final List<String> tailscaleIPs;
  final bool online;
  final String backendState;

  factory NodeStatus.fromJson(Map<String, dynamic> json) {
    return NodeStatus(
      hostname: json['hostname'] as String? ?? '',
      dnsName: json['dnsName'] as String? ?? '',
      tailscaleIPs: (json['tailscaleIPs'] as List<dynamic>? ?? const []).cast<String>(),
      online: json['online'] as bool? ?? false,
      backendState: json['backendState'] as String? ?? '',
    );
  }
}
