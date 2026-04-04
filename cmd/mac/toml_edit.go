package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// appendToTOMLArray adds value to the named array key within [packages] in the
// TOML file at path. It preserves all comments and formatting. The write is
// atomic: content is written to a temp file and renamed on success.
//
// State machine:
//
//	START
//	  │
//	  ▼
//	scan lines
//	  │
//	  ├─► found [packages] section?
//	  │       │
//	  │       └─► scan for key = [
//	  │               │
//	  │               ├─► found: look for closing ]
//	  │               │       │
//	  │               │       ├─► inline [] → key = ["value"]
//	  │               │       ├─► inline ["x"] → key = ["x", "value"]
//	  │               │       └─► multiline → insert "    \"value\"," before ]
//	  │               │
//	  │               └─► not found (hit next section or EOF)
//	  │                       └─► append key = ["value"] after [packages] content
//	  │
//	  └─► [packages] not found → append [packages]\nkey = ["value"]
func appendToTOMLArray(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	quotedValue := `"` + value + `"`

	// Find the [packages] section.
	packagesSectionIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "[packages]" {
			packagesSectionIdx = i
			break
		}
	}

	// Find where [packages] section ends (start of next section or EOF).
	packagesSectionEnd := len(lines)
	if packagesSectionIdx >= 0 {
		for i := packagesSectionIdx + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "#") {
				packagesSectionEnd = i
				break
			}
		}
	}

	// Find the key within [packages].
	keyStart := -1
	if packagesSectionIdx >= 0 {
		for i := packagesSectionIdx + 1; i < packagesSectionEnd; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, key+" =") {
				keyStart = i
				break
			}
		}
	}

	var result []string

	switch {
	case keyStart >= 0:
		// Key found — determine inline vs multiline.
		keyLine := lines[keyStart]
		bracketIdx := strings.Index(keyLine, "[")
		if bracketIdx < 0 {
			return fmt.Errorf("unrecognized format for key %q in %s", key, path)
		}
		closingIdx := strings.LastIndex(keyLine, "]")

		if closingIdx > bracketIdx {
			// Inline array on one line.
			inner := strings.TrimSpace(keyLine[bracketIdx+1 : closingIdx])
			if inlineContains(inner, quotedValue) {
				return nil // already present
			}
			var newInner string
			if inner == "" {
				newInner = quotedValue
			} else {
				newInner = inner + ", " + quotedValue
			}
			lines[keyStart] = keyLine[:bracketIdx+1] + newInner + keyLine[closingIdx:]
			result = lines
		} else {
			// Multiline array: find the closing ] on its own line.
			closingLineIdx := -1
			for i := keyStart + 1; i < packagesSectionEnd; i++ {
				if strings.TrimSpace(lines[i]) == "]" {
					closingLineIdx = i
					break
				}
			}
			if closingLineIdx < 0 {
				return fmt.Errorf("unrecognized format: could not find closing ] for %q in %s", key, path)
			}
			// Check if value already present (skip comment lines).
			for i := keyStart + 1; i < closingLineIdx; i++ {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "#") {
					continue
				}
				entry := strings.TrimRight(strings.TrimSpace(trimmed), ",")
				if entry == quotedValue {
					return nil // already present
				}
			}
			// Detect indentation from existing entries.
			indent := "    "
			for i := keyStart + 1; i < closingLineIdx; i++ {
				line := lines[i]
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || strings.HasPrefix(trimmed, "#") {
					continue
				}
				indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				break
			}
			newLine := indent + quotedValue + ","
			result = append(lines[:closingLineIdx], append([]string{newLine}, lines[closingLineIdx:]...)...)
		}

	case packagesSectionIdx >= 0:
		// [packages] exists but key is missing — append key after last content line.
		lastContent := packagesSectionIdx
		for i := packagesSectionIdx + 1; i < packagesSectionEnd; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				lastContent = i
			}
		}
		newLine := key + ` = [` + quotedValue + `]`
		insertAt := lastContent + 1
		result = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)

	default:
		// No [packages] section — append at end.
		toAppend := []string{}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			toAppend = append(toAppend, "")
		}
		toAppend = append(toAppend, "[packages]", key+` = [`+quotedValue+`]`)
		result = append(lines, toAppend...)
	}

	return atomicWriteFile(path, []byte(strings.Join(result, "\n")))
}

// inlineContains reports whether quotedValue is an element of the comma-separated
// inner string of an inline TOML array (e.g. `"git", "ripgrep"`).
func inlineContains(inner, quotedValue string) bool {
	for _, part := range strings.Split(inner, ",") {
		if strings.TrimSpace(part) == quotedValue {
			return true
		}
	}
	return false
}

// atomicWriteFile writes data to a temp file adjacent to path, then renames it.
// This prevents partial writes from corrupting the destination file.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".mac-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
