import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mxonepeer_client/src/node.dart';
import 'package:mxonepeer_client/src/node_detail_page.dart';
import 'package:mxonepeer_client/src/node_list_page.dart';

void main() {
  testWidgets('node list shows peer and detail error keeps list', (tester) async {
    final nodes = [
      TailNode(
        hostname: 'peer',
        dnsName: 'peer.tailnet.ts.net',
        tailscaleIPs: const ['100.64.0.2'],
        online: false,
      ),
    ];

    await tester.pumpWidget(MaterialApp(
      home: Builder(
        builder: (context) {
          return NodeListPage(
            nodes: nodes,
            onOpen: (node) {
              Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => NodeDetailPage(
                    ip: node.tailscaleIPs.first,
                    loader: (_) async {
                      throw Exception('fail');
                    },
                  ),
                ),
              );
            },
          );
        },
      ),
    ));

    expect(find.text('peer'), findsOneWidget);
    expect(find.text('离线'), findsOneWidget);
    expect(find.text('100.64.0.2'), findsOneWidget);

    await tester.tap(find.text('peer'));
    await tester.pumpAndSettle();

    expect(find.text('读不到状态'), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(find.text('peer'), findsOneWidget);
  });

  testWidgets('self node shows hostname and 本机 as separate texts', (tester) async {
    await tester.pumpWidget(MaterialApp(
      home: NodeListPage(
        nodes: [
          TailNode(
            hostname: 'verylonghostname-that-should-ellipsize',
            dnsName: 'mac.tailnet.ts.net',
            tailscaleIPs: const ['100.64.0.1'],
            online: true,
            self: true,
          ),
        ],
        onOpen: (_) {},
      ),
    ));

    expect(
      find.text('verylonghostname-that-should-ellipsize'),
      findsOneWidget,
    );
    expect(find.text('本机'), findsOneWidget);
    expect(
      find.text('verylonghostname-that-should-ellipsize本机'),
      findsNothing,
    );
  });
}
