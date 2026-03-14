package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

func RunSudo(name string, args ...string) (string, error) {
	full := append([]string{name}, args...)
	return Run("sudo", full...)
}

func RunPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunShell(command string) (string, error) {
	return Run("/bin/bash", "-c", command)
}

func Which(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

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
