package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

// fakeReleaseClient is a test double for ReleaseClient.
type fakeReleaseClient struct {
	latestTag   string
	latestErr   error
	files       map[string][]byte
	downloadErr map[string]error
}

func newFakeClient(latestTag string) *fakeReleaseClient {
	return &fakeReleaseClient{
		latestTag:   latestTag,
		files:       make(map[string][]byte),
		downloadErr: make(map[string]error),
	}
}

func (f *fakeReleaseClient) LatestRelease(_ string) (string, error) {
	return f.latestTag, f.latestErr
}

func (f *fakeReleaseClient) DownloadFile(url string) ([]byte, error) {
	if err, ok := f.downloadErr[url]; ok {
		return nil, err
	}
	if data, ok := f.files[url]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("unexpected download URL in test: %s", url)
}

// binaryURL returns the expected download URL for a given tag and arch.
func binaryURL(tag, arch string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/mac-darwin-%s", githubDLBase, githubRepo, tag, arch)
}

// checksumsURL returns the expected checksums URL for a given tag.
func checksumsURL(tag string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/checksums.txt", githubDLBase, githubRepo, tag)
}

// makeChecksums builds a checksums.txt line for the given binary data and filename.
func makeChecksums(data []byte, filename string) []byte {
	sum := sha256.Sum256(data)
	return []byte(fmt.Sprintf("%x  %s\n", sum, filename))
}

// withUpgradeFiles populates client with valid binaries + a combined checksums file
// for both arm64 and amd64, using binaryContent for both arches.
// This ensures tests pass regardless of the host architecture.
func withUpgradeFiles(client *fakeReleaseClient, tag string, binaryContent []byte) {
	sumURL := checksumsURL(tag)
	var combined []byte
	for _, a := range []string{"arm64", "amd64"} {
		client.files[binaryURL(tag, a)] = binaryContent
		combined = append(combined, makeChecksums(binaryContent, fmt.Sprintf("mac-darwin-%s", a))...)
	}
	client.files[sumURL] = combined
}

// sudoMvCalled reports whether any sudo mv call was recorded.
func sudoMvCalled(calls []string) bool {
	for _, c := range calls {
		if strings.HasPrefix(c, "sudo mv") {
			return true
		}
	}
	return false
}

func TestRunUpgrade_AlreadyUpToDate(t *testing.T) {
	saved := version
	version = "v0.4.1"
	defer func() { version = saved }()

	client := newFakeClient("v0.4.1")
	runner := testutil.NewFakeRunner()

	if err := runUpgrade(runner, client, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sudoMvCalled(runner.Calls()) {
		t.Error("sudo mv should not be called when already up to date")
	}
}

func TestRunUpgrade_AlreadyAhead(t *testing.T) {
	saved := version
	version = "v2.0.0"
	defer func() { version = saved }()

	// Latest is older than current — no upgrade needed.
	client := newFakeClient("v1.0.0")
	runner := testutil.NewFakeRunner()

	if err := runUpgrade(runner, client, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sudoMvCalled(runner.Calls()) {
		t.Error("sudo mv should not be called when already ahead of latest release")
	}
}

func TestRunUpgrade_UpgradeAvailable(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	const tag = "v0.4.1"
	client := newFakeClient(tag)
	withUpgradeFiles(client, tag, []byte("fake binary content"))
	runner := testutil.NewFakeRunner()

	if err := runUpgrade(runner, client, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sudoMvCalled(runner.Calls()) {
		t.Error("expected sudo mv to be called for the upgrade install")
	}
}

func TestRunUpgrade_CheckOnly(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	// check-only: no downloads, no install.
	client := newFakeClient("v0.4.1")
	runner := testutil.NewFakeRunner()

	if err := runUpgrade(runner, client, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sudoMvCalled(runner.Calls()) {
		t.Error("sudo mv should not be called in check-only mode")
	}
}

func TestRunUpgrade_DevBuild(t *testing.T) {
	saved := version
	version = "v0.4.0-3-ge411f44-dirty" // invalid semver, dev build
	defer func() { version = saved }()

	const tag = "v0.4.1"
	client := newFakeClient(tag)
	withUpgradeFiles(client, tag, []byte("fake binary content"))
	runner := testutil.NewFakeRunner()

	// Dev builds have invalid semver — upgrade should proceed without comparison.
	if err := runUpgrade(runner, client, false); err != nil {
		t.Fatalf("dev build should proceed with upgrade, got error: %v", err)
	}
	if !sudoMvCalled(runner.Calls()) {
		t.Error("expected sudo mv to be called for dev build upgrade")
	}
}

func TestRunUpgrade_LatestReleaseError(t *testing.T) {
	client := newFakeClient("")
	client.latestErr = errors.New("network timeout")
	runner := testutil.NewFakeRunner()

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error when LatestRelease fails")
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("error should mention network timeout, got: %v", err)
	}
}

func TestRunUpgrade_NoReleasesFound(t *testing.T) {
	client := newFakeClient("")
	client.latestErr = fmt.Errorf("no releases found for %s — publish a release first", githubRepo)
	runner := testutil.NewFakeRunner()

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error when no releases exist")
	}
	if !strings.Contains(err.Error(), "no releases found") {
		t.Errorf("error should mention no releases, got: %v", err)
	}
}

func TestRunUpgrade_BinaryDownloadError(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	const tag = "v0.4.1"
	client := newFakeClient(tag)
	// Fail downloads for both arches — the test is arch-agnostic.
	for _, a := range []string{"arm64", "amd64"} {
		client.downloadErr[binaryURL(tag, a)] = errors.New("connection reset")
	}
	runner := testutil.NewFakeRunner()

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error when binary download fails")
	}
	if !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("error should mention connection reset, got: %v", err)
	}
}

func TestRunUpgrade_ChecksumsDownloadError(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	const tag = "v0.4.1"
	binaryContent := []byte("fake binary content")
	client := newFakeClient(tag)
	for _, a := range []string{"arm64", "amd64"} {
		client.files[binaryURL(tag, a)] = binaryContent
	}
	client.downloadErr[checksumsURL(tag)] = errors.New("checksums unavailable")
	runner := testutil.NewFakeRunner()

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error when checksums download fails")
	}
	if !strings.Contains(err.Error(), "checksums unavailable") {
		t.Errorf("error should mention checksums unavailable, got: %v", err)
	}
}

func TestRunUpgrade_ChecksumMismatch(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	const tag = "v0.4.1"
	binaryContent := []byte("fake binary content")
	wrongHash := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	corruptChecksums := []byte(
		wrongHash + "  mac-darwin-arm64\n" +
			wrongHash + "  mac-darwin-amd64\n",
	)

	client := newFakeClient(tag)
	for _, a := range []string{"arm64", "amd64"} {
		client.files[binaryURL(tag, a)] = binaryContent
	}
	client.files[checksumsURL(tag)] = corruptChecksums
	runner := testutil.NewFakeRunner()

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error on checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("error should mention checksum mismatch, got: %v", err)
	}
}

func TestRunUpgrade_SudoMvFails(t *testing.T) {
	saved := version
	version = "v0.4.0"
	defer func() { version = saved }()

	const tag = "v0.4.1"
	client := newFakeClient(tag)
	withUpgradeFiles(client, tag, []byte("fake binary content"))

	installErr := errors.New("permission denied")
	runner := &failSudoRunner{inner: testutil.NewFakeRunner(), err: installErr}

	err := runUpgrade(runner, client, false)
	if err == nil {
		t.Fatal("expected error when sudo mv fails")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error should mention permission denied, got: %v", err)
	}
}

// failSudoRunner wraps a FakeRunner and fails all RunSudo calls with a given error.
type failSudoRunner struct {
	inner *testutil.FakeRunner
	err   error
}

func (f *failSudoRunner) Run(name string, args ...string) (string, error) {
	return f.inner.Run(name, args...)
}
func (f *failSudoRunner) RunSudo(_ string, _ ...string) (string, error) {
	return "", f.err
}
func (f *failSudoRunner) RunPassthrough(name string, args ...string) error {
	return f.inner.RunPassthrough(name, args...)
}
func (f *failSudoRunner) RunShell(command string) (string, error) {
	return f.inner.RunShell(command)
}
func (f *failSudoRunner) Which(name string) bool        { return f.inner.Which(name) }
func (f *failSudoRunner) ExpandHome(path string) string { return f.inner.ExpandHome(path) }
func (f *failSudoRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	return f.inner.WriteFile(path, data, perm)
}

// --- verifyChecksum unit tests ---

func TestVerifyChecksum_Match(t *testing.T) {
	data := []byte("hello mac")
	sum := sha256.Sum256(data)
	checksums := []byte(fmt.Sprintf("%x  mac-darwin-arm64\n", sum))

	if err := verifyChecksum(data, "mac-darwin-arm64", checksums); err != nil {
		t.Fatalf("expected no error for matching checksum, got: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	data := []byte("hello mac")
	checksums := []byte("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  mac-darwin-arm64\n")

	err := verifyChecksum(data, "mac-darwin-arm64", checksums)
	if err == nil {
		t.Fatal("expected error for mismatched checksum")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("error should say checksum mismatch, got: %v", err)
	}
}

func TestVerifyChecksum_FilenameNotFound(t *testing.T) {
	data := []byte("hello mac")
	checksums := []byte("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  mac-darwin-amd64\n")

	err := verifyChecksum(data, "mac-darwin-arm64", checksums)
	if err == nil {
		t.Fatal("expected error when filename not in checksums")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

// ── startVersionCheck ─────────────────────────────────────────────────────────

func TestStartVersionCheck_NewerAvailable(t *testing.T) {
	version = "v1.0.0"
	client := newFakeClient("v1.1.0")

	ch := startVersionCheck(client)
	tag, ok := <-ch
	if !ok {
		t.Fatal("expected a tag on the channel, channel was closed empty")
	}
	if tag != "v1.1.0" {
		t.Errorf("expected v1.1.0, got %s", tag)
	}
}

func TestStartVersionCheck_AlreadyUpToDate(t *testing.T) {
	version = "v1.1.0"
	client := newFakeClient("v1.1.0")

	ch := startVersionCheck(client)
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel closed with no value when already up to date")
	}
}

func TestStartVersionCheck_NetworkError(t *testing.T) {
	version = "v1.0.0"
	client := newFakeClient("")
	client.latestErr = errors.New("network error")

	ch := startVersionCheck(client)
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel closed with no value on network error")
	}
}

func TestStartVersionCheck_DevBuild(t *testing.T) {
	version = "v1.0.0-3-gabcdef-dirty"
	client := newFakeClient("v1.1.0")

	ch := startVersionCheck(client)
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel closed with no value for dev build")
	}
}
