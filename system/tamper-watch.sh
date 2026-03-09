#!/usr/bin/env bash
set -euo pipefail

HOSTS_FILE="/etc/hosts"
MARKER_START="# ===== FMHY BLOCKLIST START ====="
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APPLY_SCRIPT="$SCRIPT_DIR/apply-hosts.sh"
LOG_TAG="fmhy-tamper-watch"
TAMPER_LOG="/tmp/fmhy-tamper-watch.log"

log() {
  local msg="[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"
  logger -t "$LOG_TAG" "$*" 2>/dev/null
  echo "$msg" >> "$TAMPER_LOG"
}

if ! grep -q "$MARKER_START" "$HOSTS_FILE" 2>/dev/null; then
  log "TAMPER DETECTED: FMHY blocklist section missing from $HOSTS_FILE"
  log "Re-applying blocklist..."

  if bash "$APPLY_SCRIPT"; then
    log "Blocklist re-applied successfully"
  else
    log "ERROR: Failed to re-apply blocklist"
    exit 1
  fi

  REAL_USER="${SUDO_USER:-$USER}"
  if [[ "$(uname -s)" == "Darwin" ]] && command -v osascript &>/dev/null; then
    sudo -u "$REAL_USER" osascript -e '
      display notification "The FMHY blocklist was removed from /etc/hosts and has been restored." with title "Binge Blocker" subtitle "Tamper detected"
    ' 2>/dev/null || true
  fi
else
  :
fi

if [[ -f "$TAMPER_LOG" ]] && [[ $(stat -f%z "$TAMPER_LOG" 2>/dev/null || stat --format=%s "$TAMPER_LOG" 2>/dev/null || echo 0) -gt 1048576 ]]; then
  tail -100 "$TAMPER_LOG" > "$TAMPER_LOG.tmp"
  mv "$TAMPER_LOG.tmp" "$TAMPER_LOG"
fi
