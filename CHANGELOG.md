# Changelog

All notable changes are documented here. Full release notes and binaries are at
[github.com/idioticdev/mac/releases](https://github.com/idioticdev/mac/releases).

This project follows [Semantic Versioning](https://semver.org).

---

## [Unreleased]

### Added
- `mac doctor` — checks prerequisites (SSH keys, Homebrew health, stow, mas) before apply
- `mac diff` now exits 1 when changes are detected, enabling CI drift detection
- Apply failure summary — if any package or stow operation fails, a summary is printed at the end of `mac apply` instead of the misleading "complete" banner
- `mac init` now previews changes inline after setup and prompts to apply immediately
- Git clone and stow errors now show full output from the underlying command, not just the exit code

### Docs
- Added Troubleshooting section to README covering the most common failure modes
- Added `mac audit` and `mac doctor` documentation to README
- Added CHANGELOG

---

## [v0.5.0] — 2026-03-xx

### Added
- `mac upgrade` command to download and install the latest release
- Version nudge after `mac apply` if a newer release is available

### Changed
- `mac init` wizard asks about dotfiles repo and runs SSH setup inline

---

## [v0.4.0] — 2026-03-xx

### Added
- `mac audit` command — reports packages installed on the system but not tracked in config
- Key remapping via `[system].key_remapping` (persists via hidutil LaunchAgent)
- Machine identity management (`[machine]` section)

---

## [v0.3.0] — 2026-03-xx

### Added
- Config inheritance via `extends` (supports HTTP/S URLs and local paths)
- `[meta] skip` — skip sections to coexist with nix-darwin

---

## [v0.2.0] — 2026-02-xx

### Added
- `mac export` — generate mac.toml from current Homebrew state
- `mac init` — guided setup wizard
- Shell integration (`mac shell-init`) — auto-tracks `brew install` to mac.toml
- `[system]` section: PAM Touch ID, ~/Library unhide, screenshots directory

---

## [v0.1.0] — 2026-01-xx

Initial release.

- `mac apply` / `mac diff` / `mac validate`
- Homebrew taps, formulae, casks
- Mac App Store apps via `mas`
- Dotfiles clone + GNU Stow
- macOS defaults
- Post-install hooks
