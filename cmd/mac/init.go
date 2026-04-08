package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
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
//  2. Existing file at the default path
//  3. Interactive prompt → download → save to default path
func resolveConfigPath(flagPath string) (string, error) {
	// 1. Explicit flag
	if flagPath != "" && flagPath != "mac.toml" {
		return flagPath, nil
	}

	// 2. Default path exists
	def := defaultConfigPath()
	if _, err := os.Stat(def); err == nil {
		return def, nil
	}

	// 3. Fallback: local mac.toml in cwd (original behaviour)
	if _, err := os.Stat(flagPath); err == nil {
		return flagPath, nil
	}

	// 4. Interactive first-run prompt (URL only — for full wizard use: mac init)
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

	var url string
	form := accessibleForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Config URL (leave blank to abort)").
				Placeholder("https://raw.githubusercontent.com/you/dotfiles/main/mac.toml").
				Value(&url),
		),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("no config found — run: mac init")
	}
	url = strings.TrimSpace(url)

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
//
// When url is non-empty the wizard is skipped and the config is downloaded
// non-interactively. This is used by: mac init --url <url>
func runInit(r Runner, outputPath, url string) {
	if outputPath == "" {
		outputPath = defaultConfigPath()
	}

	// Non-interactive: URL provided directly (e.g. from install.sh or CI).
	if url != "" {
		Banner("mac init — downloading config")
		Info(fmt.Sprintf("Fetching config from %s …", url))
		data, err := downloadURL(url)
		if err != nil {
			Fail(err.Error())
			return
		}
		if err := writeConfig(outputPath, data); err != nil {
			Fail(err.Error())
			return
		}
		Ok("Config saved to " + outputPath)
		promptShellIntegration()
		printNextSteps(r, outputPath)
		return
	}

	Banner("mac init — setup wizard")

	// Warn if a config already exists.
	if _, err := os.Stat(outputPath); err == nil {
		Warn("Config already exists at " + outputPath)
		overwrite := false
		overwriteForm := accessibleForm(huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Overwrite existing config?").
					Value(&overwrite),
			),
		))
		if err := overwriteForm.Run(); err != nil || !overwrite {
			Ok("Keeping existing config.")
			printNextSteps(r, outputPath)
			return
		}
	}

	var choice string
	choiceForm := accessibleForm(huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How would you like to create your mac.toml?").
				Options(
					huh.NewOption("Download from a URL         (you have an existing config)", "url"),
					huh.NewOption("Generate from Homebrew state (recommended for existing machines)", "brew"),
					huh.NewOption("Blank starter template       (start from scratch)", "blank"),
				).
				Value(&choice),
		),
	))
	if err := choiceForm.Run(); err != nil {
		return
	}

	switch choice {
	case "url":
		initFromURL(r, outputPath)
	case "brew":
		initFromBrew(r, outputPath)
	default:
		initBlank(r, outputPath)
	}
}

func initFromURL(r Runner, dest string) {
	var url string
	urlForm := accessibleForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Config URL").
				Placeholder("https://raw.githubusercontent.com/you/dotfiles/main/mac.toml").
				Value(&url),
		),
	))
	if err := urlForm.Run(); err != nil {
		return
	}
	url = strings.TrimSpace(url)
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
	promptShellIntegration()
	printNextSteps(r, dest)
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

	if df := promptDotfilesSetup(r); df != nil {
		if err := appendDotfilesToConfig(dest, df); err != nil {
			Warn("Could not append dotfiles section: " + err.Error())
		} else {
			Ok("Dotfiles section added to config")
		}
	}

	Warn("Uncomment and edit sections as needed, then run: mac apply")
	promptShellIntegration()
	printNextSteps(r, dest)
}

func initBlank(r Runner, dest string) {
	computerName, _ := r.Run("scutil", "--get", "ComputerName")
	localHostname, _ := r.Run("scutil", "--get", "LocalHostName")
	if computerName == "" {
		computerName, _ = r.Run("hostname", "-s")
	}
	if localHostname == "" {
		localHostname = computerName
	}

	df := promptDotfilesSetup(r)

	dotfilesSection := buildDotfilesSection(df)

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

%s
[system]
pam_tid = true   # Enable Touch ID for sudo

[hooks]
post_install = [
    "softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true",
]
`, computerName, localHostname, dotfilesSection)

	if err := writeConfig(dest, []byte(content)); err != nil {
		Fail(err.Error())
		return
	}
	Ok("Starter config written to " + dest)
	promptShellIntegration()
	printNextSteps(r, dest)
}

// dotfilesSetup holds values collected by promptDotfilesSetup.
type dotfilesSetup struct {
	repo     string
	dest     string
	stowPkgs []string
}

// stdinByteReader wraps an io.Reader to return one byte per Read call.
// huh's accessible mode creates a new bufio.Scanner per field — without this
// wrapper, the first scanner consumes the entire pipe buffer, starving later
// fields. Reading one byte at a time keeps each scanner from pre-buffering
// beyond its own line.
type stdinByteReader struct{ r io.Reader }

func (s *stdinByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return s.r.Read(p[:1])
}

// isCharDevice reports whether src is a real terminal (character device).
func isCharDevice(src io.Reader) bool {
	if f, ok := src.(*os.File); ok {
		fi, err := f.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice != 0
	}
	return false
}

// promptDotfilesSetup asks whether the user has a dotfiles repo and collects
// the URL, destination, and stow packages. It runs SSH setup inline so that
// the first mac apply can clone without interruption. Returns nil if skipped.
//
// opts are applied to each huh.Form before Run() — useful in tests for
// injecting an io.Reader via f.WithAccessible(true) and f.WithInput(r).
func promptDotfilesSetup(r Runner, opts ...func(*huh.Form)) *dotfilesSetup {
	// Default: when stdin is not a real TTY (e.g. a pipe in tests or CI),
	// enable accessible mode and wrap stdin in a byte-at-a-time reader.
	// The byte wrapper prevents huh's per-field bufio.Scanner from consuming
	// input meant for later fields. opts (from tests) may override this.
	defaultReader := func(f *huh.Form) {
		stdin := os.Stdin
		if !isCharDevice(stdin) {
			f.WithAccessible(true)
			f.WithInput(&stdinByteReader{stdin})
		}
	}

	applyOpts := func(f *huh.Form) {
		defaultReader(f)
		for _, opt := range opts {
			opt(f)
		}
	}

	var hasDotfiles bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Do you have a dotfiles repo?").
				Value(&hasDotfiles),
		),
	)
	applyOpts(confirmForm)
	if err := confirmForm.Run(); err != nil || !hasDotfiles {
		return nil
	}

	var repo, dest, stowPkgsStr string
	detailForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Repo URL").
				Placeholder("git@github.com:you/dotfiles.git").
				Value(&repo),
			huh.NewInput().
				Title("Destination").
				Placeholder("~/.dotfiles").
				Value(&dest),
			huh.NewInput().
				Title("Stow packages (comma-separated, or leave blank to skip)").
				Value(&stowPkgsStr),
		),
	)
	applyOpts(detailForm)
	if err := detailForm.Run(); err != nil {
		return nil
	}

	repo = strings.TrimSpace(repo)
	if repo == "" {
		Warn("No repo URL provided — skipping dotfiles setup")
		return nil
	}

	dest = strings.TrimSpace(dest)
	if dest == "" {
		dest = "~/.dotfiles"
	}

	var stowPkgs []string
	for _, p := range strings.Split(strings.TrimSpace(stowPkgsStr), ",") {
		if p := strings.TrimSpace(p); p != "" {
			stowPkgs = append(stowPkgs, p)
		}
	}

	ensureSSHForClone(repo, r)

	return &dotfilesSetup{repo: repo, dest: dest, stowPkgs: stowPkgs}
}

// buildDotfilesSection returns the TOML dotfiles section for initBlank.
// Returns a commented-out placeholder when df is nil.
func buildDotfilesSection(df *dotfilesSetup) string {
	if df == nil {
		return `# [dotfiles]
# repo          = "git@github.com:you/dotfiles.git"
# dest          = "~/.dotfiles"
# method        = "stow"
# stow_packages = ["zsh", "git"]
`
	}

	var sb strings.Builder
	sb.WriteString("[dotfiles]\n")
	fmt.Fprintf(&sb, "repo   = %q\n", df.repo)
	fmt.Fprintf(&sb, "dest   = %q\n", df.dest)
	sb.WriteString(`method = "stow"` + "\n")
	if len(df.stowPkgs) > 0 {
		quoted := make([]string, len(df.stowPkgs))
		for i, p := range df.stowPkgs {
			quoted[i] = fmt.Sprintf("%q", p)
		}
		sb.WriteString("stow_packages = [" + strings.Join(quoted, ", ") + "]\n")
	}
	return sb.String()
}

// appendDotfilesToConfig appends a [dotfiles] section to an existing TOML file.
func appendDotfilesToConfig(configPath string, df *dotfilesSetup) error {
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s", buildDotfilesSection(df))
	return err
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

func printNextSteps(_ Runner, configPath string) {
	cfg, err := ResolveConfig(configPath)
	if err != nil {
		fmt.Println()
		fmt.Printf("  Next steps:\n")
		fmt.Printf("    mac diff   -c %s\n", configPath)
		fmt.Printf("    mac apply  -c %s\n", configPath)
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("  Previewing changes …\n")
	fmt.Println()
	hasChanges := runDiff(cfg)
	fmt.Println()

	if hasChanges {
		fi, err := os.Stdin.Stat()
		isTTY := err == nil && (fi.Mode()&os.ModeCharDevice) != 0
		if isTTY {
			applyNow := true
			applyForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Apply now?").
						Value(&applyNow),
				),
			)
			if applyForm.Run() == nil && applyNow {
				runApply(cfg)
				fmt.Println()
				return
			}
		}
		fmt.Printf("  Run when ready: mac apply -c %s\n", configPath)
	} else {
		fmt.Println("  No changes to apply — system already matches config.")
	}
	fmt.Println()
}

// shellRCPath returns the rc file path for the user's current shell,
// or ("", false) if the shell is unsupported or undetectable.
func shellRCPath() (string, bool) {
	shell := os.Getenv("SHELL")
	home := os.Getenv("HOME")
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc"), true
	case strings.Contains(shell, "bash"):
		return filepath.Join(home, ".bash_profile"), true
	default:
		return "", false
	}
}

// promptShellIntegration asks the user whether to add the mac shell-init
// eval line to their shell rc file. It is a no-op if already present.
func promptShellIntegration() {
	rcPath, ok := shellRCPath()
	if !ok {
		Info("Shell integration: add this line to your shell config manually:")
		fmt.Println(`      eval "$(mac shell-init)"`)
		return
	}

	const evalLine = `eval "$(mac shell-init)"`

	// Check if already wired up.
	existing, _ := os.ReadFile(rcPath)
	if strings.Contains(string(existing), evalLine) {
		Ok("Shell integration already in " + rcPath)
		return
	}

	addShellIntegration := true
	shellForm := accessibleForm(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add brew tracking to " + rcPath + "?").
				Value(&addShellIntegration),
		),
	))
	if shellForm.Run() != nil || !addShellIntegration {
		Info("Skipped. To add later: echo 'eval \"$(mac shell-init)\"' >> " + rcPath)
		return
	}

	block := "\n# mac shell integration — track brew installs in mac.toml\n" + evalLine + "\n"
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		Warn("Could not write to " + rcPath + ": " + err.Error())
		Info("Add this line manually: " + evalLine)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		Warn("Could not write to " + rcPath + ": " + err.Error())
		Info("Add this line manually: " + evalLine)
		return
	}

	Ok("Shell integration added to " + rcPath)
	Info("Restart your terminal or run: source " + rcPath)
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
