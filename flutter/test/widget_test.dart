import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mxonepeer_client/main.dart';

void main() {
  testWidgets('app shell does not start the engine', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MxonepeerApp(
          home: Scaffold(body: Text('MXOnePeer')),
        ),
      ),
    );

    expect(find.text('MXOnePeer'), findsOneWidget);
    final theme = Theme.of(tester.element(find.text('MXOnePeer')));
    expect(theme.useMaterial3, isTrue);
    expect(theme.colorScheme.primary, mxonepeerTheme().colorScheme.primary);
    expect(find.text('0'), findsNothing);
  });
}
