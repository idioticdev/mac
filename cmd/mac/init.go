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

	// 5. Interactive first-run prompt (URL only — for full wizard use: mac init)
	return runInitPrompt(def)
}

// runInitPrompt is the lightweight first-run flow triggered when mac apply
// finds no config. It only handles the URL-download path; for the full
// guided setup wizard run: mac init
func runInitPrompt(dest string) (string, error) {
	Banner("First-run setup")
	Info("No config found at " + dest)
	Info("Enter the raw URL to your mac.toml, or run: mac init")
	Info("Example: https://raw.githubusercontent.com/you/dotfiles/main/mac.toml")
	fmt.Print("\n  Config URL (or ENTER to abort): ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading input: %w", err)
	}
	url := strings.TrimSpace(line)

	if url == "" {
		return "", fmt.Errorf("no config found — run: mac init")
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

// runInit is the full guided setup wizard invoked by: mac init [--output path]
// It offers three paths:
//  1. Download from a URL
//  2. Generate from current Homebrew state (mac export)
//  3. Write a blank starter template
func runInit(r Runner, outputPath string) {
	if outputPath == "" {
		outputPath = defaultConfigPath()
	}

	Banner("mac init — setup wizard")

	// Warn if a config already exists.
	if _, err := os.Stat(outputPath); err == nil {
		Warn("Config already exists at " + outputPath)
		fmt.Print("  Overwrite? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			Ok("Keeping existing config. Run: mac apply")
			return
		}
	}

	fmt.Println()
	fmt.Println("  How would you like to create your mac.toml?")
	fmt.Println()
	fmt.Println("    [1] Download from a URL         (you have an existing config)")
	fmt.Println("    [2] Generate from Homebrew state (recommended for existing machines)")
	fmt.Println("    [3] Blank starter template       (start from scratch)")
	fmt.Println()
	fmt.Print("  Choice [1-3]: ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(line)

	switch choice {
	case "1":
		initFromURL(outputPath)
	case "2":
		initFromBrew(r, outputPath)
	default:
		initBlank(outputPath)
	}
}

func initFromURL(dest string) {
	fmt.Print("\n  Config URL: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	url := strings.TrimSpace(line)
	if url == "" {
		Fail("No URL provided")
		return
	}
	Info(fmt.Sprintf("Downloading from %s …", url))
	data, err := downloadURL(url)
	if err != nil {
		Fail(err.Error())
		return
	}
	if err := writeConfig(dest, data); err != nil {
		Fail(err.Error())
		return
	}
	Ok("Config saved to " + dest)
	printNextSteps(dest)
}

func initFromBrew(r Runner, dest string) {
	Info("Reading current Homebrew state …")
	toml := generateExportTOML(r)
	if err := writeConfig(dest, []byte(toml)); err != nil {
		Fail(err.Error())
		return
	}
	Ok("Config saved to " + dest)
	Warn("Review the generated config — defaults and system sections are commented out.")
	Warn("Uncomment and edit sections as needed, then run: mac apply")
	printNextSteps(dest)
}

func initBlank(dest string) {
	computerName, _ := DefaultRunner.Run("scutil", "--get", "ComputerName")
	localHostname, _ := DefaultRunner.Run("scutil", "--get", "LocalHostName")
	if computerName == "" {
		computerName, _ = DefaultRunner.Run("hostname", "-s")
	}
	if localHostname == "" {
		localHostname = computerName
	}

	content := fmt.Sprintf(`# ============================================================================
# mac.toml — starter config
# Edit this file, then run:  mac apply
# Full reference: https://github.com/idioticdev/mac#config-reference
# ============================================================================

[machine]
computer_name  = %q
local_hostname = %q

[packages]
formulae = [
    "git",
    "stow",
    # "neovim",
    # "ripgrep",
    # "fzf",
]
casks = [
    # "raycast",
    # "1password",
]

# [dotfiles]
# repo          = "git@github.com:you/dotfiles.git"
# dest          = "~/.dotfiles"
# method        = "stow"
# stow_packages = ["zsh", "git"]

[system]
pam_tid = true   # Enable Touch ID for sudo

[hooks]
post_install = [
    "softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true",
]
`, computerName, localHostname)

	if err := writeConfig(dest, []byte(content)); err != nil {
		Fail(err.Error())
		return
	}
	Ok("Starter config written to " + dest)
	Warn("Open the file and customise it before running: mac apply")
	printNextSteps(dest)
}

func writeConfig(dest string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
}

func printNextSteps(configPath string) {
	fmt.Println()
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    mac diff   -c %s   # preview changes\n", configPath)
	fmt.Printf("    mac apply  -c %s   # apply config\n", configPath)
	fmt.Println()
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
