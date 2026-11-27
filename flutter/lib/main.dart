import 'dart:async';
import 'dart:io';
import 'dart:ui' show AppExitResponse;

import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/engine.dart';
import 'src/local_api.dart';
import 'src/login_page.dart';
import 'src/node.dart';
import 'src/node_detail_page.dart';
import 'src/node_list_page.dart';
import 'src/session.dart';

final localApiProvider = Provider<LocalApi>((ref) => LocalApi());

ThemeData mxonepeerTheme() {
  return ThemeData(
    colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF0F6E6E)),
    useMaterial3: true,
  );
}

void main() {
  runApp(const ProviderScope(child: MxonepeerApp()));
}

class MxonepeerApp extends StatelessWidget {
  const MxonepeerApp({super.key, this.home});

  /// When set, the engine is not started. Widget tests use this.
  final Widget? home;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MXOnePeer',
      theme: mxonepeerTheme(),
      home: home ?? const EngineHome(),
    );
  }
}

class EngineHome extends ConsumerStatefulWidget {
  const EngineHome({super.key});

  @override
  ConsumerState<EngineHome> createState() => _EngineHomeState();
}

class _EngineHomeState extends ConsumerState<EngineHome> {
  AppLifecycleListener? _lifecycle;
  Process? _owned;
  bool _ownedExited = false;
  Timer? _timer;
  ClientSession _session = const ClientSession(backendState: '', online: false);
  List<TailNode> _nodes = const [];
  String _submitError = '';
  String? _blockMessage;
  bool _ready = false;
  bool _refreshing = false;
  bool _refreshAgain = false;
  bool _exitRequested = false;

  @override
  void initState() {
    super.initState();
    _lifecycle = AppLifecycleListener(onExitRequested: _onExitRequested);
    unawaited(_boot());
  }

  Future<AppExitResponse> _onExitRequested() async {
    _exitRequested = true;
    final owned = _owned;
    _owned = null;
    if (owned != null) {
      owned.kill();
    }
    return AppExitResponse.exit;
  }

  @override
  void dispose() {
    _timer?.cancel();
    _lifecycle?.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final api = ref.read(localApiProvider);
    final health = await api.checkHealth();
    if (!mounted) {
      return;
    }
    switch (chooseEngine(health.statusCode, health.name)) {
      case EngineChoice.busy:
        setState(() => _blockMessage = '端口被占用');
        return;
      case EngineChoice.reuse:
        break;
      case EngineChoice.spawn:
        final started = await _spawnOwnedEngine();
        if (!mounted) {
          return;
        }
        if (!started) {
          setState(() => _blockMessage = '引擎没有启动');
          return;
        }
        final after = await _waitForOwnEngine();
        if (!mounted) {
          return;
        }
        final block = spawnWaitBlockMessage(after, _ownedExited);
        if (block != null) {
          if (!_ownedExited) {
            _owned?.kill();
          }
          _owned = null;
          setState(() => _blockMessage = block);
          return;
        }
        break;
    }
    setState(() => _ready = true);
    await _refresh();
    if (!mounted) {
      return;
    }
    _timer = Timer.periodic(const Duration(seconds: 3), (_) {
      unawaited(_refresh());
    });
  }

  Future<bool> _spawnOwnedEngine() async {
    String machine;
    try {
      machine = await readUnameMachine();
    } catch (_) {
      return false;
    }
    final binary = await findEngineBinary(engineArch(machine));
    if (binary == null) {
      return false;
    }
    final hostname = await readComputerName();
    try {
      final process = await startEngineProcess(
        binary: binary,
        stateDir: clientStateDir(),
        hostname: hostname,
      );
      _owned = process;
      _ownedExited = false;
      if (_exitRequested) {
        _owned = null;
        process.kill();
        return false;
      }
      process.stdout.listen((_) {}, onError: (_) {});
      process.stderr.listen((_) {}, onError: (_) {});
      unawaited(process.exitCode.then((_) {
        _ownedExited = true;
      }, onError: (_) {
        _ownedExited = true;
      }));
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<EngineChoice> _waitForOwnEngine() async {
    final api = ref.read(localApiProvider);
    var latest = EngineChoice.spawn;
    for (var i = 0; i < 20; i++) {
      if (!mounted) {
        return latest;
      }
      final health = await api.checkHealth();
      latest = chooseEngine(health.statusCode, health.name);
      if (latest != EngineChoice.spawn) {
        return latest;
      }
      await Future<void>.delayed(const Duration(milliseconds: 250));
    }
    return latest;
  }

  Future<void> _refresh() async {
    if (!_ready || !mounted) {
      return;
    }
    if (_refreshing) {
      _refreshAgain = true;
      return;
    }
    _refreshing = true;
    try {
      await _refreshOnce();
    } finally {
      _refreshing = false;
      if (_refreshAgain && mounted) {
        _refreshAgain = false;
        await _refresh();
      }
    }
  }

  Future<void> _refreshOnce() async {
    final api = ref.read(localApiProvider);
    final ClientSession session;
    try {
      session = await api.getSession();
    } catch (_) {
      return;
    }
    if (!mounted) {
      return;
    }
    List<TailNode>? nodes;
    if (session.online) {
      try {
        nodes = await api.getNodes();
      } catch (_) {}
      if (!mounted) {
        return;
      }
    }
    final wasOnline = _session.online;
    setState(() {
      _session = session;
      if (!session.online) {
        _nodes = const [];
      } else if (nodes != null) {
        _nodes = nodes;
      }
    });
    if (wasOnline && !session.online) {
      Navigator.of(context).popUntil((route) => route.isFirst);
    }
  }

  Future<void> _submit(String key) async {
    try {
      await ref.read(localApiProvider).saveAuthKey(key);
      if (!mounted) {
        return;
      }
      setState(() => _submitError = '');
      await _refresh();
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _submitError = _errorText(error));
    }
  }

  Future<void> _login() async {
    try {
      await ref.read(localApiProvider).startLogin();
      await _refresh();
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _submitError = _errorText(error));
    }
  }

  void _open(TailNode node) {
    final ip = node.tailscaleIPs.isEmpty ? '' : node.tailscaleIPs.first;
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => NodeDetailPage(
          ip: ip,
          loader: (address) {
            if (address.isEmpty) {
              return Future<NodeStatus>.error(LocalApiException('读不到状态'));
            }
            return ref.read(localApiProvider).nodeStatus(address);
          },
          pickUpgrade: () async {
            final file = await openFile();
            if (file == null) {
              return null;
            }
            return UpgradePick(file.name, await file.readAsBytes());
          },
          sendUpgrade: (runtime, filename, bytes) {
            return ref.read(localApiProvider).uploadPackage(
              ip,
              runtime,
              filename,
              bytes,
            );
          },
        ),
      ),
    );
  }

  String _errorText(Object error) {
    if (error is LocalApiException) {
      return error.message;
    }
    return '$error';
  }

  @override
  Widget build(BuildContext context) {
    if (_blockMessage != null) {
      return Scaffold(body: Center(child: Text(_blockMessage!)));
    }
    if (!_ready) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final error = _submitError.isNotEmpty ? _submitError : _session.loginError;
    if (!_session.online) {
      return LoginPage(
        session: _session,
        error: error,
        onSubmit: _submit,
        onLogin: _login,
      );
    }
    return NodeListPage(nodes: _nodes, onOpen: _open);
  }
}
