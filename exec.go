package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes a command, returning combined stdout+stderr and any error.
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

// RunPassthrough executes a command, inheriting stdin/stdout/stderr.
func RunPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunShell runs a command string through /bin/bash.
func RunShell(command string) (string, error) {
	return Run("/bin/bash", "-c", command)
}

// Which checks if a binary is available on PATH.
func Which(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ExpandHome replaces a leading ~ with $HOME.
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
