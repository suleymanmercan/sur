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
  install.sh --update
  install.sh --uninstall
  install.sh --uninstall --purge

Options:
  --update     download and replace the installed binary with the latest release
  --uninstall  remove the installed sur binary
  --purge      with --uninstall, also remove ${LEGACY_TASK_DIR} and ${STATE_DIR}
EOF
}

UPDATE=0
UNINSTALL=0
PURGE=0
for arg in "$@"; do
  case "$arg" in
    --update) UPDATE=1 ;;
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
case "$OS" in
  linux|darwin) ;;
  *) fail "unsupported OS: $OS" ;;
esac

# resolve latest version tag
info "resolving latest release..."
LATEST_URL="https://github.com/${REPO}/releases/latest"
VERSION=$(curl -fsSL -o /dev/null -w '%{url_effective}' "$LATEST_URL" 2>/dev/null | sed 's|.*/tag/v||') \
  || VERSION=$(wget -q --server-response --spider "$LATEST_URL" 2>&1 | grep Location | sed 's|.*/tag/v||' | tr -d ' \r')
if [ -z "$VERSION" ]; then
  fail "could not resolve latest version"
fi
info "latest version: v${VERSION}"

# GoReleaser archive format: sur_<version>_<os>_<arch>.tar.gz
ASSET="sur_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"
URL="${BASE_URL}/${ASSET}"
CHECKSUM_URL="${BASE_URL}/checksums.txt"

info "detected: ${OS}/${ARCH}"
if [ "$UPDATE" -eq 1 ] && [ -x "${INSTALL_DIR}/${BINARY}" ]; then
  info "current: $("${INSTALL_DIR}/${BINARY}" --version 2>/dev/null || printf 'unknown')"
fi
info "downloading ${ASSET}..."

# download to temp dir
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

TMP_ARCHIVE="${TMP_DIR}/${ASSET}"
TMP_CHECKSUMS="${TMP_DIR}/checksums.txt"

download "$URL"           "$TMP_ARCHIVE"   "$ASSET"
download "$CHECKSUM_URL"  "$TMP_CHECKSUMS" "checksums.txt"

# verify checksum
if command -v sha256sum >/dev/null 2>&1; then
  info "verifying checksum..."
  EXPECTED=$(grep "${ASSET}" "$TMP_CHECKSUMS" | awk '{print $1}')
  if [ -z "$EXPECTED" ]; then
    fail "checksum for ${ASSET} not found in checksums.txt"
  fi
  ACTUAL=$(sha256sum "$TMP_ARCHIVE" | awk '{print $1}')
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    fail "checksum mismatch! expected=${EXPECTED} got=${ACTUAL}"
  fi
  ok "checksum verified"
else
  info "sha256sum not found, skipping checksum verification"
fi

# extract binary from archive
info "extracting..."
tar -xzf "$TMP_ARCHIVE" -C "$TMP_DIR"

TMP_BINARY="${TMP_DIR}/${BINARY}"
if [ ! -f "$TMP_BINARY" ]; then
  fail "binary '${BINARY}' not found in archive"
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP_BINARY" "${INSTALL_DIR}/${BINARY}"

if ! "${INSTALL_DIR}/${BINARY}" --version >/dev/null 2>&1; then
  fail "installed binary did not run correctly"
fi

ok "sur installed to ${INSTALL_DIR}/${BINARY}"
info "version: $("${INSTALL_DIR}/${BINARY}" --version 2>/dev/null || printf 'unknown')"
info "run: sur check"
