import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

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

  Future<void> uploadPackage(
    String ip,
    String runtime,
    String filename,
    List<int> bytes,
  ) async {
    const boundary = 'mxonepeer-upgrade';
    final _HttpBody response;
    try {
      response = await _send(
        'POST',
        '/upgrade',
        bytes: _upgradeBody(boundary, ip, runtime, filename, bytes),
        contentType: ContentType(
          'multipart',
          'form-data',
          parameters: {'boundary': boundary},
        ),
        timeout: const Duration(seconds: 120),
      );
    } on TimeoutException {
      throw LocalApiException('传不上去');
    } on SocketException {
      throw LocalApiException('传不上去');
    } on IOException {
      throw LocalApiException('传不上去');
    }
    if (response.statusCode == 204) {
      return;
    }
    if (response.statusCode == 400 || response.statusCode == 500) {
      throw LocalApiException(_errorField(response.body));
    }
    throw LocalApiException('传不上去');
  }

  Future<_HttpBody> _send(
    String method,
    String path, {
    String? body,
    List<int>? bytes,
    ContentType? contentType,
    Duration timeout = const Duration(seconds: 3),
  }) async {
    final request = await _client.openUrl(method, Uri.parse('$baseUrl$path'));
    if (body != null) {
      request.headers.contentType = contentType ?? ContentType.json;
      request.write(body);
    } else if (bytes != null) {
      request.headers.contentType = contentType;
      request.add(bytes);
    }
    final response = await request.close().timeout(timeout);
    final text = await utf8.decoder.bind(response).join().timeout(timeout);
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

  String _errorField(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['error'] is String) {
        return decoded['error'] as String;
      }
    } catch (_) {}
    return body.trim();
  }

  List<int> _upgradeBody(
    String boundary,
    String ip,
    String runtime,
    String filename,
    List<int> bytes,
  ) {
    final buffer = BytesBuilder();
    void addText(String text) {
      buffer.add(utf8.encode(text));
    }

    void addField(String name, String value) {
      addText('--$boundary\r\n');
      addText('Content-Disposition: form-data; name="$name"\r\n\r\n');
      addText('$value\r\n');
    }

    addField('ip', ip);
    addField('runtime', runtime);
    addText('--$boundary\r\n');
    addText(
      'Content-Disposition: form-data; name="package"; filename="${_basename(filename)}"\r\n',
    );
    addText('Content-Type: application/octet-stream\r\n\r\n');
    buffer.add(bytes);
    addText('\r\n--$boundary--\r\n');
    return buffer.takeBytes();
  }

  String _basename(String filename) {
    var name = filename.replaceAll('\\', '/');
    final slash = name.lastIndexOf('/');
    if (slash >= 0) {
      name = name.substring(slash + 1);
    }
    return name;
  }
}

class _HttpBody {
  const _HttpBody(this.statusCode, this.body);

  final int statusCode;
  final String body;
}
