import 'package:flutter/material.dart';

import 'node.dart';
import 'upgrade_name.dart';

class UpgradePick {
  const UpgradePick(this.name, this.bytes);

  final String name;
  final List<int> bytes;
}

class NodeDetailPage extends StatefulWidget {
  const NodeDetailPage({
    super.key,
    required this.ip,
    required this.loader,
    this.pickUpgrade,
    this.sendUpgrade,
    this.openBrowser,
  });

  final String ip;
  final Future<NodeStatus> Function(String ip) loader;
  final Future<UpgradePick?> Function()? pickUpgrade;
  final Future<void> Function(String runtime, String filename, List<int> bytes)?
      sendUpgrade;
  final Future<void> Function(int port, String path)? openBrowser;

  @override
  State<NodeDetailPage> createState() => _NodeDetailPageState();
}

class _NodeDetailPageState extends State<NodeDetailPage> {
  final TextEditingController _port = TextEditingController();
  final TextEditingController _path = TextEditingController();
  NodeStatus? _status;
  bool _failed = false;
  bool _sending = false;
  UpgradePick? _pick;
  String? _message;
  String _browserError = '';

  @override
  void dispose() {
    _port.dispose();
    _path.dispose();
    super.dispose();
  }

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

  Future<void> _pickFile() async {
    final pickUpgrade = widget.pickUpgrade;
    if (pickUpgrade == null) {
      return;
    }
    final pick = await pickUpgrade();
    if (!mounted || pick == null) {
      return;
    }
    if (!upgradeFilenameOk(_status!.runtime, pick.name)) {
      setState(() {
        _message = '文件名不正确';
      });
      return;
    }
    setState(() {
      _pick = pick;
      _message = null;
    });
  }

  Future<void> _send() async {
    final pick = _pick;
    final status = _status;
    final sendUpgrade = widget.sendUpgrade;
    if (pick == null || status == null || sendUpgrade == null || _sending) {
      return;
    }
    setState(() {
      _sending = true;
    });
    try {
      await sendUpgrade(status.runtime, pick.name, pick.bytes);
      if (!mounted) return;
      setState(() {
        _message = '节点正在重启';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _message = '$error';
      });
    } finally {
      if (mounted) {
        setState(() {
          _sending = false;
        });
      }
    }
  }

  Future<void> _openBrowser() async {
    final opener = widget.openBrowser;
    if (opener == null) {
      return;
    }
    final port = int.tryParse(_port.text.trim());
    if (port == null) {
      setState(() => _browserError = '端口不正确');
      return;
    }
    setState(() => _browserError = '');
    try {
      await opener(port, _path.text);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _browserError = '$error');
    }
  }

  @override
  Widget build(BuildContext context) {
    final status = _status;
    final canUpgrade = status != null &&
        !_failed &&
        canUpgradeRuntime(status.runtime) &&
        widget.pickUpgrade != null &&
        widget.sendUpgrade != null;
    return Scaffold(
      appBar: AppBar(),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          if (_failed) const Text('读不到状态'),
          if (!_failed && status != null) ...[
            Text(status.hostname),
            Text(status.dnsName),
            for (final ip in status.tailscaleIPs) Text(ip),
            Text(status.online ? '在线' : '离线'),
            Text(status.backendState),
            if (status.runtime.isNotEmpty) Text(status.runtime),
          ],
          if (canUpgrade) ...[
            FilledButton(
              onPressed: _pickFile,
              child: const Text('选择文件'),
            ),
            FilledButton(
              onPressed: _pick == null || _sending ? null : _send,
              child: const Text('发送'),
            ),
            if (_pick != null) Text(_pick!.name),
          ],
          if (_message != null) Text(_message!),
          FilledButton(
            onPressed: _load,
            child: const Text('刷新'),
          ),
          if (widget.openBrowser != null) ...[
            const SizedBox(height: 24),
            TextField(
              controller: _port,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '端口'),
            ),
            TextField(
              controller: _path,
              decoration: const InputDecoration(labelText: '路径'),
            ),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: _openBrowser,
              child: const Text('打开'),
            ),
            if (_browserError.isNotEmpty) Text(_browserError),
          ],
        ],
      ),
    );
  }
}
