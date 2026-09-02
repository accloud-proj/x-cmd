#!/usr/bin/env sh
set -eu

REPOSITORY="accloud-proj/x-cmd"
VERSION="latest"
INSTALL_DIR="${HOME}/.local/bin"
GITHUB_MIRROR=""
GITHUB_HOST=""

usage() {
  cat <<'EOF'
Install x-cmd from GitHub Releases.

Usage: install.sh [options]
  --version VERSION       Release version, for example v0.2.0 (default: latest)
  --install-dir DIR       Installation directory (default: ~/.local/bin)
  --github-mirror URL     Prefix for complete GitHub URLs, for example https://ghproxy.net
  --github-host HOST      Replace github.com while preserving the path, for example downloads.example.com
  -h, --help              Show this help
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:?missing value for --version}"; shift 2 ;;
    --install-dir) INSTALL_DIR="${2:?missing value for --install-dir}"; shift 2 ;;
    --github-mirror) GITHUB_MIRROR="${2:?missing value for --github-mirror}"; shift 2 ;;
    --github-host) GITHUB_HOST="${2:?missing value for --github-host}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [ -n "$GITHUB_MIRROR" ] && [ -n "$GITHUB_HOST" ]; then
  echo "--github-mirror and --github-host cannot be used together" >&2
  exit 2
fi

if [ -n "$GITHUB_HOST" ]; then
  case "$GITHUB_HOST" in
    http://*|https://*) ;;
    *) GITHUB_HOST="https://${GITHUB_HOST}" ;;
  esac
  GITHUB_HOST="${GITHUB_HOST%/}"
fi

case "$(uname -s)" in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) echo "Unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  armv7l|armv7) ARCH="arm" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

ASSET="x-cmd_${OS}_${ARCH}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  RELEASE_PATH="latest/download"
else
  case "$VERSION" in v*) ;; *) VERSION="v${VERSION}" ;; esac
  RELEASE_PATH="download/${VERSION}"
fi

github_url() {
  target="https://github.com/${REPOSITORY}/releases/${RELEASE_PATH}/$1"
  if [ -n "$GITHUB_MIRROR" ]; then
    printf '%s/%s' "${GITHUB_MIRROR%/}" "$target"
  elif [ -n "$GITHUB_HOST" ]; then
    printf '%s/%s/releases/%s/%s' "$GITHUB_HOST" "$REPOSITORY" "$RELEASE_PATH" "$1"
  else
    printf '%s' "$target"
  fi
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

download() {
  url="$1"
  destination="$2"
  if command -v curl >/dev/null 2>&1; then
    curl --fail --location --retry 3 --output "$destination" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget --tries=3 --output-document="$destination" "$url"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

echo "Downloading ${ASSET}..."
download "$(github_url "$ASSET")" "$TMP_DIR/$ASSET"
download "$(github_url checksums.txt)" "$TMP_DIR/checksums.txt"

EXPECTED="$(awk -v asset="$ASSET" '$2 == asset || $2 == "*" asset { print $1; exit }' "$TMP_DIR/checksums.txt")"
if [ -z "$EXPECTED" ]; then
  echo "No checksum found for ${ASSET}" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$TMP_DIR/$ASSET" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$TMP_DIR/$ASSET" | awk '{print $1}')"
fi
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum verification failed for ${ASSET}" >&2
  exit 1
fi

tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP_DIR/x-cmd" "$INSTALL_DIR/x-cmd"
if [ -n "$GITHUB_MIRROR" ]; then
  "$INSTALL_DIR/x-cmd" config set --github-mirror "$GITHUB_MIRROR"
elif [ -n "$GITHUB_HOST" ]; then
  "$INSTALL_DIR/x-cmd" config set --github-host "$GITHUB_HOST"
fi
echo "Installed x-cmd to ${INSTALL_DIR}/x-cmd"
case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *) echo "Add ${INSTALL_DIR} to PATH to run x-cmd directly." ;; esac