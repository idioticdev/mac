package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// isSSHURL reports whether repo uses an SSH transport (git@ or ssh:// scheme).
func isSSHURL(repo string) bool {
	return strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh://")
}

// sshHostFromURL extracts the hostname from an SSH repo URL.
//
//	git@github.com:user/repo.git  → "github.com"
//	ssh://git@gitlab.com/user/repo → "gitlab.com"
func sshHostFromURL(repo string) string {
	if strings.HasPrefix(repo, "git@") {
		rest := repo[len("git@"):]
		if host, _, ok := strings.Cut(rest, ":"); ok {
			return host
		}
		return rest
	}
	if strings.HasPrefix(repo, "ssh://") {
		rest := repo[len("ssh://"):]
		if _, after, ok := strings.Cut(rest, "@"); ok {
			rest = after
		}
		if host, _, ok := strings.Cut(rest, "/"); ok {
			return host
		}
		return rest
	}
	return ""
}

// sshKeyNames is the ordered list of standard SSH private key file names to check.
var sshKeyNames = []string{"id_ed25519", "id_ecdsa", "id_rsa"}

// findExistingSSHKeyIn returns the path of the first valid SSH private key found
// in sshDir, or "" if none exist. Each candidate is validated with ssh-keygen -l
// to catch corrupt/partial files from interrupted keygen runs.
func findExistingSSHKeyIn(sshDir string, r Runner) string {
	for _, name := range sshKeyNames {
		path := filepath.Join(sshDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if _, err := r.Run("ssh-keygen", "-l", "-f", path); err == nil {
			return path
		}
		Warn("Key " + path + " appears corrupt — will generate a new one")
	}
	return ""
}

// findExistingSSHKey returns the first valid key in ~/.ssh, or "".
func findExistingSSHKey(r Runner) string {
	home, _ := os.UserHomeDir()
	return findExistingSSHKeyIn(filepath.Join(home, ".ssh"), r)
}

// sshConnected reports whether SSH authentication to host is working.
// Handles GitHub's quirk of exiting 1 on success with "successfully authenticated"
// in the output.
func sshConnected(host string, r Runner) bool {
	out, err := r.Run("ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-T", "git@"+host)
	if err == nil {
		return true
	}
	return strings.Contains(out, "successfully authenticated")
}

const sshConfigBlock = `
Host *
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_ed25519
`

// writeSSHConfigTo appends the macOS SSH agent persistence block to
// sshDir/config if not already present. Idempotent.
func writeSSHConfigTo(sshDir string, r Runner) {
	configPath := filepath.Join(sshDir, "config")
	existing, _ := os.ReadFile(configPath)
	if strings.Contains(string(existing), "UseKeychain yes") {
		return
	}
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		Warn("Could not create " + sshDir + ": " + err.Error())
		return
	}
	newContent := append(existing, []byte(sshConfigBlock)...)
	if err := r.WriteFile(configPath, newContent, 0o600); err != nil {
		Warn("Could not write " + configPath + ": " + err.Error())
		return
	}
	Ok("SSH config updated (~/.ssh/config)")
}

// writeSSHConfig appends the macOS SSH persistence block to ~/.ssh/config.
func writeSSHConfig(r Runner) {
	home, _ := os.UserHomeDir()
	writeSSHConfigTo(filepath.Join(home, ".ssh"), r)
}

// displayPublicKey prints the public key at keyPath.pub to the terminal and
// shows instructions for adding it to the Git host.
func displayPublicKey(keyPath, host string) {
	pubPath := keyPath + ".pub"
	data, err := os.ReadFile(pubPath)
	if err != nil {
		Warn("Could not read public key: " + pubPath)
		return
	}
	pubKey := strings.TrimSpace(string(data))

	fmt.Fprintln(os.Stderr)
	Info("Your SSH public key:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  "+pubKey)
	fmt.Fprintln(os.Stderr, "")

	if host == "github.com" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(pubKey)
		if cmd.Run() == nil {
			Ok("Key copied to clipboard")
		} else {
			Info("Copy key:  pbcopy < " + pubPath)
		}
		Info("Add key at: https://github.com/settings/keys")
	} else {
		Info("Add this key to the SSH keys settings on " + host)
	}
	fmt.Fprintln(os.Stderr)
}

// ensureSSHForClone checks that SSH authentication is ready for repo.
// If not, it guides the user through key generation and registration.
// Only called on first clone (dest does not yet exist).
func ensureSSHForClone(repo string, r Runner) {
	if !isSSHURL(repo) {
		return
	}
	host := sshHostFromURL(repo)
	if host == "" {
		return
	}

	if sshConnected(host, r) {
		Info("SSH authentication ready")
		return
	}

	Banner("SSH Setup")
	Info("SSH not configured for " + host)

	home, _ := os.UserHomeDir()
	defaultKeyPath := filepath.Join(home, ".ssh", "id_ed25519")

	keyPath := findExistingSSHKey(r)
	if keyPath == "" {
		Info("Generating SSH key …")
		if err := r.RunPassthrough("ssh-keygen", "-t", "ed25519", "-f", defaultKeyPath); err != nil {
			Warn("Key generation failed: " + err.Error())
			return
		}
		keyPath = defaultKeyPath
	} else {
		Info("Found existing key: " + keyPath)
	}

	// Load into agent. Non-fatal: UseKeychain in ~/.ssh/config handles persistence.
	r.Run("ssh-add", "--apple-use-keychain", keyPath)

	writeSSHConfig(r)
	displayPublicKey(keyPath, host)

	if Confirm("Have you added the key to your Git host?") {
		if sshConnected(host, r) {
			Ok("SSH authentication verified")
		} else {
			Warn("SSH still failing — git clone may fail")
			Warn("Verify the key was saved correctly at " + host)
		}
	}
}
