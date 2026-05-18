#!/bin/sh
set -e

REPO="suleymanmercan/sur"
BINARY="sur"
INSTALL_DIR="/usr/local/bin"

# colors
RED='\033[0;31m'
GREEN='\033[0;32m'
DIM='\033[0;90m'
RESET='\033[0m'

info()  { printf "${DIM}→${RESET} %s\n" "$1"; }
ok()    { printf "${GREEN}✓${RESET} %s\n" "$1"; }
fail()  { printf "${RED}✗${RESET} %s\n" "$1"; exit 1; }

# root check
if [ "$(id -u)" -ne 0 ]; then
  fail "please run with sudo: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sudo bash"
fi

# detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       fail "unsupported architecture: $ARCH" ;;
esac

# detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
  fail "sur only supports Linux (detected: $OS)"
fi

ASSET="sur-linux-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

info "detected: linux/${ARCH}"
info "downloading $ASSET..."

# download
TMP=$(mktemp)
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP" "$URL"
else
  fail "curl or wget required"
fi

chmod +x "$TMP"
mv "$TMP" "${INSTALL_DIR}/${BINARY}"

ok "sur installed to ${INSTALL_DIR}/${BINARY}"
info "run: sur check"