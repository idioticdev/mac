package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// ── isSSHURL ──────────────────────────────────────────────────────────────────

func TestIsSSHURL_GitAt(t *testing.T) {
	if !isSSHURL("git@github.com:user/dotfiles.git") {
		t.Error("expected git@ URL to be SSH")
	}
}

func TestIsSSHURL_SSHScheme(t *testing.T) {
	if !isSSHURL("ssh://git@github.com/user/dotfiles.git") {
		t.Error("expected ssh:// URL to be SSH")
	}
}

func TestIsSSHURL_HTTPS(t *testing.T) {
	if isSSHURL("https://github.com/user/dotfiles.git") {
		t.Error("expected https:// URL to not be SSH")
	}
}

func TestIsSSHURL_Empty(t *testing.T) {
	if isSSHURL("") {
		t.Error("expected empty string to not be SSH")
	}
}

// ── sshHostFromURL ────────────────────────────────────────────────────────────

func TestSSHHostFromURL_GitAtGitHub(t *testing.T) {
	got := sshHostFromURL("git@github.com:user/repo.git")
	if got != "github.com" {
		t.Errorf("got %q, want %q", got, "github.com")
	}
}

func TestSSHHostFromURL_GitAtGitLab(t *testing.T) {
	got := sshHostFromURL("git@gitlab.com:user/repo")
	if got != "gitlab.com" {
		t.Errorf("got %q, want %q", got, "gitlab.com")
	}
}

func TestSSHHostFromURL_SSHSchemeWithUser(t *testing.T) {
	got := sshHostFromURL("ssh://git@github.com/user/repo.git")
	if got != "github.com" {
		t.Errorf("got %q, want %q", got, "github.com")
	}
}

func TestSSHHostFromURL_Empty(t *testing.T) {
	got := sshHostFromURL("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ── findExistingSSHKeyIn ──────────────────────────────────────────────────────

func TestFindExistingSSHKeyIn_ValidKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("fake-key"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()
	r.When("ssh-keygen", "-l", "-f", keyPath).Returns("2048 SHA256:abc... (ED25519)", nil)

	got := findExistingSSHKeyIn(dir, r)
	if got != keyPath {
		t.Errorf("got %q, want %q", got, keyPath)
	}
}

func TestFindExistingSSHKeyIn_CorruptKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()
	r.When("ssh-keygen", "-l", "-f", keyPath).Returns("", errors.New("invalid format"))

	got := findExistingSSHKeyIn(dir, r)
	if got != "" {
		t.Errorf("expected empty string for corrupt key, got %q", got)
	}
}

func TestFindExistingSSHKeyIn_NoKey(t *testing.T) {
	dir := t.TempDir() // empty directory

	r := testutil.NewFakeRunner()
	got := findExistingSSHKeyIn(dir, r)
	if got != "" {
		t.Errorf("expected empty string when no keys present, got %q", got)
	}
	if len(r.Calls()) != 0 {
		t.Errorf("expected no runner calls when no key files exist, got %v", r.Calls())
	}
}

// ── sshConnected ─────────────────────────────────────────────────────────────

func TestSSHConnected_ExitZero(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-T", "git@github.com").
		Returns("", nil)

	if !sshConnected("github.com", r) {
		t.Error("expected sshConnected to return true when command succeeds")
	}
}

func TestSSHConnected_GitHubAuthenticated(t *testing.T) {
	// GitHub exits 1 but prints "successfully authenticated" — should be treated as success.
	r := testutil.NewFakeRunner()
	r.When("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-T", "git@github.com").
		Returns("Hi user! You have successfully authenticated.", errors.New("exit status 1"))

	if !sshConnected("github.com", r) {
		t.Error("expected sshConnected to return true for GitHub-style success output")
	}
}

func TestSSHConnected_Failure(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-T", "git@github.com").
		Returns("Permission denied (publickey).", errors.New("exit status 255"))

	if sshConnected("github.com", r) {
		t.Error("expected sshConnected to return false when permission denied")
	}
}

// ── writeSSHConfigTo ──────────────────────────────────────────────────────────

func TestWriteSSHConfigTo_NewConfig(t *testing.T) {
	dir := t.TempDir()
	r := testutil.NewFakeRunner()

	writeSSHConfigTo(dir, r)

	configPath := filepath.Join(dir, "config")
	if !r.CalledWriteFile(configPath) {
		t.Error("expected WriteFile to be called for ~/.ssh/config")
	}
}

func TestWriteSSHConfigTo_Idempotent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	existing := "Host *\n  UseKeychain yes\n  AddKeysToAgent yes\n"
	if err := os.WriteFile(configPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	r := testutil.NewFakeRunner()
	writeSSHConfigTo(dir, r)

	if r.CalledWriteFile(configPath) {
		t.Error("expected WriteFile to NOT be called when config already has UseKeychain yes")
	}
}

// ── ensureSSHForClone routing ─────────────────────────────────────────────────

func TestEnsureSSHForClone_HTTPSNoOp(t *testing.T) {
	r := testutil.NewFakeRunner()
	ensureSSHForClone("https://github.com/user/dotfiles.git", r)

	for _, call := range r.Calls() {
		if len(call) >= 3 && call[:3] == "ssh" {
			t.Errorf("expected no ssh calls for HTTPS URL, got %q", call)
		}
	}
}

func TestEnsureSSHForClone_AlreadyConnected(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.When("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-T", "git@github.com").
		Returns("", nil)

	ensureSSHForClone("git@github.com:user/dotfiles.git", r)

	// Should not attempt key generation or passthrough calls.
	for _, call := range r.Calls() {
		if call == "ssh-keygen -t ed25519 -f "+filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519") {
			t.Error("expected no keygen call when SSH is already connected")
		}
	}
}
