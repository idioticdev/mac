package main

import (
	"fmt"
	"os"
	"strings"
)

const version = "1.0.0"

func usage() {
	fmt.Println(`mac — Declarative macOS Configuration

Usage:
  mac [command] [options]

Commands:
  apply       Apply the configuration (default)
  diff        Show what would change without applying
  audit       Check running system for drift from config
  validate    Check the config file for errors
  export      Generate a mac.toml from current system state
  init        Interactive setup wizard — create your first mac.toml
  uninstall   Remove everything mac applied
  version     Print version

Options:
  -c, --config <path>    Config file (default: ~/.config/mac/mac.toml)
  -o, --output <path>    Output path for export / init (default: stdout / config path)
  -h, --help             Show this help

Uninstall options:
  --dry-run   Show what would be removed without making changes
  --yes, -y   Skip confirmation prompts

Environment:
  MAC_CONFIG   Override config path (same as -c)

Examples:
  mac                            # apply config (prompts for URL on first run)
  mac apply -c ~/my-setup.toml   # apply a specific config file
  mac diff                       # preview all changes (safe, no writes)
  mac diff -c company.toml       # preview a company config on your machine
  mac validate                   # check config for errors
  mac export                     # print mac.toml from current Homebrew state
  mac export -o ~/mac.toml       # save to file instead of stdout
  mac init                       # guided setup wizard
  mac uninstall --dry-run        # preview what uninstall would do
  mac uninstall --yes            # uninstall without prompts
  mac audit                      # check system for drift from config
  mac shell-init                 # print shell integration for brew tracking`)
}

func main() {
	args := os.Args[1:]

	// Handle internal / simple commands before the general flag parser.
	if len(args) > 0 && args[0] == "shell-init" {
		runShellInit()
		return
	}
	if len(args) > 0 && args[0] == "_track" {
		isCask := false
		pkg := ""
		for _, a := range args[1:] {
			switch a {
			case "--cask":
				isCask = true
			default:
				if !strings.HasPrefix(a, "-") && pkg == "" {
					pkg = stripVersion(a)
				}
			}
		}
		if pkg != "" {
			runTrack(pkg, isCask)
		}
		return
	}

	command := "apply"
	configFlag := ""
	outputFlag := ""
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
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --output requires a path")
				os.Exit(1)
			}
			i++
			outputFlag = args[i]
		case "-y", "--yes":
			yes = true
		case "--dry-run":
			dryRun = true
		case "apply", "diff", "audit", "validate", "export", "init", "uninstall":
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

	// export and init don't need a config file to load.
	if command == "export" {
		runExport(DefaultRunner)
		return
	}
	if command == "init" {
		runInit(DefaultRunner, outputFlag)
		return
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
	case "audit":
		if !runAudit(cfg, DefaultRunner) {
			os.Exit(1)
		}
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

	// Validate [meta] skip section names.
	for _, s := range cfg.Meta.Skip {
		if !ValidSections[s] {
			valid := make([]string, 0, len(ValidSections))
			for k := range ValidSections {
				valid = append(valid, k)
			}
			Fail(fmt.Sprintf("Unknown section in [meta] skip: %q (valid: %s)",
				s, strings.Join(valid, ", ")))
			os.Exit(1)
		}
	}
	if len(cfg.Meta.Skip) > 0 {
		Info(fmt.Sprintf("Skipped sections: %s", strings.Join(cfg.Meta.Skip, ", ")))
	}

	for _, msg := range validateKeyRemapping(cfg) {
		Fail(msg)
	}

	if s := cfg.Shell.Default; s != "" && !isValidShellPath(s) {
		Fail(fmt.Sprintf("Invalid shell.default %q: must be an absolute path using only [a-zA-Z0-9/_.-]", s))
		os.Exit(1)
	}
}

// runDiff previews all changes by running the full apply pipeline with a
// DryRunRunner — no writes are made to the system.
func runDiff(cfg *Config) {
	Header()
	Info("dry run — showing what would change")
	doApply(cfg, NewDryRunRunner(DefaultRunner), makeSkipSet(cfg))
}

func runApply(cfg *Config) {
	Header()

	Info("Requesting administrator privileges …")
	if err := DefaultRunner.RunPassthrough("sudo", "-v"); err != nil {
		Fail("Could not acquire sudo. Aborting.")
		os.Exit(1)
	}

	go keepSudoAlive()

	doApply(cfg, DefaultRunner, makeSkipSet(cfg))

	Done()
}

// doApply runs all apply operations via r, skipping sections listed in skip.
// Used by both runApply (real runner) and runDiff (DryRunRunner).
func doApply(cfg *Config, r Runner, skip map[string]bool) {
	if !skip["machine"] {
		applyMachine(cfg, r)
	}
	if !skip["packages"] {
		EnsureHomebrew(r)
		InstallTaps(cfg, r)
		InstallFormulae(cfg, r)
		InstallCasks(cfg, r)
	}
	if !skip["mas"] {
		InstallMAS(cfg, r)
	}
	if !skip["dotfiles"] {
		applyDotfiles(cfg, r)
	}
	if !skip["shell"] {
		applyShell(cfg, r)
	}
	if !skip["defaults"] {
		applyDefaults(cfg, r)
	}
	if !skip["system"] {
		applySystem(cfg, r)
	}
	if !skip["hooks"] {
		runHooks(cfg, r)
	}
	// Restart UI services only when defaults or system were applied.
	if !skip["defaults"] || !skip["system"] {
		restartServices(r)
	}
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
