package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
