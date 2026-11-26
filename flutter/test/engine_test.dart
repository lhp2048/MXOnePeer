import 'package:flutter_test/flutter_test.dart';
import 'package:mxonepeer_client/src/engine.dart';

void main() {
  test('arch and reuse decision', () {
    expect(engineArch('arm64'), 'arm64');
    expect(engineArch('x86_64'), 'amd64');
    expect(chooseEngine(200, 'mxonepeer-client'), EngineChoice.reuse);
    expect(chooseEngine(null, null), EngineChoice.spawn);
    expect(chooseEngine(200, 'other'), EngineChoice.busy);
  });
}
