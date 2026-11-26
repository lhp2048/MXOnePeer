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

  test('spawn wait stays blocked when health never becomes this client', () {
    expect(spawnWaitBlockMessage(EngineChoice.reuse, false), isNull);
    expect(spawnWaitBlockMessage(EngineChoice.busy, false), '端口被占用');
    expect(spawnWaitBlockMessage(EngineChoice.spawn, true), '端口被占用');
    expect(spawnWaitBlockMessage(EngineChoice.spawn, false), '引擎没有启动');
  });
}
