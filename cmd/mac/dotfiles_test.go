package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// ── checkStowConflicts ────────────────────────────────────────────────────

func TestCheckStowConflicts_NoTargets(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	targetDir := filepath.Join(dir, "home")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, ".zshrc"), []byte("# zsh"), 0o644); err != nil {
		t.Fatal(err)
	}
	// targetDir exists but has no files.
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	conflicts, err := checkStowConflicts(pkgDir, targetDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}

func TestCheckStowConflicts_RegularFileConflict(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	targetDir := filepath.Join(dir, "home")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, ".zshrc"), []byte("# pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A regular file at the target — this is a conflict.
	if err := os.WriteFile(filepath.Join(targetDir, ".zshrc"), []byte("# existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := checkStowConflicts(pkgDir, targetDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0] != filepath.Join(targetDir, ".zshrc") {
		t.Errorf("expected conflict on .zshrc, got %v", conflicts)
	}
}

func TestCheckStowConflicts_SymlinkNotConflict(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	targetDir := filepath.Join(dir, "home")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkgFile := filepath.Join(pkgDir, ".zshrc")
	if err := os.WriteFile(pkgFile, []byte("# pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink at the target — already stowed, not a conflict.
	if err := os.Symlink(pkgFile, filepath.Join(targetDir, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	conflicts, err := checkStowConflicts(pkgDir, targetDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for symlink target, got %v", conflicts)
	}
}

func TestCheckStowConflicts_DirectoryNotConflict(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	targetDir := filepath.Join(dir, "home")
	// pkg has .config/nvim/init.lua; home has .config/ as a real directory.
	if err := os.MkdirAll(filepath.Join(pkgDir, ".config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, ".config", "nvim", "init.lua"), []byte("-- nvim"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Real directory at home/.config/ — stow unfolds this, not a conflict.
	if err := os.MkdirAll(filepath.Join(targetDir, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}

	conflicts, err := checkStowConflicts(pkgDir, targetDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts when only dirs overlap, got %v", conflicts)
	}
}

func TestCheckStowConflicts_NestedFileConflict(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	targetDir := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(pkgDir, ".config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, ".config", "nvim", "init.lua"), []byte("-- pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Regular file at the nested target path — this is a conflict.
	if err := os.MkdirAll(filepath.Join(targetDir, ".config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, ".config", "nvim", "init.lua"), []byte("-- existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := checkStowConflicts(pkgDir, targetDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(targetDir, ".config", "nvim", "init.lua")
	if len(conflicts) != 1 || conflicts[0] != want {
		t.Errorf("expected nested conflict at %s, got %v", want, conflicts)
	}
}

// ── applyDotfiles (conflict integration) ─────────────────────────────────

func TestApplyDotfiles_SkipsStowWhenConflictsExist(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	home := filepath.Join(dir, "home")

	// Create pkg dir with a file.
	if err := os.MkdirAll(filepath.Join(dest, "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "zsh", ".zshrc"), []byte("# pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Conflicting regular file in the fake home.
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner().WhenWhich("stow", true).SetHome(home)
	r.When("git", "-C", dest, "pull", "--ff-only").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"zsh"},
		},
	}

	applyDotfiles(cfg, r)

	if r.CalledWith("stow", "-d", dest, "-t", home, "--restow", "zsh") {
		t.Error("stow should not be called when conflicts exist")
	}
}

func TestApplyDotfiles_NoRepo(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	applyDotfiles(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when repo is empty, got %v", r.Calls())
	}
}

func TestApplyDotfiles_CloneWhenDestMissing(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles") // does not exist yet

	r := testutil.NewFakeRunner()
	r.When("git", "clone", "https://github.com/user/dotfiles", dest).Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo: "https://github.com/user/dotfiles",
			Dest: dest,
		},
	}

	applyDotfiles(cfg, r)

	if !r.CalledWith("git", "clone", "https://github.com/user/dotfiles", dest) {
		t.Error("expected git clone to be called")
	}
	if r.CalledWith("git", "-C", dest, "pull", "--ff-only") {
		t.Error("pull should not be called when dest does not exist")
	}
}

func TestApplyDotfiles_PullWhenDestExists(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()
	r.When("git", "-C", dest, "pull", "--ff-only").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo: "https://github.com/user/dotfiles",
			Dest: dest,
		},
	}

	applyDotfiles(cfg, r)

	if r.CalledWith("git", "clone", "https://github.com/user/dotfiles", dest) {
		t.Error("clone should not be called when dest already exists")
	}
	if !r.CalledWith("git", "-C", dest, "pull", "--ff-only") {
		t.Error("expected git pull to be called")
	}
}

func TestApplyDotfiles_StowMethod(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create stow package directories.
	for _, pkg := range []string{"zsh", "git"} {
		if err := os.MkdirAll(filepath.Join(dest, pkg), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r := testutil.NewFakeRunner().WhenWhich("stow", true)
	r.When("git", "-C", dest, "pull", "--ff-only").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"zsh", "git"},
		},
	}

	applyDotfiles(cfg, r)

	home := "/home/testuser"
	for _, pkg := range []string{"zsh", "git"} {
		if !r.CalledWith("stow", "-d", dest, "-t", home, "--restow", pkg) {
			t.Errorf("expected stow call for package %q", pkg)
		}
	}
}

func TestApplyDotfiles_StowInstallsStowWhenMissing(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dest, "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner().WhenWhich("stow", false)
	r.When("git", "-C", dest, "pull", "--ff-only").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"zsh"},
		},
	}

	applyDotfiles(cfg, r)

	if !r.CalledWith("brew", "install", "stow") {
		t.Error("expected brew install stow when stow not found")
	}
}

func TestApplyDotfiles_StowSkipsMissingPackageDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	// Do NOT create the "nvim" subdirectory.

	r := testutil.NewFakeRunner().WhenWhich("stow", true)
	r.When("git", "-C", dest, "pull", "--ff-only").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"nvim"},
		},
	}

	applyDotfiles(cfg, r)

	home := "/home/testuser"
	if r.CalledWith("stow", "-d", dest, "-t", home, "--restow", "nvim") {
		t.Error("stow should not be called for a missing package directory")
	}
}

func TestApplyDotfiles_ExpandsHomeDest(t *testing.T) {
	r := testutil.NewFakeRunner()
	// The FakeRunner.ExpandHome maps ~/foo → /home/testuser/foo.
	// The dest "/home/testuser/dotfiles" won't exist, so clone should be called.
	r.When("git", "clone", "https://github.com/user/dotfiles", "/home/testuser/dotfiles").Returns("", nil)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo: "https://github.com/user/dotfiles",
			Dest: "~/dotfiles",
		},
	}

	applyDotfiles(cfg, r)

	if !r.CalledWith("git", "clone", "https://github.com/user/dotfiles", "/home/testuser/dotfiles") {
		t.Error("expected git clone with expanded home path")
	}
}

func TestApplyDotfiles_CloneFailureAbortsStow(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	// dest doesn't exist → will try to clone.

	r := testutil.NewFakeRunner().WhenWhich("stow", true)
	// Clone returns an error.
	r.When("git", "clone", "https://github.com/user/dotfiles", dest).Returns("", os.ErrPermission)

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"zsh"},
		},
	}

	applyDotfiles(cfg, r)

	home := "/home/testuser"
	if r.CalledWith("stow", "-d", dest, "-t", home, "--restow", "zsh") {
		t.Error("stow should not run after clone failure")
	}
}

// ── uninstallDotfiles ─────────────────────────────────────────────────────

func TestUninstallDotfiles_NoRepo(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	uninstallDotfiles(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when repo is empty, got %v", r.Calls())
	}
}

func TestUninstallDotfiles_StowUnstowsPackages(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "stow",
			StowPackages: []string{"zsh", "git"},
		},
	}

	uninstallDotfiles(cfg, r)

	home := "/home/testuser"
	for _, pkg := range []string{"zsh", "git"} {
		if !r.CalledWith("stow", "-d", dest, "-t", home, "-D", pkg) {
			t.Errorf("expected stow -D for package %q", pkg)
		}
	}
}

func TestUninstallDotfiles_RemovesRepoDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo: "https://github.com/user/dotfiles",
			Dest: dest,
		},
	}

	uninstallDotfiles(cfg, r)

	if !r.CalledWith("rm", "-rf", dest) {
		t.Error("expected rm -rf of dotfiles repo")
	}
}

func TestUninstallDotfiles_MissingRepoSkipped(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles") // does not exist

	r := testutil.NewFakeRunner()

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo: "https://github.com/user/dotfiles",
			Dest: dest,
		},
	}

	uninstallDotfiles(cfg, r)

	if r.CalledWith("rm", "-rf", dest) {
		t.Error("should not attempt rm -rf when repo dir does not exist")
	}
}

func TestUninstallDotfiles_CloneOnlySkipsStow(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Repo:         "https://github.com/user/dotfiles",
			Dest:         dest,
			Method:       "clone-only",
			StowPackages: []string{"zsh"},
		},
	}

	uninstallDotfiles(cfg, r)

	home := "/home/testuser"
	if r.CalledWith("stow", "-d", dest, "-t", home, "-D", "zsh") {
		t.Error("stow -D should not be called for clone-only method")
	}
	// Repo removal still happens.
	if !r.CalledWith("rm", "-rf", dest) {
		t.Error("expected rm -rf even for clone-only method")
	}
}
