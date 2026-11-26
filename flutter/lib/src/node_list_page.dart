import 'package:flutter/material.dart';

import 'node.dart';

class NodeListPage extends StatelessWidget {
  const NodeListPage({
    super.key,
    required this.nodes,
    required this.onOpen,
  });

  final List<TailNode> nodes;
  final ValueChanged<TailNode> onOpen;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: ListView.builder(
        itemCount: nodes.length,
        itemBuilder: (context, index) {
          final node = nodes[index];
          return ListTile(
            title: Row(
              children: [
                Flexible(
                  child: Text(
                    node.hostname,
                    overflow: TextOverflow.ellipsis,
                    maxLines: 1,
                  ),
                ),
                if (node.self) ...[
                  const SizedBox(width: 8),
                  const Text('本机'),
                ],
              ],
            ),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(node.online ? '在线' : '离线'),
                for (final ip in node.tailscaleIPs) Text(ip),
              ],
            ),
            onTap: () => onOpen(node),
          );
        },
      ),
    );
  }
}
