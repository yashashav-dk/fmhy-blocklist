#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

DOMAINS_URL="https://raw.githubusercontent.com/yashashav-dk/fmhy-blocklist/main/domains.txt"

info()  { echo -e "${GREEN}[+]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
step()  { echo -e "${CYAN}[>]${NC} $*"; }

echo ""
echo "========================================"
echo "  FMHY Binge Blocker — NextDNS Setup"
echo "========================================"
echo ""

step "STEP 1: Create your NextDNS configuration"
echo ""
echo "  1. Go to https://my.nextdns.io/signup"
echo "  2. Create an account (free tier = 300k queries/month)"
echo "  3. In your configuration, go to the 'Denylist' tab"
echo "  4. Click 'Add a blocklist'"
echo "  5. Paste this URL:"
echo ""
echo -e "     ${CYAN}${DOMAINS_URL}${NC}"
echo ""
echo "  6. Save the configuration"
echo "  7. Note your configuration ID (looks like: abc123)"
echo ""
read -rp "Press Enter when done, or Ctrl+C to exit..."

echo ""
step "STEP 2: Install NextDNS CLI (recommended)"
echo ""

if command -v nextdns &>/dev/null; then
  info "NextDNS CLI already installed: $(nextdns version)"
else
  echo "  The NextDNS CLI routes all DNS through NextDNS automatically."
  echo "  Install it with:"
  echo ""
  echo "    sh -c \"\$(curl -fsSL https://nextdns.io/install)\""
  echo ""
  read -rp "Install now? [y/N] " INSTALL_CLI
  if [[ "${INSTALL_CLI,,}" == "y" ]]; then
    sh -c "$(curl -fsSL https://nextdns.io/install)" || {
      warn "CLI install failed. You can still configure DNS manually (Step 3)."
    }
  fi
fi

echo ""
step "STEP 3: Point your Mac's DNS to NextDNS"
echo ""

if command -v nextdns &>/dev/null; then
  echo "  Since you have the CLI, run:"
  echo ""
  echo "    sudo nextdns install -config YOUR_CONFIG_ID -report-client-info"
  echo ""
  echo "  Replace YOUR_CONFIG_ID with your actual ID from Step 1."
else
  echo "  Without the CLI, set DNS manually:"
  echo ""
  echo "  macOS: System Settings > Network > Wi-Fi > Details > DNS"
  echo "    Remove existing entries, add:"
  echo "      Primary:   Your NextDNS IPv4 (shown on setup page)"
  echo "      Secondary: Your NextDNS IPv6 (shown on setup page)"
  echo ""
  echo "  Or use the profile download from https://apple.nextdns.io"
  echo "  (this installs a DNS profile that persists across networks)"
fi

echo ""
step "STEP 4: Lock yourself out (the whole point)"
echo ""
echo -e "  ${YELLOW}This is the most important step.${NC}"
echo ""
echo "  1. Have a trusted friend set the NextDNS account password"
echo "  2. They should also set up 2FA on the account"
echo "  3. You should NOT know the password or have the 2FA device"
echo ""
echo "  This way you cannot:"
echo "    - Remove domains from the denylist"
echo "    - Disable NextDNS filtering"
echo "    - Delete the configuration"
echo ""
echo "  The blocklist auto-updates via GitHub — no account access needed."
echo ""

info "NextDNS setup guide complete."
info "Denylist URL for reference:"
echo "  $DOMAINS_URL"
echo ""
warn "Remember: the auto-update pipeline keeps domains.txt current."
warn "NextDNS re-fetches the list periodically — no manual maintenance needed."
echo ""
