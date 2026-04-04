package main

import (
	"fmt"
	"os"
	"strings"
)

// runShellInit prints a bash/zsh shell function that wraps brew. The user adds
// this to their shell config once:
//
//	eval "$(mac shell-init)"
//
// After that, every successful `brew install` automatically records the package
// in mac.toml via `mac _track`.
func runShellInit() {
	fmt.Print(`# mac shell integration — add to ~/.zshrc or ~/.bashrc:
#   eval "$(mac shell-init)"
brew() {
    command brew "$@"
    local _mac_status=$?
    if [ $_mac_status -eq 0 ] && [ "${1:-}" = "install" ]; then
        local _mac_cask=false
        local _mac_pkgs=()
        local _mac_arg
        for _mac_arg in "${@:2}"; do
            case "$_mac_arg" in
                --cask) _mac_cask=true ;;
                --*|-*) ;;
                *) _mac_pkgs+=("${_mac_arg%%@*}") ;;
            esac
        done
        for _mac_pkg in "${_mac_pkgs[@]}"; do
            if $_mac_cask; then
                mac _track --cask "$_mac_pkg" 2>/dev/null || true
            else
                mac _track "$_mac_pkg" 2>/dev/null || true
            fi
        done
    fi
    return $_mac_status
}
`)
}

// runTrack adds pkg to the mac.toml config file. It is called internally by
// the shell wrapper emitted by runShellInit after a successful brew install.
// Errors are printed to stderr and the process exits 0 so the shell wrapper's
// `|| true` remains a reliable no-op for the user's terminal session.
func runTrack(pkg string, isCask bool) {
	configPath := os.Getenv("MAC_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath()
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "mac: config not found at %s — run: mac init\n", configPath)
		return
	}

	cfg, err := Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mac: could not load config (%s) — run: mac add %s\n", err, pkg)
		return
	}

	if isCask {
		for _, c := range cfg.Packages.Casks {
			if c == pkg {
				return // already tracked, silent no-op
			}
		}
	} else {
		for _, f := range cfg.Packages.Formulae {
			if f == pkg {
				return // already tracked, silent no-op
			}
		}
	}

	key := "formulae"
	if isCask {
		key = "casks"
	}

	if err := appendToTOMLArray(configPath, key, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "mac: could not update config (%s) — run: mac add %s\n", err, pkg)
		return
	}

	Ok(pkg + " → added to mac.toml")
}

// stripVersion removes a version suffix from a brew package name.
// e.g. "git@2.40.0" → "git", "node@20" → "node".
func stripVersion(pkg string) string {
	if idx := strings.Index(pkg, "@"); idx >= 0 {
		return pkg[:idx]
	}
	return pkg
}
