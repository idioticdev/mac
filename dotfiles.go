package dotfiles

import (
	"os"
	"path/filepath"

	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

func Apply(cfg *config.Config) {
	if cfg.Dotfiles.Repo == "" {
		return
	}

	ui.Banner("Dotfiles")

	dest := runexec.ExpandHome(cfg.Dotfiles.Dest)
	method := cfg.Dotfiles.Method
	if method == "" {
		method = "clone-only"
	}

	// Clone or pull
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		ui.Ok("Dotfiles repo exists at " + dest)
		ui.Info("Pulling latest …")
		if _, err := runexec.Run("git", "-C", dest, "pull", "--ff-only"); err != nil {
			ui.Warn("Pull failed (dirty tree?)")
		}
	} else {
		ui.Info("Cloning " + cfg.Dotfiles.Repo + " → " + dest)
		if _, err := runexec.Run("git", "clone", cfg.Dotfiles.Repo, dest); err != nil {
			ui.Fail("Clone failed: " + err.Error())
			return
		}
		ui.Ok("Cloned")
	}

	// Stow
	if method == "stow" {
		if !runexec.Which("stow") {
			ui.Info("Installing GNU Stow …")
			runexec.Run("brew", "install", "stow")
		}

		home, _ := os.UserHomeDir()
		for _, pkg := range cfg.Dotfiles.StowPackages {
			pkgDir := filepath.Join(dest, pkg)
			if info, err := os.Stat(pkgDir); err != nil || !info.IsDir() {
				ui.Warn("Stow package directory not found: " + pkgDir)
				continue
			}

			ui.Info("Stowing " + pkg + " …")
			if _, err := runexec.Run("stow", "-d", dest, "-t", home, "--restow", pkg); err != nil {
				ui.Warn(pkg + " had conflicts: " + err.Error())
			} else {
				ui.Ok(pkg)
			}
		}
	}
}
