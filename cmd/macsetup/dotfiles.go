package main

import (
	"os"
	"path/filepath"
)

func applyDotfiles(cfg *Config) {
	if cfg.Dotfiles.Repo == "" {
		return
	}

	Banner("Dotfiles")

	dest := ExpandHome(cfg.Dotfiles.Dest)
	method := cfg.Dotfiles.Method
	if method == "" {
		method = "clone-only"
	}

	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		Ok("Dotfiles repo exists at " + dest)
		Info("Pulling latest …")
		if _, err := Run("git", "-C", dest, "pull", "--ff-only"); err != nil {
			Warn("Pull failed (dirty tree?)")
		}
	} else {
		Info("Cloning " + cfg.Dotfiles.Repo + " → " + dest)
		if _, err := Run("git", "clone", cfg.Dotfiles.Repo, dest); err != nil {
			Fail("Clone failed: " + err.Error())
			return
		}
		Ok("Cloned")
	}

	if method == "stow" {
		if !Which("stow") {
			Info("Installing GNU Stow …")
			Run("brew", "install", "stow")
		}

		home, _ := os.UserHomeDir()
		for _, pkg := range cfg.Dotfiles.StowPackages {
			pkgDir := filepath.Join(dest, pkg)
			if info, err := os.Stat(pkgDir); err != nil || !info.IsDir() {
				Warn("Stow package directory not found: " + pkgDir)
				continue
			}

			Info("Stowing " + pkg + " …")
			if _, err := Run("stow", "-d", dest, "-t", home, "--restow", pkg); err != nil {
				Warn(pkg + " had conflicts: " + err.Error())
			} else {
				Ok(pkg)
			}
		}
	}
}
