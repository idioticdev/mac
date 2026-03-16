package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
type DryRunRunner struct {
	inner Runner
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
	}
	return false
}

func (d *DryRunRunner) Run(name string, args ...string) (string, error) {
	if isReadOnly(name, args) {
		return d.inner.Run(name, args...)
	}
	Info(fmt.Sprintf("[dry-run] %s", strings.Join(append([]string{name}, args...), " ")))
	return "", nil
}

func (d *DryRunRunner) RunSudo(name string, args ...string) (string, error) {
	Info(fmt.Sprintf("[dry-run] sudo %s", strings.Join(append([]string{name}, args...), " ")))
	return "", nil
}

func (d *DryRunRunner) RunPassthrough(name string, args ...string) error {
	Info(fmt.Sprintf("[dry-run] %s", strings.Join(append([]string{name}, args...), " ")))
	return nil
}

func (d *DryRunRunner) RunShell(command string) (string, error) {
	Info("[dry-run] shell: " + command)
	return "", nil
}

func (d *DryRunRunner) Which(name string) bool {
	return d.inner.Which(name)
}

func (d *DryRunRunner) ExpandHome(path string) string {
	return d.inner.ExpandHome(path)
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
