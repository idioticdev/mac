package main

import (
	"fmt"
	"os"
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
  -c, --config <path>   Config file (default: ~/.config/macsetup/macsetup.toml)
  -h, --help            Show this help

Environment:
  MACSETUP_CONFIG   Override config path (same as -c)

Examples:
  macsetup                          # apply config (prompts for URL on first run)
  macsetup apply -c ~/my-setup.toml # apply a specific config file
  macsetup validate                 # check config for errors
  macsetup diff                     # preview changes`)
}

func main() {
	args := os.Args[1:]

	command := "apply"
	configFlag := ""

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
			configFlag = args[i]
		case "apply", "diff", "validate":
			command = args[i]
		default:
			if args[i][0] != '-' {
				configFlag = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[i])
				usage()
				os.Exit(1)
			}
		}
		i++
	}

	configPath, err := resolveConfigPath(configFlag)
	if err != nil {
		Fail(err.Error())
		os.Exit(1)
	}

	cfg, err := ResolveConfig(configPath)
	if err != nil {
		Fail(fmt.Sprintf("Failed to load config: %s", err))
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

func runValidate(cfg *Config, path string) {
	Ok(fmt.Sprintf("Config %s is valid", path))
	if len(cfg.Extends) > 0 {
		fmt.Printf("  Extends: %d base config(s)\n", len(cfg.Extends))
	}
	fmt.Printf("  Formulae: %d, Casks: %d, Taps: %d\n",
		len(cfg.Packages.Formulae), len(cfg.Packages.Casks), len(cfg.Packages.Taps))
	fmt.Printf("  MAS apps: %d\n", len(cfg.MAS.Apps))
	fmt.Printf("  Defaults domains: %d\n", len(cfg.Defaults))
	fmt.Printf("  Stow packages: %d\n", len(cfg.Dotfiles.StowPackages))
	fmt.Printf("  Post-install hooks: %d\n", len(cfg.Hooks.PostInstall))
}

func runDiff(cfg *Config) {
	Header()
	fmt.Println("  (dry run — showing what would change)")

	if len(cfg.Packages.Formulae) > 0 {
		Banner("Homebrew Formulae")
		installed := diffInstalledFormulae()
		for _, pkg := range cfg.Packages.Formulae {
			if installed[pkg] {
				Ok(pkg + " (already installed)")
			} else {
				Info(pkg + " (would install)")
			}
		}
	}

	if len(cfg.Packages.Casks) > 0 {
		Banner("Homebrew Casks")
		installed := diffInstalledCasks()
		for _, cask := range cfg.Packages.Casks {
			if installed[cask] {
				Ok(cask + " (already installed)")
			} else {
				Info(cask + " (would install)")
			}
		}
	}

	if len(cfg.Defaults) > 0 {
		Banner("macOS Defaults")
		for domain, entries := range cfg.Defaults {
			Info("Domain: " + domain)
			for key, val := range entries {
				current := diffCurrentDefault(domain, key)
				desired := fmt.Sprintf("%v", val)
				if current == desired {
					Ok(fmt.Sprintf("  %s = %v (no change)", key, val))
				} else {
					Warn(fmt.Sprintf("  %s: %s → %v", key, current, val))
				}
			}
		}
	}

	if BoolVal(cfg.System.PamTID) {
		Banner("System Tweaks")
		Info("Touch ID for sudo: would enable if not present")
	}
}

func runApply(cfg *Config) {
	Header()

	fmt.Println("  Requesting administrator privileges …")
	if err := RunPassthrough("sudo", "-v"); err != nil {
		Fail("Could not acquire sudo. Aborting.")
		os.Exit(1)
	}

	go keepSudoAlive()

	applyMachine(cfg)
	EnsureHomebrew()
	InstallTaps(cfg)
	InstallFormulae(cfg)
	InstallCasks(cfg)
	InstallMAS(cfg)
	applyDotfiles(cfg)
	applyShell(cfg)
	applyDefaults(cfg)
	applySystem(cfg)
	runHooks(cfg)
	restartServices()

	Done()
}

func keepSudoAlive() {
	for {
		Run("sudo", "-n", "true")
		Run("sleep", "50")
	}
}

func diffInstalledFormulae() map[string]bool {
	out, _ := Run("brew", "list", "--formula", "-1")
	return toSet(out)
}

func diffInstalledCasks() map[string]bool {
	out, _ := Run("brew", "list", "--cask", "-1")
	return toSet(out)
}

func diffCurrentDefault(domain, key string) string {
	out, err := Run("defaults", "read", domain, key)
	if err != nil {
		return "(not set)"
	}
	return out
}
