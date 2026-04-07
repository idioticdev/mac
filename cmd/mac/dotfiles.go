package main

import (
	"os"
	"path/filepath"
)

// checkStowConflicts walks pkgDir and returns paths in targetDir that are
// regular files — these will block stow from creating symlinks.
func checkStowConflicts(pkgDir, targetDir string) ([]string, error) {
	var conflicts []string
	err := filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(pkgDir, path)
		target := filepath.Join(targetDir, rel)
		fi, err := os.Lstat(target)
		if err != nil {
			return nil // target doesn't exist — no conflict
		}
		if fi.Mode().IsRegular() {
			conflicts = append(conflicts, target)
		}
		return nil
	})
	return conflicts, err
}

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
		if err := r.RunPassthrough("git", "clone", cfg.Dotfiles.Repo, dest); err != nil {
			Fail("Clone failed — check git output above for details")
			return
		}
		Ok("Cloned")
	}

	if method == "stow" {
		if !r.Which("stow") {
			Info("Installing GNU Stow …")
			r.Run("brew", "install", "stow")
		}

		home := r.ExpandHome("~")
		for _, pkg := range cfg.Dotfiles.StowPackages {
			pkgDir := filepath.Join(dest, pkg)
			if info, err := os.Stat(pkgDir); err != nil || !info.IsDir() {
				Warn("Stow package directory not found: " + pkgDir)
				continue
			}

			if conflicts, err := checkStowConflicts(pkgDir, home); err != nil {
				Warn("Could not check stow conflicts for " + pkg + ": " + err.Error())
			} else if len(conflicts) > 0 {
				Warn(pkg + " has stow conflicts — back up these files before applying:")
				for _, c := range conflicts {
					Info("  mv " + c + " " + c + ".bak")
				}
				Warn("Skipping stow for " + pkg)
				continue
			}

			Info("Stowing " + pkg + " …")
			if err := r.RunPassthrough("stow", "-d", dest, "-t", home, "--restow", pkg); err != nil {
				Warn(pkg + " had conflicts — check stow output above for details")
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
		home := r.ExpandHome("~")
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
