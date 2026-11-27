bool canUpgradeRuntime(String runtime) {
  return runtime.startsWith('go/') || runtime.startsWith('python/');
}

bool upgradeFilenameOk(String runtime, String name) {
  final parts = runtime.split('/');
  if (parts.length != 3 || parts[2].isEmpty || parts[2] == 'unknown') {
    return false;
  }
  if (name.contains('/') || name.contains('\\')) {
    return false;
  }
  final osName = parts[1];
  final arch = parts[2];
  if (parts[0] == 'go') {
    final base = 'mxonepeer-$osName-$arch';
    return osName == 'windows' ? name == '$base.exe' : name == base;
  }
  if (parts[0] == 'python') {
    return name == 'mxonepeer-python-$osName-$arch.zip';
  }
  return false;
}
