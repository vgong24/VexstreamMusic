#!/bin/bash
# VexStream Music V4 — human-first macOS source launcher
# No Git, Python, or preinstalled Go is required.
set -eu

SOURCE_REF="${1:-main}"
REPOSITORY="vgong24/VexstreamMusic"
GO_VERSION="1.23.2"

case "$(/usr/bin/uname -m)" in
  arm64)
    GO_ARCH="arm64"
    GO_ARCHIVE="go${GO_VERSION}.darwin-arm64.tar.gz"
    GO_SHA256="d87031194fe3e01abdcaf3c7302148ade97a7add6eac3fec26765bcb3207b80f"
    ;;
  x86_64)
    GO_ARCH="amd64"
    GO_ARCHIVE="go${GO_VERSION}.darwin-amd64.tar.gz"
    GO_SHA256="445c0ef19d8692283f4c3a92052cc0568f5a048f4e546105f58e991d4aea54f5"
    ;;
  *)
    printf 'VexStream does not currently have a macOS build route for architecture: %s\n' "$(/usr/bin/uname -m)" >&2
    exit 2
    ;;
esac

API_REF="$(printf '%s' "$SOURCE_REF" | /usr/bin/sed 's#/#%2F#g')"
COMMIT="$(/usr/bin/curl -fsSL -H 'User-Agent: VexStream-Music-human-launcher' "https://api.github.com/repos/${REPOSITORY}/commits/${API_REF}" \
  | /usr/bin/sed -n 's/^[[:space:]]*"sha":[[:space:]]*"\([0-9a-f][0-9a-f]*\)".*/\1/p' \
  | /usr/bin/head -n 1)"
if ! printf '%s' "$COMMIT" | /usr/bin/grep -Eq '^[0-9a-f]{40}$'; then
  printf "GitHub did not return an exact commit for '%s'.\n" "$SOURCE_REF" >&2
  exit 3
fi

APP_SUPPORT="$HOME/Library/Application Support/VexStreamMusic"
INSTALL_ROOT="$APP_SUPPORT/source-builds/$COMMIT"
BINARY="$INSTALL_ROOT/VexStreamMusic-4.0.0-macOS-$GO_ARCH"
RECEIPT="$INSTALL_ROOT/BUILD-INFO.txt"
GO_ROOT="$APP_SUPPORT/toolchains/go${GO_VERSION}-darwin-${GO_ARCH}"
GO="$GO_ROOT/go/bin/go"
GOFMT="$GO_ROOT/go/bin/gofmt"

/bin/mkdir -p "$APP_SUPPORT/toolchains" "$INSTALL_ROOT"

if [ -x "$BINARY" ] && [ -f "$RECEIPT" ]; then
  RECEIPT_COMMIT="$(/usr/bin/sed -n 's/^sourceCommit=//p' "$RECEIPT" | /usr/bin/head -n 1)"
  RECEIPT_SHA="$(/usr/bin/sed -n 's/^binarySha256=//p' "$RECEIPT" | /usr/bin/head -n 1)"
  OBSERVED_SHA="$(/usr/bin/shasum -a 256 "$BINARY" | /usr/bin/awk '{print $1}')"
  if [ "$RECEIPT_COMMIT" = "$COMMIT" ] && [ "$RECEIPT_SHA" = "$OBSERVED_SHA" ]; then
    printf 'VexStream: launching cached V4 build from %s\n' "$COMMIT"
    exec "$BINARY"
  fi
fi

if [ ! -x "$GO" ]; then
  TMP_GO="$(/usr/bin/mktemp -d "${TMPDIR:-/tmp}/vexstream-go.XXXXXX")"
  trap '/bin/rm -rf "$TMP_GO"' EXIT
  printf 'VexStream: obtaining private Go %s build tool...\n' "$GO_VERSION"
  /usr/bin/curl -fL "https://go.dev/dl/$GO_ARCHIVE" -o "$TMP_GO/$GO_ARCHIVE"
  OBSERVED_GO_SHA="$(/usr/bin/shasum -a 256 "$TMP_GO/$GO_ARCHIVE" | /usr/bin/awk '{print $1}')"
  if [ "$OBSERVED_GO_SHA" != "$GO_SHA256" ]; then
    printf 'Go archive digest mismatch: %s\n' "$OBSERVED_GO_SHA" >&2
    exit 4
  fi
  /bin/rm -rf "$GO_ROOT"
  /bin/mkdir -p "$GO_ROOT"
  /usr/bin/tar -xzf "$TMP_GO/$GO_ARCHIVE" -C "$GO_ROOT"
  /bin/rm -rf "$TMP_GO"
  trap - EXIT
fi

if [ ! -x "$GO" ]; then
  printf 'Private Go toolchain formation failed.\n' >&2
  exit 5
fi

WORK="$(/usr/bin/mktemp -d "${TMPDIR:-/tmp}/vexstream-v4-source.XXXXXX")"
trap '/bin/rm -rf "$WORK"' EXIT

printf 'VexStream: downloading exact source %s...\n' "$COMMIT"
/usr/bin/curl -fL -H 'User-Agent: VexStream-Music-human-launcher' "https://github.com/${REPOSITORY}/archive/${COMMIT}.zip" -o "$WORK/source.zip"
/usr/bin/ditto -x -k "$WORK/source.zip" "$WORK/source"
SOURCE_ROOT="$(/usr/bin/find "$WORK/source" -mindepth 1 -maxdepth 1 -type d | /usr/bin/head -n 1)"
if [ -z "$SOURCE_ROOT" ]; then
  printf 'VexStream source archive did not contain a repository root.\n' >&2
  exit 6
fi

FORMAT_LIST="$WORK/gofmt-files.txt"
/usr/bin/find "$SOURCE_ROOT/v2" -type f -name '*.go' -print > "$FORMAT_LIST"
FORMAT_DELTA="$("$GOFMT" -d $(/bin/cat "$FORMAT_LIST"))"
if [ -n "$FORMAT_DELTA" ]; then
  printf '%s\n' "$FORMAT_DELTA" >&2
  printf 'V4 source is not gofmt-clean and will not be built by the human launcher.\n' >&2
  exit 7
fi

printf 'VexStream: checking V4 source...\n'
(
  cd "$SOURCE_ROOT"
  "$GO" test ./v2/...
  GOOS=darwin GOARCH="$GO_ARCH" CGO_ENABLED=0 "$GO" vet ./v2/...
)

FORMING="$INSTALL_ROOT/VexStreamMusic-4.0.0.forming"
printf 'VexStream: building the macOS app...\n'
(
  cd "$SOURCE_ROOT"
  GOOS=darwin GOARCH="$GO_ARCH" CGO_ENABLED=0 "$GO" build \
    -trimpath \
    -ldflags "-s -w -X main.buildRef=$COMMIT" \
    -o "$FORMING" \
    ./v2/cmd/vexstream-v4
)
/bin/mv -f "$FORMING" "$BINARY"
/bin/chmod u+x "$BINARY"
/usr/bin/xattr -d com.apple.quarantine "$BINARY" 2>/dev/null || true
BINARY_SHA="$(/usr/bin/shasum -a 256 "$BINARY" | /usr/bin/awk '{print $1}')"
{
  printf 'schemaVersion=vexstream.human-source-build/v1\n'
  printf 'appVersion=4.0.0\n'
  printf 'sourceRef=%s\n' "$SOURCE_REF"
  printf 'sourceCommit=%s\n' "$COMMIT"
  printf 'goVersion=go%s\n' "$GO_VERSION"
  printf 'platform=darwin-%s\n' "$GO_ARCH"
  printf 'binarySha256=%s\n' "$BINARY_SHA"
} > "$RECEIPT"

/bin/rm -rf "$WORK"
trap - EXIT

printf 'VexStream: ready.\n'
printf 'Source: %s\n' "$COMMIT"
printf 'App:    %s\n' "$BINARY"
printf 'Keep this Terminal window open while VexStream is running. Use System status -> Exit VexStream to stop it cleanly.\n'
exec "$BINARY"
