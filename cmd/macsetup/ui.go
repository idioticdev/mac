package main

import (
	"fmt"
	"os"

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
	fmt.Println()
	fmt.Println(bannerStyle.Render("▸ " + msg))
}

func Info(msg string) {
	fmt.Printf("  %s %s\n", symbolInfo, msg)
}

func Ok(msg string) {
	fmt.Printf("  %s %s\n", symbolOk, msg)
}

func Warn(msg string) {
	fmt.Printf("  %s %s\n", symbolWarn, msg)
}

func Fail(msg string) {
	fmt.Printf("  %s %s\n", symbolFail, msg)
}

func Skip(msg string) {
	fmt.Printf("  %s %s (skipped)\n", symbolSkip, msg)
}

func Header() {
	title := "macsetup — v1.0.0\nDeclarative macOS Configuration"
	fmt.Println()
	fmt.Println(headerStyle.Render(title))
	fmt.Println()
}

func Done() {
	fmt.Println()
	fmt.Println(doneStyle.Render("✅ macsetup complete!"))
	Warn("Some changes may require a logout/restart to take full effect.")
}

func Logger() *log.Logger {
	return logger
}
