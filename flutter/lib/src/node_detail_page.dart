import 'package:flutter/material.dart';

import 'node.dart';

class NodeDetailPage extends StatefulWidget {
  const NodeDetailPage({
    super.key,
    required this.ip,
    required this.loader,
  });

  final String ip;
  final Future<NodeStatus> Function(String ip) loader;

  @override
  State<NodeDetailPage> createState() => _NodeDetailPageState();
}

class _NodeDetailPageState extends State<NodeDetailPage> {
  NodeStatus? _status;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final status = await widget.loader(widget.ip);
      if (!mounted) return;
      setState(() {
        _status = status;
        _failed = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _failed = true;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          if (_failed) const Text('读不到状态'),
          if (!_failed && _status != null) ...[
            Text(_status!.hostname),
            Text(_status!.dnsName),
            for (final ip in _status!.tailscaleIPs) Text(ip),
            Text(_status!.online ? '在线' : '离线'),
            Text(_status!.backendState),
          ],
          FilledButton(
            onPressed: _load,
            child: const Text('刷新'),
          ),
        ],
      ),
    );
  }
}
