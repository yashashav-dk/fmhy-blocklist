#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

BLOCKLIST_URL="https://raw.githubusercontent.com/yashashav-dk/fmhy-blocklist/main/blocklist.txt"

info()  { echo -e "${GREEN}[+]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
step()  { echo -e "${CYAN}[>]${NC} $*"; }

echo ""
echo "============================================"
echo "  FMHY Binge Blocker — Library Mac Setup"
echo "  (No sudo required)"
echo "============================================"
echo ""

warn "This Mac has MDM restrictions. Available layers:"
echo "  [Layer 1] uBlock Origin in Brave — AVAILABLE"
echo "  [Layer 2] /etc/hosts            — NOT AVAILABLE (no sudo)"
echo "  [Layer 3] NextDNS               — PARTIAL (CLI needs sudo, but DNS profile works)"
echo ""

step "Disabling DNS-over-HTTPS in browsers (user-level policies)..."

BRAVE_POLICY_DIR="$HOME/Library/Application Support/BraveSoftware/Brave-Browser/policies/managed"
mkdir -p "$BRAVE_POLICY_DIR"
cat > "$BRAVE_POLICY_DIR/fmhy-doh-policy.json" <<'POLICY'
{
  "DnsOverHttpsMode": "off"
}
POLICY
info "DoH disabled for Brave (user-level policy)"

CHROME_POLICY_DIR="$HOME/Library/Application Support/Google/Chrome/policies/managed"
mkdir -p "$CHROME_POLICY_DIR"
cat > "$CHROME_POLICY_DIR/fmhy-doh-policy.json" <<'POLICY'
{
  "DnsOverHttpsMode": "off"
}
POLICY
info "DoH disabled for Chrome (user-level policy)"

echo ""
warn "Restart Brave and Chrome for DoH policy to take effect."

echo ""
step "Brave + uBlock Origin setup:"
echo ""
echo "  1. Open Brave"
echo "  2. Go to: brave://settings/extensions"
echo "  3. Under 'Manifest V2 Extensions', enable uBlock Origin"
echo "  4. Open uBlock Origin settings (click the extension icon > gear)"
echo "  5. Go to 'Filter lists' tab"
echo "  6. Scroll to bottom, click 'Import...'"
echo "  7. Paste this URL:"
echo ""
echo -e "     ${CYAN}${BLOCKLIST_URL}${NC}"
echo ""
echo "  8. Click 'Apply changes'"
echo ""
echo "  uBlock will auto-refresh this list every few hours."
echo ""

step "NextDNS (optional, recommended):"
echo ""
echo "  Even without sudo, you can use a NextDNS DNS profile:"
echo ""
echo "  1. Go to https://apple.nextdns.io"
echo "  2. Enter your NextDNS configuration ID"
echo "  3. Download and install the DNS profile"
echo "     (System Settings > Privacy & Security > Profiles)"
echo ""
echo "  This routes ALL DNS through NextDNS system-wide,"
echo "  no sudo or CLI needed."
echo ""

warn "Chrome on this Mac has almost zero blocking coverage."
echo "  MV3 kills real uBlock by April 2026 and you can't modify /etc/hosts."
echo "  Recommendation: don't use Chrome for anything on this machine."
echo ""

info "Library Mac setup complete."
echo ""
info "Active protection:"
echo "  - DoH disabled in Brave + Chrome (user-level policy)"
echo "  - uBlock Origin filter list (after manual Brave setup above)"
echo "  - NextDNS (after profile install above)"
echo ""
