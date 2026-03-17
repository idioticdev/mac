# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mac** is a declarative macOS configuration tool written in Go. It manages an entire Mac setup from a single TOML config file: Homebrew packages, Mac App Store apps, dotfiles (via GNU Stow), macOS system defaults, PAM Touch ID, post-install hooks, and machine identity.

See `AGENTS.md` for agent workflow rules (beads issue tracking, session completion protocol, non-interactive shell flags).

## Build & Development

```bash
make build        # Build for current platform → ./mac
make install      # Build + install to /usr/local/bin
make build-all    # Cross-compile arm64 + amd64
make universal    # Universal binary via lipo
make fmt          # go fmt
make lint         # go vet
make test         # Run tests
make clean        # Remove build artifacts
```

Uses Go 1.22+. `devbox.json` provides the dev shell (run `devbox shell`).

## Architecture

All source lives in `cmd/mac/`. The CLI exposes three commands: `apply`, `diff`, `validate`.

### Apply Execution Flow

1. Load & resolve config chain (`extends` inheritance, cycle detection in `extends.go`)
2. Keep sudo alive in a background goroutine
3. Apply in order: machine identity → Homebrew install → taps → formulae → casks → MAS apps → dotfiles clone+stow → default shell → macOS defaults → PAM Touch ID → hooks → restart services

### Key Files

| File | Responsibility |
|------|---------------|
| `main.go` | CLI setup; `apply`/`diff`/`validate`/`version` commands |
| `config.go` | Config struct + TOML unmarshaling |
| `extends.go` | Config inheritance chain resolution with cycle detection |
| `packages.go` | Homebrew (taps/formulae/casks) + MAS operations |
| `defaults.go` | `defaults write` for macOS system preferences |
| `dotfiles.go` | Git clone + GNU Stow integration |
| `machine.go` | ComputerName & HostName |
| `system.go` | PAM Touch ID, library unhiding, shell defaults, services |
| `hooks.go` | Post-install shell command execution |
| `exec.go` | Shell helpers: `Run`, `RunPassthrough`, `RunSudo` |
| `ui.go` | Terminal UI (charmbracelet/lipgloss + charmbracelet/log) |

### Config System

TOML with sections: `[machine]`, `[packages]`, `[mas]`, `[dotfiles]`, `[shell]`, `[defaults.*]`, `[system]`, `[hooks]`. The `extends` key chains configs — child values win, package lists merge. See `mac.toml` for a full example.

### Idempotency Pattern

Every operation checks current state before acting (e.g., `brew list` before install, `defaults read` before write). All operations must be safe to run repeatedly.

## Release

GitHub Actions (`.github/workflows/release.yml`) builds cross-platform binaries on `v*` tags and publishes them as GitHub Releases. The `bootstrap.sh` one-liner fetches the latest binary — no Go required on the target machine.

## gstack

Use the `/browse` skill from gstack for all web browsing. Never use `mcp__claude-in-chrome__*` tools.

Available gstack skills:
- `/plan-ceo-review` — Review a plan from a CEO/product perspective
- `/plan-eng-review` — Review a plan from an engineering perspective
- `/review` — Code review
- `/ship` — Ship a feature end-to-end
- `/browse` — Browse the web (use this instead of MCP browser tools)
- `/qa` — QA testing
- `/setup-browser-cookies` — Set up browser cookies for authenticated browsing
- `/retro` — Run a retrospective

If gstack skills aren't working, run `cd .claude/skills/gstack && ./setup` to build the binary and register skills.
