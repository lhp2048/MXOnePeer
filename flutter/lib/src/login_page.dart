import 'package:flutter/material.dart';

import 'session.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({
    super.key,
    required this.session,
    required this.error,
    required this.onSubmit,
    required this.onLogin,
  });

  final ClientSession session;
  final String error;
  final Future<void> Function(String key) onSubmit;
  final Future<void> Function() onLogin;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _key = TextEditingController();

  @override
  void dispose() {
    _key.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Text('MXOnePeer'),
          TextField(
            controller: _key,
            obscureText: true,
            decoration: const InputDecoration(labelText: 'auth key'),
          ),
          FilledButton(
            onPressed: () => widget.onSubmit(_key.text),
            child: const Text('保存 auth key'),
          ),
          OutlinedButton(
            onPressed: widget.onLogin,
            child: const Text('用 Tailscale 账号登录'),
          ),
          if (widget.session.loginUrl.isNotEmpty) SelectableText(widget.session.loginUrl),
          if (widget.error.isNotEmpty) Text(widget.error),
        ],
      ),
    );
  }
}
