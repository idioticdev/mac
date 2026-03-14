#!/usr/bin/env bash
# ============================================================================
# bootstrap.sh — One-liner bootstrap for macsetup
# ============================================================================
# Usage (fresh Mac):
#   curl -fsSL https://raw.githubusercontent.com/youruser/macsetup/main/bootstrap.sh | bash
#
# Headless (skip config prompt):
#   CONFIG_URL=https://raw.githubusercontent.com/you/dotfiles/main/macsetup.toml \
#     curl -fsSL https://raw.githubusercontent.com/youruser/macsetup/main/bootstrap.sh | bash
#
# What this does:
#   1. Installs Xcode Command Line Tools (if missing)
#   2. Installs Homebrew (if missing)
#   3. Detects architecture (arm64 / amd64)
#   4. Downloads the latest pre-built macsetup binary from GitHub Releases
#   5. Installs it to /usr/local/bin
#   6. Prompts for your macsetup.toml URL and runs apply
# ============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()  { printf "  ${CYAN}→${NC} %s\n" "$1"; }
ok()    { printf "  ${GREEN}✓${NC} %s\n" "$1"; }
warn()  { printf "  ${YELLOW}!${NC} %s\n" "$1"; }
fail()  { printf "  ${RED}✗${NC} %s\n" "$1"; exit 1; }

# -------------------------------------------------------------------
# CONFIGURATION
# -------------------------------------------------------------------
GITHUB_REPO="youruser/macsetup"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/macsetup"
CONFIG_FILE="$CONFIG_DIR/macsetup.toml"
# -------------------------------------------------------------------

printf "\n${BOLD}${BLUE}"
printf "  ┌─────────────────────────────────────┐\n"
printf "  │     macsetup — Bootstrap Script      │\n"
printf "  └─────────────────────────────────────┘\n"
printf "${NC}\n"

# ---- Step 1: Xcode Command Line Tools ------------------------------------
if xcode-select -p &>/dev/null; then
    ok "Xcode Command Line Tools already installed"
else
    info "Installing Xcode Command Line Tools …"
    xcode-select --install 2>/dev/null || true

    echo ""
    echo "    A dialog should have appeared asking to install the tools."
    echo "    Please complete the installation, then press ENTER to continue."
    read -r

    if ! xcode-select -p &>/dev/null; then
        fail "Xcode Command Line Tools installation failed. Please install manually and re-run."
    fi
    ok "Xcode Command Line Tools installed"
fi

# ---- Step 2: Homebrew -----------------------------------------------------
if command -v brew &>/dev/null; then
    ok "Homebrew already installed"
else
    info "Installing Homebrew …"
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

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
TAG="$(curl -fsSL "$API_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
if [[ -z "$TAG" ]]; then
    fail "Could not determine latest release. Check https://github.com/${GITHUB_REPO}/releases"
fi
ok "Latest release: $TAG"

# ---- Step 5: Download pre-built binary ------------------------------------
BINARY_NAME="macsetup-darwin-${ARCH}"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${BINARY_NAME}"
TMP_BIN="$(mktemp)"

info "Downloading ${BINARY_NAME} …"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"; then
    rm -f "$TMP_BIN"
    fail "Download failed: $DOWNLOAD_URL"
fi
ok "Downloaded $BINARY_NAME"

# ---- Step 6: Install binary -----------------------------------------------
info "Installing to ${INSTALL_DIR}/macsetup …"
chmod +x "$TMP_BIN"
sudo mkdir -p "$INSTALL_DIR"
sudo mv "$TMP_BIN" "${INSTALL_DIR}/macsetup"
ok "Installed: ${INSTALL_DIR}/macsetup  (${TAG})"

# ---- Step 7: Fetch config -------------------------------------------------
printf "\n${BOLD}Almost there!${NC}\n\n"

if [[ -n "${CONFIG_URL:-}" ]]; then
    # Headless mode: CONFIG_URL provided via env
    info "Fetching config from $CONFIG_URL …"
    mkdir -p "$CONFIG_DIR"
    if ! curl -fsSL "$CONFIG_URL" -o "$CONFIG_FILE"; then
        fail "Could not download config from: $CONFIG_URL"
    fi
    ok "Config saved to $CONFIG_FILE"
elif [[ -f "$CONFIG_FILE" ]]; then
    ok "Existing config found: $CONFIG_FILE"
else
    echo "  Enter the raw GitHub URL to your macsetup.toml, or press ENTER to skip."
    echo "  Example: https://raw.githubusercontent.com/you/dotfiles/main/macsetup.toml"
    echo ""
    printf "  Config URL: "
    read -r CONFIG_URL

    if [[ -n "$CONFIG_URL" ]]; then
        info "Fetching config …"
        mkdir -p "$CONFIG_DIR"
        if ! curl -fsSL "$CONFIG_URL" -o "$CONFIG_FILE"; then
            fail "Could not download config from: $CONFIG_URL"
        fi
        ok "Config saved to $CONFIG_FILE"
    else
        warn "Skipped. Run later with:  macsetup apply -c /path/to/macsetup.toml"
    fi
fi

# ---- Step 8: Apply (if config available) ----------------------------------
if [[ -f "$CONFIG_FILE" ]]; then
    echo ""
    read -rp "  Run macsetup apply now? [Y/n] " answer
    answer=${answer:-Y}
    if [[ "$answer" =~ ^[Yy]$ ]]; then
        echo ""
        macsetup apply -c "$CONFIG_FILE"
    fi
fi

printf "\n${BOLD}${GREEN}Done!${NC}\n\n"
echo "  To apply your config at any time:"
echo "    macsetup apply -c $CONFIG_FILE"
echo ""
