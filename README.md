# mac

**One file. One command. Your entire Mac, configured.**

`mac` is a declarative macOS configuration tool written in Go. It handles Homebrew packages, Mac App Store apps, dotfiles (via GNU Stow), macOS system defaults, PAM Touch ID, and post-install hooks — all from a single TOML config.

## Why?

| Tool | Packages | Dotfiles | System Defaults | PAM / Touch ID | One Config | Simple Install |
|------|----------|----------|-----------------|----------------|------------|----------------|
| **mac** | ✅ | ✅ (Stow) | ✅ | ✅ | ✅ TOML | ✅ one binary |
| nix-darwin | ✅ | ✅ | ✅ | ✅ | Nix (steep curve) | ❌ |
| Devbox | ✅ | ❌ | ❌ | ❌ | JSON | ✅ |
| Homebrew Bundle | ✅ | ❌ | ❌ | ❌ | Brewfile | ✅ |
| Ansible | ✅ | ✅ | ✅ | ✅ | YAML (heavy) | ❌ |

## Quick Start — Fresh Mac

One command to go from a factory-fresh Mac to your full setup:

```bash
curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/bootstrap.sh | bash
```

The bootstrap handles everything: Xcode CLI Tools → Homebrew → download pre-built binary → install to `/usr/local/bin` → prompt for your config URL (or generate a starter config) → apply. No Go installation required.

For a fully headless install (CI / zero interaction):

```bash
CONFIG_URL=https://raw.githubusercontent.com/you/dotfiles/main/mac.toml \
  curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/bootstrap.sh | bash
```

## Quick Start — Existing Setup

```bash
# Clone
git clone https://github.com/idioticdev/mac.git ~/.mac
cd ~/.mac

# Edit config
$EDITOR mac.toml

# Build and install
make install

# Run
mac apply
```

## Usage

```
mac [command] [options]

Commands:
  apply       Apply the configuration (default)
  diff        Show what would change without applying
  validate    Check the config file for errors
  version     Print version

Options:
  -c, --config <path>   Config file (default: ~/.config/mac/mac.toml)
  -h, --help            Show this help

Environment:
  MAC_CONFIG   Override config path (same as -c)
```

### Preview changes before applying

```bash
mac diff
```

Shows which packages would be installed, which defaults would change, and what system tweaks would be applied — without touching anything.

### Validate your config

```bash
mac validate
```

## Building

```bash
make build             # Standard build
make install           # Build + install to /usr/local/bin
make build-all         # Cross-compile named release artifacts (arm64 + amd64)
make universal         # Universal binary via lipo (Intel + Apple Silicon)
make fmt               # Format source
make lint              # Run go vet
```

## Config Reference

### `[machine]`

```toml
[machine]
computer_name  = "my-air"
local_hostname = "my-air"
```

To find your current values:

```bash
scutil --get ComputerName   # → "My MacBook Air"  (shown in Sharing preferences)
scutil --get LocalHostName  # → "My-MacBook-Air"  (Bonjour / .local hostname)
```

### `[packages]`

```toml
[packages]
taps     = ["homebrew/cask-fonts"]
formulae = ["git", "neovim", "ripgrep", "fd", "fzf", "stow"]
casks    = ["wezterm", "raycast", "1password"]
```

### `[mas]` — Mac App Store

```toml
[mas]
apps = [
    { id = 497799835, name = "Xcode" },
]
```

### `[dotfiles]`

```toml
[dotfiles]
repo           = "git@github.com:you/dotfiles.git"
dest           = "~/.dotfiles"
method         = "stow"
stow_packages  = ["zsh", "nvim", "git", "tmux"]
```

### `[shell]`

```toml
[shell]
default = "/bin/zsh"
```

### `[defaults.*]` — macOS Preferences

```toml
[defaults.NSGlobalDomain]
AppleShowAllExtensions = true
KeyRepeat              = 2
AppleInterfaceStyle    = "Dark"

[defaults."com.apple.finder"]
ShowPathbar       = true
AppleShowAllFiles = true

[defaults."com.apple.dock"]
autohide = true
tilesize = 36
```

### `[system]`

```toml
[system]
pam_tid         = true
reveal_library  = true
screenshots_dir = "~/Screenshots"
```

### `extends` — Template inheritance

Layer your config on top of a base template (e.g. a company preset):

```toml
extends = [
    "https://raw.githubusercontent.com/acme/mac/main/company.toml",
]
```

Merge semantics:
- **Packages / taps / MAS apps** — union (deduplicated, base first)
- **Defaults** — deep merge, your config wins on conflict
- **Hooks** — concatenated (base hooks run first)
- **machine / shell / dotfiles** — your config wins entirely if non-empty
- **system** — field-by-field, your non-nil values win

Chains are supported (a base can also extend). Cycles are detected and reported as errors.

### `[hooks]`

```toml
[hooks]
post_install = [
    "softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true",
]
```

## Discovering macOS Defaults

```bash
defaults domains | tr ',' '\n' | sort        # List all domains
defaults read com.apple.dock                  # Read a domain

# Diff trick
defaults read > /tmp/before.plist
# (change something in System Settings)
defaults read > /tmp/after.plist
diff /tmp/before.plist /tmp/after.plist
```

## Project Structure

```
mac/
├── bootstrap.sh                 # One-liner bootstrap for fresh Macs
├── mac.toml                     # Example config (copy and edit)
├── Makefile
├── go.mod / go.sum
├── .github/workflows/
│   └── release.yml              # Publishes pre-built binaries on git tags
└── cmd/mac/
    ├── main.go                  # CLI entry point: apply / diff / validate
    ├── config.go                # Config struct and TOML loader
    ├── extends.go               # Template inheritance + merge logic
    ├── init.go                  # First-run config URL prompt
    ├── packages.go              # Homebrew + MAS installation
    ├── defaults.go              # macOS defaults write
    ├── dotfiles.go              # Git clone + GNU Stow
    ├── machine.go               # ComputerName / HostName
    ├── system.go                # PAM, ~/Library, shell, services
    ├── hooks.go                 # Post-install hooks
    ├── exec.go                  # Shell execution helpers
    └── ui.go                    # Colored terminal output
```

## Idempotency

Every operation checks current state before acting. Safe to re-run after any config change.

## Requirements

- macOS 13+ (Sonoma+ recommended for `sudo_local` PAM support)
- Internet connection
- Go 1.22+ (only needed if building from source; not required for the bootstrap install)

## License

MIT
