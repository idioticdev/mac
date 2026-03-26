package main

import (
	"strings"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestGenerateExportTOML_EmptyBrew(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("scutil", "--get", "ComputerName").Returns("Test Mac", nil)
	r.When("scutil", "--get", "LocalHostName").Returns("test-mac", nil)
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("", nil)

	out := generateExportTOML(r)

	if !strings.Contains(out, `computer_name  = "Test Mac"`) {
		t.Errorf("expected computer_name in output, got:\n%s", out)
	}
	if !strings.Contains(out, `local_hostname = "test-mac"`) {
		t.Errorf("expected local_hostname in output, got:\n%s", out)
	}
	if !strings.Contains(out, "formulae = []") {
		t.Errorf("expected empty formulae = [], got:\n%s", out)
	}
	if !strings.Contains(out, "casks = []") {
		t.Errorf("expected empty casks = [], got:\n%s", out)
	}
	// Dotfiles stub must be present (commented out)
	if !strings.Contains(out, "# [dotfiles]") {
		t.Errorf("expected dotfiles stub in output")
	}
	// nix-darwin note must be present
	if !strings.Contains(out, "nix-darwin") {
		t.Errorf("expected nix-darwin note in output")
	}
}

func TestGenerateExportTOML_WithPackages(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("scutil", "--get", "ComputerName").Returns("My Machine", nil)
	r.When("scutil", "--get", "LocalHostName").Returns("my-machine", nil)
	r.When("brew", "list", "--formula", "-1").Returns("git\nneovim\nripgrep\n", nil)
	r.When("brew", "list", "--cask", "-1").Returns("arc\nraycast\n", nil)
	r.When("brew", "tap").Returns("homebrew/core\n", nil)

	out := generateExportTOML(r)

	for _, f := range []string{"git", "neovim", "ripgrep"} {
		if !strings.Contains(out, `"`+f+`"`) {
			t.Errorf("expected formula %q in output", f)
		}
	}
	for _, c := range []string{"arc", "raycast"} {
		if !strings.Contains(out, `"`+c+`"`) {
			t.Errorf("expected cask %q in output", c)
		}
	}
	if !strings.Contains(out, `"homebrew/core"`) {
		t.Errorf("expected tap homebrew/core in output")
	}
}

func TestGenerateExportTOML_FallbackHostname(t *testing.T) {
	// When scutil fails, should fall back to hostname -s
	r := testutil.NewFakeRunner()
	r.When("scutil", "--get", "ComputerName").Returns("", nil)
	r.When("scutil", "--get", "LocalHostName").Returns("", nil)
	r.When("hostname", "-s").Returns("fallback-host", nil)
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("", nil)

	out := generateExportTOML(r)

	if !strings.Contains(out, `"fallback-host"`) {
		t.Errorf("expected fallback hostname in output, got:\n%s", out)
	}
}

func TestExportLines_Empty(t *testing.T) {
	result := exportLines("")
	if len(result) != 0 {
		t.Errorf("exportLines(\"\") = %v, want empty slice", result)
	}
}

func TestExportLines_Whitespace(t *testing.T) {
	result := exportLines("  \n  \n  ")
	if len(result) != 0 {
		t.Errorf("exportLines(whitespace) = %v, want empty slice", result)
	}
}

func TestExportLines_Normal(t *testing.T) {
	result := exportLines("git\nneovim\nripgrep\n")
	if len(result) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(result), result)
	}
	if result[0] != "git" || result[1] != "neovim" || result[2] != "ripgrep" {
		t.Errorf("unexpected lines: %v", result)
	}
}
