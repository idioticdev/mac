package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runDoctor checks prerequisites and diagnoses common issues.
func runDoctor(cfg *Config, r Runner) {
	Header()
	Info("checking system prerequisites …")
	fmt.Fprintln(os.Stderr)

	issues := 0

	// macOS version check
	osVer, _ := r.Run("sw_vers", "-productVersion")
	if osVer != "" {
		parts := strings.Split(osVer, ".")
		major := 0
		if len(parts) > 0 {
			fmt.Sscanf(parts[0], "%d", &major)
		}
		if major >= 13 {
			Ok(fmt.Sprintf("macOS %s", osVer))
		} else {
			Warn(fmt.Sprintf("macOS %s — version 13 (Ventura) or newer recommended", osVer))
			issues++
		}
	}

	// Homebrew
	if r.Which("brew") {
		Ok("Homebrew installed")
	} else {
		Fail("Homebrew not found — install it: https://brew.sh")
		issues++
	}

	// Config file
	if cfg != nil {
		Ok(fmt.Sprintf("Config loaded (%d formulae, %d casks)", len(cfg.Packages.Formulae), len(cfg.Packages.Casks)))
	}

	// SSH key (needed for git@ dotfiles repos)
	if cfg != nil && strings.HasPrefix(cfg.Dotfiles.Repo, "git@") {
		fmt.Fprintln(os.Stderr)
		Banner("Dotfiles SSH")
		sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
		hasKey := false
		for _, keyFile := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
			if _, err := os.Stat(filepath.Join(sshDir, keyFile)); err == nil {
				hasKey = true
				Ok(fmt.Sprintf("SSH key found: ~/.ssh/%s", keyFile))
				break
			}
		}
		if !hasKey {
			Warn("No SSH key found in ~/.ssh — dotfiles clone may fail")
			Info("Generate one: ssh-keygen -t ed25519 -C \"your@email.com\"")
			Info("Then add the public key to GitHub: https://github.com/settings/keys")
			issues++
		}

		// Test GitHub SSH connectivity
		Info("Testing SSH connectivity to github.com …")
		if _, err := r.Run("ssh", "-T", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes", "git@github.com"); err != nil {
			// ssh -T to GitHub exits non-zero even on success (it prints "Hi user!")
			// so we check stderr for the "successfully authenticated" message
			out, _ := r.Run("ssh", "-T", "-o", "ConnectTimeout=5", "git@github.com")
			if strings.Contains(out, "successfully authenticated") {
				Ok("GitHub SSH authentication works")
			} else {
				Warn("Could not verify GitHub SSH connectivity")
				Info("Run: ssh -T git@github.com")
				issues++
			}
		}
	}

	// Stow (if method=stow)
	if cfg != nil && cfg.Dotfiles.Method == "stow" {
		fmt.Fprintln(os.Stderr)
		Banner("GNU Stow")
		if r.Which("stow") {
			Ok("stow installed")
		} else {
			Warn("stow not installed — required for dotfiles.method = \"stow\"")
			Info("Install: brew install stow")
			issues++
		}
	}

	// mas CLI (if [mas] section is non-empty)
	if cfg != nil && len(cfg.MAS.Apps) > 0 {
		fmt.Fprintln(os.Stderr)
		Banner("Mac App Store")
		if r.Which("mas") {
			Ok("mas installed")
		} else {
			Fail(fmt.Sprintf("mas not installed — required for %d MAS app(s) in config", len(cfg.MAS.Apps)))
			Info("Install: brew install mas")
			issues++
		}
	}

	// Homebrew health
	fmt.Fprintln(os.Stderr)
	Banner("Homebrew Health")
	Info("running brew doctor …")
	if err := r.RunPassthrough("brew", "doctor"); err != nil {
		Warn("brew doctor reported issues — see above for details")
		issues++
	}

	// Summary
	fmt.Fprintln(os.Stderr)
	if issues == 0 {
		Ok("All checks passed — ready to run: mac apply")
	} else {
		Warn(fmt.Sprintf("%d issue(s) found — fix the above before running mac apply", issues))
	}
}
