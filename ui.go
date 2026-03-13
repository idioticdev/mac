package ui

import "fmt"

const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	cyan   = "\033[0;36m"
	bold   = "\033[1m"
	nc     = "\033[0m"
)

func Banner(msg string) {
	fmt.Printf("\n%s%s▸ %s%s\n", blue, bold, msg, nc)
}

func Info(msg string) {
	fmt.Printf("  %s→%s %s\n", cyan, nc, msg)
}

func Ok(msg string) {
	fmt.Printf("  %s✓%s %s\n", green, nc, msg)
}

func Warn(msg string) {
	fmt.Printf("  %s⚠%s %s\n", yellow, nc, msg)
}

func Fail(msg string) {
	fmt.Printf("  %s✗%s %s\n", red, nc, msg)
}

func Skip(msg string) {
	fmt.Printf("  %s–%s %s (skipped)\n", yellow, nc, msg)
}

func Header() {
	fmt.Printf("\n%s%s", bold, blue)
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Println("  │         macsetup — v1.0.0            │")
	fmt.Println("  │   Declarative macOS Configuration    │")
	fmt.Printf("  └─────────────────────────────────────┘\n%s\n", nc)
}

func Done() {
	fmt.Printf("\n%s%s✅ macsetup complete!%s\n", green, bold, nc)
	Warn("Some changes may require a logout/restart to take full effect.")
}
