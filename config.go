package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the top-level configuration structure matching macsetup.toml.
type Config struct {
	Machine  MachineConfig  `toml:"machine"`
	Packages PackagesConfig `toml:"packages"`
	MAS      MASConfig      `toml:"mas"`
	Dotfiles DotfilesConfig `toml:"dotfiles"`
	Shell    ShellConfig    `toml:"shell"`
	Defaults map[string]map[string]interface{} `toml:"defaults"`
	System   SystemConfig   `toml:"system"`
	Hooks    HooksConfig    `toml:"hooks"`
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
	PamTID         *bool  `toml:"pam_tid"`
	RevealLibrary  *bool  `toml:"reveal_library"`
	ScreenshotsDir string `toml:"screenshots_dir"`
}

type HooksConfig struct {
	PostInstall []string `toml:"post_install"`
}

// Load reads and parses the TOML config file.
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

// BoolVal safely dereferences a *bool, returning false if nil.
func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
