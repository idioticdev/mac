#!/usr/bin/env bash
# ============================================================================
# bootstrap.sh — One-liner bootstrap for macsetup
# ============================================================================
# Usage (fresh Mac):
#   bash <(curl -sL https://raw.githubusercontent.com/youruser/macsetup/main/bootstrap.sh)
#
# What this does:
#   1. Installs Xcode Command Line Tools (for git, clang)
#   2. Installs Homebrew (if missing)
#   3. Installs Go via Homebrew
#   4. Clones your macsetup repo
#   5. Builds the macsetup binary
#   6. Runs macsetup apply
# ============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()  { printf "  ${CYAN}→${NC} %s\n" "$1"; }
ok()    { printf "  ${GREEN}✓${NC} %s\n" "$1"; }
fail()  { printf "  ${RED}✗${NC} %s\n" "$1"; exit 1; }

# -------------------------------------------------------------------
# CONFIGURATION — edit these to match your repo
# -------------------------------------------------------------------
REPO_URL="https://github.com/youruser/macsetup.git"   # HTTPS for fresh Macs without SSH keys
REPO_SSH="git@github.com:youruser/macsetup.git"        # Used after SSH keys are set up
CLONE_DIR="$HOME/.macsetup"
INSTALL_DIR="/usr/local/bin"
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

    # Wait for installation to complete
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

# ---- Step 3: Go -----------------------------------------------------------
if command -v go &>/dev/null; then
    ok "Go already installed ($(go version | awk '{print $3}'))"
else
    info "Installing Go via Homebrew …"
    brew install go
    ok "Go installed"
fi

# ---- Step 4: Clone / update repo ------------------------------------------
if [[ -d "$CLONE_DIR" ]]; then
    ok "Repo already cloned at $CLONE_DIR"
    info "Pulling latest …"
    git -C "$CLONE_DIR" pull --ff-only 2>/dev/null || true
else
    info "Cloning macsetup repo …"
    # Try HTTPS first (no SSH keys on fresh Mac), fall back to SSH
    if git clone "$REPO_URL" "$CLONE_DIR" 2>/dev/null; then
        ok "Cloned via HTTPS"
    elif git clone "$REPO_SSH" "$CLONE_DIR" 2>/dev/null; then
        ok "Cloned via SSH"
    else
        fail "Could not clone repo. Check REPO_URL in this script."
    fi
fi

# ---- Step 5: Build --------------------------------------------------------
info "Building macsetup …"
cd "$CLONE_DIR"
go build -o macsetup ./cmd/macsetup
ok "Built: $CLONE_DIR/macsetup"

# ---- Step 6: Install binary -----------------------------------------------
info "Installing to $INSTALL_DIR/macsetup …"
sudo mkdir -p "$INSTALL_DIR"
sudo cp macsetup "$INSTALL_DIR/macsetup"
sudo chmod +x "$INSTALL_DIR/macsetup"
ok "Installed: $INSTALL_DIR/macsetup"

# ---- Step 7: Run ----------------------------------------------------------
printf "\n${BOLD}Build complete!${NC}\n\n"
echo "  To apply your config:"
echo ""
echo "    cd $CLONE_DIR"
echo "    macsetup apply"
echo ""
echo "  Or with a custom config:"
echo ""
echo "    macsetup apply -c /path/to/macsetup.toml"
echo ""

# Ask if they want to run now
read -rp "  Run macsetup apply now? [Y/n] " answer
answer=${answer:-Y}
if [[ "$answer" =~ ^[Yy]$ ]]; then
    echo ""
    macsetup apply -c "$CLONE_DIR/macsetup.toml"
fi
