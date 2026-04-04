package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestApplySystem_NothingSet(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	applySystem(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when no system config, got %v", r.Calls())
	}
}

func TestApplyRevealLibrary(t *testing.T) {
	r := testutil.NewFakeRunner()
	// Just verify chflags nohidden is called (home dir varies, use CalledWith partial match via Calls).
	applyRevealLibrary(r)
	found := false
	for _, c := range r.Calls() {
		if len(c) > 16 && c[:16] == "chflags nohidden" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected chflags nohidden call, got %v", r.Calls())
	}
}

func TestRestartServices(t *testing.T) {
	r := testutil.NewFakeRunner()
	restartServices(r)

	for _, svc := range []string{"Finder", "Dock", "SystemUIServer"} {
		if !r.CalledWith("killall", svc) {
			t.Errorf("expected killall %s", svc)
		}
	}
}

func TestApplyShell_EmptyTarget(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	applyShell(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when shell not configured, got %v", r.Calls())
	}
}

// ── isValidShellPath ─────────────────────────────────────────────────────────

func TestIsValidShellPath_Valid(t *testing.T) {
	cases := []string{
		"/bin/zsh",
		"/usr/local/bin/fish",
		"/opt/homebrew/bin/bash",
		"/usr/bin/zsh-5.9",
		"/bin/sh",
	}
	for _, c := range cases {
		if !isValidShellPath(c) {
			t.Errorf("expected valid: %q", c)
		}
	}
}

func TestIsValidShellPath_Invalid(t *testing.T) {
	cases := []string{
		"",
		"/",
		"bin/zsh",           // relative
		"/bin/zsh; evil",    // semicolon
		"/bin/zsh' && cmd",  // single quote
		"/bin/z$h",          // dollar sign
		"/bin/z sh",         // space
		"/bin/zsh\nmalicious", // newline
	}
	for _, c := range cases {
		if isValidShellPath(c) {
			t.Errorf("expected invalid: %q", c)
		}
	}
}

// ── applyShell injection guard ───────────────────────────────────────────────

func TestApplyShell_InvalidPathRejected(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{Shell: ShellConfig{Default: "/bin/zsh'; evil command; echo '"}}
	applyShell(cfg, r)
	// Should make no system calls — invalid path is rejected before any action.
	for _, c := range r.Calls() {
		if strings.Contains(c, "chsh") || strings.Contains(c, "tee") || strings.Contains(c, "sudo") {
			t.Errorf("should not have made call %q for invalid shell path", c)
		}
	}
}

func TestApplyShell_NoRunShellCall(t *testing.T) {
	// Regression: applyShell must never use RunShell for /etc/shells manipulation.
	// Shell string interpolation of cfg.Shell.Default was the injection vector.
	r := testutil.NewFakeRunner()
	cfg := &Config{Shell: ShellConfig{Default: "/bin/zsh"}}
	applyShell(cfg, r)
	for _, c := range r.Calls() {
		if strings.HasPrefix(c, "shell:") {
			t.Errorf("applyShell must not use RunShell (injection risk), got: %q", c)
		}
	}
}

func TestApplySystem_ExpandsScreenshotsDir(t *testing.T) {
	// ExpandHome in FakeRunner maps ~/foo → /home/testuser/foo.
	// Since /home/testuser/Screenshots won't normally exist, MkdirAll creates it.
	// We just verify no panic and the runner doesn't get unexpected calls.
	r := testutil.NewFakeRunner()
	cfg := &Config{
		System: SystemConfig{
			ScreenshotsDir: "~/Screenshots",
		},
	}
	// applySystem calls ExpandHome and then os.MkdirAll — no runner calls expected for this path.
	applySystem(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no runner calls for screenshots dir, got %v", r.Calls())
	}
}

func TestApplySystem_PamTIDEnabled(t *testing.T) {
	tr := true
	r := testutil.NewFakeRunner()
	cfg := &Config{
		System: SystemConfig{PamTID: &tr},
	}
	// applyPamTID checks real files on disk — it will either find pam_tid.so
	// already present or call RunShell. In a test environment the PAM files
	// may not exist, so RunShell gets called. We just verify no panic.
	applySystem(cfg, r)
	// No assertion on exact calls since PAM file state is OS-dependent.
}

func TestApplySystem_RevealLibrary(t *testing.T) {
	tr := true
	r := testutil.NewFakeRunner()
	cfg := &Config{
		System: SystemConfig{RevealLibrary: &tr},
	}
	applySystem(cfg, r)

	found := false
	for _, c := range r.Calls() {
		if len(c) > 16 && c[:16] == "chflags nohidden" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected chflags nohidden in calls, got %v", r.Calls())
	}
}

func TestFileContains_NotExist(t *testing.T) {
	if fileContains("/nonexistent/path/to/file", "anything") {
		t.Error("fileContains on missing file should return false")
	}
}

// ── uninstallSystem ──────────────────────────────────────────────────────────

func TestUninstallSystem_NothingSet(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	uninstallSystem(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when no system config, got %v", r.Calls())
	}
}

func TestUninstallSystem_ShellWarnsOnly(t *testing.T) {
	// Uninstall should not call chsh — just warn.
	r := testutil.NewFakeRunner()
	cfg := &Config{Shell: ShellConfig{Default: "/bin/zsh"}}
	uninstallSystem(cfg, r)

	if r.CalledWith("chsh", "-s", "/bin/zsh") {
		t.Error("uninstall should not call chsh — it only warns")
	}
}

func TestUninstallPamTID_AbsentSkipped(t *testing.T) {
	// This test only runs when pam_tid.so is not already configured on this machine.
	// On developer machines where Touch ID for sudo is enabled the PAM file will
	// contain the line and uninstallPamTID will (correctly) try to remove it.
	if fileContains("/etc/pam.d/sudo_local", "pam_tid.so") || fileContains("/etc/pam.d/sudo", "pam_tid.so") {
		t.Skip("pam_tid.so is present on this machine — test only valid when absent")
	}

	r := testutil.NewFakeRunner()
	uninstallPamTID(r)

	for _, c := range r.Calls() {
		if len(c) > 5 && c[:5] == "shell" {
			t.Errorf("unexpected shell call when pam_tid.so absent: %q", c)
		}
	}
}

// ── buildHidutilJSON ─────────────────────────────────────────────────────────

func TestBuildHidutilJSON_SingleMapping(t *testing.T) {
	mappings := []KeyRemapConfig{{From: "caps_lock", To: "escape"}}
	got, err := buildHidutilJSON(mappings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify both HID values appear in the output.
	if !strings.Contains(got, "30064771129") { // 0x700000039 decimal
		t.Errorf("expected CapsLock HID value in output, got: %s", got)
	}
	if !strings.Contains(got, "30064771113") { // 0x700000029 decimal
		t.Errorf("expected Escape HID value in output, got: %s", got)
	}
}

func TestBuildHidutilJSON_TwoMappings(t *testing.T) {
	mappings := []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
		{From: "left_fn", To: "left_control"},
	}
	got, err := buildHidutilJSON(mappings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "UserKeyMapping") {
		t.Errorf("expected UserKeyMapping key, got: %s", got)
	}
}

func TestBuildHidutilJSON_UnknownFromKey(t *testing.T) {
	mappings := []KeyRemapConfig{{From: "caps_lck", To: "escape"}}
	_, err := buildHidutilJSON(mappings)
	if err == nil {
		t.Error("expected error for unknown From key, got nil")
	}
}

func TestBuildHidutilJSON_UnknownToKey(t *testing.T) {
	mappings := []KeyRemapConfig{{From: "caps_lock", To: "esc"}}
	_, err := buildHidutilJSON(mappings)
	if err == nil {
		t.Error("expected error for unknown To key, got nil")
	}
}

// ── validateKeyRemapping ─────────────────────────────────────────────────────

func TestValidateKeyRemapping_Valid(t *testing.T) {
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
		{From: "left_fn", To: "left_control"},
	}}}
	if errs := validateKeyRemapping(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateKeyRemapping_UnknownKey(t *testing.T) {
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lck", To: "esc"},
	}}}
	errs := validateKeyRemapping(cfg)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors (bad from + bad to), got %d: %v", len(errs), errs)
	}
}

// ── applyKeyRemapping ────────────────────────────────────────────────────────

func TestApplyKeyRemapping_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	applyKeyRemapping(&Config{}, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty config, got: %v", r.Calls())
	}
}

func TestApplyKeyRemapping_UnknownKey(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "not_a_key", To: "escape"},
	}}}
	applyKeyRemapping(cfg, r)
	if r.CalledWriteFile(filepath.Join(r.ExpandHome(keyRemapPlistDir), "com.local.mac-keyremap.plist")) {
		t.Error("should not write plist when key name is unknown")
	}
}

func TestApplyKeyRemapping_WritesAndLoads(t *testing.T) {
	origDir := keyRemapPlistDir
	keyRemapPlistDir = t.TempDir()
	t.Cleanup(func() { keyRemapPlistDir = origDir })

	r := testutil.NewFakeRunner()
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
	}}}
	applyKeyRemapping(cfg, r)

	plistPath := filepath.Join(keyRemapPlistDir, "com.local.mac-keyremap.plist")
	if !r.CalledWriteFile(plistPath) {
		t.Errorf("expected WriteFile call for %s", plistPath)
	}
	if !r.CalledWith("launchctl", "load", plistPath) {
		t.Errorf("expected launchctl load call for %s", plistPath)
	}
}

func TestApplyKeyRemapping_Idempotent(t *testing.T) {
	origDir := keyRemapPlistDir
	tmp := t.TempDir()
	keyRemapPlistDir = tmp
	t.Cleanup(func() { keyRemapPlistDir = origDir })

	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
	}}}

	// Pre-write a plist containing the expected JSON so idempotency check triggers.
	hidutilJSON, _ := buildHidutilJSON(cfg.System.KeyRemapping)
	plistPath := filepath.Join(tmp, "com.local.mac-keyremap.plist")
	os.WriteFile(plistPath, []byte(buildKeyRemapPlist(hidutilJSON)), 0644)

	r := testutil.NewFakeRunner()
	applyKeyRemapping(cfg, r)

	if r.CalledWith("launchctl", "load", plistPath) {
		t.Error("should not reload launchctl when plist is already up to date")
	}
}

// ── uninstallKeyRemapping ────────────────────────────────────────────────────

func TestUninstallKeyRemapping_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	uninstallKeyRemapping(&Config{}, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty config, got: %v", r.Calls())
	}
}

func TestUninstallKeyRemapping_PlistAbsent(t *testing.T) {
	origDir := keyRemapPlistDir
	keyRemapPlistDir = t.TempDir()
	t.Cleanup(func() { keyRemapPlistDir = origDir })

	r := testutil.NewFakeRunner()
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
	}}}
	uninstallKeyRemapping(cfg, r)

	plistPath := filepath.Join(keyRemapPlistDir, "com.local.mac-keyremap.plist")
	if r.CalledWith("launchctl", "unload", plistPath) {
		t.Error("should not call launchctl unload when plist is absent")
	}
}

func TestUninstallKeyRemapping_RemovesAndResets(t *testing.T) {
	origDir := keyRemapPlistDir
	tmp := t.TempDir()
	keyRemapPlistDir = tmp
	t.Cleanup(func() { keyRemapPlistDir = origDir })

	plistPath := filepath.Join(tmp, "com.local.mac-keyremap.plist")
	os.WriteFile(plistPath, []byte("<plist/>"), 0644)

	r := testutil.NewFakeRunner()
	cfg := &Config{System: SystemConfig{KeyRemapping: []KeyRemapConfig{
		{From: "caps_lock", To: "escape"},
	}}}
	uninstallKeyRemapping(cfg, r)

	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Error("expected plist file to be removed")
	}
	if !r.CalledWith("launchctl", "unload", plistPath) {
		t.Error("expected launchctl unload call")
	}
	if !r.CalledWith("hidutil", "property", "--set", `{"UserKeyMapping":[]}`) {
		t.Error("expected hidutil reset call")
	}
}
