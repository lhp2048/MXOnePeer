class ClientSession {
  const ClientSession({
    required this.backendState,
    required this.online,
    this.hostname = '',
    this.requestedHostname = '',
    this.tailscaleIPs = const [],
    this.loginUrl = '',
    this.authKeyConfigured = false,
    this.authKeyMasked = '',
    this.loginError = '',
  });

  final String backendState;
  final bool online;
  final String hostname;
  final String requestedHostname;
  final List<String> tailscaleIPs;
  final String loginUrl;
  final bool authKeyConfigured;
  final String authKeyMasked;
  final String loginError;

  factory ClientSession.fromJson(Map<String, dynamic> json) {
    return ClientSession(
      backendState: json['backendState'] as String? ?? '',
      online: json['online'] as bool? ?? false,
      hostname: json['hostname'] as String? ?? '',
      requestedHostname: json['requestedHostname'] as String? ?? '',
      tailscaleIPs: (json['tailscaleIPs'] as List<dynamic>? ?? const []).cast<String>(),
      loginUrl: json['loginUrl'] as String? ?? '',
      authKeyConfigured: json['authKeyConfigured'] as bool? ?? false,
      authKeyMasked: json['authKeyMasked'] as String? ?? '',
      loginError: json['loginError'] as String? ?? '',
    );
  }
}
