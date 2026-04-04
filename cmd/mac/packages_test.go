package main

import (
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestInstallFormulae_AlreadyInstalled(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\nfd\n", nil)

	cfg := &Config{
		Packages: PackagesConfig{
			Formulae: []string{"git", "ripgrep"},
		},
	}

	InstallFormulae(cfg, r)

	// brew install should NOT have been called since everything is installed.
	if r.CalledWith("brew", "install", "git") {
		t.Error("should not have called brew install git")
	}
	if r.CalledWith("brew", "install", "ripgrep") {
		t.Error("should not have called brew install ripgrep")
	}
}

func TestInstallFormulae_NewPackage(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("git\n", nil)

	cfg := &Config{
		Packages: PackagesConfig{
			Formulae: []string{"git", "ripgrep"},
		},
	}

	InstallFormulae(cfg, r)

	if !r.CalledWith("brew", "install", "ripgrep") {
		t.Error("expected brew install ripgrep to be called")
	}
	// git is already installed, no install call.
	if r.CalledWith("brew", "install", "git") {
		t.Error("should not have called brew install git (already installed)")
	}
}

func TestInstallFormulae_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	InstallFormulae(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls, got %v", r.Calls())
	}
}

func TestInstallCasks_AlreadyInstalled(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\nraycast\n", nil)

	cfg := &Config{
		Packages: PackagesConfig{
			Casks: []string{"wezterm", "raycast"},
		},
	}

	InstallCasks(cfg, r)

	if r.CalledWith("brew", "install", "--cask", "wezterm") {
		t.Error("should not have called brew install --cask wezterm (already installed)")
	}
}

func TestInstallCasks_NewCask(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\n", nil)

	cfg := &Config{
		Packages: PackagesConfig{
			Casks: []string{"wezterm", "raycast"},
		},
	}

	InstallCasks(cfg, r)

	if !r.CalledWith("brew", "install", "--cask", "raycast") {
		t.Error("expected brew install --cask raycast to be called")
	}
}

func TestInstallCasks_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	InstallCasks(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls, got %v", r.Calls())
	}
}

func TestInstallTaps_NewTap(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "tap").Returns("homebrew/core\n", nil)

	cfg := &Config{
		Packages: PackagesConfig{
			Taps: []string{"homebrew/core", "homebrew/cask-fonts"},
		},
	}

	InstallTaps(cfg, r)

	// homebrew/core already tapped, homebrew/cask-fonts is new.
	if r.CalledWith("brew", "tap", "homebrew/core") {
		t.Error("should not re-tap homebrew/core")
	}
	if !r.CalledWith("brew", "tap", "homebrew/cask-fonts") {
		t.Error("expected brew tap homebrew/cask-fonts")
	}
}

func TestInstallTaps_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	InstallTaps(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls, got %v", r.Calls())
	}
}

func TestInstallMAS_AlreadyInstalled(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("497799835 Xcode\n", nil)

	cfg := &Config{
		MAS: MASConfig{
			Apps: []MASApp{{ID: 497799835, Name: "Xcode"}},
		},
	}

	InstallMAS(cfg, r)

	if r.CalledWith("mas", "install", "497799835") {
		t.Error("should not install Xcode (already installed)")
	}
}

func TestInstallMAS_NewApp(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("", nil)

	cfg := &Config{
		MAS: MASConfig{
			Apps: []MASApp{{ID: 497799835, Name: "Xcode"}},
		},
	}

	InstallMAS(cfg, r)

	if !r.CalledWith("mas", "install", "497799835") {
		t.Error("expected mas install 497799835 to be called")
	}
}

func TestInstallMAS_InstallsMasCLI(t *testing.T) {
	// mas not present → should install it via brew.
	r := testutil.NewFakeRunner().WhenWhich("mas", false)
	r.When("mas", "list").Returns("", nil)

	cfg := &Config{
		MAS: MASConfig{
			Apps: []MASApp{{ID: 1, Name: "TestApp"}},
		},
	}

	InstallMAS(cfg, r)

	if !r.CalledWith("brew", "install", "mas") {
		t.Error("expected brew install mas when mas not present")
	}
}

func TestInstallMAS_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	InstallMAS(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls, got %v", r.Calls())
	}
}

func TestEnsureHomebrew_AlreadyInstalled(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("brew", true)
	r.When("brew", "update", "--quiet").Returns("", nil)

	EnsureHomebrew(r)

	if r.CalledWith("/bin/bash", "-c",
		`eval "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`) {
		t.Error("should not run homebrew install script when brew already present")
	}
	if !r.CalledWith("brew", "update", "--quiet") {
		t.Error("expected brew update --quiet to always run")
	}
}

func TestToSet(t *testing.T) {
	out := "git\nripgrep\nfd\n"
	s := toSet(out)
	for _, want := range []string{"git", "ripgrep", "fd"} {
		if !s[want] {
			t.Errorf("expected %q in set", want)
		}
	}
	if s[""] {
		t.Error("empty string should not be in set")
	}
}

func TestToSet_Empty(t *testing.T) {
	s := toSet("")
	if len(s) != 0 {
		t.Errorf("expected empty set, got %v", s)
	}
}

// ── uninstallPackages ────────────────────────────────────────────────────────

func TestUninstallPackages_InstalledFormula(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\n", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("", nil)

	cfg := &Config{Packages: PackagesConfig{Formulae: []string{"git", "ripgrep"}}}
	uninstallPackages(cfg, r)

	if !r.CalledWith("brew", "uninstall", "git") {
		t.Error("expected brew uninstall git")
	}
	if !r.CalledWith("brew", "uninstall", "ripgrep") {
		t.Error("expected brew uninstall ripgrep")
	}
}

func TestUninstallPackages_NotInstalledFormulaSkipped(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("", nil)

	cfg := &Config{Packages: PackagesConfig{Formulae: []string{"git"}}}
	uninstallPackages(cfg, r)

	if r.CalledWith("brew", "uninstall", "git") {
		t.Error("should not uninstall git when not installed")
	}
}

func TestUninstallPackages_InstalledCask(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\n", nil)
	r.When("brew", "tap").Returns("", nil)

	cfg := &Config{Packages: PackagesConfig{Casks: []string{"wezterm"}}}
	uninstallPackages(cfg, r)

	if !r.CalledWith("brew", "uninstall", "--cask", "wezterm") {
		t.Error("expected brew uninstall --cask wezterm")
	}
}

func TestUninstallPackages_TappedTap(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("homebrew/cask-fonts\n", nil)

	cfg := &Config{Packages: PackagesConfig{Taps: []string{"homebrew/cask-fonts"}}}
	uninstallPackages(cfg, r)

	if !r.CalledWith("brew", "untap", "homebrew/cask-fonts") {
		t.Error("expected brew untap homebrew/cask-fonts")
	}
}

func TestUninstallPackages_UntappedTapSkipped(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("brew", "tap").Returns("homebrew/core\n", nil)

	cfg := &Config{Packages: PackagesConfig{Taps: []string{"homebrew/cask-fonts"}}}
	uninstallPackages(cfg, r)

	if r.CalledWith("brew", "untap", "homebrew/cask-fonts") {
		t.Error("should not untap homebrew/cask-fonts when not tapped")
	}
}

func TestUninstallPackages_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	uninstallPackages(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty packages config, got %v", r.Calls())
	}
}

// ── uninstallMAS ─────────────────────────────────────────────────────────────

func TestUninstallMAS_InstalledApp(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("497799835 Xcode 15.0\n", nil)

	cfg := &Config{MAS: MASConfig{Apps: []MASApp{{ID: 497799835, Name: "Xcode"}}}}
	uninstallMAS(cfg, r)

	if !r.CalledWith("mas", "uninstall", "497799835") {
		t.Error("expected mas uninstall 497799835")
	}
}

func TestUninstallMAS_NotInstalledSkipped(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", true)
	r.When("mas", "list").Returns("", nil)

	cfg := &Config{MAS: MASConfig{Apps: []MASApp{{ID: 497799835, Name: "Xcode"}}}}
	uninstallMAS(cfg, r)

	if r.CalledWith("mas", "uninstall", "497799835") {
		t.Error("should not uninstall Xcode when not installed")
	}
}

func TestUninstallMAS_MasCliMissing(t *testing.T) {
	r := testutil.NewFakeRunner().WhenWhich("mas", false)

	cfg := &Config{MAS: MASConfig{Apps: []MASApp{{ID: 497799835, Name: "Xcode"}}}}
	uninstallMAS(cfg, r)

	if r.CalledWith("mas", "uninstall", "497799835") {
		t.Error("should not attempt uninstall when mas CLI is missing")
	}
}

func TestUninstallMAS_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	uninstallMAS(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty MAS config, got %v", r.Calls())
	}
}
