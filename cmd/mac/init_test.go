package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/youruser/mac/cmd/mac/testutil"
)

// TestResolveConfigPath_MACCONFIGIgnored verifies that the MAC_CONFIG env var
// is no longer honored after its removal from resolveConfigPath.
func TestResolveConfigPath_MACCONFIGIgnored(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "mac", "mac.toml")
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MAC_CONFIG", "/totally/wrong/path.toml")

	got, err := resolveConfigPath("")
	if err != nil {
		t.Fatal(err)
	}
	if got != real {
		t.Errorf("expected default path %q, got %q — MAC_CONFIG should be ignored", real, got)
	}
}

// TestRunInit_URL_DownloadsAndSaves verifies that mac init --url downloads the
// config and writes it to the output path without any interactive prompts.
func TestRunInit_URL_DownloadsAndSaves(t *testing.T) {
	const content = `[packages]
formulae = ["git"]
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "mac.toml")

	runInit(testutil.NewFakeRunner(), dest, srv.URL)

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if !strings.Contains(string(data), `"git"`) {
		t.Errorf("expected downloaded content, got:\n%s", data)
	}
}

// TestRunInit_URL_DownloadFails verifies that a network failure is reported and
// no file is written.
func TestRunInit_URL_DownloadFails(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "mac.toml")

	// Port 0 is never listening — connection refused immediately.
	runInit(testutil.NewFakeRunner(), dest, "http://127.0.0.1:0/nope")

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("expected no file to be written on download failure")
	}
}

// TestRunInit_ExistingConfig_PreservesFileOnDecline is a regression test for the
// "used computer" bug: when a config already exists and the user declines to
// overwrite it, the config must not be modified.
func TestRunInit_ExistingConfig_PreservesFileOnDecline(t *testing.T) {
	const original = "[packages]\nformulae = [\"git\"]\n"
	dir := t.TempDir()
	dest := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(dest, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	pr, pw, _ := os.Pipe()
	os.Stdin = pr
	pw.WriteString("n\n")
	pw.Close()
	defer func() { os.Stdin = oldStdin }()

	runInit(testutil.NewFakeRunner(), dest, "")

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("config file missing after runInit: %v", err)
	}
	if string(data) != original {
		t.Errorf("config was modified when user declined overwrite\ngot:  %q\nwant: %q", string(data), original)
	}
}

// ── promptDotfilesSetup ───────────────────────────────────────────────────────

// slowReader wraps an io.Reader to return one byte per Read call, preventing
// bufio.Scanner from pre-buffering past the current line. This lets multiple
// huh fields share a single reader without the first field consuming all input.
type slowReader struct{ r *strings.Reader }

func (s *slowReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return s.r.Read(p[:1])
}

// testFormOpts returns huh form options that force accessible mode and inject
// the given string as sequential stdin for all forms in a promptDotfilesSetup call.
func testFormOpts(input string) []func(*huh.Form) {
	r := &slowReader{strings.NewReader(input)}
	return []func(*huh.Form){
		func(f *huh.Form) {
			f.WithAccessible(true)
			f.WithInput(r)
		},
	}
}

func TestPromptDotfilesSetup_Skipped(t *testing.T) {
	r := testutil.NewFakeRunner()
	// User answers "n" — skip dotfiles.
	result := promptDotfilesSetup(r, testFormOpts("n\n")...)
	if result != nil {
		t.Errorf("expected nil when user skips, got %+v", result)
	}
}

func TestPromptDotfilesSetup_EmptyRepo(t *testing.T) {
	r := testutil.NewFakeRunner()
	// User says yes then provides no URL — detail form submits empty repo.
	result := promptDotfilesSetup(r, testFormOpts("y\n\n\n\n")...)
	if result != nil {
		t.Errorf("expected nil when repo URL is empty, got %+v", result)
	}
}

func TestPromptDotfilesSetup_HTTPSRepo_NoSSHSetup(t *testing.T) {
	r := testutil.NewFakeRunner()
	// HTTPS repo — ensureSSHForClone should not run ssh-keygen or ssh-add.
	// Confirm: "y", then repo URL, dest (blank = default), stow pkgs (blank).
	result := promptDotfilesSetup(r, testFormOpts("y\nhttps://github.com/you/dotfiles.git\n\n\n")...)
	if result == nil {
		t.Fatal("expected non-nil result for HTTPS repo")
	}
	if result.repo != "https://github.com/you/dotfiles.git" {
		t.Errorf("unexpected repo: %s", result.repo)
	}
	if result.dest != "~/.dotfiles" {
		t.Errorf("expected default dest, got %s", result.dest)
	}
	// No SSH commands should be issued for an HTTPS URL.
	for _, call := range r.Calls() {
		if strings.Contains(call, "ssh-keygen") || strings.Contains(call, "ssh-add") {
			t.Errorf("unexpected SSH call for HTTPS repo: %s", call)
		}
	}
}

func TestPromptDotfilesSetup_CustomDestAndPackages(t *testing.T) {
	r := testutil.NewFakeRunner()
	input := "y\nhttps://github.com/you/dotfiles.git\n~/.config/dotfiles\nzsh, git, nvim\n"
	result := promptDotfilesSetup(r, testFormOpts(input)...)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.dest != "~/.config/dotfiles" {
		t.Errorf("expected custom dest, got %s", result.dest)
	}
	if len(result.stowPkgs) != 3 || result.stowPkgs[0] != "zsh" || result.stowPkgs[2] != "nvim" {
		t.Errorf("unexpected stow packages: %v", result.stowPkgs)
	}
}

// ── buildDotfilesSection ─────────────────────────────────────────────────────

func TestBuildDotfilesSection_Nil_ReturnsCommented(t *testing.T) {
	s := buildDotfilesSection(nil)
	if !strings.Contains(s, "# [dotfiles]") {
		t.Error("expected commented-out section when nil")
	}
}

func TestBuildDotfilesSection_WithValues(t *testing.T) {
	df := &dotfilesSetup{
		repo:     "git@github.com:you/dotfiles.git",
		dest:     "~/.dotfiles",
		stowPkgs: []string{"zsh", "git"},
	}
	s := buildDotfilesSection(df)
	if !strings.Contains(s, `[dotfiles]`) {
		t.Error("expected [dotfiles] header")
	}
	if !strings.Contains(s, `"git@github.com:you/dotfiles.git"`) {
		t.Error("expected repo value")
	}
	if !strings.Contains(s, `stow_packages = ["zsh", "git"]`) {
		t.Error("expected stow_packages")
	}
	if strings.Contains(s, "#") {
		t.Error("expected no comments when dotfiles are configured")
	}
}

func TestBuildDotfilesSection_NoStowPkgs(t *testing.T) {
	df := &dotfilesSetup{repo: "git@github.com:you/dotfiles.git", dest: "~/.dotfiles"}
	s := buildDotfilesSection(df)
	if strings.Contains(s, "stow_packages") {
		t.Error("expected no stow_packages line when empty")
	}
}

// ── initBlank ────────────────────────────────────────────────────────────────

func TestInitBlank_NoRepo_WritesCommentedDotfiles(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "mac.toml")

	r := testutil.NewFakeRunner()
	r.When("scutil", "--get", "ComputerName").Returns("test-mac", nil)
	r.When("scutil", "--get", "LocalHostName").Returns("test-mac", nil)

	// User skips dotfiles prompt.
	oldStdin := os.Stdin
	pr, pw, _ := os.Pipe()
	os.Stdin = pr
	pw.WriteString("n\n")
	pw.Close()
	defer func() { os.Stdin = oldStdin }()

	initBlank(r, dest)

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# [dotfiles]") {
		t.Error("expected commented dotfiles section when user skips")
	}
	if !strings.Contains(content, `computer_name  = "test-mac"`) {
		t.Error("expected machine name in config")
	}
}

func TestInitBlank_WithRepo_WritesDotfilesSection(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "mac.toml")

	r := testutil.NewFakeRunner()
	r.When("scutil", "--get", "ComputerName").Returns("my-mac", nil)
	r.When("scutil", "--get", "LocalHostName").Returns("my-mac", nil)
	// sshConnected check for HTTPS repo won't call SSH — no stub needed.

	oldStdin := os.Stdin
	pr, pw, _ := os.Pipe()
	os.Stdin = pr
	// yes, repo URL (HTTPS so no SSH prompt), default dest, packages
	pw.WriteString("y\nhttps://github.com/you/dotfiles.git\n\nzsh, git\n")
	pw.Close()
	defer func() { os.Stdin = oldStdin }()

	initBlank(r, dest)

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "# [dotfiles]") {
		t.Error("expected uncommented dotfiles section when repo provided")
	}
	if !strings.Contains(content, `[dotfiles]`) {
		t.Error("expected [dotfiles] section")
	}
	if !strings.Contains(content, `stow_packages = ["zsh", "git"]`) {
		t.Errorf("expected stow_packages in config; got:\n%s", content)
	}
}
