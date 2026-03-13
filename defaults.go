package defaults

import (
	"fmt"
	"math"

	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

func Apply(cfg *config.Config) {
	if len(cfg.Defaults) == 0 {
		return
	}

	ui.Banner("macOS Defaults")

	for domain, entries := range cfg.Defaults {
		ui.Info("Domain: " + domain)

		for key, val := range entries {
			args := buildArgs(domain, key, val)
			if args == nil {
				ui.Warn(fmt.Sprintf("  %s: unsupported type %T", key, val))
				continue
			}

			if _, err := runexec.Run("defaults", args...); err != nil {
				ui.Fail(fmt.Sprintf("  %s: %s", key, err.Error()))
			} else {
				ui.Ok(fmt.Sprintf("  %s = %v", key, val))
			}
		}
	}
}

// buildArgs constructs the arguments for `defaults write <domain> <key> -type <val>`.
// TOML types map to defaults types:
//
//	bool   → -bool true/false
//	int64  → -int N
//	float64 → -float N  (or -int if it's a whole number)
//	string → -string S
func buildArgs(domain, key string, val interface{}) []string {
	base := []string{"write", domain, key}

	switch v := val.(type) {
	case bool:
		boolStr := "false"
		if v {
			boolStr = "true"
		}
		return append(base, "-bool", boolStr)

	case int64:
		return append(base, "-int", fmt.Sprintf("%d", v))

	case float64:
		// TOML parses integers as int64, but the generic map[string]interface{}
		// from BurntSushi/toml may give float64 for some values.
		// If it's a whole number, use -int; otherwise -float.
		if v == math.Trunc(v) {
			return append(base, "-int", fmt.Sprintf("%d", int64(v)))
		}
		return append(base, "-float", fmt.Sprintf("%g", v))

	case string:
		return append(base, "-string", v)

	default:
		return nil
	}
}
