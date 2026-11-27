import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

class NodeBrowserPage extends StatefulWidget {
  const NodeBrowserPage({
    super.key,
    required this.title,
    required this.url,
  });

  final String title;
  final String url;

  @override
  State<NodeBrowserPage> createState() => _NodeBrowserPageState();
}

class _NodeBrowserPageState extends State<NodeBrowserPage> {
  late final WebViewController _controller;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.title)),
      body: WebViewWidget(controller: _controller),
    );
  }
}
