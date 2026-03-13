package packages

import (
	"fmt"
	"os"
	"strings"

	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

// EnsureHomebrew installs Homebrew if not present and runs update.
func EnsureHomebrew() {
	ui.Banner("Homebrew")

	if runexec.Which("brew") {
		ui.Ok("Homebrew already installed")
	} else {
		ui.Info("Installing Homebrew …")
		err := runexec.RunPassthrough("/bin/bash", "-c",
			`$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
		if err != nil {
			ui.Fail("Homebrew installation failed: " + err.Error())
			return
		}
		// Add to PATH for Apple Silicon
		if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
			shellenv, _ := runexec.Run("/opt/homebrew/bin/brew", "shellenv")
			for _, line := range strings.Split(shellenv, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "export PATH=") {
					val := strings.TrimPrefix(line, "export PATH=")
					val = strings.Trim(val, `"`)
					os.Setenv("PATH", val)
				}
			}
		}
		ui.Ok("Homebrew installed")
	}

	ui.Info("Updating Homebrew …")
	runexec.Run("brew", "update", "--quiet")
}

// InstallTaps adds Homebrew taps.
func InstallTaps(cfg *config.Config) {
	if len(cfg.Packages.Taps) == 0 {
		return
	}

	ui.Banner("Homebrew Taps")
	tapped, _ := runexec.Run("brew", "tap")
	tappedSet := toSet(tapped)

	for _, tap := range cfg.Packages.Taps {
		if tappedSet[tap] {
			ui.Ok(tap + " (already tapped)")
		} else {
			ui.Info("Tapping " + tap + " …")
			if _, err := runexec.Run("brew", "tap", tap); err != nil {
				ui.Fail(tap + ": " + err.Error())
			} else {
				ui.Ok(tap)
			}
		}
	}
}

// InstallFormulae installs Homebrew formulae.
func InstallFormulae(cfg *config.Config) {
	if len(cfg.Packages.Formulae) == 0 {
		return
	}

	ui.Banner("Homebrew Formulae")
	installed, _ := runexec.Run("brew", "list", "--formula", "-1")
	installedSet := toSet(installed)

	for _, pkg := range cfg.Packages.Formulae {
		if installedSet[pkg] {
			ui.Ok(pkg + " (installed)")
		} else {
			ui.Info("Installing " + pkg + " …")
			if err := runexec.RunPassthrough("brew", "install", pkg); err != nil {
				ui.Fail(pkg + ": " + err.Error())
			} else {
				ui.Ok(pkg)
			}
		}
	}
}

// InstallCasks installs Homebrew casks.
func InstallCasks(cfg *config.Config) {
	if len(cfg.Packages.Casks) == 0 {
		return
	}

	ui.Banner("Homebrew Casks")
	installed, _ := runexec.Run("brew", "list", "--cask", "-1")
	installedSet := toSet(installed)

	for _, cask := range cfg.Packages.Casks {
		if installedSet[cask] {
			ui.Ok(cask + " (installed)")
		} else {
			ui.Info("Installing " + cask + " …")
			if err := runexec.RunPassthrough("brew", "install", "--cask", cask); err != nil {
				ui.Fail(cask + ": " + err.Error())
			} else {
				ui.Ok(cask)
			}
		}
	}
}

// InstallMAS installs Mac App Store apps using mas.
func InstallMAS(cfg *config.Config) {
	if len(cfg.MAS.Apps) == 0 {
		return
	}

	ui.Banner("Mac App Store")

	if !runexec.Which("mas") {
		ui.Info("Installing mas CLI …")
		runexec.Run("brew", "install", "mas")
	}

	masList, _ := runexec.Run("mas", "list")

	for _, app := range cfg.MAS.Apps {
		idStr := fmt.Sprintf("%d", app.ID)
		if strings.Contains(masList, idStr) {
			ui.Ok(fmt.Sprintf("%s (%d) (installed)", app.Name, app.ID))
		} else {
			ui.Info(fmt.Sprintf("Installing %s (%d) …", app.Name, app.ID))
			if _, err := runexec.Run("mas", "install", idStr); err != nil {
				ui.Fail(fmt.Sprintf("%s: %s", app.Name, err.Error()))
			} else {
				ui.Ok(app.Name)
			}
		}
	}
}

// toSet splits newline-delimited output into a map for O(1) lookups.
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
