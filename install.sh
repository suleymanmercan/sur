#!/bin/sh
set -e

REPO="suleymanmercan/sur"
BINARY="sur"
INSTALL_DIR="/usr/local/bin"
LEGACY_TASK_DIR="/etc/sur"
STATE_DIR="/var/lib/sur"

# colors
RED='\033[0;31m'
GREEN='\033[0;32m'
DIM='\033[0;90m'
RESET='\033[0m'

info()  { printf "${DIM}→${RESET} %s\n" "$1"; }
ok()    { printf "${GREEN}✓${RESET} %s\n" "$1"; }
fail()  { printf "${RED}✗${RESET} %s\n" "$1"; exit 1; }

download() {
  url="$1"
  out="$2"
  label="$3"

  if command -v curl >/dev/null 2>&1; then
    if ! curl -fsSL "$url" -o "$out"; then
      fail "could not download ${label}: ${url}"
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -qO "$out" "$url"; then
      fail "could not download ${label}: ${url}"
    fi
  else
    fail "curl or wget required"
  fi
}

usage() {
  cat <<EOF
sur installer

Usage:
  install.sh
  install.sh --uninstall
  install.sh --uninstall --purge

Options:
  --uninstall  remove the installed sur binary
  --purge      with --uninstall, also remove ${LEGACY_TASK_DIR} and ${STATE_DIR}
EOF
}

UNINSTALL=0
PURGE=0
for arg in "$@"; do
  case "$arg" in
    --uninstall) UNINSTALL=1 ;;
    --purge) PURGE=1 ;;
    -h|--help) usage; exit 0 ;;
    *) fail "unknown option: $arg" ;;
  esac
done

# root check
if [ "$(id -u)" -ne 0 ]; then
  fail "please run with sudo: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sudo bash"
fi

if [ "$UNINSTALL" -eq 1 ]; then
  info "uninstalling sur..."
  rm -f "${INSTALL_DIR}/${BINARY}"
  if [ "$PURGE" -eq 1 ]; then
    rm -rf "${LEGACY_TASK_DIR}"
    rm -rf "${STATE_DIR}"
    ok "sur removed, including ${LEGACY_TASK_DIR} and ${STATE_DIR}"
  else
    ok "sur binary removed"
    info "kept ${LEGACY_TASK_DIR} and ${STATE_DIR}"
    info "run with --uninstall --purge to remove them too"
  fi
  exit 0
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

download "$URL" "$TMP" "$ASSET"
download "$CHECKSUM_URL" "$CHECKSUM" "${ASSET}.sha256"

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
