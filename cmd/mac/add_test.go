package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunTrack_FormulaAdded verifies that a new formula is appended to mac.toml.
func TestRunTrack_FormulaAdded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(path, []byte(`[packages]
formulae = ["git"]
`), 0644); err != nil {
		t.Fatal(err)
	}

	runTrack(path, "ripgrep", false)

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"ripgrep"`) {
		t.Errorf("expected ripgrep in config, got:\n%s", data)
	}
}

// TestRunTrack_CaskAdded verifies that a new cask is appended to the casks array.
func TestRunTrack_CaskAdded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(path, []byte(`[packages]
casks = ["wezterm"]
`), 0644); err != nil {
		t.Fatal(err)
	}

	runTrack(path, "arc", true)

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"arc"`) {
		t.Errorf("expected arc in config, got:\n%s", data)
	}
}

// TestRunTrack_AlreadyPresent verifies that a duplicate is silently skipped.
func TestRunTrack_AlreadyPresent(t *testing.T) {
	original := `[packages]
formulae = ["git", "ripgrep"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	runTrack(path, "git", false)

	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Errorf("file should be unchanged for duplicate, got:\n%s", data)
	}
}

// TestRunTrack_ConfigNotFound verifies graceful handling when mac.toml is absent.
func TestRunTrack_ConfigNotFound(t *testing.T) {
	// Should not panic; prints to stderr and returns.
	runTrack("/nonexistent/path/mac.toml", "git", false)
}

// TestRunShellInit verifies the emitted shell function contains required parts.
func TestRunShellInit(t *testing.T) {
	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runShellInit()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	checks := []struct {
		desc string
		want string
	}{
		{"defines brew function", "brew()"},
		{"calls real brew", "command brew"},
		{"calls mac _track", "mac _track"},
		{"handles --cask flag", "--cask"},
		{"strips @version suffix", `%%@*}`},
	}
	for _, tc := range checks {
		if !strings.Contains(output, tc.want) {
			t.Errorf("shell-init output missing %s (%q)\noutput:\n%s", tc.desc, tc.want, output)
		}
	}
}

// TestStripVersion verifies @version suffix removal.
func TestStripVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"git@2.40.0", "git"},
		{"node@20", "node"},
		{"ripgrep", "ripgrep"},
		{"git", "git"},
	}
	for _, tc := range cases {
		got := stripVersion(tc.in)
		if got != tc.want {
			t.Errorf("stripVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
