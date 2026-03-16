package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// defaultConfigPath returns ~/.config/mac/mac.toml.
func defaultConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(base, "mac", "mac.toml")
}

// resolveConfigPath returns the config path to use, running the interactive
// first-run prompt if no config exists at the default location.
//
// Priority:
//  1. Explicit -c / --config flag (caller passes non-empty override)
//  2. MAC_CONFIG env var
//  3. Existing file at the default path
//  4. Interactive prompt → download → save to default path
func resolveConfigPath(flagPath string) (string, error) {
	// 1. Explicit flag
	if flagPath != "" && flagPath != "mac.toml" {
		return flagPath, nil
	}

	// 2. Env var
	if env := os.Getenv("MAC_CONFIG"); env != "" {
		return env, nil
	}

	// 3. Default path exists
	def := defaultConfigPath()
	if _, err := os.Stat(def); err == nil {
		return def, nil
	}

	// 4. Fallback: local mac.toml in cwd (original behaviour)
	if _, err := os.Stat(flagPath); err == nil {
		return flagPath, nil
	}

	// 5. Interactive first-run prompt
	return runInitPrompt(def)
}

// runInitPrompt asks the user for their mac.toml URL, downloads it, and
// saves it to dest. Returns the saved path on success.
func runInitPrompt(dest string) (string, error) {
	Banner("First-run setup")
	Info("No config found. Enter the raw URL to your mac.toml to get started.")
	Info("Example: https://raw.githubusercontent.com/you/dotfiles/main/mac.toml")
	fmt.Print("\n  Config URL (or ENTER to skip): ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading input: %w", err)
	}
	url := strings.TrimSpace(line)

	if url == "" {
		return "", fmt.Errorf("no config found — run: mac apply -c /path/to/mac.toml")
	}

	Info(fmt.Sprintf("Downloading config from %s …", url))
	data, err := downloadURL(url)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("saving config: %w", err)
	}

	Ok(fmt.Sprintf("Config saved to %s", dest))
	return dest, nil
}

func downloadURL(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return data, nil
}
