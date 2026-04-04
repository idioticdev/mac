# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mac** is a declarative macOS configuration tool written in Go. It manages an entire Mac setup from a single TOML config file: Homebrew packages, Mac App Store apps, dotfiles (via GNU Stow), macOS system defaults, PAM Touch ID, post-install hooks, and machine identity.

See `AGENTS.md` for agent workflow rules (beads issue tracking, session completion protocol, non-interactive shell flags).

## Build & Development

```bash
just              # List all available recipes
just build        # Build for current platform → ./mac
just install      # Build + install to /usr/local/bin
just build-all    # Cross-compile named release artifacts (arm64 + amd64)
just universal    # Universal binary via lipo
just fmt          # go fmt
just lint         # go vet
just test         # Run tests
just check        # fmt + lint + test (pre-commit gate)
just clean        # Remove build artifacts
```

Uses Go 1.22+. `devbox.json` provides the dev shell (run `devbox shell`). Install just: `brew install just`

## Architecture

All source lives in `cmd/mac/`. The CLI exposes: `apply`, `diff`, `validate`, `export`, `init`, `uninstall`, `shell-init`, and the internal `_track` command (called by the shell wrapper).

### Apply Execution Flow

1. Load & resolve config chain (`extends` inheritance, cycle detection in `extends.go`)
2. Keep sudo alive in a background goroutine
3. Apply in order: machine identity → Homebrew install → taps → formulae → casks → MAS apps → dotfiles clone+stow → default shell → macOS defaults → PAM Touch ID → hooks → restart services

### Key Files

| File | Responsibility |
|------|---------------|
| `main.go` | CLI entry point; command routing; shell path validation in `validate` |
| `config.go` | Config struct + TOML unmarshaling |
| `extends.go` | Config inheritance chain resolution with cycle detection |
| `packages.go` | Homebrew (taps/formulae/casks) + MAS operations |
| `defaults.go` | `defaults write` for macOS system preferences |
| `dotfiles.go` | Git clone + GNU Stow integration |
| `machine.go` | ComputerName & HostName |
| `system.go` | PAM Touch ID, library unhiding, shell defaults, services; `isValidShellPath` |
| `hooks.go` | Post-install shell command execution |
| `exec.go` | Shell helpers: `Run`, `RunPassthrough`, `RunSudo` |
| `ui.go` | Terminal UI (charmbracelet/lipgloss + charmbracelet/log) |
| `add.go` | `shell-init` (prints brew wrapper) + `_track` (adds pkg to mac.toml) |
| `toml_edit.go` | Atomic in-place TOML array append, preserving comments and formatting |

### Config System

TOML with sections: `[machine]`, `[packages]`, `[mas]`, `[dotfiles]`, `[shell]`, `[defaults.*]`, `[system]`, `[hooks]`. The `extends` key chains configs — child values win, package lists merge. See `mac.toml` for a full example.

### Idempotency Pattern

Every operation checks current state before acting (e.g., `brew list` before install, `defaults read` before write). All operations must be safe to run repeatedly.

## Git Workflow (OSS Branch Policy)

All code changes must go through a pull request — never commit directly to `main`.

**For every task that involves code changes:**

1. Create a branch before writing any code:
   ```bash
   git checkout -b <type>/<short-description>   # e.g. fix/stow-path, feat/mas-upgrade
   ```
2. Make commits on the branch (follow Conventional Commits: `feat:`, `fix:`, `docs:`, etc.)
3. Push and open a PR against `main`:
   ```bash
   git push -u origin <branch>
   gh pr create --title "..." --body "..."
   ```
4. Do **not** push directly to `main` under any circumstances.

Branch naming mirrors `CONTRIBUTING.md`: `fix/`, `feat/`, `docs/`, `refactor/`, `chore/`.

## Release

GitHub Actions (`.github/workflows/release.yml`) builds cross-platform binaries on `v*` tags and publishes them as GitHub Releases. The `install.sh` one-liner (`curl -fsSL https://mac.idiotic.dev/install.sh | bash`) fetches the latest binary — no Go required on the target machine.

## gstack

Use the `/browse` skill from gstack for all web browsing. Never use `mcp__claude-in-chrome__*` tools.

Available gstack skills:
- `/plan-ceo-review` — Review a plan from a CEO/product perspective
- `/plan-eng-review` — Review a plan from an engineering perspective
- `/plan-design-review` — Designer's eye review of a plan
- `/design-consultation` — Create a full design system and DESIGN.md
- `/design-html` — Generate HTML design mockups
- `/design-review` — Visual QA audit of a live site with fixes
- `/review` — Pre-landing PR code review
- `/ship` — Ship a feature end-to-end (tests, changelog, PR)
- `/land-and-deploy` — Merge PR, wait for CI/deploy, verify production
- `/canary` — Post-deploy canary monitoring
- `/benchmark` — Performance regression detection
- `/browse` — Browse the web (use this instead of MCP browser tools)
- `/connect-chrome` — Connect to a running Chrome instance
- `/qa` — QA test a site and fix bugs found
- `/qa-only` — QA test a site, report only (no fixes)
- `/setup-browser-cookies` — Import browser cookies for authenticated testing
- `/setup-deploy` — Configure deployment settings for /land-and-deploy
- `/retro` — Weekly engineering retrospective
- `/investigate` — Systematic root cause debugging
- `/document-release` — Post-ship documentation update
- `/codex` — OpenAI Codex second opinion (review, challenge, consult)
- `/cso` — Chief Security Officer security audit
- `/autoplan` — Auto-run all plan reviews with auto-decisions
- `/careful` — Safety guardrails for destructive commands
- `/freeze` — Restrict edits to a specific directory
- `/guard` — Full safety mode (careful + freeze combined)
- `/unfreeze` — Clear the freeze boundary
- `/gstack-upgrade` — Upgrade gstack to latest version
- `/learn` — Learn a new skill from a URL or description

If gstack skills aren't working, run `cd ~/.claude/skills/gstack && ./setup` to build the binary and register skills.
