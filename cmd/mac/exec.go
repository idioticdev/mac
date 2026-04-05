package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// Runner abstracts command execution for testability.
type Runner interface {
	Run(name string, args ...string) (string, error)
	RunSudo(name string, args ...string) (string, error)
	RunPassthrough(name string, args ...string) error
	RunShell(command string) (string, error)
	Which(name string) bool
	ExpandHome(path string) string
	WriteFile(path string, data []byte, perm os.FileMode) error
}

// SystemRunner is the real implementation that executes commands on the OS.
type SystemRunner struct{}

// DefaultRunner is the package-level Runner used by all apply functions.
var DefaultRunner Runner = &SystemRunner{}

func (s *SystemRunner) Run(name string, args ...string) (string, error) {
	return Run(name, args...)
}

func (s *SystemRunner) RunSudo(name string, args ...string) (string, error) {
	return RunSudo(name, args...)
}

func (s *SystemRunner) RunPassthrough(name string, args ...string) error {
	return RunPassthrough(name, args...)
}

func (s *SystemRunner) RunShell(command string) (string, error) {
	return RunShell(command)
}

func (s *SystemRunner) Which(name string) bool {
	return Which(name)
}

func (s *SystemRunner) ExpandHome(path string) string {
	return ExpandHome(path)
}

func (s *SystemRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Run executes a command and returns its combined stdout/stderr output.
func Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

// RunSudo executes a command with sudo.
func RunSudo(name string, args ...string) (string, error) {
	full := append([]string{name}, args...)
	return Run("sudo", full...)
}

// RunPassthrough executes a command with inherited stdin/stdout/stderr.
func RunPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunShell executes a command string via /bin/bash -c.
func RunShell(command string) (string, error) {
	return Run("/bin/bash", "-c", command)
}

// Which reports whether an executable is available in PATH.
func Which(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// DryRunRunner wraps any Runner and no-ops all mutating operations,
// logging what would have run. Read-only operations pass through to
// the inner runner so that state-check calls (brew list, mas list, etc.)
// still return real data for an accurate dry-run preview.
//
// Changed is set to true when at least one meaningful mutating operation
// (package install, defaults write, dotfiles clone, etc.) was detected.
// Infrastructure ops like "brew update" do not count.
type DryRunRunner struct {
	inner   Runner
	Changed bool
}

// NewDryRunRunner returns a DryRunRunner backed by inner.
func NewDryRunRunner(inner Runner) *DryRunRunner {
	return &DryRunRunner{inner: inner}
}

// isReadOnly returns true for commands that only read system state.
// Everything not on this allowlist is treated as mutating and skipped.
func isReadOnly(name string, args []string) bool {
	switch name {
	case "brew":
		if len(args) == 0 {
			return false
		}
		switch args[0] {
		case "list":
			return true
		case "tap":
			// "brew tap" with no tap name = list taps (read-only).
			// "brew tap <name>" = add tap (mutating).
			return len(args) == 1
		}
	case "mas":
		return len(args) > 0 && args[0] == "list"
	case "defaults":
		return len(args) > 0 && args[0] == "read"
	case "sysctl":
		return true
	case "ssh":
		// ssh -T is a connectivity test; batch-mode flags are read-only.
		return slices.Contains(args, "-T")
	case "ssh-keygen":
		// ssh-keygen -l lists/validates a key without modifying it.
		return slices.Contains(args, "-l")
	}
	return false
}

func (d *DryRunRunner) Run(name string, args ...string) (string, error) {
	if isReadOnly(name, args) {
		return d.inner.Run(name, args...)
	}
	// Infrastructure ops: show in diff output but don't count as config drift.
	// - "brew update" is maintenance, not a package change.
	// - "killall" restarts UI services; it's a side effect of applying changes,
	//   not a change itself — and always runs unconditionally.
	if isInfrastructureOp(name, args) {
		Info(humanSummary(name, args))
		return "", nil
	}
	d.Changed = true
	Info(humanSummary(name, args))
	return "", nil
}

// isInfrastructureOp returns true for commands that are always run as
// side effects but don't represent meaningful config drift on their own.
func isInfrastructureOp(name string, args []string) bool {
	if name == "brew" && len(args) > 0 && args[0] == "update" {
		return true
	}
	if name == "killall" {
		return true
	}
	return false
}

func (d *DryRunRunner) RunSudo(name string, args ...string) (string, error) {
	d.Changed = true
	Info(humanSudoSummary(name, args))
	return "", nil
}

func (d *DryRunRunner) RunPassthrough(name string, args ...string) error {
	// Skip sudo authentication pings — not meaningful in dry-run.
	if name == "sudo" && len(args) == 1 && args[0] == "-v" {
		return nil
	}
	d.Changed = true
	Info(humanSummary(name, args))
	return nil
}

func (d *DryRunRunner) RunShell(command string) (string, error) {
	d.Changed = true
	display := command
	if len(display) > 80 {
		display = display[:77] + "…"
	}
	Info("would run: " + display)
	return "", nil
}

// humanSummary maps a command + args to a concise human-readable description
// for use in dry-run / diff output.
func humanSummary(name string, args []string) string {
	switch name {
	case "brew":
		if len(args) == 0 {
			break
		}
		switch args[0] {
		case "install":
			if len(args) >= 3 && args[1] == "--cask" {
				return "would install cask: " + args[2]
			}
			if len(args) >= 2 {
				return "would install formula: " + args[1]
			}
		case "uninstall":
			if len(args) >= 3 && args[1] == "--cask" {
				return "would uninstall cask: " + args[2]
			}
			if len(args) >= 2 {
				return "would uninstall formula: " + args[1]
			}
		case "tap":
			if len(args) >= 2 {
				return "would tap: " + args[1]
			}
		case "untap":
			if len(args) >= 2 {
				return "would untap: " + args[1]
			}
		case "update":
			return "would update Homebrew"
		}
	case "mas":
		if len(args) >= 2 {
			switch args[0] {
			case "install":
				return "would install MAS app: " + args[1]
			case "uninstall":
				return "would uninstall MAS app: " + args[1]
			}
		}
	case "defaults":
		if len(args) >= 3 && args[0] == "write" {
			val := ""
			if len(args) >= 5 {
				val = " → " + strings.Join(args[4:], " ")
			}
			return fmt.Sprintf("would set %s %s%s", args[1], args[2], val)
		}
		if len(args) >= 3 && args[0] == "delete" {
			return fmt.Sprintf("would delete default %s %s", args[1], args[2])
		}
	case "git":
		if len(args) >= 3 && args[0] == "clone" {
			return "would clone dotfiles: " + args[1]
		}
		if len(args) >= 2 && args[0] == "-C" {
			return "would pull dotfiles in " + args[1]
		}
	case "stow":
		if len(args) > 0 {
			return "would stow: " + args[len(args)-1]
		}
	case "chsh":
		for i, a := range args {
			if a == "-s" && i+1 < len(args) {
				return "would change default shell: " + args[i+1]
			}
		}
	case "killall":
		if len(args) >= 1 {
			return "would restart service: " + args[0]
		}
	case "launchctl":
		if len(args) >= 2 {
			switch args[0] {
			case "load":
				return "would load LaunchAgent: " + args[1]
			case "unload":
				return "would unload LaunchAgent: " + args[1]
			}
		}
	case "hidutil":
		if len(args) >= 3 && args[0] == "property" && args[1] == "--set" {
			return "would configure hidutil: " + args[2]
		}
	case "rm":
		return "would remove: " + strings.Join(args, " ")
	case "chflags":
		return "would set file flags: " + strings.Join(args, " ")
	}
	// Fallback: show the command verbatim.
	cmd := name
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}
	return "[would run] " + cmd
}

// humanSudoSummary maps a sudo command + args to a human-readable description.
func humanSudoSummary(name string, args []string) string {
	switch name {
	case "scutil":
		if len(args) >= 3 && args[0] == "--set" {
			return fmt.Sprintf("would set %s: %s", args[1], args[2])
		}
	}
	return humanSummary(name, args)
}

func (d *DryRunRunner) Which(name string) bool {
	return d.inner.Which(name)
}

func (d *DryRunRunner) ExpandHome(path string) string {
	return d.inner.ExpandHome(path)
}

func (d *DryRunRunner) WriteFile(path string, _ []byte, _ os.FileMode) error {
	Info("would write: " + path)
	return nil
}

// ExpandHome expands a leading ~ to the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return fmt.Sprintf("%s%s", home, path[1:])
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	return path
}
