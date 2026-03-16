package main

import (
	"errors"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestBuildDefaultsArgs_Bool(t *testing.T) {
	args := buildDefaultsArgs("com.apple.dock", "autohide", true)
	want := []string{"write", "com.apple.dock", "autohide", "-bool", "true"}
	assertStringSliceEqual(t, want, args)

	args = buildDefaultsArgs("com.apple.dock", "autohide", false)
	want = []string{"write", "com.apple.dock", "autohide", "-bool", "false"}
	assertStringSliceEqual(t, want, args)
}

func TestBuildDefaultsArgs_Int64(t *testing.T) {
	args := buildDefaultsArgs("com.apple.dock", "tilesize", int64(48))
	want := []string{"write", "com.apple.dock", "tilesize", "-int", "48"}
	assertStringSliceEqual(t, want, args)
}

func TestBuildDefaultsArgs_Float64_WholeNumber(t *testing.T) {
	// Whole float64 should become -int.
	args := buildDefaultsArgs("com.apple.dock", "tilesize", float64(48.0))
	want := []string{"write", "com.apple.dock", "tilesize", "-int", "48"}
	assertStringSliceEqual(t, want, args)
}

func TestBuildDefaultsArgs_Float64_Fractional(t *testing.T) {
	args := buildDefaultsArgs("com.apple.trackpad", "speed", float64(1.5))
	want := []string{"write", "com.apple.trackpad", "speed", "-float", "1.5"}
	assertStringSliceEqual(t, want, args)
}

func TestBuildDefaultsArgs_String(t *testing.T) {
	args := buildDefaultsArgs("com.apple.dock", "orientation", "bottom")
	want := []string{"write", "com.apple.dock", "orientation", "-string", "bottom"}
	assertStringSliceEqual(t, want, args)
}

func TestBuildDefaultsArgs_UnsupportedType(t *testing.T) {
	args := buildDefaultsArgs("domain", "key", []string{"unsupported"})
	if args != nil {
		t.Errorf("expected nil for unsupported type, got %v", args)
	}
}

func TestApplyDefaults_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	applyDefaults(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty defaults, got %v", r.Calls())
	}
}

func TestApplyDefaults_CallsDefaultsWrite(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("defaults", "write", "com.apple.dock", "autohide", "-bool", "true").Returns("", nil)
	r.When("defaults", "write", "com.apple.dock", "tilesize", "-int", "48").Returns("", nil)

	cfg := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock": {
				"autohide": true,
				"tilesize": int64(48),
			},
		},
	}

	applyDefaults(cfg, r)

	if !r.CalledWith("defaults", "write", "com.apple.dock", "autohide", "-bool", "true") {
		t.Error("expected defaults write for autohide")
	}
	if !r.CalledWith("defaults", "write", "com.apple.dock", "tilesize", "-int", "48") {
		t.Error("expected defaults write for tilesize")
	}
}

func TestApplyDefaults_MultipleDomains(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock":   {"autohide": true},
			"com.apple.finder": {"ShowStatusBar": true},
		},
	}

	applyDefaults(cfg, r)

	if !r.CalledWith("defaults", "write", "com.apple.dock", "autohide", "-bool", "true") {
		t.Error("expected defaults write for dock autohide")
	}
	if !r.CalledWith("defaults", "write", "com.apple.finder", "ShowStatusBar", "-bool", "true") {
		t.Error("expected defaults write for finder ShowStatusBar")
	}
}

// ── uninstallDefaults ────────────────────────────────────────────────────────

func TestUninstallDefaults_DeletesKeys(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock": {
				"autohide": true,
				"tilesize": int64(48),
			},
		},
	}

	uninstallDefaults(cfg, r)

	if !r.CalledWith("defaults", "delete", "com.apple.dock", "autohide") {
		t.Error("expected defaults delete for autohide")
	}
	if !r.CalledWith("defaults", "delete", "com.apple.dock", "tilesize") {
		t.Error("expected defaults delete for tilesize")
	}
}

func TestUninstallDefaults_AbsentKeyHandledGracefully(t *testing.T) {
	r := testutil.NewFakeRunner()
	// Simulate defaults delete returning non-zero (key already absent).
	r.When("defaults", "delete", "com.apple.dock", "autohide").Returns("", errors.New("exit status 1"))

	cfg := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock": {"autohide": true},
		},
	}

	// Should not panic — absent key is treated as a skip.
	uninstallDefaults(cfg, r)
}

func TestUninstallDefaults_MultipleDomains(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock":   {"autohide": true},
			"com.apple.finder": {"ShowStatusBar": false},
		},
	}

	uninstallDefaults(cfg, r)

	if !r.CalledWith("defaults", "delete", "com.apple.dock", "autohide") {
		t.Error("expected defaults delete for dock autohide")
	}
	if !r.CalledWith("defaults", "delete", "com.apple.finder", "ShowStatusBar") {
		t.Error("expected defaults delete for finder ShowStatusBar")
	}
}

func TestUninstallDefaults_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	uninstallDefaults(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty defaults config, got %v", r.Calls())
	}
}

// assertStringSliceEqual is a helper to compare string slices.
func assertStringSliceEqual(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("length mismatch: want %d %v, got %d %v", len(want), want, len(got), got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("index %d: want %q, got %q", i, want[i], got[i])
		}
	}
}
