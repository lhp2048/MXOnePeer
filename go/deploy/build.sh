#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
mkdir -p dist
build() {
  echo "build $1/$2"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w" -o "dist/mxonepeer-$1-$2$3" ./cmd/mxonepeer
}
build darwin amd64 ""
build darwin arm64 ""
build linux amd64 ""
build windows amd64 ".exe"
