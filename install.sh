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
CHECKSUM_URL="${URL}.sha256"

info "detected: linux/${ARCH}"
info "downloading $ASSET..."

# download
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
TMP="${TMP_DIR}/${ASSET}"
CHECKSUM="${TMP_DIR}/${ASSET}.sha256"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMP"
  curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP" "$URL"
  wget -qO "$CHECKSUM" "$CHECKSUM_URL"
else
  fail "curl or wget required"
fi

if ! command -v sha256sum >/dev/null 2>&1; then
  fail "sha256sum required to verify download"
fi

info "verifying checksum..."
(cd "$TMP_DIR" && sha256sum -c "${ASSET}.sha256") >/dev/null

mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP" "${INSTALL_DIR}/${BINARY}"

if ! "${INSTALL_DIR}/${BINARY}" --version >/dev/null 2>&1; then
  fail "installed binary did not run correctly"
fi

ok "sur installed to ${INSTALL_DIR}/${BINARY}"
info "run: sur check"
