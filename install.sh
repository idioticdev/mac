#!/usr/bin/env bash
# ============================================================================
# install.sh — One-liner bootstrap for mac
# ============================================================================
# Usage (fresh Mac):
#   curl -fsSL https://mac.idiotic.dev/install.sh | bash
#
# Headless (skip config prompt):
#   CONFIG_URL=https://raw.githubusercontent.com/you/dotfiles/main/mac.toml \
#     curl -fsSL https://mac.idiotic.dev/install.sh | bash
#
# What this does:
#   1. Installs Xcode Command Line Tools (if missing)
#   2. Installs Homebrew (if missing)
#   3. Detects architecture (arm64 / amd64)
#   4. Downloads the latest pre-built mac binary from GitHub Releases
#   5. Installs it to /usr/local/bin
#   6. Prompts for your mac.toml URL (or generates a starter config) and runs apply
# ============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()  { printf "  ${CYAN}→${NC} %s\n" "$1"; }
ok()    { printf "  ${GREEN}✓${NC} %s\n" "$1"; }
warn()  { printf "  ${YELLOW}!${NC} %s\n" "$1"; }
fail()  { printf "  ${RED}✗${NC} %s\n" "$1"; exit 1; }

# ---- Sudo: authenticate once, keep alive, revoke on exit ------------------
_SUDO_KEEPALIVE_PID=""
_sudo_cleanup() {
    [[ -n "$_SUDO_KEEPALIVE_PID" ]] && kill "$_SUDO_KEEPALIVE_PID" 2>/dev/null || true
    sudo -k  # revoke cached credentials
}
trap _sudo_cleanup EXIT

info "This script needs sudo access for Homebrew and binary install."
sudo -v < /dev/tty
# Refresh the credential every 50s so it doesn't expire mid-run
(while true; do sudo -n true 2>/dev/null; sleep 50; done) &
_SUDO_KEEPALIVE_PID=$!
disown "$_SUDO_KEEPALIVE_PID" 2>/dev/null || true

# -------------------------------------------------------------------
# CONFIGURATION
# -------------------------------------------------------------------
GITHUB_REPO="idioticdev/mac"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/mac"
CONFIG_FILE="$CONFIG_DIR/mac.toml"
# -------------------------------------------------------------------

printf "\n${BOLD}${BLUE}"
printf "  ┌─────────────────────────────────────┐\n"
printf "  │        mac — Bootstrap Script        │\n"
printf "  └─────────────────────────────────────┘\n"
printf "${NC}\n"

# ---- Step 1: Xcode Command Line Tools ------------------------------------
if xcode-select -p &>/dev/null; then
    ok "Xcode Command Line Tools already installed"
else
    info "Installing Xcode Command Line Tools …"
    # Triggers the native macOS install dialog, then polls until complete.
    # xcode-select --install exits non-zero (it launched the GUI, not the tools),
    # so we suppress the error and poll xcode-select -p every 5s.
    xcode-select --install 2>/dev/null || true
    echo ""
    warn "A dialog appeared — click Install to continue."
    echo "    Waiting for Xcode Command Line Tools to finish installing…"
    echo ""
    WAIT=0
    while ! xcode-select -p &>/dev/null; do
        sleep 5
        WAIT=$((WAIT + 5))
        printf "    [%ds elapsed]\r" "$WAIT"
        if [[ $WAIT -ge 1800 ]]; then
            fail "Xcode Command Line Tools installation timed out after 30 min. Install manually then re-run."
        fi
    done
    echo ""
    ok "Xcode Command Line Tools installed"
fi

# ---- Step 2: Homebrew -----------------------------------------------------
if command -v brew &>/dev/null; then
    ok "Homebrew already installed"
else
    info "Installing Homebrew …"
    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

    # Add to PATH for current session (Apple Silicon)
    if [[ -f /opt/homebrew/bin/brew ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    fi
    ok "Homebrew installed"
fi

# ---- Step 3: Detect architecture ------------------------------------------
MACHINE="$(uname -m)"
if [[ "$MACHINE" == "arm64" ]]; then
    ARCH="arm64"
elif [[ "$MACHINE" == "x86_64" ]]; then
    ARCH="amd64"
else
    fail "Unsupported architecture: $MACHINE"
fi
ok "Architecture: $ARCH"

# ---- Step 4: Fetch latest release tag -------------------------------------
info "Fetching latest release …"
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
API_RESPONSE="$(curl -sSL "$API_URL")"
if echo "$API_RESPONSE" | grep -q '"message": *"Not Found"'; then
    fail "No releases found for ${GITHUB_REPO}. A maintainer must publish a release first: https://github.com/${GITHUB_REPO}/releases"
fi
TAG="$(echo "$API_RESPONSE" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
if [[ -z "$TAG" ]]; then
    fail "Could not determine latest release. Check https://github.com/${GITHUB_REPO}/releases"
fi
ok "Latest release: $TAG"

# ---- Step 5: Download pre-built binary + checksums -----------------------
BINARY_NAME="mac-darwin-${ARCH}"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${BINARY_NAME}"
CHECKSUMS_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/checksums.txt"
TMP_BIN="$(mktemp)"
TMP_SUMS="$(mktemp)"

info "Downloading ${BINARY_NAME} …"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"; then
    rm -f "$TMP_BIN" "$TMP_SUMS"
    fail "Download failed: $DOWNLOAD_URL"
fi
ok "Downloaded $BINARY_NAME"

info "Verifying checksum …"
if ! curl -fsSL "$CHECKSUMS_URL" -o "$TMP_SUMS"; then
    rm -f "$TMP_BIN" "$TMP_SUMS"
    fail "Could not download checksums: $CHECKSUMS_URL"
fi
# Verify: extract the expected hash for this binary and compare.
EXPECTED="$(grep "${BINARY_NAME}$" "$TMP_SUMS" | awk '{print $1}')"
if [[ -z "$EXPECTED" ]]; then
    rm -f "$TMP_BIN" "$TMP_SUMS"
    fail "Checksum for ${BINARY_NAME} not found in checksums.txt"
fi
ACTUAL="$(shasum -a 256 "$TMP_BIN" | awk '{print $1}')"
if [[ "$ACTUAL" != "$EXPECTED" ]]; then
    rm -f "$TMP_BIN" "$TMP_SUMS"
    fail "Checksum mismatch for ${BINARY_NAME} (got ${ACTUAL}, expected ${EXPECTED})"
fi
rm -f "$TMP_SUMS"
ok "Checksum verified"

# ---- Step 6: Install binary -----------------------------------------------
info "Installing to ${INSTALL_DIR}/mac …"
chmod +x "$TMP_BIN"
sudo mkdir -p "$INSTALL_DIR"
sudo mv "$TMP_BIN" "${INSTALL_DIR}/mac"
ok "Installed: ${INSTALL_DIR}/mac  (${TAG})"

# ---- Step 7: Fetch or generate config ------------------------------------
printf "\n${BOLD}Almost there!${NC}\n\n"

if [[ -n "${CONFIG_URL:-}" ]]; then
    # Headless mode: download config non-interactively via mac init --url
    mac init --url "$CONFIG_URL" --output "$CONFIG_FILE"
elif [[ -f "$CONFIG_FILE" ]]; then
    ok "Existing config found: $CONFIG_FILE"
else
    # No config found — run the guided setup wizard.
    mac init --output "$CONFIG_FILE"
fi

# ---- Step 8: Apply (if config available) ----------------------------------
if [[ -f "$CONFIG_FILE" ]]; then
    echo ""
    read -rp "  Run mac apply now? [Y/n] " answer
    answer=${answer:-Y}
    if [[ "$answer" =~ ^[Yy]$ ]]; then
        echo ""
        mac apply -c "$CONFIG_FILE"
    fi
fi

printf "\n${BOLD}${GREEN}Done!${NC}\n\n"
echo "  To apply your config at any time:"
echo "    mac apply -c $CONFIG_FILE"
echo ""
