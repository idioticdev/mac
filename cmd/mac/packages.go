package main

import (
	"fmt"
	"os"
	"strings"
)

func EnsureHomebrew(r Runner) {
	Banner("Homebrew")

	if r.Which("brew") {
		Ok("Homebrew already installed")
	} else {
		Info("Installing Homebrew …")
		err := r.RunPassthrough("/bin/bash", "-c",
			`$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
		if err != nil {
			Fail("Homebrew installation failed: " + err.Error())
			return
		}
		if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
			shellenv, _ := r.Run("/opt/homebrew/bin/brew", "shellenv")
			for _, line := range strings.Split(shellenv, "\n") {
				line = strings.TrimSpace(line)
				if val, ok := strings.CutPrefix(line, "export PATH="); ok {
					val = strings.Trim(val, `"`)
					os.Setenv("PATH", val)
				}
			}
		}
		Ok("Homebrew installed")
	}

	Info("Updating Homebrew …")
	r.Run("brew", "update", "--quiet")
}

func InstallTaps(cfg *Config, r Runner) {
	if len(cfg.Packages.Taps) == 0 {
		return
	}

	Banner("Homebrew Taps")
	tapped, _ := r.Run("brew", "tap")
	tappedSet := toSet(tapped)

	for _, tap := range cfg.Packages.Taps {
		if tappedSet[tap] {
			Ok(tap + " (already tapped)")
		} else {
			Info("Tapping " + tap + " …")
			if _, err := r.Run("brew", "tap", tap); err != nil {
				Fail(tap + ": " + err.Error())
			} else {
				Ok(tap)
			}
		}
	}
}

func InstallFormulae(cfg *Config, r Runner) {
	if len(cfg.Packages.Formulae) == 0 {
		return
	}

	Banner("Homebrew Formulae")
	installed, _ := r.Run("brew", "list", "--formula", "-1")
	installedSet := toSet(installed)

	for _, pkg := range cfg.Packages.Formulae {
		if installedSet[pkg] {
			Ok(pkg + " (installed)")
		} else {
			Info("Installing " + pkg + " …")
			if err := r.RunPassthrough("brew", "install", pkg); err != nil {
				Fail(pkg + ": " + err.Error())
			} else {
				Ok(pkg)
			}
		}
	}
}

func InstallCasks(cfg *Config, r Runner) {
	if len(cfg.Packages.Casks) == 0 {
		return
	}

	Banner("Homebrew Casks")
	installed, _ := r.Run("brew", "list", "--cask", "-1")
	installedSet := toSet(installed)

	for _, cask := range cfg.Packages.Casks {
		if installedSet[cask] {
			Ok(cask + " (installed)")
		} else {
			Info("Installing " + cask + " …")
			if err := r.RunPassthrough("brew", "install", "--cask", cask); err != nil {
				Fail(cask + ": " + err.Error())
			} else {
				Ok(cask)
			}
		}
	}
}

func InstallMAS(cfg *Config, r Runner) {
	if len(cfg.MAS.Apps) == 0 {
		return
	}

	Banner("Mac App Store")

	if !r.Which("mas") {
		Info("Installing mas CLI …")
		r.Run("brew", "install", "mas")
	}

	masList, _ := r.Run("mas", "list")

	for _, app := range cfg.MAS.Apps {
		idStr := fmt.Sprintf("%d", app.ID)
		if strings.Contains(masList, idStr) {
			Ok(fmt.Sprintf("%s (%d) (installed)", app.Name, app.ID))
		} else {
			Info(fmt.Sprintf("Installing %s (%d) …", app.Name, app.ID))
			if _, err := r.Run("mas", "install", idStr); err != nil {
				Fail(fmt.Sprintf("%s: %s", app.Name, err.Error()))
			} else {
				Ok(app.Name)
			}
		}
	}
}

func uninstallPackages(cfg *Config, r Runner) {
	if len(cfg.Packages.Formulae) == 0 && len(cfg.Packages.Casks) == 0 && len(cfg.Packages.Taps) == 0 {
		return
	}

	Banner("Homebrew Packages")

	if len(cfg.Packages.Formulae) > 0 {
		installed, _ := r.Run("brew", "list", "--formula", "-1")
		installedSet := toSet(installed)
		for _, pkg := range cfg.Packages.Formulae {
			if !installedSet[pkg] {
				Skip(pkg)
				continue
			}
			Info("Uninstalling " + pkg + " …")
			if err := r.RunPassthrough("brew", "uninstall", pkg); err != nil {
				Fail(pkg + ": " + err.Error())
			} else {
				Ok(pkg + " removed")
			}
		}
	}

	if len(cfg.Packages.Casks) > 0 {
		installed, _ := r.Run("brew", "list", "--cask", "-1")
		installedSet := toSet(installed)
		for _, cask := range cfg.Packages.Casks {
			if !installedSet[cask] {
				Skip(cask)
				continue
			}
			Info("Uninstalling cask " + cask + " …")
			if err := r.RunPassthrough("brew", "uninstall", "--cask", cask); err != nil {
				Fail(cask + ": " + err.Error())
			} else {
				Ok(cask + " removed")
			}
		}
	}

	if len(cfg.Packages.Taps) > 0 {
		tapped, _ := r.Run("brew", "tap")
		tappedSet := toSet(tapped)
		for _, tap := range cfg.Packages.Taps {
			if !tappedSet[tap] {
				Skip(tap)
				continue
			}
			Info("Untapping " + tap + " …")
			if _, err := r.Run("brew", "untap", tap); err != nil {
				Fail(tap + ": " + err.Error())
			} else {
				Ok(tap + " untapped")
			}
		}
	}
}

func uninstallMAS(cfg *Config, r Runner) {
	if len(cfg.MAS.Apps) == 0 {
		return
	}

	Banner("Mac App Store")

	if !r.Which("mas") {
		Warn("mas CLI not found — skipping MAS uninstall")
		return
	}

	masList, _ := r.Run("mas", "list")
	for _, app := range cfg.MAS.Apps {
		idStr := fmt.Sprintf("%d", app.ID)
		if !strings.Contains(masList, idStr) {
			Skip(fmt.Sprintf("%s (%d)", app.Name, app.ID))
			continue
		}
		Info(fmt.Sprintf("Uninstalling %s (%d) …", app.Name, app.ID))
		if _, err := r.Run("mas", "uninstall", idStr); err != nil {
			Fail(fmt.Sprintf("%s: %s", app.Name, err.Error()))
		} else {
			Ok(app.Name + " removed")
		}
	}
}

func toSet(output string) map[string]bool {
	s := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			s[line] = true
		}
	}
	return s
}
