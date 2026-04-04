package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// ResolveConfig loads a config file and recursively resolves any extends,
// returning a fully merged Config with base configs layered beneath it.
func ResolveConfig(path string) (*Config, error) {
	visited := map[string]bool{}
	return resolveConfig(path, visited)
}

func resolveConfig(source string, visited map[string]bool) (*Config, error) {
	if visited[source] {
		return nil, fmt.Errorf("cycle detected in extends chain: %s", source)
	}
	visited[source] = true

	cfg, err := fetchConfig(source)
	if err != nil {
		return nil, err
	}

	if len(cfg.Extends) == 0 {
		return cfg, nil
	}

	// Start with an empty base, merge each extended config in order.
	merged := &Config{}
	for _, base := range cfg.Extends {
		baseCfg, err := resolveConfig(base, visited)
		if err != nil {
			return nil, fmt.Errorf("extends %q: %w", base, err)
		}
		merged = mergeConfigs(merged, baseCfg)
	}

	// User config wins over all bases.
	return mergeConfigs(merged, cfg), nil
}

// fetchConfig loads a Config from a URL or local file path.
func fetchConfig(source string) (*Config, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") {
		return nil, fmt.Errorf("extends URL must use https://, got: %s", source)
	}
	if strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source) //nolint:noctx
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", source, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetching %s: HTTP %d", source, resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", source, err)
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", source, err)
	}
	return &cfg, nil
}

// mergeConfigs merges override on top of base.
// - Packages/MAS: union (deduplicated, order preserved — base first)
// - Defaults: deep merge, override wins on key conflict
// - Hooks: concatenated (base first)
// - Machine, Shell, Dotfiles: override wins if non-zero
// - System: field-by-field, override's non-nil/non-empty values win
func mergeConfigs(base, override *Config) *Config {
	out := &Config{}

	// Packages — union
	out.Packages.Taps = unionStrings(base.Packages.Taps, override.Packages.Taps)
	out.Packages.Formulae = unionStrings(base.Packages.Formulae, override.Packages.Formulae)
	out.Packages.Casks = unionStrings(base.Packages.Casks, override.Packages.Casks)

	// MAS — union by app ID
	out.MAS.Apps = unionMASApps(base.MAS.Apps, override.MAS.Apps)

	// Defaults — deep merge
	out.Defaults = mergeDefaults(base.Defaults, override.Defaults)

	// Hooks — concatenate (base first)
	out.Hooks.PostInstall = append(base.Hooks.PostInstall, override.Hooks.PostInstall...)

	// Machine — override wins if non-zero
	if override.Machine.ComputerName != "" || override.Machine.LocalHostname != "" {
		out.Machine = override.Machine
	} else {
		out.Machine = base.Machine
	}

	// Shell — override wins if non-zero
	if override.Shell.Default != "" {
		out.Shell = override.Shell
	} else {
		out.Shell = base.Shell
	}

	// Dotfiles — override wins if non-zero
	if override.Dotfiles.Repo != "" {
		out.Dotfiles = override.Dotfiles
	} else {
		out.Dotfiles = base.Dotfiles
	}

	// System — field-by-field
	out.System.ScreenshotsDir = override.System.ScreenshotsDir
	if out.System.ScreenshotsDir == "" {
		out.System.ScreenshotsDir = base.System.ScreenshotsDir
	}
	out.System.PamTID = override.System.PamTID
	if out.System.PamTID == nil {
		out.System.PamTID = base.System.PamTID
	}
	out.System.RevealLibrary = override.System.RevealLibrary
	if out.System.RevealLibrary == nil {
		out.System.RevealLibrary = base.System.RevealLibrary
	}

	return out
}

func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(a, b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func unionMASApps(a, b []MASApp) []MASApp {
	seen := map[int]bool{}
	out := make([]MASApp, 0, len(a)+len(b))
	for _, app := range append(a, b...) {
		if !seen[app.ID] {
			seen[app.ID] = true
			out = append(out, app)
		}
	}
	return out
}

func mergeDefaults(base, override map[string]map[string]any) map[string]map[string]any {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := map[string]map[string]any{}
	for domain, keys := range base {
		out[domain] = make(map[string]any, len(keys))
		for k, v := range keys {
			out[domain][k] = v
		}
	}
	for domain, keys := range override {
		if out[domain] == nil {
			out[domain] = make(map[string]any, len(keys))
		}
		for k, v := range keys {
			out[domain][k] = v
		}
	}
	return out
}
