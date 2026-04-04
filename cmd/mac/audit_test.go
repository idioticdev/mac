package main

import (
	"fmt"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// ── untrackedItems ────────────────────────────────────────────────────────────

func TestUntrackedItems_AllTracked(t *testing.T) {
	managed := map[string]bool{"git": true, "jq": true}
	got := untrackedItems("git\njq\n", managed)
	if len(got) != 0 {
		t.Errorf("expected no untracked, got %v", got)
	}
}

func TestUntrackedItems_SomeUntracked(t *testing.T) {
	managed := map[string]bool{"git": true}
	got := untrackedItems("git\njq\ncurl\n", managed)
	if len(got) != 2 {
		t.Errorf("expected 2 untracked, got %v", got)
	}
	if got[0] != "curl" || got[1] != "jq" {
		t.Errorf("expected sorted [curl jq], got %v", got)
	}
}

func TestUntrackedItems_EmptyInstalled(t *testing.T) {
	managed := map[string]bool{"git": true}
	got := untrackedItems("", managed)
	if len(got) != 0 {
		t.Errorf("expected no untracked for empty installed, got %v", got)
	}
}

// ── expectedDefaultsReadValue ─────────────────────────────────────────────────

func TestExpectedDefaultsReadValue_Bool(t *testing.T) {
	if got := expectedDefaultsReadValue(true); got != "1" {
		t.Errorf("want 1, got %q", got)
	}
	if got := expectedDefaultsReadValue(false); got != "0" {
		t.Errorf("want 0, got %q", got)
	}
}

func TestExpectedDefaultsReadValue_Int(t *testing.T) {
	if got := expectedDefaultsReadValue(int64(42)); got != "42" {
		t.Errorf("want 42, got %q", got)
	}
}

func TestExpectedDefaultsReadValue_FloatWhole(t *testing.T) {
	if got := expectedDefaultsReadValue(float64(3.0)); got != "3" {
		t.Errorf("want 3, got %q", got)
	}
}

func TestExpectedDefaultsReadValue_FloatFractional(t *testing.T) {
	if got := expectedDefaultsReadValue(float64(1.5)); got != "1.5" {
		t.Errorf("want 1.5, got %q", got)
	}
}

func TestExpectedDefaultsReadValue_String(t *testing.T) {
	if got := expectedDefaultsReadValue("hello"); got != "hello" {
		t.Errorf("want hello, got %q", got)
	}
}

func TestExpectedDefaultsReadValue_UnsupportedType(t *testing.T) {
	if got := expectedDefaultsReadValue([]int{1, 2}); got != "" {
		t.Errorf("want empty for unsupported type, got %q", got)
	}
}

// ── auditFormulae ─────────────────────────────────────────────────────────────

func TestAuditFormulae_AllTracked(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("git\njq", nil)
	cfg := &Config{Packages: PackagesConfig{Formulae: []string{"git", "jq"}}}
	if got := auditFormulae(cfg, r); got != 0 {
		t.Errorf("expected 0 drift, got %d", got)
	}
}

func TestAuditFormulae_Untracked(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("git\njq\ncurl", nil)
	cfg := &Config{Packages: PackagesConfig{Formulae: []string{"git"}}}
	if got := auditFormulae(cfg, r); got != 2 {
		t.Errorf("expected 2 drift (jq, curl), got %d", got)
	}
}

func TestAuditFormulae_NothingInstalled(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	cfg := &Config{}
	if got := auditFormulae(cfg, r); got != 0 {
		t.Errorf("expected 0 drift when nothing installed and nothing in config, got %d", got)
	}
}

// ── auditCasks ────────────────────────────────────────────────────────────────

func TestAuditCasks_Untracked(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--cask", "-1").Returns("1password\niterm2", nil)
	cfg := &Config{Packages: PackagesConfig{Casks: []string{"1password"}}}
	if got := auditCasks(cfg, r); got != 1 {
		t.Errorf("expected 1 drift (iterm2), got %d", got)
	}
}

func TestAuditCasks_AllTracked(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--cask", "-1").Returns("iterm2", nil)
	cfg := &Config{Packages: PackagesConfig{Casks: []string{"iterm2"}}}
	if got := auditCasks(cfg, r); got != 0 {
		t.Errorf("expected 0 drift, got %d", got)
	}
}

// ── auditMAS ──────────────────────────────────────────────────────────────────

func TestAuditMAS_NoMasCLI(t *testing.T) {
	r := testutil.NewFakeRunner()
	// mas not in which map → returns false
	cfg := &Config{}
	if got := auditMAS(cfg, r); got != 0 {
		t.Errorf("expected 0 when mas not installed, got %d", got)
	}
}

func TestAuditMAS_Untracked(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("497799835 Xcode\n409183694 Keynote", nil)
	cfg := &Config{MAS: MASConfig{Apps: []MASApp{{ID: 497799835, Name: "Xcode"}}}}
	if got := auditMAS(cfg, r); got != 1 {
		t.Errorf("expected 1 drift (Keynote untracked), got %d", got)
	}
}

func TestAuditMAS_AllTracked(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("497799835 Xcode", nil)
	cfg := &Config{MAS: MASConfig{Apps: []MASApp{{ID: 497799835, Name: "Xcode"}}}}
	if got := auditMAS(cfg, r); got != 0 {
		t.Errorf("expected 0 drift, got %d", got)
	}
}

// ── auditDefaults ─────────────────────────────────────────────────────────────

func TestAuditDefaults_NoDrift(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("defaults", "read", "NSGlobalDomain", "NSAutomaticSpellingCorrectionEnabled").Returns("0", nil)
	cfg := &Config{
		Defaults: map[string]map[string]any{
			"NSGlobalDomain": {"NSAutomaticSpellingCorrectionEnabled": false},
		},
	}
	if got := auditDefaults(cfg, r); got != 0 {
		t.Errorf("expected 0 drift, got %d", got)
	}
}

func TestAuditDefaults_Drifted(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("defaults", "read", "NSGlobalDomain", "NSAutomaticSpellingCorrectionEnabled").Returns("1", nil)
	cfg := &Config{
		Defaults: map[string]map[string]any{
			"NSGlobalDomain": {"NSAutomaticSpellingCorrectionEnabled": false},
		},
	}
	if got := auditDefaults(cfg, r); got != 1 {
		t.Errorf("expected 1 drift, got %d", got)
	}
}

func TestAuditDefaults_KeyMissing(t *testing.T) {
	r := testutil.NewFakeRunner()
	// defaults read returns error when key absent — FakeRunner returns "", nil by default.
	// To simulate a missing key we need an error return.
	r.When("defaults", "read", "NSGlobalDomain", "SomeKey").Returns("", fmt.Errorf("not found"))
	cfg := &Config{
		Defaults: map[string]map[string]any{
			"NSGlobalDomain": {"SomeKey": "expected-value"},
		},
	}
	if got := auditDefaults(cfg, r); got != 1 {
		t.Errorf("expected 1 drift for missing key, got %d", got)
	}
}

func TestAuditDefaults_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	if got := auditDefaults(cfg, r); got != 0 {
		t.Errorf("expected 0 drift for empty defaults config, got %d", got)
	}
}

// ── runAudit ──────────────────────────────────────────────────────────────────

func TestRunAudit_CleanReturnsTrue(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	cfg := &Config{}
	if !runAudit(cfg, r) {
		t.Error("expected true (no drift) for empty system + empty config")
	}
}

func TestRunAudit_DriftReturnsFalse(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("untracked-pkg", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	cfg := &Config{}
	if runAudit(cfg, r) {
		t.Error("expected false (drift found) when untracked package installed")
	}
}
