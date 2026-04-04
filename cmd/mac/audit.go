package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// runAudit compares the running system against cfg and reports drift.
// Returns true if no drift was found (clean system).
func runAudit(cfg *Config, r Runner) bool {
	Header()
	Info("auditing system against config …")
	fmt.Fprintln(os.Stderr)

	drifted := 0
	drifted += auditFormulae(cfg, r)
	drifted += auditCasks(cfg, r)
	drifted += auditMAS(cfg, r)
	drifted += auditDefaults(cfg, r)

	fmt.Fprintln(os.Stderr)
	if drifted == 0 {
		Ok("No drift detected — system matches config")
		return true
	}
	Warn(fmt.Sprintf("%d drift(s) detected", drifted))
	return false
}

// auditFormulae reports formulae installed on the system but not in cfg.
// Returns the number of drifted items.
func auditFormulae(cfg *Config, r Runner) int {
	installed, _ := r.Run("brew", "list", "--formula", "-1")
	managed := toSet(strings.Join(cfg.Packages.Formulae, "\n"))
	untracked := untrackedItems(installed, managed)

	if len(untracked) == 0 && len(cfg.Packages.Formulae) == 0 {
		return 0
	}

	Banner("Homebrew Formulae")
	for _, f := range untracked {
		Warn(fmt.Sprintf("%s (installed, not in config)", f))
	}
	if len(untracked) == 0 {
		Ok(fmt.Sprintf("all %d formula(e) tracked", len(cfg.Packages.Formulae)))
	}
	return len(untracked)
}

// auditCasks reports casks installed on the system but not in cfg.
// Returns the number of drifted items.
func auditCasks(cfg *Config, r Runner) int {
	installed, _ := r.Run("brew", "list", "--cask", "-1")
	managed := toSet(strings.Join(cfg.Packages.Casks, "\n"))
	untracked := untrackedItems(installed, managed)

	if len(untracked) == 0 && len(cfg.Packages.Casks) == 0 {
		return 0
	}

	Banner("Homebrew Casks")
	for _, c := range untracked {
		Warn(fmt.Sprintf("%s (installed, not in config)", c))
	}
	if len(untracked) == 0 {
		Ok(fmt.Sprintf("all %d cask(s) tracked", len(cfg.Packages.Casks)))
	}
	return len(untracked)
}

// auditMAS reports Mac App Store apps installed but not in cfg.
// Returns the number of drifted items.
func auditMAS(cfg *Config, r Runner) int {
	if !r.Which("mas") {
		return 0
	}

	masList, _ := r.Run("mas", "list")
	managedIDs := make(map[string]bool, len(cfg.MAS.Apps))
	for _, app := range cfg.MAS.Apps {
		managedIDs[fmt.Sprintf("%d", app.ID)] = true
	}

	var untracked []string
	for _, line := range strings.Split(masList, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		id := strings.SplitN(line, " ", 2)[0]
		if !managedIDs[id] {
			untracked = append(untracked, line)
		}
	}

	if len(untracked) == 0 && len(cfg.MAS.Apps) == 0 {
		return 0
	}

	Banner("Mac App Store")
	for _, app := range untracked {
		Warn(fmt.Sprintf("%s (installed, not in config)", app))
	}
	if len(untracked) == 0 {
		Ok(fmt.Sprintf("all %d app(s) tracked", len(cfg.MAS.Apps)))
	}
	return len(untracked)
}

// auditDefaults reports defaults in cfg whose current system value differs.
// Returns the number of drifted items.
func auditDefaults(cfg *Config, r Runner) int {
	if len(cfg.Defaults) == 0 {
		return 0
	}

	Banner("macOS Defaults")

	domains := make([]string, 0, len(cfg.Defaults))
	for d := range cfg.Defaults {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	drifted := 0
	for _, domain := range domains {
		entries := cfg.Defaults[domain]
		keys := make([]string, 0, len(entries))
		for k := range entries {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			expected := expectedDefaultsReadValue(entries[key])
			if expected == "" {
				continue // unsupported type — validate catches these
			}

			current, err := r.Run("defaults", "read", domain, key)
			if err != nil {
				Warn(fmt.Sprintf("%s %s: not set (expected %s)", domain, key, expected))
				drifted++
				continue
			}

			if current != expected {
				Warn(fmt.Sprintf("%s %s: got %q, want %q", domain, key, current, expected))
				drifted++
			} else {
				Ok(fmt.Sprintf("%s %s = %s", domain, key, expected))
			}
		}
	}
	return drifted
}

// expectedDefaultsReadValue returns the string that 'defaults read' produces
// for a value written by 'defaults write'.
func expectedDefaultsReadValue(val any) string {
	switch v := val.(type) {
	case bool:
		if v {
			return "1"
		}
		return "0"
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		if v == math.Trunc(v) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case string:
		return v
	}
	return ""
}

// untrackedItems returns sorted items from the newline-separated installed list
// that are not in managed.
func untrackedItems(installed string, managed map[string]bool) []string {
	var out []string
	for _, line := range strings.Split(installed, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !managed[line] {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}
