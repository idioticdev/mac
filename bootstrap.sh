#!/usr/bin/env bash
# ============================================================================
# bootstrap.sh — One-liner bootstrap for mac
# ============================================================================
# Usage (fresh Mac):
#   curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/bootstrap.sh | bash
#
# Headless (skip config prompt):
#   CONFIG_URL=https://raw.githubusercontent.com/you/dotfiles/main/mac.toml \
#     curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/bootstrap.sh | bash
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
BINARY_NAME="mac-darwin-${ARCH}"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${BINARY_NAME}"
TMP_BIN="$(mktemp)"

info "Downloading ${BINARY_NAME} …"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"; then
    rm -f "$TMP_BIN"
    fail "Download failed: $DOWNLOAD_URL"
fi
ok "Downloaded $BINARY_NAME"

# ---- Step 6: Install binary -----------------------------------------------
info "Installing to ${INSTALL_DIR}/mac …"
chmod +x "$TMP_BIN"
sudo mkdir -p "$INSTALL_DIR"
sudo mv "$TMP_BIN" "${INSTALL_DIR}/mac"
ok "Installed: ${INSTALL_DIR}/mac  (${TAG})"

# ---- Step 7: Fetch or generate config ------------------------------------
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
    echo "  Do you have a mac.toml config URL? (leave blank to generate a starter config)"
    echo "  Example: https://raw.githubusercontent.com/you/dotfiles/main/mac.toml"
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
        info "Generating starter config at $CONFIG_FILE …"
        mkdir -p "$CONFIG_DIR"
        COMPUTER_NAME="$(scutil --get ComputerName 2>/dev/null || hostname -s)"
        LOCAL_HOSTNAME="$(scutil --get LocalHostName 2>/dev/null || hostname -s)"
        cat > "$CONFIG_FILE" <<TOML
# ============================================================================
# mac.toml — starter config generated by bootstrap
# Edit this file, then run:  mac apply
# Full reference: https://github.com/${GITHUB_REPO}#config-reference
# ============================================================================

[machine]
computer_name  = "${COMPUTER_NAME}"
local_hostname = "${LOCAL_HOSTNAME}"

[packages]
formulae = [
    "git",
    "stow",
    # "neovim",
    # "ripgrep",
    # "fzf",
]
casks = [
    # "raycast",
    # "1password",
]

# [dotfiles]
# repo          = "git@github.com:you/dotfiles.git"
# dest          = "~/.dotfiles"
# method        = "stow"
# stow_packages = ["zsh", "git"]

[system]
pam_tid = true   # Enable Touch ID for sudo

[hooks]
post_install = [
    "softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true",
]
TOML
        ok "Starter config written to $CONFIG_FILE"
        echo ""
        warn "Open $CONFIG_FILE and customise before running mac apply."
    fi
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
