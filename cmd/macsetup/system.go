package main

import (
	"os"
	"strings"
)

func applySystem(cfg *Config) {
	anySet := BoolVal(cfg.System.PamTID) ||
		BoolVal(cfg.System.RevealLibrary) ||
		cfg.System.ScreenshotsDir != ""

	if !anySet {
		return
	}

	Banner("System Tweaks")

	if BoolVal(cfg.System.PamTID) {
		applyPamTID()
	}
	if BoolVal(cfg.System.RevealLibrary) {
		applyRevealLibrary()
	}
	if cfg.System.ScreenshotsDir != "" {
		dir := ExpandHome(cfg.System.ScreenshotsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			Fail("Could not create screenshots dir: " + err.Error())
		} else {
			Ok("Screenshots directory: " + dir)
		}
	}
}

func applyShell(cfg *Config) {
	target := cfg.Shell.Default
	if target == "" {
		return
	}

	Banner("Default Shell")

	currentShell := os.Getenv("SHELL")
	if currentShell == target {
		Ok("Shell already set to " + target)
		return
	}

	shells, _ := os.ReadFile("/etc/shells")
	if !strings.Contains(string(shells), target) {
		Info("Adding " + target + " to /etc/shells …")
		RunShell("echo '" + target + "' | sudo tee -a /etc/shells >/dev/null")
	}

	Info("Changing default shell to " + target + " …")
	if err := RunPassthrough("chsh", "-s", target); err != nil {
		Fail("Could not change shell: " + err.Error())
	} else {
		Ok("Default shell → " + target)
	}
}

func applyPamTID() {
	pamLine := "auth       sufficient     pam_tid.so"
	sudoLocal := "/etc/pam.d/sudo_local"
	sudoFile := "/etc/pam.d/sudo"

	if fileContains(sudoLocal, "pam_tid.so") {
		Ok("Touch ID for sudo already enabled (sudo_local)")
		return
	}
	if fileContains(sudoFile, "pam_tid.so") {
		Ok("Touch ID for sudo already enabled")
		return
	}

	Info("Enabling Touch ID for sudo …")

	if _, err := os.Stat(sudoLocal); err == nil {
		RunShell("echo '" + pamLine + "' | sudo tee -a " + sudoLocal + " >/dev/null")
	} else {
		RunShell("echo '" + pamLine + "' | sudo tee " + sudoLocal + " >/dev/null")
	}
	Ok("Touch ID for sudo enabled")
}

func applyRevealLibrary() {
	home, _ := os.UserHomeDir()
	if _, err := Run("chflags", "nohidden", home+"/Library"); err != nil {
		Warn("Could not reveal ~/Library: " + err.Error())
	} else {
		Ok("~/Library revealed")
	}
}

func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

func restartServices() {
	Banner("Restarting Affected Services")

	services := []string{"Finder", "Dock", "SystemUIServer"}
	for _, svc := range services {
		if _, err := Run("killall", svc); err != nil {
			Skip(svc + " (not running)")
		} else {
			Ok("Restarted " + svc)
		}
	}
}
