package main

import (
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
