package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfig_SingleNoExtends(t *testing.T) {
	toml := `
[packages]
formulae = ["git"]
`
	path := writeTempTOML(t, toml)
	cfg, err := ResolveConfig(path)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if len(cfg.Packages.Formulae) != 1 || cfg.Packages.Formulae[0] != "git" {
		t.Errorf("formulae: got %v", cfg.Packages.Formulae)
	}
}

func TestResolveConfig_TwoLevelInheritance(t *testing.T) {
	dir := t.TempDir()

	baseTOML := `
[packages]
formulae = ["git", "ripgrep"]
casks    = ["wezterm"]

[machine]
computer_name = "base-machine"
`
	childTOML := `
[packages]
formulae = ["fd"]

[machine]
computer_name = "child-machine"
`
	basePath := filepath.Join(dir, "base.toml")
	childPath := filepath.Join(dir, "child.toml")

	if err := os.WriteFile(basePath, []byte(baseTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	// Child extends base; we need to embed the basePath in the TOML.
	childContent := "extends = [\"" + basePath + "\"]\n" + childTOML
	if err := os.WriteFile(childPath, []byte(childContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ResolveConfig(childPath)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}

	// Packages are unioned: git, ripgrep from base; fd from child.
	wantFormulae := map[string]bool{"git": true, "ripgrep": true, "fd": true}
	for _, f := range cfg.Packages.Formulae {
		delete(wantFormulae, f)
	}
	if len(wantFormulae) != 0 {
		t.Errorf("missing formulae: %v", wantFormulae)
	}

	// Casks from base should be present.
	if len(cfg.Packages.Casks) != 1 || cfg.Packages.Casks[0] != "wezterm" {
		t.Errorf("casks: got %v", cfg.Packages.Casks)
	}

	// Child's machine wins.
	if cfg.Machine.ComputerName != "child-machine" {
		t.Errorf("computer_name: want child-machine, got %q", cfg.Machine.ComputerName)
	}
}

func TestResolveConfig_CycleDetection(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.toml")
	bPath := filepath.Join(dir, "b.toml")

	aContent := "extends = [\"" + bPath + "\"]\n"
	bContent := "extends = [\"" + aPath + "\"]\n"

	if err := os.WriteFile(aPath, []byte(aContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(bContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveConfig(aPath)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestResolveConfig_SelfReference(t *testing.T) {
	dir := t.TempDir()
	selfPath := filepath.Join(dir, "self.toml")
	content := "extends = [\"" + selfPath + "\"]\n"
	if err := os.WriteFile(selfPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveConfig(selfPath)
	if err == nil {
		t.Fatal("expected self-reference cycle error, got nil")
	}
}

func TestResolveConfig_URLExtends(t *testing.T) {
	baseTOML := `
[packages]
formulae = ["curl", "wget"]
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(baseTOML))
	}))
	defer srv.Close()

	dir := t.TempDir()
	childContent := "extends = [\"" + srv.URL + "/base.toml\"]\n[packages]\nformulae = [\"jq\"]\n"
	childPath := filepath.Join(dir, "child.toml")
	if err := os.WriteFile(childPath, []byte(childContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ResolveConfig(childPath)
	if err != nil {
		t.Fatalf("ResolveConfig with URL extends: %v", err)
	}

	wantFormulae := map[string]bool{"curl": true, "wget": true, "jq": true}
	for _, f := range cfg.Packages.Formulae {
		delete(wantFormulae, f)
	}
	if len(wantFormulae) != 0 {
		t.Errorf("missing formulae from merged config: %v", wantFormulae)
	}
}

func TestResolveConfig_URLExtends_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	childContent := "extends = [\"" + srv.URL + "/missing.toml\"]\n"
	childPath := filepath.Join(dir, "child.toml")
	if err := os.WriteFile(childPath, []byte(childContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveConfig(childPath)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

func TestMergeConfigs_PackageUnion(t *testing.T) {
	base := &Config{
		Packages: PackagesConfig{
			Formulae: []string{"git", "curl"},
			Casks:    []string{"firefox"},
			Taps:     []string{"homebrew/core"},
		},
	}
	override := &Config{
		Packages: PackagesConfig{
			Formulae: []string{"curl", "ripgrep"}, // curl is duplicate
			Casks:    []string{"wezterm"},
		},
	}

	merged := mergeConfigs(base, override)

	// curl should appear only once
	count := 0
	for _, f := range merged.Packages.Formulae {
		if f == "curl" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("curl should appear exactly once, got %d times in %v", count, merged.Packages.Formulae)
	}

	// Should have git, curl, ripgrep
	wantF := map[string]bool{"git": true, "curl": true, "ripgrep": true}
	for _, f := range merged.Packages.Formulae {
		delete(wantF, f)
	}
	if len(wantF) != 0 {
		t.Errorf("missing formulae: %v", wantF)
	}

	// Taps from base preserved
	if len(merged.Packages.Taps) != 1 || merged.Packages.Taps[0] != "homebrew/core" {
		t.Errorf("taps: got %v", merged.Packages.Taps)
	}
}

func TestMergeConfigs_HooksConcatenation(t *testing.T) {
	base := &Config{
		Hooks: HooksConfig{PostInstall: []string{"echo base1", "echo base2"}},
	}
	override := &Config{
		Hooks: HooksConfig{PostInstall: []string{"echo child1"}},
	}

	merged := mergeConfigs(base, override)
	if len(merged.Hooks.PostInstall) != 3 {
		t.Errorf("hooks: want 3, got %d: %v", len(merged.Hooks.PostInstall), merged.Hooks.PostInstall)
	}
	if merged.Hooks.PostInstall[0] != "echo base1" {
		t.Errorf("hooks[0]: want 'echo base1', got %q", merged.Hooks.PostInstall[0])
	}
	if merged.Hooks.PostInstall[2] != "echo child1" {
		t.Errorf("hooks[2]: want 'echo child1', got %q", merged.Hooks.PostInstall[2])
	}
}

func TestMergeConfigs_MachineOverride(t *testing.T) {
	base := &Config{
		Machine: MachineConfig{ComputerName: "base-name", LocalHostname: "base-host"},
	}
	override := &Config{
		Machine: MachineConfig{ComputerName: "override-name"},
	}

	merged := mergeConfigs(base, override)
	if merged.Machine.ComputerName != "override-name" {
		t.Errorf("computer_name: want override-name, got %q", merged.Machine.ComputerName)
	}
}

func TestMergeConfigs_MachineBaseWinsWhenOverrideEmpty(t *testing.T) {
	base := &Config{
		Machine: MachineConfig{ComputerName: "base-name"},
	}
	override := &Config{}

	merged := mergeConfigs(base, override)
	if merged.Machine.ComputerName != "base-name" {
		t.Errorf("computer_name: want base-name, got %q", merged.Machine.ComputerName)
	}
}

func TestMergeConfigs_SystemFieldByField(t *testing.T) {
	tr := true
	fa := false
	base := &Config{
		System: SystemConfig{PamTID: &tr, ScreenshotsDir: "~/Desktop/Screenshots"},
	}
	override := &Config{
		System: SystemConfig{RevealLibrary: &fa},
	}

	merged := mergeConfigs(base, override)
	if merged.System.PamTID == nil || !*merged.System.PamTID {
		t.Error("pam_tid should be true from base")
	}
	if merged.System.RevealLibrary == nil || *merged.System.RevealLibrary {
		t.Error("reveal_library should be false from override")
	}
	if merged.System.ScreenshotsDir != "~/Desktop/Screenshots" {
		t.Errorf("screenshots_dir: got %q", merged.System.ScreenshotsDir)
	}
}

func TestMergeConfigs_DefaultsDeepMerge(t *testing.T) {
	base := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock": {"tilesize": int64(48), "autohide": true},
		},
	}
	override := &Config{
		Defaults: map[string]map[string]any{
			"com.apple.dock":    {"tilesize": int64(36)}, // override wins
			"com.apple.finder": {"ShowStatusBar": true},
		},
	}

	merged := mergeConfigs(base, override)
	dock := merged.Defaults["com.apple.dock"]
	if dock == nil {
		t.Fatal("dock domain missing")
	}
	if dock["tilesize"] != int64(36) {
		t.Errorf("tilesize: want 36, got %v", dock["tilesize"])
	}
	// autohide from base should survive
	if dock["autohide"] != true {
		t.Errorf("autohide: want true, got %v", dock["autohide"])
	}
	if merged.Defaults["com.apple.finder"] == nil {
		t.Error("finder domain should be present")
	}
}

func TestUnionStrings_Dedup(t *testing.T) {
	result := unionStrings([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	want := []string{"a", "b", "c", "d"}
	if len(result) != len(want) {
		t.Errorf("want %v, got %v", want, result)
	}
	for i, s := range result {
		if s != want[i] {
			t.Errorf("index %d: want %q, got %q", i, want[i], s)
		}
	}
}

func TestUnionMASApps_DedupByID(t *testing.T) {
	a := []MASApp{{ID: 1, Name: "App A"}, {ID: 2, Name: "App B"}}
	b := []MASApp{{ID: 2, Name: "App B dup"}, {ID: 3, Name: "App C"}}

	result := unionMASApps(a, b)
	if len(result) != 3 {
		t.Fatalf("want 3 apps, got %d: %v", len(result), result)
	}
	// ID 2 from 'a' should be kept (first wins).
	for _, app := range result {
		if app.ID == 2 && app.Name != "App B" {
			t.Errorf("ID 2 should keep first occurrence name 'App B', got %q", app.Name)
		}
	}
}
