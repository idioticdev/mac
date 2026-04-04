# mac

**One file. One command. Your entire Mac, configured.**

`mac` is a declarative macOS configuration tool written in Go. It manages Homebrew packages, Mac App Store apps, dotfiles (via GNU Stow), macOS system defaults, PAM Touch ID, post-install hooks, and machine identity — all from a single TOML config.

## Choose your path

```
Are you on a fresh Mac?
  └─ Yes ──► Path 1: Fresh Mac          (bootstrap one-liner, ~10 min)
  └─ No  ──► Do you use Homebrew?
               └─ Yes ──► Path 2: Existing Homebrew   (export + apply, ~5 min)
               └─ No  ──► Do you use nix-darwin?
                            └─ Yes ──► Path 3: nix-darwin   (coexist, ~5 min)
                            └─ No  ──► Path 2: Existing Homebrew
```

---

## Path 1: Fresh Mac

One command to go from factory-reset to your full setup:

```bash
curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/install.sh | bash
```

The bootstrap handles everything automatically:

1. Installs Xcode Command Line Tools
2. Installs Homebrew
3. Downloads the latest `mac` binary to `/usr/local/bin`
4. Runs `mac init` — a guided wizard to create your config

After the wizard, review changes before anything is applied:

```bash
mac diff       # preview all changes — safe, nothing is written
mac apply      # apply when ready
```

**Headless / CI install** (skip all prompts):

```bash
CONFIG_URL=https://raw.githubusercontent.com/you/dotfiles/main/mac.toml \
  curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/install.sh | bash
```

---

## Path 2: Existing Homebrew Machine

Your machine already has Homebrew and some packages. Generate a starter config from your current state:

```bash
# Install mac
curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/install.sh | bash

# Generate mac.toml from current Homebrew state
mac export > ~/.config/mac/mac.toml

# Edit the generated file (dotfiles, defaults, system tweaks are commented stubs)
$EDITOR ~/.config/mac/mac.toml

# Preview what would change — safe, nothing is written
mac diff

# Apply when satisfied
mac apply
```

`mac export` reads your current `brew list` and `brew tap` output and generates a complete mac.toml with commented-out stubs for sections you may want to add later.

Since `mac apply` is idempotent, packages already installed are skipped — it only installs what's missing.

---

## Path 3: Existing nix-darwin Machine

You can run `mac` alongside nix-darwin. The key is telling `mac` which sections nix-darwin owns so they don't fight for control.

```bash
# Install mac binary
curl -fsSL https://raw.githubusercontent.com/idioticdev/mac/main/install.sh | bash

# Generate a starter config from your current Homebrew state
mac export > ~/.config/mac/mac.toml

# Edit the file — add a [meta] skip block for nix-darwin-managed sections
$EDITOR ~/.config/mac/mac.toml
```

Add this near the top of your `mac.toml`:

```toml
# nix-darwin owns these — skip them to avoid conflicts
[meta]
skip = ["machine", "shell", "system"]
```

**Conflict reference:**

| Section | nix-darwin conflict | Recommendation |
|---|---|---|
| `[packages]` | None — Homebrew ≠ nix | Safe to use |
| `[dotfiles]` | Low (unless using home-manager) | Safe to use |
| `[defaults.*]` | High — both write defaults | Skip if nix-darwin sets the same keys |
| `[machine]` | High — `networking.hostName` | Skip |
| `[system]` | Medium — PAM, Library | Skip |
| `[shell]` | Medium — `users.users.<name>.shell` | Skip |
| `[hooks]` | None | Safe to use |

**Gradual migration path:**

```toml
# Phase 1: mac handles Homebrew + dotfiles; nix-darwin keeps the rest
[meta]
skip = ["machine", "shell", "system", "defaults"]

# Phase 2: Add defaults once you confirm no nix-darwin overlap
# (remove "defaults" from skip list)

# Phase 3: Full migration — remove [meta] skip after nix-darwin is gone
```

Preview before applying:

```bash
mac diff       # shows what would change, nothing is written
mac apply      # apply when ready
```

---

## Safe Testing Ground — Company Configs

Creating or reviewing a company mac.toml? Test it on your machine without touching anything:

```bash
# Preview the company config against your current state
mac diff -c https://raw.githubusercontent.com/acme/mac/main/company.toml

# Or test a local file
mac diff -c ~/acme-mac.toml
```

`mac diff` runs the entire apply pipeline as a dry run. Nothing is written. You see exactly which packages would install, which defaults would change, and which system tweaks would run — including sections that the old `mac diff` missed (machine name, dotfiles, shell, hooks).

Test a layered personal + company config:

```toml
# personal.toml
extends = ["https://raw.githubusercontent.com/acme/mac/main/company.toml"]

[packages]
casks = ["wezterm", "arc"]   # personal additions on top of company config
```

```bash
mac diff -c ~/personal.toml    # preview the merged result
mac apply -c ~/personal.toml   # apply when ready
```

---

## Usage

```
mac [command] [options]

Commands:
  apply       Apply the configuration (default)
  diff        Preview all changes — safe, nothing is written
  validate    Check the config file for errors
  export      Generate a mac.toml from current Homebrew state (stdout)
  init        Guided setup wizard — create your first mac.toml
  uninstall   Remove everything mac applied
  version     Print version

Options:
  -c, --config <path>    Config file (default: ~/.config/mac/mac.toml)
  -o, --output <path>    Output path for export / init
  -h, --help             Show this help

Examples:
  mac diff                        # preview changes
  mac diff -c company.toml        # preview a company config on your machine
  mac apply                       # apply config
  mac apply -c ~/my-setup.toml    # apply a specific config
  mac export                      # print mac.toml from current state
  mac export -o ~/mac.toml        # save export to file (via redirect or -o)
  mac init                        # guided setup wizard
  mac validate                    # check config for errors
  mac uninstall --dry-run         # preview what uninstall would do
```

---

## Config Reference

### `[meta]`

```toml
[meta]
# Skip these sections during apply and diff.
# Useful for coexisting with nix-darwin or other config managers.
# Valid values: machine, packages, mas, dotfiles, shell, defaults, system, hooks
skip = ["machine", "shell", "system"]
```

### `[machine]`

```toml
[machine]
computer_name  = "my-air"
local_hostname = "my-air"   # Bonjour / .local hostname
```

Find your current values:

```bash
scutil --get ComputerName   # shown in System Settings → Sharing
scutil --get LocalHostName  # Bonjour / .local hostname
```

### `[packages]`

```toml
[packages]
taps     = ["homebrew/cask-fonts"]
formulae = ["git", "neovim", "ripgrep", "fd", "fzf", "stow"]
casks    = ["wezterm", "raycast", "1password"]
```

### `[mas]` — Mac App Store

Requires the `mas` CLI (`brew install mas`).

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
dest           = "~/.dotfiles"   # where to clone (default: ~/.dotfiles)
method         = "stow"          # "stow" | "clone-only" (default: clone-only)
stow_packages  = ["zsh", "nvim", "git", "tmux"]
```

| `method` | Behavior |
|----------|----------|
| `clone-only` | Clone (or pull) the repo. No symlinking. Default when omitted. |
| `stow` | Clone/pull, then run `stow -t ~ <package>` for each entry in `stow_packages`. Installs GNU Stow via Homebrew if missing. |

`stow_packages` lists subdirectory names inside `dest` to activate. Each becomes `stow -d <dest> -t ~ --restow <package>`.

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

Discover defaults keys with the diff trick:

```bash
defaults read > /tmp/before.plist
# Change something in System Settings
defaults read > /tmp/after.plist
diff /tmp/before.plist /tmp/after.plist
```

### `[system]`

```toml
[system]
pam_tid         = true            # Enable Touch ID for sudo
reveal_library  = true            # Unhide ~/Library
screenshots_dir = "~/Screenshots" # Created if missing

# Key remapping — persists across reboots via hidutil LaunchAgent.
# mac apply writes ~/Library/LaunchAgents/com.local.mac-keyremap.plist
# and loads it immediately. mac uninstall removes it and resets hidutil.
key_remapping = [
    { from = "caps_lock", to = "escape" },
    { from = "left_fn",   to = "left_control" },
]
```

**`key_remapping` — available key names:**

| Category | Keys |
|----------|------|
| Special | `caps_lock`, `escape`, `return`, `tab`, `space`, `backspace`, `delete_forward` |
| Left modifiers | `left_shift`, `left_control`, `left_option`, `left_command`, `left_fn` |
| Right modifiers | `right_shift`, `right_control`, `right_option`, `right_command` |
| Function keys | `f1` – `f12` |

> **Note:** `left_fn` is the Globe/Fn key on Apple Silicon keyboards. It is silently ignored on Intel Macs — use [Karabiner-Elements](https://karabiner-elements.pqrs.org/) instead.

### `extends` — Template inheritance

```toml
extends = [
    "https://raw.githubusercontent.com/acme/mac/main/company.toml",
]
```

Merge semantics: packages/taps/MAS apps union (deduplicated); defaults deep-merge (child wins on conflict); hooks concatenated (base first); machine/shell/dotfiles child wins if non-empty.

### `[hooks]`

```toml
[hooks]
post_install = [
    "softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true",
]
```

---

## Why mac?

| Tool | Packages | Dotfiles | Defaults | Touch ID | Key Remap | One Config | Simple Install |
|------|----------|----------|----------|----------|-----------|------------|----------------|
| **mac** | ✅ | ✅ Stow | ✅ | ✅ | ✅ | ✅ TOML | ✅ one binary |
| nix-darwin | ✅ | ✅ | ✅ | ✅ | ❌ | Nix (steep) | ❌ |
| Homebrew Bundle | ✅ | ❌ | ❌ | ❌ | ❌ | Brewfile | ✅ |
| Ansible | ✅ | ✅ | ✅ | ✅ | ❌ | YAML (heavy) | ❌ |

---

## Building from Source

```bash
just build      # build → ./mac
just install    # build + install to /usr/local/bin
just test       # run tests
just check      # fmt + lint + test
```

Install just: `brew install just`. Requires Go 1.22+.

---

## Idempotency

Every operation checks current state before acting. Safe to re-run after any config change. `mac diff` always shows the current delta.

## Requirements

- macOS 13+ (Sonoma+ recommended for `sudo_local` PAM support)
- Internet connection for bootstrap and remote configs
- Go 1.22+ only if building from source

## License

MIT
