package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Adaptive hex colors — calibrated for both dark and light terminals.
var (
	colorPrimary = lipgloss.AdaptiveColor{Dark: "#79B8FF", Light: "#0366D6"}
	colorSuccess = lipgloss.AdaptiveColor{Dark: "#A8CC8C", Light: "#2E7D32"}
	colorWarning = lipgloss.AdaptiveColor{Dark: "#E3B341", Light: "#855E00"}
	colorError   = lipgloss.AdaptiveColor{Dark: "#F97583", Light: "#CB2431"}
	colorMuted   = lipgloss.AdaptiveColor{Dark: "#6A737D", Light: "#959DA5"}
)

var (
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	symbolOk   = lipgloss.NewStyle().Foreground(colorSuccess).Render("✓")
	symbolInfo = lipgloss.NewStyle().Foreground(colorPrimary).Render("→")
	symbolWarn = lipgloss.NewStyle().Foreground(colorWarning).Render("!")
	symbolFail = lipgloss.NewStyle().Foreground(colorError).Render("✗")
	symbolSkip = lipgloss.NewStyle().Foreground(colorMuted).Render("·")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 2).
			Align(lipgloss.Center)

	doneStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})
)

func Banner(msg string) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bannerStyle.Render("▸ "+msg))
}

func Info(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symbolInfo, msg)
}

func Ok(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symbolOk, msg)
}

func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symbolWarn, msg)
}

// failRecorder is set during mac apply to collect failure messages for the
// end-of-run summary. It is nil outside of apply runs.
var failRecorder func(string)

func Fail(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symbolFail, msg)
	if failRecorder != nil {
		failRecorder(msg)
	}
}

func Skip(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s (skipped)\n", symbolSkip, msg)
}

func Header() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, headerStyle.Render("mac · declarative macOS"))
	fmt.Fprintln(os.Stderr)
}

func Done() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, doneStyle.Render("  ✓ mac complete"))
	Warn("Some changes may require a logout or restart.")
}

func Logger() *log.Logger {
	return logger
}

// Confirm prints a [y/N] prompt and returns true only if the user types "y" or "yes".
func Confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "\n  %s %s [y/N] ", symbolWarn, prompt)
	var response string
	fmt.Scanln(&response) //nolint:errcheck
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}
