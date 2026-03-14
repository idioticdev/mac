package main

import (
	"fmt"
	"os"
	"strings"
)

func EnsureHomebrew() {
	Banner("Homebrew")

	if Which("brew") {
		Ok("Homebrew already installed")
	} else {
		Info("Installing Homebrew …")
		err := RunPassthrough("/bin/bash", "-c",
			`$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
		if err != nil {
			Fail("Homebrew installation failed: " + err.Error())
			return
		}
		if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
			shellenv, _ := Run("/opt/homebrew/bin/brew", "shellenv")
			for _, line := range strings.Split(shellenv, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "export PATH=") {
					val := strings.TrimPrefix(line, "export PATH=")
					val = strings.Trim(val, `"`)
					os.Setenv("PATH", val)
				}
			}
		}
		Ok("Homebrew installed")
	}

	Info("Updating Homebrew …")
	Run("brew", "update", "--quiet")
}

func InstallTaps(cfg *Config) {
	if len(cfg.Packages.Taps) == 0 {
		return
	}

	Banner("Homebrew Taps")
	tapped, _ := Run("brew", "tap")
	tappedSet := toSet(tapped)

	for _, tap := range cfg.Packages.Taps {
		if tappedSet[tap] {
			Ok(tap + " (already tapped)")
		} else {
			Info("Tapping " + tap + " …")
			if _, err := Run("brew", "tap", tap); err != nil {
				Fail(tap + ": " + err.Error())
			} else {
				Ok(tap)
			}
		}
	}
}

func InstallFormulae(cfg *Config) {
	if len(cfg.Packages.Formulae) == 0 {
		return
	}

	Banner("Homebrew Formulae")
	installed, _ := Run("brew", "list", "--formula", "-1")
	installedSet := toSet(installed)

	for _, pkg := range cfg.Packages.Formulae {
		if installedSet[pkg] {
			Ok(pkg + " (installed)")
		} else {
			Info("Installing " + pkg + " …")
			if err := RunPassthrough("brew", "install", pkg); err != nil {
				Fail(pkg + ": " + err.Error())
			} else {
				Ok(pkg)
			}
		}
	}
}

func InstallCasks(cfg *Config) {
	if len(cfg.Packages.Casks) == 0 {
		return
	}

	Banner("Homebrew Casks")
	installed, _ := Run("brew", "list", "--cask", "-1")
	installedSet := toSet(installed)

	for _, cask := range cfg.Packages.Casks {
		if installedSet[cask] {
			Ok(cask + " (installed)")
		} else {
			Info("Installing " + cask + " …")
			if err := RunPassthrough("brew", "install", "--cask", cask); err != nil {
				Fail(cask + ": " + err.Error())
			} else {
				Ok(cask)
			}
		}
	}
}

func InstallMAS(cfg *Config) {
	if len(cfg.MAS.Apps) == 0 {
		return
	}

	Banner("Mac App Store")

	if !Which("mas") {
		Info("Installing mas CLI …")
		Run("brew", "install", "mas")
	}

	masList, _ := Run("mas", "list")

	for _, app := range cfg.MAS.Apps {
		idStr := fmt.Sprintf("%d", app.ID)
		if strings.Contains(masList, idStr) {
			Ok(fmt.Sprintf("%s (%d) (installed)", app.Name, app.ID))
		} else {
			Info(fmt.Sprintf("Installing %s (%d) …", app.Name, app.ID))
			if _, err := Run("mas", "install", idStr); err != nil {
				Fail(fmt.Sprintf("%s: %s", app.Name, err.Error()))
			} else {
				Ok(app.Name)
			}
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
