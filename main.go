package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/youruser/macsetup/internal/config"
	"github.com/youruser/macsetup/internal/defaults"
	"github.com/youruser/macsetup/internal/dotfiles"
	"github.com/youruser/macsetup/internal/hooks"
	"github.com/youruser/macsetup/internal/machine"
	"github.com/youruser/macsetup/internal/packages"
	"github.com/youruser/macsetup/internal/system"
	"github.com/youruser/macsetup/internal/ui"
)

const version = "1.0.0"

func usage() {
	fmt.Println(`macsetup — Declarative macOS Configuration

Usage:
  macsetup [command] [options]

Commands:
  apply       Apply the configuration (default)
  diff        Show what would change without applying
  validate    Check the config file for errors
  version     Print version

Options:
  -c, --config <path>   Config file (default: ./macsetup.toml)
  -h, --help            Show this help

Examples:
  macsetup                          # apply ./macsetup.toml
  macsetup apply -c ~/my-setup.toml # apply custom config
  macsetup validate                 # check config for errors
  macsetup diff                     # preview changes`)
}

func main() {
	args := os.Args[1:]

	command := "apply"
	configPath := "macsetup.toml"

	// Parse arguments
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			usage()
			os.Exit(0)
		case "version", "--version":
			fmt.Println("macsetup v" + version)
			os.Exit(0)
		case "-c", "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --config requires a path")
				os.Exit(1)
			}
			i++
			configPath = args[i]
		case "apply", "diff", "validate":
			command = args[i]
		default:
			// If it looks like a file path (not a flag), treat as config
			if args[i][0] != '-' {
				configPath = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[i])
				usage()
				os.Exit(1)
			}
		}
		i++
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		ui.Fail(fmt.Sprintf("Failed to load config: %s", err))
		os.Exit(1)
	}

	switch command {
	case "validate":
		runValidate(cfg, configPath)
	case "diff":
		runDiff(cfg)
	case "apply":
		runApply(cfg)
	}
}

func runValidate(cfg *config.Config, path string) {
	ui.Ok(fmt.Sprintf("Config %s is valid", path))
	fmt.Printf("  Formulae: %d, Casks: %d, Taps: %d\n",
		len(cfg.Packages.Formulae), len(cfg.Packages.Casks), len(cfg.Packages.Taps))
	fmt.Printf("  MAS apps: %d\n", len(cfg.MAS.Apps))
	fmt.Printf("  Defaults domains: %d\n", len(cfg.Defaults))
	fmt.Printf("  Stow packages: %d\n", len(cfg.Dotfiles.StowPackages))
	fmt.Printf("  Post-install hooks: %d\n", len(cfg.Hooks.PostInstall))
}

func runDiff(cfg *config.Config) {
	ui.Header()
	fmt.Println("  (dry run — showing what would change)")

	// Packages diff
	if len(cfg.Packages.Formulae) > 0 {
		ui.Banner("Homebrew Formulae")
		installed := installedFormulae()
		for _, pkg := range cfg.Packages.Formulae {
			if installed[pkg] {
				ui.Ok(pkg + " (already installed)")
			} else {
				ui.Info(pkg + " (would install)")
			}
		}
	}

	if len(cfg.Packages.Casks) > 0 {
		ui.Banner("Homebrew Casks")
		installed := installedCasks()
		for _, cask := range cfg.Packages.Casks {
			if installed[cask] {
				ui.Ok(cask + " (already installed)")
			} else {
				ui.Info(cask + " (would install)")
			}
		}
	}

	// Defaults diff
	if len(cfg.Defaults) > 0 {
		ui.Banner("macOS Defaults")
		for domain, entries := range cfg.Defaults {
			ui.Info("Domain: " + domain)
			for key, val := range entries {
				current := currentDefault(domain, key)
				desired := fmt.Sprintf("%v", val)
				if current == desired {
					ui.Ok(fmt.Sprintf("  %s = %v (no change)", key, val))
				} else {
					ui.Warn(fmt.Sprintf("  %s: %s → %v", key, current, val))
				}
			}
		}
	}

	// System
	if config.BoolVal(cfg.System.PamTID) {
		ui.Banner("System Tweaks")
		ui.Info("Touch ID for sudo: would enable if not present")
	}
}

func runApply(cfg *config.Config) {
	ui.Header()

	// Acquire sudo upfront
	fmt.Println("  Requesting administrator privileges …")
	sudoCmd := exec.Command("sudo", "-v")
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	if err := sudoCmd.Run(); err != nil {
		ui.Fail("Could not acquire sudo. Aborting.")
		os.Exit(1)
	}

	// Keep sudo alive in background
	go keepSudoAlive()

	machine.Apply(cfg)
	packages.EnsureHomebrew()
	packages.InstallTaps(cfg)
	packages.InstallFormulae(cfg)
	packages.InstallCasks(cfg)
	packages.InstallMAS(cfg)
	dotfiles.Apply(cfg)
	system.ApplyShell(cfg)
	defaults.Apply(cfg)
	system.Apply(cfg)
	hooks.Run(cfg)
	system.RestartServices()

	ui.Done()
}

func keepSudoAlive() {
	for {
		exec.Command("sudo", "-n", "true").Run()
		// Sleep 50 seconds (below 5-minute default timeout)
		exec.Command("sleep", "50").Run()
	}
}

// Helper lookups for diff mode.

func installedFormulae() map[string]bool {
	out, _ := exec.Command("brew", "list", "--formula", "-1").Output()
	return toSet(string(out))
}

func installedCasks() map[string]bool {
	out, _ := exec.Command("brew", "list", "--cask", "-1").Output()
	return toSet(string(out))
}

func currentDefault(domain, key string) string {
	out, err := exec.Command("defaults", "read", domain, key).Output()
	if err != nil {
		return "(not set)"
	}
	return string(out[:len(out)-1]) // trim newline
}

func toSet(s string) map[string]bool {
	m := make(map[string]bool)
	for _, line := range splitLines(s) {
		if line != "" {
			m[line] = true
		}
	}
	return m
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
