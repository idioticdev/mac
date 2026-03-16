package main

import (
	"fmt"
	"math"
)

func applyDefaults(cfg *Config, r Runner) {
	if len(cfg.Defaults) == 0 {
		return
	}

	Banner("macOS Defaults")

	for domain, entries := range cfg.Defaults {
		Info("Domain: " + domain)

		for key, val := range entries {
			args := buildDefaultsArgs(domain, key, val)
			if args == nil {
				Warn(fmt.Sprintf("  %s: unsupported type %T", key, val))
				continue
			}

			if _, err := r.Run("defaults", args...); err != nil {
				Fail(fmt.Sprintf("  %s: %s", key, err.Error()))
			} else {
				Ok(fmt.Sprintf("  %s = %v", key, val))
			}
		}
	}
}

func uninstallDefaults(cfg *Config, r Runner) {
	if len(cfg.Defaults) == 0 {
		return
	}

	Banner("macOS Defaults")

	for domain, entries := range cfg.Defaults {
		Info("Domain: " + domain)
		for key := range entries {
			if _, err := r.Run("defaults", "delete", domain, key); err != nil {
				// defaults delete exits non-zero when the key is already absent — that's fine.
				Skip(fmt.Sprintf("  %s (already absent)", key))
			} else {
				Ok(fmt.Sprintf("  %s deleted", key))
			}
		}
	}

	Warn("Prior values were not captured; keys were deleted, not restored to their original values.")
}

func buildDefaultsArgs(domain, key string, val any) []string {
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
