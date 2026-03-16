package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp toml: %v", err)
	}
	return path
}

func TestLoad_Empty(t *testing.T) {
	path := writeTempTOML(t, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Packages.Formulae) != 0 {
		t.Errorf("expected empty formulae, got %v", cfg.Packages.Formulae)
	}
}

func TestLoad_Packages(t *testing.T) {
	toml := `
[packages]
formulae = ["git", "ripgrep", "fd"]
casks    = ["wezterm", "raycast"]
taps     = ["homebrew/cask-fonts"]
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Packages.Formulae) != 3 {
		t.Errorf("formulae: want 3, got %d", len(cfg.Packages.Formulae))
	}
	if cfg.Packages.Formulae[0] != "git" {
		t.Errorf("formulae[0]: want git, got %q", cfg.Packages.Formulae[0])
	}
	if len(cfg.Packages.Casks) != 2 {
		t.Errorf("casks: want 2, got %d", len(cfg.Packages.Casks))
	}
	if len(cfg.Packages.Taps) != 1 {
		t.Errorf("taps: want 1, got %d", len(cfg.Packages.Taps))
	}
}

func TestLoad_MAS(t *testing.T) {
	toml := `
[[mas.apps]]
id   = 497799835
name = "Xcode"

[[mas.apps]]
id   = 1295203466
name = "Microsoft Remote Desktop"
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.MAS.Apps) != 2 {
		t.Fatalf("mas.apps: want 2, got %d", len(cfg.MAS.Apps))
	}
	if cfg.MAS.Apps[0].ID != 497799835 {
		t.Errorf("mas.apps[0].id: want 497799835, got %d", cfg.MAS.Apps[0].ID)
	}
	if cfg.MAS.Apps[0].Name != "Xcode" {
		t.Errorf("mas.apps[0].name: want Xcode, got %q", cfg.MAS.Apps[0].Name)
	}
}

func TestLoad_Machine(t *testing.T) {
	toml := `
[machine]
computer_name  = "my-mac"
local_hostname = "my-mac-local"
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Machine.ComputerName != "my-mac" {
		t.Errorf("computer_name: want my-mac, got %q", cfg.Machine.ComputerName)
	}
	if cfg.Machine.LocalHostname != "my-mac-local" {
		t.Errorf("local_hostname: want my-mac-local, got %q", cfg.Machine.LocalHostname)
	}
}

func TestLoad_SystemBoolPointers(t *testing.T) {
	toml := `
[system]
pam_tid        = true
reveal_library = false
screenshots_dir = "~/Screenshots"
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.System.PamTID == nil {
		t.Fatal("pam_tid should not be nil")
	}
	if !*cfg.System.PamTID {
		t.Error("pam_tid: want true")
	}
	if cfg.System.RevealLibrary == nil {
		t.Fatal("reveal_library should not be nil")
	}
	if *cfg.System.RevealLibrary {
		t.Error("reveal_library: want false")
	}
	if cfg.System.ScreenshotsDir != "~/Screenshots" {
		t.Errorf("screenshots_dir: got %q", cfg.System.ScreenshotsDir)
	}
}

func TestLoad_Dotfiles(t *testing.T) {
	toml := `
[dotfiles]
repo          = "https://github.com/user/dotfiles"
dest          = "~/.dotfiles"
method        = "stow"
stow_packages = ["zsh", "git", "nvim"]
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Dotfiles.Repo != "https://github.com/user/dotfiles" {
		t.Errorf("repo: got %q", cfg.Dotfiles.Repo)
	}
	if cfg.Dotfiles.Method != "stow" {
		t.Errorf("method: got %q", cfg.Dotfiles.Method)
	}
	if len(cfg.Dotfiles.StowPackages) != 3 {
		t.Errorf("stow_packages: want 3, got %d", len(cfg.Dotfiles.StowPackages))
	}
}

func TestLoad_Hooks(t *testing.T) {
	toml := `
[hooks]
post_install = ["echo hello", "touch /tmp/test-hook"]
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Hooks.PostInstall) != 2 {
		t.Fatalf("post_install: want 2, got %d", len(cfg.Hooks.PostInstall))
	}
	if cfg.Hooks.PostInstall[0] != "echo hello" {
		t.Errorf("post_install[0]: got %q", cfg.Hooks.PostInstall[0])
	}
}

func TestLoad_Defaults(t *testing.T) {
	toml := `
[defaults."com.apple.dock"]
tilesize   = 48
autohide   = true
orientation = "bottom"
`
	path := writeTempTOML(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	dock, ok := cfg.Defaults["com.apple.dock"]
	if !ok {
		t.Fatal("expected com.apple.dock domain")
	}
	if dock["autohide"] != true {
		t.Errorf("autohide: want true, got %v", dock["autohide"])
	}
	if dock["orientation"] != "bottom" {
		t.Errorf("orientation: want bottom, got %v", dock["orientation"])
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/mac.toml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	path := writeTempTOML(t, "[[[ invalid toml")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func TestBoolVal(t *testing.T) {
	if BoolVal(nil) {
		t.Error("BoolVal(nil) should return false")
	}
	tr := true
	if !BoolVal(&tr) {
		t.Error("BoolVal(&true) should return true")
	}
	fa := false
	if BoolVal(&fa) {
		t.Error("BoolVal(&false) should return false")
	}
}
