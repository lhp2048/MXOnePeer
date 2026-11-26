#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
mkdir -p dist
build() {
  echo "build $1/$2 $4"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w" -o "dist/$4-$1-$2$3" "./cmd/$4"
}
build darwin amd64 "" mxonepeer
build darwin arm64 "" mxonepeer
build linux amd64 "" mxonepeer
build windows amd64 ".exe" mxonepeer
build darwin amd64 "" mxonepeer-client
build darwin arm64 "" mxonepeer-client
