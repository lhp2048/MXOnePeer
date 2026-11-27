import 'package:flutter_test/flutter_test.dart';
import 'package:mxonepeer_client/src/upgrade_name.dart';

void main() {
  test('filename matches runtime', () {
    expect(upgradeFilenameOk('go/linux/amd64', 'mxonepeer-linux-amd64'), isTrue);
    expect(upgradeFilenameOk('go/windows/amd64', 'mxonepeer-windows-amd64.exe'), isTrue);
    expect(upgradeFilenameOk('python/linux/amd64', 'mxonepeer-python-linux-amd64.zip'), isTrue);
    expect(upgradeFilenameOk('go/linux/amd64', 'mxonepeer-darwin-arm64'), isFalse);
    expect(canUpgradeRuntime('go/linux/amd64'), isTrue);
    expect(canUpgradeRuntime('python/windows/amd64'), isTrue);
    expect(canUpgradeRuntime('client/darwin/arm64'), isFalse);
    expect(canUpgradeRuntime(''), isFalse);
  });
}
