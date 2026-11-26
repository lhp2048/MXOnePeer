import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mxonepeer_client/src/login_page.dart';
import 'package:mxonepeer_client/src/session.dart';

void main() {
  testWidgets('login page shows auth key and account login', (tester) async {
    await tester.pumpWidget(MaterialApp(
      home: LoginPage(
        session: ClientSession(
          backendState: 'NeedsLogin',
          online: false,
          loginUrl: 'https://login.tailscale.com/a/example',
          loginError: '',
        ),
        error: 'auth key 必须以 tskey-auth- 开头。',
        onSubmit: (_) async {},
        onLogin: () async {},
      ),
    ));
    expect(find.text('用 Tailscale 账号登录'), findsOneWidget);
    expect(find.text('https://login.tailscale.com/a/example'), findsOneWidget);
    expect(find.text('auth key 必须以 tskey-auth- 开头。'), findsOneWidget);
  });
}
