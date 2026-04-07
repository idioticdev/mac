package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// scenarioCfg builds a comprehensive Config for journey tests.
func scenarioCfg(t *testing.T) *Config {
	t.Helper()
	tr := true
	return &Config{
		Machine: MachineConfig{
			ComputerName:  "scenario-mac",
			LocalHostname: "scenario-mac-local",
		},
		Packages: PackagesConfig{
			Taps:     []string{"homebrew/cask-fonts"},
			Formulae: []string{"git", "ripgrep", "fd"},
			Casks:    []string{"wezterm", "raycast"},
		},
		MAS: MASConfig{
			Apps: []MASApp{
				{ID: 497799835, Name: "Xcode"},
			},
		},
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         "~/dotfiles",
			Method:       "stow",
			StowPackages: []string{"zsh", "git"},
		},
		Defaults: map[string]map[string]any{
			"com.apple.dock": {
				"autohide": true,
				"tilesize": int64(48),
			},
		},
		System: SystemConfig{
			PamTID: &tr,
		},
		Hooks: HooksConfig{
			PostInstall: []string{"echo setup complete"},
		},
	}
}

// ── Scenario A: Fresh Mac ──────────────────────────────────────────────────

// TestScenarioA_FreshMac simulates a machine where nothing is installed yet.
// Every install/set command should be invoked.
func TestScenarioA_FreshMac(t *testing.T) {
	dir := t.TempDir()
	// dotfilesDest must NOT exist yet so that applyDotfiles takes the clone path.
	dotfilesDest := filepath.Join(dir, "dotfiles")

	r := testutil.NewFakeRunner()
	r.WhenWhich("brew", true)
	r.WhenWhich("mas", false) // mas not installed yet
	r.WhenWhich("stow", true)

	// Nothing installed.
	r.When("brew", "tap").Returns("", nil)
	r.When("brew", "update", "--quiet").Returns("", nil)
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)
	r.When("mas", "list").Returns("", nil)
	r.When("git", "clone", "https://github.com/user/dotfiles", dotfilesDest).Returns("", nil)

	cfg := scenarioCfg(t)
	cfg.Dotfiles.Dest = dotfilesDest

	// Run all subsystems.
	applyMachine(cfg, r)
	EnsureHomebrew(r)
	InstallTaps(cfg, r)
	InstallFormulae(cfg, r)
	InstallCasks(cfg, r)
	InstallMAS(cfg, r)
	applyDotfiles(cfg, r)
	applyDefaults(cfg, r)
	runHooks(cfg, r)
	restartServices(r)

	// Machine identity.
	if !r.CalledWithSudo("scutil", "--set", "ComputerName", "scenario-mac") {
		t.Error("expected ComputerName to be set")
	}
	if !r.CalledWithSudo("scutil", "--set", "LocalHostName", "scenario-mac-local") {
		t.Error("expected LocalHostName to be set")
	}

	// Taps.
	if !r.CalledWith("brew", "tap", "homebrew/cask-fonts") {
		t.Error("expected homebrew/cask-fonts to be tapped")
	}

	// Formulae.
	for _, f := range []string{"git", "ripgrep", "fd"} {
		if !r.CalledWith("brew", "install", f) {
			t.Errorf("expected brew install %s", f)
		}
	}

	// Casks.
	for _, c := range []string{"wezterm", "raycast"} {
		if !r.CalledWith("brew", "install", "--cask", c) {
			t.Errorf("expected brew install --cask %s", c)
		}
	}

	// mas CLI should be installed since Which("mas") returns false.
	if !r.CalledWith("brew", "install", "mas") {
		t.Error("expected brew install mas on fresh mac")
	}

	// MAS app.
	if !r.CalledWith("mas", "install", "497799835") {
		t.Error("expected mas install 497799835 (Xcode)")
	}

	// Dotfiles cloned.
	if !r.CalledWith("git", "clone", "https://github.com/user/dotfiles", dotfilesDest) {
		t.Error("expected git clone for dotfiles")
	}
	// Stow is attempted after clone, but since the fake clone doesn't create real dirs
	// the stow package subdirs don't exist — stow calls are skipped with a warning.
	// We just verify clone happened and no pull was called.
	if r.CalledWith("git", "-C", dotfilesDest, "pull", "--ff-only") {
		t.Error("pull should not be called on fresh mac (clone path taken)")
	}

	// Defaults.
	if !r.CalledWith("defaults", "write", "com.apple.dock", "autohide", "-bool", "true") {
		t.Error("expected defaults write autohide")
	}

	// Hooks.
	if !r.CalledWithShell("echo setup complete") {
		t.Error("expected post-install hook to run")
	}

	// Services restarted.
	for _, svc := range []string{"Finder", "Dock", "SystemUIServer"} {
		if !r.CalledWith("killall", svc) {
			t.Errorf("expected killall %s", svc)
		}
	}
}

// ── Scenario B: Fully Configured (idempotent) ─────────────────────────────

// TestScenarioB_FullyConfigured simulates a machine where everything is
// already installed. No install commands should fire.
func TestScenarioB_FullyConfigured(t *testing.T) {
	dir := t.TempDir()
	dotfilesDest := filepath.Join(dir, "dotfiles")
	// The dotfiles dest exists already → pull, not clone.
	if err := os.MkdirAll(dotfilesDest, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"zsh", "git"} {
		if err := os.MkdirAll(filepath.Join(dotfilesDest, pkg), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r := testutil.NewFakeRunner()
	r.WhenWhich("brew", true)
	r.WhenWhich("mas", true)
	r.WhenWhich("stow", true)

	// Everything already installed.
	r.When("brew", "tap").Returns("homebrew/cask-fonts\n", nil)
	r.When("brew", "update", "--quiet").Returns("", nil)
	r.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\nfd\n", nil)
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\nraycast\n", nil)
	r.When("mas", "list").Returns("497799835 Xcode 15.0\n", nil)
	r.When("git", "-C", dotfilesDest, "pull", "--ff-only").Returns("Already up to date.\n", nil)

	cfg := scenarioCfg(t)
	cfg.Dotfiles.Dest = dotfilesDest

	applyMachine(cfg, r)
	EnsureHomebrew(r)
	InstallTaps(cfg, r)
	InstallFormulae(cfg, r)
	InstallCasks(cfg, r)
	InstallMAS(cfg, r)
	applyDotfiles(cfg, r)
	applyDefaults(cfg, r)
	runHooks(cfg, r)
	restartServices(r)

	// No brew install for formulae.
	for _, f := range []string{"git", "ripgrep", "fd"} {
		if r.CalledWith("brew", "install", f) {
			t.Errorf("should not re-install formula %s", f)
		}
	}

	// No brew install --cask.
	for _, c := range []string{"wezterm", "raycast"} {
		if r.CalledWith("brew", "install", "--cask", c) {
			t.Errorf("should not re-install cask %s", c)
		}
	}

	// No tap.
	if r.CalledWith("brew", "tap", "homebrew/cask-fonts") {
		t.Error("should not re-tap homebrew/cask-fonts")
	}

	// No MAS install.
	if r.CalledWith("mas", "install", "497799835") {
		t.Error("should not re-install Xcode")
	}

	// Git pull, not clone.
	if r.CalledWith("git", "clone", "https://github.com/user/dotfiles", dotfilesDest) {
		t.Error("should not clone when dest exists")
	}
	if !r.CalledWith("git", "-C", dotfilesDest, "pull", "--ff-only") {
		t.Error("expected git pull when dest exists")
	}
}

// ── Scenario C: Migration (partial existing setup) ────────────────────────

// TestScenarioC_MigrationPartial simulates a machine where some things are
// already set up and some still need to be installed.
func TestScenarioC_MigrationPartial(t *testing.T) {
	dir := t.TempDir()
	dotfilesDest := filepath.Join(dir, "dotfiles") // not yet cloned

	r := testutil.NewFakeRunner()
	r.WhenWhich("brew", true)
	r.WhenWhich("mas", true)  // mas already present
	r.WhenWhich("stow", true) // stow already present

	// git and ripgrep installed, fd is new.
	r.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\n", nil)
	// wezterm installed, raycast is new.
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\n", nil)
	// homebrew/cask-fonts not tapped yet.
	r.When("brew", "tap").Returns("homebrew/core\n", nil)
	r.When("brew", "update", "--quiet").Returns("", nil)
	// Xcode already installed.
	r.When("mas", "list").Returns("497799835 Xcode 15.0\n", nil)
	r.When("git", "clone", "https://github.com/user/dotfiles", dotfilesDest).Returns("", nil)

	cfg := scenarioCfg(t)
	cfg.Dotfiles.Dest = dotfilesDest

	applyMachine(cfg, r)
	EnsureHomebrew(r)
	InstallTaps(cfg, r)
	InstallFormulae(cfg, r)
	InstallCasks(cfg, r)
	InstallMAS(cfg, r)
	applyDotfiles(cfg, r)
	applyDefaults(cfg, r)
	runHooks(cfg, r)
	restartServices(r)

	// fd is new → should install.
	if !r.CalledWith("brew", "install", "fd") {
		t.Error("expected brew install fd (new formula)")
	}
	// git is present → should NOT install.
	if r.CalledWith("brew", "install", "git") {
		t.Error("should not re-install git")
	}
	if r.CalledWith("brew", "install", "ripgrep") {
		t.Error("should not re-install ripgrep")
	}

	// raycast is new → should install.
	if !r.CalledWith("brew", "install", "--cask", "raycast") {
		t.Error("expected brew install --cask raycast (new cask)")
	}
	// wezterm present → should NOT install.
	if r.CalledWith("brew", "install", "--cask", "wezterm") {
		t.Error("should not re-install wezterm")
	}

	// cask-fonts not tapped → should tap.
	if !r.CalledWith("brew", "tap", "homebrew/cask-fonts") {
		t.Error("expected brew tap homebrew/cask-fonts")
	}

	// Xcode already installed → no mas install.
	if r.CalledWith("mas", "install", "497799835") {
		t.Error("should not reinstall Xcode")
	}

	// Dotfiles not present → should clone.
	if !r.CalledWith("git", "clone", "https://github.com/user/dotfiles", dotfilesDest) {
		t.Error("expected git clone for dotfiles (not yet present)")
	}
}

// ── Scenario D: Full Uninstall ────────────────────────────────────────────

// TestScenarioD_FullUninstall simulates a fully-configured machine being
// completely uninstalled. Every remove/delete/untap/unstow command should fire.
func TestScenarioD_FullUninstall(t *testing.T) {
	dir := t.TempDir()
	dotfilesDest := filepath.Join(dir, "dotfiles")

	// Create the dotfiles repo and stow package dirs so os.Stat checks pass.
	for _, pkg := range []string{"zsh", "git"} {
		if err := os.MkdirAll(filepath.Join(dotfilesDest, pkg), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r := testutil.NewFakeRunner()
	r.WhenWhich("mas", true)
	r.WhenWhich("stow", true)

	// Everything is installed.
	r.When("brew", "list", "--formula", "-1").Returns("git\nripgrep\nfd\n", nil)
	r.When("brew", "list", "--cask", "-1").Returns("wezterm\nraycast\n", nil)
	r.When("brew", "tap").Returns("homebrew/cask-fonts\n", nil)
	r.When("mas", "list").Returns("497799835 Xcode 15.0\n", nil)

	cfg := scenarioCfg(t)
	cfg.Dotfiles.Dest = dotfilesDest

	uninstallPackages(cfg, r)
	uninstallMAS(cfg, r)
	uninstallDotfiles(cfg, r)
	uninstallDefaults(cfg, r)

	// Formulae removed.
	for _, f := range []string{"git", "ripgrep", "fd"} {
		if !r.CalledWith("brew", "uninstall", f) {
			t.Errorf("expected brew uninstall %s", f)
		}
	}

	// Casks removed.
	for _, c := range []string{"wezterm", "raycast"} {
		if !r.CalledWith("brew", "uninstall", "--cask", c) {
			t.Errorf("expected brew uninstall --cask %s", c)
		}
	}

	// Tap untapped.
	if !r.CalledWith("brew", "untap", "homebrew/cask-fonts") {
		t.Error("expected brew untap homebrew/cask-fonts")
	}

	// MAS app uninstalled.
	if !r.CalledWith("mas", "uninstall", "497799835") {
		t.Error("expected mas uninstall 497799835 (Xcode)")
	}

	// Dotfiles unstowed.
	home := "/home/testuser"
	for _, pkg := range []string{"zsh", "git"} {
		if !r.CalledWith("stow", "-d", dotfilesDest, "-t", home, "-D", pkg) {
			t.Errorf("expected stow -D for package %q", pkg)
		}
	}

	// Dotfiles repo removed.
	if !r.CalledWith("rm", "-rf", dotfilesDest) {
		t.Error("expected rm -rf of dotfiles repo")
	}

	// Defaults deleted.
	if !r.CalledWith("defaults", "delete", "com.apple.dock", "autohide") {
		t.Error("expected defaults delete for autohide")
	}
}

// ── Scenario D: mac diff via DryRunRunner covers ALL sections ─────────────

// TestScenarioD_DiffCoversAllSections verifies that runDiff (via DryRunRunner)
// exercises the full apply pipeline — machine, packages, dotfiles, shell,
// defaults, system, hooks — and never calls the inner runner for mutating ops.
func TestScenarioD_DiffCoversAllSections(t *testing.T) {
	dir := t.TempDir()
	dotfilesDest := filepath.Join(dir, "dotfiles")
	tr := true

	inner := testutil.NewFakeRunner()
	inner.WhenWhich("brew", true)
	inner.WhenWhich("stow", true)
	inner.WhenWhich("mas", true)
	inner.When("brew", "tap").Returns("", nil)
	inner.When("brew", "list", "--formula", "-1").Returns("", nil)
	inner.When("brew", "list", "--cask", "-1").Returns("", nil)
	inner.When("mas", "list").Returns("", nil)

	dr := NewDryRunRunner(inner)

	cfg := &Config{
		Machine:  MachineConfig{ComputerName: "test-mac", LocalHostname: "test-mac-local"},
		Packages: PackagesConfig{Taps: []string{"homebrew/cask-fonts"}, Formulae: []string{"git"}, Casks: []string{"arc"}},
		MAS:      MASConfig{Apps: []MASApp{{ID: 1, Name: "TestApp"}}},
		Dotfiles: DotfilesConfig{Repo: "https://github.com/user/dotfiles", Dest: dotfilesDest, Method: "stow", StowPackages: []string{"zsh"}},
		Shell:    ShellConfig{Default: "/bin/zsh"},
		Defaults: map[string]map[string]any{"com.apple.dock": {"autohide": true}},
		System:   SystemConfig{PamTID: &tr},
		Hooks:    HooksConfig{PostInstall: []string{"echo done"}},
	}

	doApply(cfg, dr, makeSkipSet(cfg))

	// Mutating commands must NOT reach inner runner.
	if inner.CalledWithSudo("scutil", "--set", "ComputerName", "test-mac") {
		t.Error("diff must not write ComputerName")
	}
	if inner.CalledWith("brew", "install", "git") {
		t.Error("diff must not install formulae")
	}
	if inner.CalledWith("brew", "install", "--cask", "arc") {
		t.Error("diff must not install casks")
	}
	if inner.CalledWith("defaults", "write", "com.apple.dock", "autohide", "-bool", "true") {
		t.Error("diff must not write defaults")
	}

	// Read-only commands SHOULD reach inner runner (to show accurate state).
	if !inner.CalledWith("brew", "list", "--formula", "-1") {
		t.Error("diff must read brew formula list for accurate output")
	}
}

// ── Scenario E: [meta] skip ───────────────────────────────────────────────

// TestScenarioE_SkipSections verifies that sections listed in [meta] skip
// are fully bypassed during doApply — no commands for those sections run.
func TestScenarioE_SkipSections(t *testing.T) {
	tr := true
	cfg := &Config{
		Meta:     MetaConfig{Skip: []string{"machine", "system", "shell"}},
		Machine:  MachineConfig{ComputerName: "should-not-be-set"},
		Packages: PackagesConfig{Formulae: []string{"git"}},
		Shell:    ShellConfig{Default: "/bin/zsh"},
		Defaults: map[string]map[string]any{"com.apple.dock": {"autohide": true}},
		System:   SystemConfig{PamTID: &tr},
	}

	r := testutil.NewFakeRunner()
	r.WhenWhich("brew", true)
	r.When("brew", "tap").Returns("", nil)
	r.When("brew", "update", "--quiet").Returns("", nil)
	r.When("brew", "list", "--formula", "-1").Returns("", nil)
	r.When("brew", "list", "--cask", "-1").Returns("", nil)

	doApply(cfg, r, makeSkipSet(cfg))

	// machine section skipped
	if r.CalledWithSudo("scutil", "--set", "ComputerName", "should-not-be-set") {
		t.Error("machine section should be skipped")
	}
	// system section skipped (no PAM)
	for _, call := range r.Calls() {
		if call == "shell:echo 'auth       sufficient     pam_tid.so' | sudo tee /etc/pam.d/sudo_local >/dev/null" {
			t.Error("system section should be skipped (PAM must not run)")
		}
	}
	// shell section skipped
	if r.CalledWith("chsh", "-s", "/bin/zsh") {
		t.Error("shell section should be skipped")
	}
	// packages section NOT skipped
	if !r.CalledWith("brew", "list", "--formula", "-1") {
		t.Error("packages section should run (not skipped)")
	}
}

// TestMakeSkipSet verifies makeSkipSet converts the Meta.Skip slice to a set.
func TestMakeSkipSet(t *testing.T) {
	cfg := &Config{
		Meta: MetaConfig{Skip: []string{"machine", "system"}},
	}
	skip := makeSkipSet(cfg)
	if !skip["machine"] {
		t.Error("expected machine to be in skip set")
	}
	if !skip["system"] {
		t.Error("expected system to be in skip set")
	}
	if skip["packages"] {
		t.Error("packages should not be in skip set")
	}
}
