package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	githubRepo    = "idioticdev/mac"
	githubAPIBase = "https://api.github.com"
	githubDLBase  = "https://github.com"
)

// ReleaseClient abstracts GitHub release fetching for testability.
type ReleaseClient interface {
	// LatestRelease returns the latest release tag (e.g. "v0.4.1") for the given repo.
	LatestRelease(repo string) (string, error)
	// DownloadFile fetches the content at url and returns the raw bytes.
	DownloadFile(url string) ([]byte, error)
}

// httpReleaseClient is the production ReleaseClient backed by http.DefaultClient.
type httpReleaseClient struct {
	apiBase string
	dlBase  string
}

func newHTTPReleaseClient() ReleaseClient {
	return &httpReleaseClient{apiBase: githubAPIBase, dlBase: githubDLBase}
}

func (c *httpReleaseClient) LatestRelease(repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase, repo)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no releases found for %s — publish a release first", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repo)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("parsing release response: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("empty tag_name in release response")
	}
	return payload.TagName, nil
}

func (c *httpReleaseClient) DownloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, url)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading download response: %w", err)
	}
	return data, nil
}

// runUpgrade downloads and installs the latest mac release.
// Returns an error if anything goes wrong; the caller is responsible for os.Exit.
//
// Upgrade flow:
//
//	os.Executable() → execPath
//	runtime.GOARCH  → arch
//	        │
//	        ▼
//	client.LatestRelease(githubRepo) → latestTag
//	        │
//	        ├─ error ──────────────► return error
//	        ▼
//	semver.IsValid(version)?
//	        ├─ false (dev build) ──► warn, proceed
//	        ▼
//	semver.Compare(version, latest) >= 0?
//	        ├─ true ───────────────► "already up to date", return nil
//	        ▼
//	checkOnly?
//	        ├─ true ───────────────► print available version, return nil
//	        ▼
//	download binary + checksums → verify sha256
//	        ├─ mismatch ───────────► return error
//	        ▼
//	chmod +x tmpBin; r.RunSudo("mv", tmpBin, execPath)
//	        ▼
//	Ok("upgraded to " + latestTag), return nil
func runUpgrade(r Runner, client ReleaseClient, checkOnly bool) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	arch := runtime.GOARCH
	if arch != "arm64" && arch != "amd64" {
		return fmt.Errorf("unsupported architecture: %s (mac only supports darwin/arm64 and darwin/amd64)", arch)
	}

	Info("Checking for latest release …")
	latestTag, err := client.LatestRelease(githubRepo)
	if err != nil {
		return err
	}

	// Normalize version for semver comparison: ensure v-prefix.
	currentVersion := version
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}

	if !semver.IsValid(currentVersion) {
		// Dev/dirty builds (e.g. "v0.4.0-3-ge411f44-dirty") are not valid semver.
		// Skip comparison and offer the upgrade.
		Warn(fmt.Sprintf("Current version %q is a dev build — skipping version comparison", version))
	} else if semver.Compare(currentVersion, latestTag) >= 0 {
		Ok(fmt.Sprintf("Already up to date (%s)", version))
		return nil
	}

	if checkOnly {
		Info(fmt.Sprintf("Upgrade available: %s → %s", version, latestTag))
		Info("Run 'mac upgrade' to install")
		return nil
	}

	Info(fmt.Sprintf("Upgrading %s → %s …", version, latestTag))

	binaryName := fmt.Sprintf("mac-darwin-%s", arch)
	binaryURL := fmt.Sprintf("%s/%s/releases/download/%s/%s", githubDLBase, githubRepo, latestTag, binaryName)
	checksumsURL := fmt.Sprintf("%s/%s/releases/download/%s/checksums.txt", githubDLBase, githubRepo, latestTag)

	Info(fmt.Sprintf("Downloading %s …", binaryName))
	binaryData, err := client.DownloadFile(binaryURL)
	if err != nil {
		return err
	}

	Info("Verifying checksum …")
	checksumsData, err := client.DownloadFile(checksumsURL)
	if err != nil {
		return err
	}
	if err := verifyChecksum(binaryData, binaryName, checksumsData); err != nil {
		return err
	}
	Ok("Checksum verified")

	// Write to a temp file, then sudo mv into place.
	tmp, err := os.CreateTemp("", "mac-upgrade-*")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op after successful mv

	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write temp file: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("could not chmod temp file: %w", err)
	}

	Info(fmt.Sprintf("Installing to %s …", execPath))
	if _, err := r.RunSudo("mv", tmpPath, execPath); err != nil {
		return fmt.Errorf("could not install: %w", err)
	}

	Ok(fmt.Sprintf("Upgraded to %s", latestTag))
	return nil
}

// verifyChecksum checks that the sha256 of data matches the expected hash in
// checksumsData (a file with lines of the form "<hash>  <filename>").
func verifyChecksum(data []byte, filename string, checksumsData []byte) error {
	expected := ""
	for _, line := range strings.Split(string(checksumsData), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum for %q not found in checksums.txt", filename)
	}

	sum := sha256.Sum256(data)
	actual := fmt.Sprintf("%x", sum)
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s (got %s, expected %s)", filename, actual, expected)
	}
	return nil
}
