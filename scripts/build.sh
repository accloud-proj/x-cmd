#!/usr/bin/env sh
set -eu

VERSION="${1:-dev}"
VERSION="${VERSION#v}"
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
OUTPUT_DIR="$ROOT/dist/linux-amd64"

cd "$ROOT"
GO_PLATFORM="$(go env GOOS)/$(go env GOARCH)"
if [ "$GO_PLATFORM" != "linux/amd64" ]; then
  echo "This script requires linux/amd64; current Go platform is $GO_PLATFORM" >&2
  exit 1
fi

echo "Running tests..."
go test ./...

mkdir -p "$OUTPUT_DIR"
echo "Building linux/amd64..."
go build \
  -trimpath \
  -ldflags "-s -w -X github.com/accloud-proj/x-cmd/internal/version.Version=$VERSION" \
  -o "$OUTPUT_DIR/x-cmd" .

echo "Build complete: $OUTPUT_DIR"