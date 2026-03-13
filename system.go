package system

import (
	"os"
	"strings"

	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

func Apply(cfg *config.Config) {
	anySet := config.BoolVal(cfg.System.PamTID) ||
		config.BoolVal(cfg.System.RevealLibrary) ||
		cfg.System.ScreenshotsDir != ""

	if !anySet {
		return
	}

	ui.Banner("System Tweaks")

	if config.BoolVal(cfg.System.PamTID) {
		applyPamTID()
	}
	if config.BoolVal(cfg.System.RevealLibrary) {
		applyRevealLibrary()
	}
	if cfg.System.ScreenshotsDir != "" {
		dir := runexec.ExpandHome(cfg.System.ScreenshotsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			ui.Fail("Could not create screenshots dir: " + err.Error())
		} else {
			ui.Ok("Screenshots directory: " + dir)
		}
	}
}

// ApplyShell sets the default login shell.
func ApplyShell(cfg *config.Config) {
	target := cfg.Shell.Default
	if target == "" {
		return
	}

	ui.Banner("Default Shell")

	currentShell := os.Getenv("SHELL")
	if currentShell == target {
		ui.Ok("Shell already set to " + target)
		return
	}

	// Ensure shell is in /etc/shells
	shells, _ := os.ReadFile("/etc/shells")
	if !strings.Contains(string(shells), target) {
		ui.Info("Adding " + target + " to /etc/shells …")
		runexec.RunShell("echo '" + target + "' | sudo tee -a /etc/shells >/dev/null")
	}

	ui.Info("Changing default shell to " + target + " …")
	if err := runexec.RunPassthrough("chsh", "-s", target); err != nil {
		ui.Fail("Could not change shell: " + err.Error())
	} else {
		ui.Ok("Default shell → " + target)
	}
}

func applyPamTID() {
	pamLine := "auth       sufficient     pam_tid.so"

	// Check sudo_local (Sonoma+)
	sudoLocal := "/etc/pam.d/sudo_local"
	sudoFile := "/etc/pam.d/sudo"

	if fileContains(sudoLocal, "pam_tid.so") {
		ui.Ok("Touch ID for sudo already enabled (sudo_local)")
		return
	}
	if fileContains(sudoFile, "pam_tid.so") {
		ui.Ok("Touch ID for sudo already enabled")
		return
	}

	ui.Info("Enabling Touch ID for sudo …")

	if _, err := os.Stat(sudoLocal); err == nil {
		// sudo_local exists (Sonoma+) — append
		runexec.RunShell("echo '" + pamLine + "' | sudo tee -a " + sudoLocal + " >/dev/null")
	} else {
		// Pre-Sonoma: create sudo_local
		runexec.RunShell("echo '" + pamLine + "' | sudo tee " + sudoLocal + " >/dev/null")
	}
	ui.Ok("Touch ID for sudo enabled")
}

func applyRevealLibrary() {
	home, _ := os.UserHomeDir()
	if _, err := runexec.Run("chflags", "nohidden", home+"/Library"); err != nil {
		ui.Warn("Could not reveal ~/Library: " + err.Error())
	} else {
		ui.Ok("~/Library revealed")
	}
}

func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// RestartServices kills and restarts Finder, Dock, SystemUIServer.
func RestartServices() {
	ui.Banner("Restarting Affected Services")

	services := []string{"Finder", "Dock", "SystemUIServer"}
	for _, svc := range services {
		if _, err := runexec.Run("killall", svc); err != nil {
			ui.Skip(svc + " (not running)")
		} else {
			ui.Ok("Restarted " + svc)
		}
	}
}
