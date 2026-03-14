package main

import (
	"fmt"
	"math"
)

func applyDefaults(cfg *Config) {
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

			if _, err := Run("defaults", args...); err != nil {
				Fail(fmt.Sprintf("  %s: %s", key, err.Error()))
			} else {
				Ok(fmt.Sprintf("  %s = %v", key, val))
			}
		}
	}
}

func buildDefaultsArgs(domain, key string, val interface{}) []string {
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
