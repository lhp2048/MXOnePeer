import 'dart:io';

enum EngineChoice { reuse, spawn, busy }

String engineArch(String unameMachine) {
  switch (unameMachine) {
    case 'arm64':
      return 'arm64';
    case 'x86_64':
      return 'amd64';
    default:
      return unameMachine;
  }
}

EngineChoice chooseEngine(int? statusCode, String? name) {
  if (statusCode == null) {
    return EngineChoice.spawn;
  }
  if (statusCode == 200 && name == 'mxonepeer-client') {
    return EngineChoice.reuse;
  }
  return EngineChoice.busy;
}

const fallbackComputerName = 'mxonepeer-mac';

String clientStateDir() {
  final home = Platform.environment['HOME'] ?? '';
  return '$home/Library/Application Support/mxonepeer-client';
}

Future<String> readUnameMachine() async {
  final result = await Process.run('uname', ['-m']);
  if (result.exitCode != 0) {
    throw StateError('uname failed');
  }
  return '${result.stdout}'.trim();
}

Future<String> readComputerName() async {
  try {
    final result = await Process.run('scutil', ['--get', 'ComputerName']);
    if (result.exitCode == 0) {
      final name = '${result.stdout}'.trim();
      if (name.isNotEmpty) {
        return name;
      }
    }
  } catch (_) {}
  return fallbackComputerName;
}

Future<String?> findEngineBinary(String arch) async {
  final beside = File(
    '${File(Platform.resolvedExecutable).parent.path}/mxonepeer-client',
  );
  if (await _isFile(beside)) {
    return beside.path;
  }

  final relative = 'go/dist/mxonepeer-client-darwin-$arch';
  final starts = <Directory>[
    Directory.current,
    File(Platform.resolvedExecutable).parent,
  ];
  for (final start in starts) {
    var dir = start.absolute;
    for (var i = 0; i < 16; i++) {
      final candidate = File('${dir.path}/$relative');
      if (await _isFile(candidate)) {
        return candidate.path;
      }
      final parent = dir.parent;
      if (parent.path == dir.path) {
        break;
      }
      dir = parent;
    }
  }
  return null;
}

Future<Process> startEngineProcess({
  required String binary,
  required String stateDir,
  required String hostname,
}) {
  return Process.start(binary, [
    '--state-dir',
    stateDir,
    '--api-host',
    '127.0.0.1',
    '--api-port',
    '19020',
    '--port',
    '18020',
    '--hostname',
    hostname,
  ]);
}

Future<bool> _isFile(File file) async {
  if (!await file.exists()) {
    return false;
  }
  return FileSystemEntity.isFile(file.path);
}
