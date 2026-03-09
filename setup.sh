#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY_NAME="fmhy-blocklist"
INSTALL_DIR="/usr/local/bin"
GO_VERSION="1.22.4"
MIN_GO_MAJOR=1
MIN_GO_MINOR=21

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
error() { echo -e "${RED}[x]${NC} $*"; exit 1; }

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin) PLATFORM="darwin" ;;
  Linux)  PLATFORM="linux" ;;
  *)      error "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *)       error "Unsupported architecture: $ARCH" ;;
esac

if [[ $EUID -ne 0 ]]; then
  warn "Re-running with sudo..."
  exec sudo bash "$0" "$@"
fi

info "Platform: $PLATFORM/$GOARCH"

check_go_version() {
  if ! command -v go &>/dev/null; then
    return 1
  fi
  local ver
  ver="$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')"
  local major minor
  major="$(echo "$ver" | cut -d. -f1)"
  minor="$(echo "$ver" | cut -d. -f2)"
  [[ "$major" -gt "$MIN_GO_MAJOR" ]] || { [[ "$major" -eq "$MIN_GO_MAJOR" ]] && [[ "$minor" -ge "$MIN_GO_MINOR" ]]; }
}

if ! check_go_version; then
  warn "Go >= ${MIN_GO_MAJOR}.${MIN_GO_MINOR} not found. Installing Go ${GO_VERSION}..."
  GO_TAR="go${GO_VERSION}.${PLATFORM}-${GOARCH}.tar.gz"
  GO_URL="https://go.dev/dl/${GO_TAR}"
  TMP_DIR="$(mktemp -d)"
  curl -fsSL "$GO_URL" -o "$TMP_DIR/$GO_TAR"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$TMP_DIR/$GO_TAR"
  rm -rf "$TMP_DIR"
  export PATH="/usr/local/go/bin:$PATH"
  info "Go ${GO_VERSION} installed"
else
  info "Go found: $(go version)"
fi

cd "$REPO_DIR"
info "Building ${BINARY_NAME}..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BINARY_NAME" .
info "Binary built: $(du -h "$BINARY_NAME" | cut -f1) stripped"

cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"
info "Installed to $INSTALL_DIR/$BINARY_NAME"

info "Running initial scrape..."
cd "$REPO_DIR"
"$INSTALL_DIR/$BINARY_NAME"

# ── Disable DoH in Chrome and Brave ──

disable_doh() {
  local browser_name="$1"
  local policy_dir="$2"
  mkdir -p "$policy_dir"
  cat > "$policy_dir/fmhy-doh-policy.json" <<'POLICY'
{
  "DnsOverHttpsMode": "off"
}
POLICY
  info "DoH disabled for $browser_name via managed policy"
}

if [[ "$PLATFORM" == "darwin" ]]; then
  CHROME_POLICY_DIR="/Library/Managed Preferences/com.google.Chrome"
  BRAVE_POLICY_DIR="/Library/Managed Preferences/com.brave.Browser"

  REAL_USER="${SUDO_USER:-$USER}"
  REAL_HOME="$(eval echo ~"$REAL_USER")"
  CHROME_USER_POLICY_DIR="$REAL_HOME/Library/Application Support/Google/Chrome/policies/managed"
  BRAVE_USER_POLICY_DIR="$REAL_HOME/Library/Application Support/BraveSoftware/Brave-Browser/policies/managed"

  disable_doh "Chrome (system)" "$CHROME_POLICY_DIR"
  disable_doh "Brave (system)" "$BRAVE_POLICY_DIR"
  disable_doh "Chrome (user)" "$CHROME_USER_POLICY_DIR"
  disable_doh "Brave (user)" "$BRAVE_USER_POLICY_DIR"

  chown -R "$REAL_USER" "$CHROME_USER_POLICY_DIR" 2>/dev/null || true
  chown -R "$REAL_USER" "$BRAVE_USER_POLICY_DIR" 2>/dev/null || true
fi

# ── Platform-specific setup ──

if [[ "$PLATFORM" == "darwin" ]]; then
  info "Applying /etc/hosts blocklist..."
  bash "$REPO_DIR/system/apply-hosts.sh"

  PLIST_SRC="$REPO_DIR/system/com.fmhy.blocklist.plist"
  PLIST_DST="/Library/LaunchDaemons/com.fmhy.blocklist.plist"
  sed "s|/path/to/apply-hosts.sh|${REPO_DIR}/system/apply-hosts.sh|g" "$PLIST_SRC" > "$PLIST_DST"
  chmod 644 "$PLIST_DST"
  chown root:wheel "$PLIST_DST"
  launchctl unload "$PLIST_DST" 2>/dev/null || true
  launchctl load -w "$PLIST_DST"
  info "launchd daemon installed: com.fmhy.blocklist"

  TAMPER_PLIST_SRC="$REPO_DIR/system/com.fmhy.tamper-watch.plist"
  TAMPER_PLIST_DST="/Library/LaunchDaemons/com.fmhy.tamper-watch.plist"
  sed "s|/path/to/tamper-watch.sh|${REPO_DIR}/system/tamper-watch.sh|g" "$TAMPER_PLIST_SRC" > "$TAMPER_PLIST_DST"
  chmod 644 "$TAMPER_PLIST_DST"
  chown root:wheel "$TAMPER_PLIST_DST"
  launchctl unload "$TAMPER_PLIST_DST" 2>/dev/null || true
  launchctl load -w "$TAMPER_PLIST_DST"
  info "launchd daemon installed: com.fmhy.tamper-watch"

elif [[ "$PLATFORM" == "linux" ]]; then
  info "Applying /etc/hosts blocklist..."
  bash "$REPO_DIR/system/apply-hosts.sh"

  CRON_FILE="/etc/cron.d/fmhy-blocklist"
  echo "15 3 * * * root bash ${REPO_DIR}/system/apply-hosts.sh" > "$CRON_FILE"
  chmod 644 "$CRON_FILE"
  info "Cron job installed: $CRON_FILE"
fi

echo ""
info "========================================="
info "  FMHY Binge Blocker — Setup Complete"
info "========================================="
echo ""
info "Layers active:"
info "  [Layer 2] /etc/hosts blocklist applied"
info "  [Layer 2] Auto-updater scheduled (03:15 UTC daily)"
info "  [Layer 4] Tamper detection daemon running"
if [[ "$PLATFORM" == "darwin" ]]; then
  info "  [DoH]     Disabled in Chrome + Brave via managed policy"
fi
echo ""
warn "Manual steps remaining:"
echo "  1. BRAVE: Settings > Extensions > enable uBlock Origin (MV2)"
echo "     Subscribe filter: https://raw.githubusercontent.com/yashashav-dk/fmhy-blocklist/main/blocklist.txt"
echo ""
echo "  2. NEXTDNS: Set up account at https://nextdns.io"
echo "     Add denylist: https://raw.githubusercontent.com/yashashav-dk/fmhy-blocklist/main/domains.txt"
echo "     Point Mac DNS to NextDNS, then give password to a friend"
echo "     Or run: bash system/setup-nextdns.sh"
echo ""
echo "  3. VERIFY: curl -v http://fmovies.ps (should fail to resolve)"
echo "     sudo launchctl list | grep fmhy (should show both daemons)"
echo ""
