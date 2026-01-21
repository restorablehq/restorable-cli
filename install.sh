#!/usr/bin/env sh
set -e

BIN_NAME="restorable"
INSTALL_DIR="/usr/local/bin"
GITHUB_REPO="restorablehq/restorable-cli"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

BIN_FILE="${BIN_NAME}-${OS}-${ARCH}"
BIN_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BIN_FILE}"
SHA_URL="${BIN_URL}.sha256"

TMP_BIN="/tmp/${BIN_NAME}"
TMP_SHA="/tmp/${BIN_NAME}.sha256"

echo "Installing ${BIN_NAME} (${OS}/${ARCH})..."

curl -fsSL "$BIN_URL" -o "$TMP_BIN"
curl -fsSL "$SHA_URL" -o "$TMP_SHA"

# Verify checksum (works on Linux + macOS)
if command -v sha256sum >/dev/null 2>&1; then
  echo "$(cat "$TMP_SHA")  $TMP_BIN" | sha256sum -c -
elif command -v shasum >/dev/null 2>&1; then
  echo "$(cat "$TMP_SHA")  $TMP_BIN" | shasum -a 256 -c -
else
  echo "No SHA256 checksum tool found"
  exit 1
fi

chmod +x "$TMP_BIN"

if [ ! -w "$INSTALL_DIR" ]; then
  sudo mv "$TMP_BIN" "$INSTALL_DIR/${BIN_NAME}"
else
  mv "$TMP_BIN" "$INSTALL_DIR/${BIN_NAME}"
fi

echo "✓ ${BIN_NAME} installed successfully"

