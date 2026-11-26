import 'dart:convert';
import 'dart:io';

import 'node.dart';
import 'session.dart';

class LocalApiException implements Exception {
  LocalApiException(this.message);

  final String message;

  @override
  String toString() => message;
}

class HealthCheck {
  const HealthCheck({this.statusCode, this.name});

  final int? statusCode;
  final String? name;
}

class LocalApi {
  LocalApi({
    HttpClient? client,
    this.baseUrl = 'http://127.0.0.1:19020',
  }) : _client = client ?? HttpClient() {
    _client.connectionTimeout = const Duration(seconds: 2);
  }

  final HttpClient _client;
  final String baseUrl;

  Future<HealthCheck> checkHealth() async {
    try {
      final response = await _send('GET', '/health');
      return HealthCheck(
        statusCode: response.statusCode,
        name: _healthName(response.body),
      );
    } catch (_) {
      return const HealthCheck();
    }
  }

  Future<ClientSession> getSession() async {
    final response = await _send('GET', '/session');
    if (response.statusCode != 200) {
      throw LocalApiException(response.body);
    }
    return ClientSession.fromJson(_object(response.body));
  }

  Future<List<TailNode>> getNodes() async {
    final response = await _send('GET', '/nodes');
    if (response.statusCode != 200) {
      throw LocalApiException(response.body);
    }
    final raw = _object(response.body)['nodes'];
    if (raw is! List) {
      return const [];
    }
    return raw.map((item) {
      return TailNode.fromJson(Map<String, dynamic>.from(item as Map));
    }).toList();
  }

  Future<void> saveAuthKey(String key) async {
    final response = await _send(
      'POST',
      '/authkey',
      body: jsonEncode({'authKey': key}),
    );
    if (response.statusCode == 400) {
      throw LocalApiException(response.body);
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw LocalApiException(response.body);
    }
  }

  Future<void> startLogin() async {
    final response = await _send('POST', '/login');
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw LocalApiException(response.body);
    }
  }

  Future<NodeStatus> nodeStatus(String ip) async {
    final response = await _send(
      'GET',
      '/node-status?ip=${Uri.encodeQueryComponent(ip)}',
    );
    if (response.statusCode != 200) {
      throw LocalApiException('读不到状态');
    }
    return NodeStatus.fromJson(_object(response.body));
  }

  Future<_HttpBody> _send(String method, String path, {String? body}) async {
    final request = await _client.openUrl(method, Uri.parse('$baseUrl$path'));
    if (body != null) {
      request.headers.contentType = ContentType.json;
      request.write(body);
    }
    final response = await request.close().timeout(const Duration(seconds: 3));
    final text = await utf8.decoder
        .bind(response)
        .join()
        .timeout(const Duration(seconds: 3));
    return _HttpBody(response.statusCode, text);
  }

  Map<String, dynamic> _object(String body) {
    final decoded = jsonDecode(body);
    return Map<String, dynamic>.from(decoded as Map);
  }

  String? _healthName(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['name'] is String) {
        return decoded['name'] as String;
      }
    } catch (_) {}
    return null;
  }
}

class _HttpBody {
  const _HttpBody(this.statusCode, this.body);

  final int statusCode;
  final String body;
}
