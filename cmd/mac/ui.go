package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	colorBlue   = lipgloss.Color("69")
	colorCyan   = lipgloss.Color("87")
	colorGreen  = lipgloss.Color("76")
	colorYellow = lipgloss.Color("220")
	colorRed    = lipgloss.Color("196")

	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlue)

	symbolOk   = lipgloss.NewStyle().Foreground(colorGreen).Render("✓")
	symbolInfo = lipgloss.NewStyle().Foreground(colorCyan).Render("→")
	symbolWarn = lipgloss.NewStyle().Foreground(colorYellow).Render("⚠")
	symbolFail = lipgloss.NewStyle().Foreground(colorRed).Render("✗")
	symbolSkip = lipgloss.NewStyle().Foreground(colorYellow).Render("–")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlue).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBlue).
			Padding(0, 2).
			Align(lipgloss.Center)

	doneStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

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

func Fail(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symbolFail, msg)
}

func Skip(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s (skipped)\n", symbolSkip, msg)
}

func Header() {
	title := "mac — v1.0.0\nDeclarative macOS Configuration"
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, headerStyle.Render(title))
	fmt.Fprintln(os.Stderr)
}

func Done() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, doneStyle.Render("✅ mac complete!"))
	Warn("Some changes may require a logout/restart to take full effect.")
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
