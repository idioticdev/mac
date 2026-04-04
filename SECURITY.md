# Security Policy

## Supported Versions

Only the latest released version of `mac` receives security fixes. We do not backport patches to older releases.

| Version | Supported |
|---------|-----------|
| Latest  | ✅        |
| Older   | ❌        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.** Doing so exposes users before a fix is available.

Instead, use one of these private channels:

1. **GitHub Private Vulnerability Reporting** (preferred): Use the [Report a Vulnerability](../../security/advisories/new) button on the Security tab of this repository.
2. **Email**: Contact the maintainer directly via email listed on the GitHub profile.

### What to Include

A useful report includes:
- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept (without weaponizing it)
- The `mac` version affected (`mac version`)
- Your macOS version

### What to Expect

- **Acknowledgment**: Within 72 hours of receiving the report.
- **Assessment**: We will confirm whether it is a valid vulnerability and its severity.
- **Fix timeline**: We aim to patch critical issues within 7 days and ship a release promptly.
- **Credit**: We will credit reporters in the release notes unless you prefer to remain anonymous.

We will not take legal action against researchers who report vulnerabilities in good faith and follow this policy.

---

## Security Considerations for Users

### `extends` and Remote Configs

The `extends` key can fetch and apply TOML from remote URLs at `mac apply` time. **Only extend configs from sources you trust.** A malicious remote config can:
- Install arbitrary Homebrew packages or casks
- Execute arbitrary shell commands via `[hooks]`
- Change system defaults and machine identity

Review any remote config with `mac diff -c <url>` before applying it.

### `[hooks]` — Shell Command Execution

The `post_install` hooks run arbitrary shell commands as your user. Treat the `[hooks]` section of any config like shell scripts — review them before applying.

### Install Script

The one-liner install script (`curl ... | bash`) fetches the latest release binary. To verify integrity:

```bash
# Download and verify checksums before running
curl -fsSL https://mac.idiotic.dev/install.sh -o install.sh
# Review the script
cat install.sh
# Then run
bash install.sh
```

Release binaries are accompanied by a `checksums.txt` (SHA-256) on the GitHub Releases page.

### `sudo` Usage

`mac apply` requests `sudo` for operations that require it (PAM configuration, `scutil` for machine naming). It does not store your password and uses a background goroutine to keep the `sudo` timestamp alive only for the duration of the apply run.
