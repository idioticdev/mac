package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// keyRemapPlistDir is the LaunchAgents directory for the key remap plist.
// Overridable in tests via t.TempDir().
var keyRemapPlistDir = "~/Library/LaunchAgents"

// keyNameToHID maps friendly key names to HID usage values for hidutil.
var keyNameToHID = map[string]uint64{
	"caps_lock":      0x700000039,
	"escape":         0x700000029,
	"return":         0x700000028,
	"tab":            0x70000002B,
	"space":          0x70000002C,
	"backspace":      0x70000002A,
	"delete_forward": 0x70000004C,
	"left_shift":     0x7000000E1,
	"right_shift":    0x7000000E5,
	"left_control":   0x7000000E0,
	"right_control":  0x7000000E4,
	"left_option":    0x7000000E2,
	"right_option":   0x7000000E6,
	"left_command":   0x7000000E3,
	"right_command":  0x7000000E7,
	// NOTE: left_fn is the Globe/Fn key on Apple Silicon. Silently ignored on Intel.
	"left_fn": 0xFF00000003,
	"f1":      0x70000003A, "f2": 0x70000003B, "f3": 0x70000003C,
	"f4": 0x70000003D, "f5": 0x70000003E, "f6": 0x70000003F,
	"f7": 0x700000040, "f8": 0x700000041, "f9": 0x700000042,
	"f10": 0x700000043, "f11": 0x700000044, "f12": 0x700000045,
}

// validateKeyRemapping returns error messages for any unknown key names.
func validateKeyRemapping(cfg *Config) []string {
	var errs []string
	for _, m := range cfg.System.KeyRemapping {
		if _, ok := keyNameToHID[m.From]; !ok {
			errs = append(errs, fmt.Sprintf("unknown key name %q in key_remapping.from", m.From))
		}
		if _, ok := keyNameToHID[m.To]; !ok {
			errs = append(errs, fmt.Sprintf("unknown key name %q in key_remapping.to", m.To))
		}
	}
	return errs
}

// buildHidutilJSON returns the JSON argument for hidutil --set from a list of mappings.
func buildHidutilJSON(mappings []KeyRemapConfig) (string, error) {
	type entry struct {
		Src uint64 `json:"HIDKeyboardModifierMappingSrc"`
		Dst uint64 `json:"HIDKeyboardModifierMappingDst"`
	}
	entries := make([]entry, 0, len(mappings))
	for _, m := range mappings {
		src, ok := keyNameToHID[m.From]
		if !ok {
			return "", fmt.Errorf("unknown key name %q", m.From)
		}
		dst, ok := keyNameToHID[m.To]
		if !ok {
			return "", fmt.Errorf("unknown key name %q", m.To)
		}
		entries = append(entries, entry{Src: src, Dst: dst})
	}
	b, err := json.Marshal(map[string][]entry{"UserKeyMapping": entries})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// buildKeyRemapPlist returns the XML plist content for the key remap LaunchAgent.
func buildKeyRemapPlist(hidutilJSON string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.local.mac-keyremap</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/hidutil</string>
        <string>property</string>
        <string>--set</string>
        <string>` + hidutilJSON + `</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`
}

func applyKeyRemapping(cfg *Config, r Runner) {
	if len(cfg.System.KeyRemapping) == 0 {
		return
	}

	hidutilJSON, err := buildHidutilJSON(cfg.System.KeyRemapping)
	if err != nil {
		Fail("Key remapping: " + err.Error())
		return
	}

	plistDir := r.ExpandHome(keyRemapPlistDir)
	plistPath := filepath.Join(plistDir, "com.local.mac-keyremap.plist")

	// Idempotency: skip if plist already contains this exact JSON.
	if existing, readErr := os.ReadFile(plistPath); readErr == nil {
		if strings.Contains(string(existing), hidutilJSON) {
			Ok("Key remapping already configured")
			return
		}
	}

	if err := os.MkdirAll(plistDir, 0755); err != nil {
		Fail("Could not create LaunchAgents dir: " + err.Error())
		return
	}

	if err := r.WriteFile(plistPath, []byte(buildKeyRemapPlist(hidutilJSON)), 0644); err != nil {
		Fail("Could not write key remap plist: " + err.Error())
		return
	}

	// Unload first (ignore error — not yet loaded is fine).
	r.Run("launchctl", "unload", plistPath) //nolint:errcheck

	if _, err := r.Run("launchctl", "load", plistPath); err != nil {
		Fail("Could not load key remap LaunchAgent: " + err.Error())
		return
	}

	Ok("Key remapping configured")
}

func uninstallKeyRemapping(cfg *Config, r Runner) {
	if len(cfg.System.KeyRemapping) == 0 {
		return
	}

	plistPath := filepath.Join(r.ExpandHome(keyRemapPlistDir), "com.local.mac-keyremap.plist")

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		Skip("Key remapping (already absent)")
		return
	}

	r.Run("launchctl", "unload", plistPath) //nolint:errcheck

	if err := os.Remove(plistPath); err != nil {
		Fail("Could not remove key remap plist: " + err.Error())
		return
	}

	// Reset all hidutil mappings immediately.
	r.Run("hidutil", "property", "--set", `{"UserKeyMapping":[]}`) //nolint:errcheck

	Ok("Key remapping removed")
}

func applySystem(cfg *Config, r Runner) {
	anySet := BoolVal(cfg.System.PamTID) ||
		BoolVal(cfg.System.RevealLibrary) ||
		cfg.System.ScreenshotsDir != "" ||
		len(cfg.System.KeyRemapping) > 0

	if !anySet {
		return
	}

	Banner("System Tweaks")

	if BoolVal(cfg.System.PamTID) {
		applyPamTID(r)
	}
	if BoolVal(cfg.System.RevealLibrary) {
		applyRevealLibrary(r)
	}
	if cfg.System.ScreenshotsDir != "" {
		dir := r.ExpandHome(cfg.System.ScreenshotsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			Fail("Could not create screenshots dir: " + err.Error())
		} else {
			Ok("Screenshots directory: " + dir)
		}
	}
	applyKeyRemapping(cfg, r)
}

func applyShell(cfg *Config, r Runner) {
	target := cfg.Shell.Default
	if target == "" {
		return
	}

	if !isValidShellPath(target) {
		Fail(fmt.Sprintf("Invalid shell.default %q: must be an absolute path using only [a-zA-Z0-9/_.-]", target))
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
		if err := appendShellEntry(r, string(shells), target); err != nil {
			Fail("Could not add to /etc/shells: " + err.Error())
			return
		}
	}

	Info("Changing default shell to " + target + " …")
	if err := r.RunPassthrough("chsh", "-s", target); err != nil {
		Fail("Could not change shell: " + err.Error())
	} else {
		Ok("Default shell → " + target)
	}
}

// isValidShellPath returns true when path is safe to use as a shell executable:
// absolute and composed only of [a-zA-Z0-9/_.-].
func isValidShellPath(path string) bool {
	if len(path) < 2 || path[0] != '/' {
		return false
	}
	for _, c := range path[1:] {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '/' || c == '_' || c == '-' || c == '.':
		default:
			return false
		}
	}
	return true
}

// appendShellEntry appends target to /etc/shells using sudo cp — no shell
// interpolation, so the path cannot inject commands regardless of its content.
func appendShellEntry(r Runner, existing, target string) error {
	tmp, err := os.CreateTemp("", "mac-shells-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(existing + target + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	_, err = r.RunSudo("cp", tmp.Name(), "/etc/shells")
	return err
}

func applyPamTID(r Runner) {
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
		r.RunShell("echo '" + pamLine + "' | sudo tee -a " + sudoLocal + " >/dev/null")
	} else {
		r.RunShell("echo '" + pamLine + "' | sudo tee " + sudoLocal + " >/dev/null")
	}
	Ok("Touch ID for sudo enabled")
}

func applyRevealLibrary(r Runner) {
	home, _ := os.UserHomeDir()
	if _, err := r.Run("chflags", "nohidden", home+"/Library"); err != nil {
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

func uninstallSystem(cfg *Config, r Runner) {
	anySet := BoolVal(cfg.System.PamTID) || BoolVal(cfg.System.RevealLibrary) ||
		cfg.Shell.Default != "" || len(cfg.System.KeyRemapping) > 0
	if !anySet {
		return
	}

	Banner("System Tweaks")

	if BoolVal(cfg.System.PamTID) {
		uninstallPamTID(r)
	}

	if BoolVal(cfg.System.RevealLibrary) {
		Warn("~/Library was revealed by mac apply — re-hide manually if desired:")
		Warn("  chflags hidden ~/Library")
	}

	if cfg.Shell.Default != "" {
		Warn("Default shell was changed by mac apply.")
		Warn("To revert manually: chsh -s /bin/zsh (or your preferred shell)")
	}

	uninstallKeyRemapping(cfg, r)
}

func uninstallPamTID(r Runner) {
	sudoLocal := "/etc/pam.d/sudo_local"

	if !fileContains(sudoLocal, "pam_tid.so") {
		Skip("Touch ID for sudo (already absent)")
		return
	}

	Info("Removing Touch ID for sudo …")
	if _, err := r.RunShell("sudo sed -i '' '/pam_tid\\.so/d' " + sudoLocal); err != nil {
		Fail("Could not remove pam_tid line: " + err.Error())
	} else {
		Ok("Touch ID for sudo disabled")
	}
}

func restartServices(r Runner) {
	Banner("Restarting Affected Services")

	services := []string{"Finder", "Dock", "SystemUIServer"}
	for _, svc := range services {
		if _, err := r.Run("killall", svc); err != nil {
			Skip(svc + " (not running)")
		} else {
			Ok("Restarted " + svc)
		}
	}
}
