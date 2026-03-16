package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

func usage() {
	fmt.Println(`mac — Declarative macOS Configuration

Usage:
  mac [command] [options]

Commands:
  apply       Apply the configuration (default)
  diff        Show what would change without applying
  validate    Check the config file for errors
  uninstall   Remove everything mac applied
  version     Print version

Options:
  -c, --config <path>   Config file (default: ~/.config/mac/mac.toml)
  -h, --help            Show this help

Uninstall options:
  --dry-run   Show what would be removed without making changes
  --yes, -y   Skip confirmation prompts

Environment:
  MAC_CONFIG   Override config path (same as -c)

Examples:
  mac                          # apply config (prompts for URL on first run)
  mac apply -c ~/my-setup.toml # apply a specific config file
  mac validate                 # check config for errors
  mac diff                     # preview changes
  mac uninstall --dry-run      # preview what uninstall would do
  mac uninstall --yes          # uninstall without prompts`)
}

func main() {
	args := os.Args[1:]

	command := "apply"
	configFlag := ""
	yes := false
	dryRun := false

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			usage()
			os.Exit(0)
		case "version", "--version":
			fmt.Println("mac v" + version)
			os.Exit(0)
		case "-c", "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --config requires a path")
				os.Exit(1)
			}
			i++
			configFlag = args[i]
		case "-y", "--yes":
			yes = true
		case "--dry-run":
			dryRun = true
		case "apply", "diff", "validate", "uninstall":
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
	case "uninstall":
		runUninstall(cfg, yes, dryRun)
	}
}

func runValidate(cfg *Config, path string) {
	Ok(fmt.Sprintf("Config %s is valid", path))
	if len(cfg.Extends) > 0 {
		Info(fmt.Sprintf("Extends: %d base config(s)", len(cfg.Extends)))
	}
	Info(fmt.Sprintf("Formulae: %d, Casks: %d, Taps: %d",
		len(cfg.Packages.Formulae), len(cfg.Packages.Casks), len(cfg.Packages.Taps)))
	Info(fmt.Sprintf("MAS apps: %d", len(cfg.MAS.Apps)))
	Info(fmt.Sprintf("Defaults domains: %d", len(cfg.Defaults)))
	Info(fmt.Sprintf("Stow packages: %d", len(cfg.Dotfiles.StowPackages)))
	Info(fmt.Sprintf("Post-install hooks: %d", len(cfg.Hooks.PostInstall)))
}

func runDiff(cfg *Config) {
	Header()
	Info("dry run — showing what would change")

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
	r := DefaultRunner
	Header()

	Info("Requesting administrator privileges …")
	if err := r.RunPassthrough("sudo", "-v"); err != nil {
		Fail("Could not acquire sudo. Aborting.")
		os.Exit(1)
	}

	go keepSudoAlive()

	applyMachine(cfg, r)
	EnsureHomebrew(r)
	InstallTaps(cfg, r)
	InstallFormulae(cfg, r)
	InstallCasks(cfg, r)
	InstallMAS(cfg, r)
	applyDotfiles(cfg, r)
	applyShell(cfg, r)
	applyDefaults(cfg, r)
	applySystem(cfg, r)
	runHooks(cfg, r)
	restartServices(r)

	Done()
}

func keepSudoAlive() {
	for {
		Run("sudo", "-n", "true")
		Run("sleep", "50")
	}
}

func runUninstall(cfg *Config, yes, dryRun bool) {
	var r Runner = DefaultRunner
	if dryRun {
		r = NewDryRunRunner(DefaultRunner)
	}

	Header()

	if dryRun {
		Info("dry-run — showing what would change")
	}

	if !dryRun {
		Info("Requesting administrator privileges …")
		if err := DefaultRunner.RunPassthrough("sudo", "-v"); err != nil {
			Fail("Could not acquire sudo. Aborting.")
			os.Exit(1)
		}
		go keepSudoAlive()
	}

	if len(cfg.Packages.Formulae) > 0 || len(cfg.Packages.Casks) > 0 || len(cfg.Packages.Taps) > 0 {
		if yes || dryRun || Confirm(fmt.Sprintf("Uninstall %d formulae, %d casks, %d taps?",
			len(cfg.Packages.Formulae), len(cfg.Packages.Casks), len(cfg.Packages.Taps))) {
			uninstallPackages(cfg, r)
		}
	}

	if len(cfg.MAS.Apps) > 0 {
		if yes || dryRun || Confirm(fmt.Sprintf("Uninstall %d Mac App Store app(s)?", len(cfg.MAS.Apps))) {
			uninstallMAS(cfg, r)
		}
	}

	if cfg.Dotfiles.Repo != "" {
		if yes || dryRun || Confirm("Unstow dotfiles and remove repo?") {
			uninstallDotfiles(cfg, r)
		}
	}

	if len(cfg.Defaults) > 0 {
		if yes || dryRun || Confirm(fmt.Sprintf("Delete %d macOS defaults key(s)?", totalDefaultsCount(cfg))) {
			uninstallDefaults(cfg, r)
		}
	}

	anySystem := BoolVal(cfg.System.PamTID) || BoolVal(cfg.System.RevealLibrary) || cfg.Shell.Default != ""
	if anySystem {
		if yes || dryRun || Confirm("Revert system tweaks?") {
			uninstallSystem(cfg, r)
		}
	}

	Done()
}

func totalDefaultsCount(cfg *Config) int {
	n := 0
	for _, keys := range cfg.Defaults {
		n += len(keys)
	}
	return n
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
