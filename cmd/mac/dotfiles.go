package main

import (
	"os"
	"path/filepath"
)

func applyDotfiles(cfg *Config, r Runner) {
	if cfg.Dotfiles.Repo == "" {
		return
	}

	Banner("Dotfiles")

	dest := r.ExpandHome(cfg.Dotfiles.Dest)
	method := cfg.Dotfiles.Method
	if method == "" {
		method = "clone-only"
	}

	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		Ok("Dotfiles repo exists at " + dest)
		Info("Pulling latest …")
		if _, err := r.Run("git", "-C", dest, "pull", "--ff-only"); err != nil {
			Warn("Pull failed (dirty tree?)")
		}
	} else {
		ensureSSHForClone(cfg.Dotfiles.Repo, r)
		Info("Cloning " + cfg.Dotfiles.Repo + " → " + dest)
		if _, err := r.Run("git", "clone", cfg.Dotfiles.Repo, dest); err != nil {
			Fail("Clone failed: " + err.Error())
			return
		}
		Ok("Cloned")
	}

	if method == "stow" {
		if !r.Which("stow") {
			Info("Installing GNU Stow …")
			r.Run("brew", "install", "stow")
		}

		home, _ := os.UserHomeDir()
		for _, pkg := range cfg.Dotfiles.StowPackages {
			pkgDir := filepath.Join(dest, pkg)
			if info, err := os.Stat(pkgDir); err != nil || !info.IsDir() {
				Warn("Stow package directory not found: " + pkgDir)
				continue
			}

			Info("Stowing " + pkg + " …")
			if _, err := r.Run("stow", "-d", dest, "-t", home, "--restow", pkg); err != nil {
				Warn(pkg + " had conflicts: " + err.Error())
			} else {
				Ok(pkg)
			}
		}
	}
}

func uninstallDotfiles(cfg *Config, r Runner) {
	if cfg.Dotfiles.Repo == "" {
		return
	}

	Banner("Dotfiles")

	dest := r.ExpandHome(cfg.Dotfiles.Dest)

	if cfg.Dotfiles.Method == "stow" {
		home, _ := os.UserHomeDir()
		for _, pkg := range cfg.Dotfiles.StowPackages {
			Info("Unstowing " + pkg + " …")
			if _, err := r.Run("stow", "-d", dest, "-t", home, "-D", pkg); err != nil {
				Warn(pkg + " had conflicts: " + err.Error())
			} else {
				Ok(pkg + " unstowed")
			}
		}
	}

	if _, err := os.Stat(dest); err == nil {
		Info("Removing dotfiles repo at " + dest + " …")
		if _, err := r.Run("rm", "-rf", dest); err != nil {
			Fail("Could not remove " + dest + ": " + err.Error())
		} else {
			Ok("Removed " + dest)
		}
	} else {
		Skip("dotfiles repo not found at " + dest)
	}
}
