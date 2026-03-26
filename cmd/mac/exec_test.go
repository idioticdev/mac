package main

import (
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// TestIsReadOnly verifies the allowlist-based command classification.
func TestIsReadOnly(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"brew", []string{"list", "--formula", "-1"}, true},
		{"brew", []string{"list", "--cask", "-1"}, true},
		{"brew", []string{"tap"}, true},                // list taps — read-only
		{"brew", []string{"tap", "homebrew/core"}, false}, // add tap — mutating
		{"brew", []string{"uninstall", "git"}, false},
		{"brew", []string{"untap", "homebrew/core"}, false},
		{"brew", []string{"install", "git"}, false},
		{"mas", []string{"list"}, true},
		{"mas", []string{"uninstall", "497799835"}, false},
		{"mas", []string{"install", "497799835"}, false},
		{"defaults", []string{"read", "com.apple.dock", "autohide"}, true},
		{"defaults", []string{"delete", "com.apple.dock", "autohide"}, false},
		{"defaults", []string{"write", "com.apple.dock", "autohide", "-bool", "true"}, false},
		{"rm", []string{"-rf", "/tmp/foo"}, false},
		{"stow", []string{"-D", "zsh"}, false},
		{"sudo", []string{"scutil", "--set", "ComputerName", "mac"}, false},
	}

	for _, tc := range cases {
		got := isReadOnly(tc.name, tc.args)
		if got != tc.want {
			t.Errorf("isReadOnly(%q, %v) = %v, want %v", tc.name, tc.args, got, tc.want)
		}
	}
}

// TestDryRunRunner_ReadOnlyPassesThrough verifies that read-only commands
// reach the inner runner and return real output.
func TestDryRunRunner_ReadOnlyPassesThrough(t *testing.T) {
	inner := testutil.NewFakeRunner()
	inner.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\n", nil)
	inner.When("brew", "tap").Returns("homebrew/core\n", nil)
	inner.When("mas", "list").Returns("497799835 Xcode\n", nil)

	d := NewDryRunRunner(inner)

	out, err := d.Run("brew", "list", "--formula", "-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Error("expected brew list output to pass through to inner runner")
	}
	if !inner.CalledWith("brew", "list", "--formula", "-1") {
		t.Error("expected inner runner to be called for brew list")
	}

	d.Run("brew", "tap")
	if !inner.CalledWith("brew", "tap") {
		t.Error("expected inner runner to be called for brew tap (list)")
	}
}

// TestDryRunRunner_MutatingCommandsSkipped verifies that mutating commands
// do not reach the inner runner.
func TestDryRunRunner_MutatingCommandsSkipped(t *testing.T) {
	inner := testutil.NewFakeRunner()
	d := NewDryRunRunner(inner)

	d.Run("brew", "uninstall", "git")
	d.Run("brew", "untap", "homebrew/core")
	d.Run("brew", "tap", "homebrew/core") // mutating form
	d.Run("mas", "uninstall", "497799835")
	d.Run("defaults", "delete", "com.apple.dock", "autohide")
	d.Run("rm", "-rf", "/tmp/dotfiles")
	d.RunSudo("scutil", "--set", "ComputerName", "newname")
	d.RunPassthrough("brew", "uninstall", "git")
	d.RunShell("sudo sed -i '' '/pam_tid/d' /etc/pam.d/sudo_local")

	if inner.CalledWith("brew", "uninstall", "git") {
		t.Error("brew uninstall should not reach inner runner in dry-run mode")
	}
	if inner.CalledWith("brew", "untap", "homebrew/core") {
		t.Error("brew untap should not reach inner runner in dry-run mode")
	}
	if inner.CalledWith("brew", "tap", "homebrew/core") {
		t.Error("brew tap <name> should not reach inner runner in dry-run mode")
	}
	if inner.CalledWith("rm", "-rf", "/tmp/dotfiles") {
		t.Error("rm -rf should not reach inner runner in dry-run mode")
	}
}

// TestDryRunRunner_WhichAndExpandHomePassThrough verifies that non-mutating
// Runner methods always delegate to the inner runner.
func TestDryRunRunner_WhichAndExpandHomePassThrough(t *testing.T) {
	inner := testutil.NewFakeRunner()
	inner.WhenWhich("stow", true)

	d := NewDryRunRunner(inner)

	if !d.Which("stow") {
		t.Error("Which should pass through to inner runner")
	}
	expanded := d.ExpandHome("~/dotfiles")
	if expanded != "/home/testuser/dotfiles" {
		t.Errorf("ExpandHome should pass through to inner runner, got %q", expanded)
	}
}

// TestHumanSummary verifies that key command/arg combinations produce
// user-friendly descriptions rather than raw command strings.
func TestHumanSummary(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"brew", []string{"install", "neovim"}, "would install formula: neovim"},
		{"brew", []string{"install", "--cask", "arc"}, "would install cask: arc"},
		{"brew", []string{"uninstall", "git"}, "would uninstall formula: git"},
		{"brew", []string{"uninstall", "--cask", "arc"}, "would uninstall cask: arc"},
		{"brew", []string{"tap", "homebrew/core"}, "would tap: homebrew/core"},
		{"brew", []string{"untap", "homebrew/core"}, "would untap: homebrew/core"},
		{"brew", []string{"update", "--quiet"}, "would update Homebrew"},
		{"mas", []string{"install", "497799835"}, "would install MAS app: 497799835"},
		{"mas", []string{"uninstall", "497799835"}, "would uninstall MAS app: 497799835"},
		{"defaults", []string{"write", "com.apple.dock", "autohide", "-bool", "true"},
			"would set com.apple.dock autohide → true"},
		{"defaults", []string{"delete", "com.apple.dock", "autohide"},
			"would delete default com.apple.dock autohide"},
		{"git", []string{"clone", "https://github.com/user/dotfiles", "/tmp/dotfiles"},
			"would clone dotfiles: https://github.com/user/dotfiles"},
		{"stow", []string{"-d", "/tmp/dotfiles", "-t", "/home/user", "--restow", "zsh"},
			"would stow: zsh"},
		{"chsh", []string{"-s", "/bin/zsh"}, "would change default shell: /bin/zsh"},
		{"killall", []string{"Dock"}, "would restart service: Dock"},
	}

	for _, tc := range cases {
		got := humanSummary(tc.name, tc.args)
		if got != tc.want {
			t.Errorf("humanSummary(%q, %v)\n  got:  %q\n  want: %q",
				tc.name, tc.args, got, tc.want)
		}
	}
}

// TestHumanSudoSummary verifies scutil --set produces a clean description.
func TestHumanSudoSummary(t *testing.T) {
	got := humanSudoSummary("scutil", []string{"--set", "ComputerName", "my-mac"})
	want := "would set ComputerName: my-mac"
	if got != want {
		t.Errorf("humanSudoSummary got %q, want %q", got, want)
	}
}

// TestDryRunRunner_SudoVSkipped verifies that sudo -v is silently no-op'd
// (no log noise, no error) since authentication is irrelevant in dry-run.
func TestDryRunRunner_SudoVSkipped(t *testing.T) {
	inner := testutil.NewFakeRunner()
	d := NewDryRunRunner(inner)
	if err := d.RunPassthrough("sudo", "-v"); err != nil {
		t.Errorf("RunPassthrough(sudo, -v) should return nil in dry-run, got %v", err)
	}
	// Inner runner must NOT be called.
	if inner.CalledWith("sudo", "-v") {
		t.Error("sudo -v should not reach inner runner in dry-run")
	}
}

// TestDryRunRunner_MutatingCallsReturnEmpty verifies that skipped mutating
// calls return empty output and nil error (so callers treat dry-run as success).
func TestDryRunRunner_MutatingCallsReturnEmpty(t *testing.T) {
	inner := testutil.NewFakeRunner()
	d := NewDryRunRunner(inner)

	out, err := d.Run("brew", "uninstall", "git")
	if err != nil {
		t.Errorf("mutating Run should return nil error in dry-run, got %v", err)
	}
	if out != "" {
		t.Errorf("mutating Run should return empty output in dry-run, got %q", out)
	}

	if err := d.RunPassthrough("brew", "uninstall", "git"); err != nil {
		t.Errorf("mutating RunPassthrough should return nil error in dry-run, got %v", err)
	}
}
