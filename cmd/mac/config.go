package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ValidSections is the set of section names accepted by [meta] skip.
var ValidSections = map[string]bool{
	"machine":  true,
	"packages": true,
	"mas":      true,
	"dotfiles": true,
	"shell":    true,
	"defaults": true,
	"system":   true,
	"hooks":    true,
}

type MetaConfig struct {
	// Skip lists section names to silently skip during apply/diff.
	// Useful for coexisting with nix-darwin or other config managers.
	// Valid values: machine, packages, mas, dotfiles, shell, defaults, system, hooks
	Skip []string `toml:"skip"`
}

type Config struct {
	Meta     MetaConfig                `toml:"meta"`
	Extends  []string                  `toml:"extends"`
	Machine  MachineConfig             `toml:"machine"`
	Packages PackagesConfig            `toml:"packages"`
	MAS      MASConfig                 `toml:"mas"`
	Dotfiles DotfilesConfig            `toml:"dotfiles"`
	Shell    ShellConfig               `toml:"shell"`
	Defaults map[string]map[string]any `toml:"defaults"`
	System   SystemConfig              `toml:"system"`
	Hooks    HooksConfig               `toml:"hooks"`
}

// makeSkipSet returns a set of section names to skip, derived from cfg.Meta.Skip.
func makeSkipSet(cfg *Config) map[string]bool {
	skip := make(map[string]bool, len(cfg.Meta.Skip))
	for _, s := range cfg.Meta.Skip {
		skip[s] = true
	}
	return skip
}

type MachineConfig struct {
	ComputerName  string `toml:"computer_name"`
	LocalHostname string `toml:"local_hostname"`
}

type PackagesConfig struct {
	Taps     []string `toml:"taps"`
	Formulae []string `toml:"formulae"`
	Casks    []string `toml:"casks"`
}

type MASApp struct {
	ID   int    `toml:"id"`
	Name string `toml:"name"`
}

type MASConfig struct {
	Apps []MASApp `toml:"apps"`
}

type DotfilesConfig struct {
	Repo         string   `toml:"repo"`
	Dest         string   `toml:"dest"`
	Method       string   `toml:"method"`
	StowPackages []string `toml:"stow_packages"`
}

type ShellConfig struct {
	Default string `toml:"default"`
}

type SystemConfig struct {
	PamTID         *bool            `toml:"pam_tid"`
	RevealLibrary  *bool            `toml:"reveal_library"`
	ScreenshotsDir string           `toml:"screenshots_dir"`
	KeyRemapping   []KeyRemapConfig `toml:"key_remapping"`
}

type KeyRemapConfig struct {
	From string `toml:"from"`
	To   string `toml:"to"`
}

type HooksConfig struct {
	PostInstall []string `toml:"post_install"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
