# Contributing to mac

Thanks for your interest in contributing. This document covers how to get set up, the workflow for making changes, and what to expect during review.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Commit Style](#commit-style)
- [Code Standards](#code-standards)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

---

## Getting Started

Before writing code, please:

1. Check [existing issues](../../issues) to avoid duplicate work.
2. For significant changes, **open an issue first** to discuss the approach. This saves everyone time if the direction needs adjustment.
3. For small fixes (typos, docs, obvious bugs), a PR is fine without a prior issue.

---

## Development Setup

**Requirements:**
- macOS (this tool is macOS-only)
- Go 1.22+
- [just](https://github.com/casey/just): `brew install just`

```bash
# Clone the repo
git clone https://github.com/idiotic/mac.git
cd mac

# Build
just build       # → ./mac

# Run all quality checks (required before submitting a PR)
just check       # fmt + lint + test
```

Available recipes:

```bash
just             # List all recipes
just build       # Build for current platform
just fmt         # go fmt
just lint        # go vet
just test        # Run tests
just check       # fmt + lint + test (pre-commit gate)
just install     # Build + install to /usr/local/bin
just clean       # Remove build artifacts
```

---

## Making Changes

1. Fork the repository and create a branch from `main`:
   ```bash
   git checkout -b fix/describe-your-change
   ```

2. Make your changes. Keep commits focused — one logical change per commit.

3. Run the full quality gate before pushing:
   ```bash
   just check
   ```

4. Push and open a pull request against `main`.

### Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Bug fix | `fix/description` | `fix/stow-path-expansion` |
| Feature | `feat/description` | `feat/mas-upgrade-command` |
| Documentation | `docs/description` | `docs/config-reference` |
| Refactor | `refactor/description` | `refactor/exec-helpers` |

---

## Pull Request Process

1. Fill out the pull request template completely.
2. Ensure CI passes — PRs with failing checks will not be reviewed.
3. Keep PRs focused. If you find an unrelated issue while working, open a separate PR.
4. A maintainer will review within a reasonable time. Be prepared for feedback and revision requests.
5. Maintainers may close PRs that don't align with the project's direction — this is not personal, and the issue discussion beforehand is the right place to align first.

---

## Commit Style

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <short description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

Examples:
```
feat: add screenshots_dir to [system] config
fix: stow --restow flag missing for re-runs
docs: document nix-darwin coexistence pattern
```

---

## Code Standards

- **Idempotency**: Every operation must be safe to run repeatedly. Check current state before acting.
- **No silent failures**: Surface errors clearly. Don't swallow errors that indicate real problems.
- **Minimal dependencies**: This binary is intentionally small. Avoid adding external dependencies unless absolutely necessary.
- **macOS-only scope**: Don't add cross-platform abstractions. This tool does one thing on one platform.
- See `.claude/rules/code-quality.md` for detailed naming, comment, and style conventions.

---

## Reporting Bugs

Use the [Bug Report](../../issues/new?template=bug_report.yml) issue template. Include:
- macOS version
- `mac` version (`mac version`)
- The relevant section of your `mac.toml` (redact personal info)
- The full error output

---

## Suggesting Features

Use the [Feature Request](../../issues/new?template=feature_request.yml) issue template. Describe the problem you're solving, not just the solution — there may be a better approach already in the works.

---

## Security

If you discover a security vulnerability, **do not open a public issue**. See [SECURITY.md](SECURITY.md) for the responsible disclosure process.

---

## Code of Conduct

All contributors are expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md). Be respectful and constructive.
