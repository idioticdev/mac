package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mac.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}

func readTemp(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	return string(data)
}

func TestAppendToTOMLArray_MultilineInsert(t *testing.T) {
	path := writeTemp(t, `[packages]
formulae = [
    "git",
    "ripgrep",
]
`)
	if err := appendToTOMLArray(path, "formulae", "neovim"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, `"neovim",`) {
		t.Errorf("expected neovim in output, got:\n%s", got)
	}
	if !strings.Contains(got, `"git",`) {
		t.Errorf("existing entries should be preserved, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_InlineSingleElement(t *testing.T) {
	path := writeTemp(t, `[packages]
formulae = ["git"]
`)
	if err := appendToTOMLArray(path, "formulae", "neovim"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, `"git", "neovim"`) {
		t.Errorf("expected inline append, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_EmptyInlineArray(t *testing.T) {
	path := writeTemp(t, `[packages]
formulae = []
`)
	if err := appendToTOMLArray(path, "formulae", "git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, `formulae = ["git"]`) {
		t.Errorf("expected formulae = [\"git\"], got:\n%s", got)
	}
}

func TestAppendToTOMLArray_AlreadyPresent_Inline(t *testing.T) {
	original := `[packages]
formulae = ["git", "ripgrep"]
`
	path := writeTemp(t, original)
	if err := appendToTOMLArray(path, "formulae", "git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if got != original {
		t.Errorf("file should be unchanged when value already present\ngot:\n%s", got)
	}
}

func TestAppendToTOMLArray_AlreadyPresent_Multiline(t *testing.T) {
	original := `[packages]
formulae = [
    "git",
    "ripgrep",
]
`
	path := writeTemp(t, original)
	if err := appendToTOMLArray(path, "formulae", "ripgrep"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	// Count occurrences of "ripgrep" — should still be exactly one.
	if count := strings.Count(got, `"ripgrep"`); count != 1 {
		t.Errorf("expected exactly 1 occurrence of ripgrep, got %d:\n%s", count, got)
	}
}

func TestAppendToTOMLArray_KeyMissingFromPackages(t *testing.T) {
	path := writeTemp(t, `[packages]
taps = ["homebrew/core"]
`)
	if err := appendToTOMLArray(path, "formulae", "git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, `formulae = ["git"]`) {
		t.Errorf("expected new formulae key, got:\n%s", got)
	}
	if !strings.Contains(got, `taps = ["homebrew/core"]`) {
		t.Errorf("existing taps should be preserved, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_PackagesSectionMissing(t *testing.T) {
	path := writeTemp(t, `[machine]
computer_name = "my-mac"
`)
	if err := appendToTOMLArray(path, "formulae", "git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, "[packages]") {
		t.Errorf("expected [packages] section to be created, got:\n%s", got)
	}
	if !strings.Contains(got, `formulae = ["git"]`) {
		t.Errorf("expected formulae key, got:\n%s", got)
	}
	if !strings.Contains(got, `computer_name = "my-mac"`) {
		t.Errorf("existing content should be preserved, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_CasksKey(t *testing.T) {
	path := writeTemp(t, `[packages]
casks = [
    "wezterm",
]
`)
	if err := appendToTOMLArray(path, "casks", "arc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, `"arc",`) {
		t.Errorf("expected arc in casks, got:\n%s", got)
	}
	if !strings.Contains(got, `"wezterm",`) {
		t.Errorf("existing casks should be preserved, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_PreservesComments(t *testing.T) {
	path := writeTemp(t, `# My mac config
[packages]
# installed tools
formulae = [
    "git",
]
`)
	if err := appendToTOMLArray(path, "formulae", "ripgrep"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readTemp(t, path)
	if !strings.Contains(got, "# My mac config") {
		t.Errorf("top-level comment should be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# installed tools") {
		t.Errorf("inline comment should be preserved, got:\n%s", got)
	}
}

func TestAppendToTOMLArray_AtomicWrite_FileNotFound(t *testing.T) {
	err := appendToTOMLArray("/nonexistent/path/mac.toml", "formulae", "git")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestInlineContains(t *testing.T) {
	cases := []struct {
		inner string
		value string
		want  bool
	}{
		{`"git", "ripgrep"`, `"git"`, true},
		{`"git", "ripgrep"`, `"neovim"`, false},
		{`"git"`, `"git"`, true},
		{``, `"git"`, false},
	}
	for _, tc := range cases {
		got := inlineContains(tc.inner, tc.value)
		if got != tc.want {
			t.Errorf("inlineContains(%q, %q) = %v, want %v", tc.inner, tc.value, got, tc.want)
		}
	}
}
