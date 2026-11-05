package connect

import (
	"regexp"
	"strings"
)

// The patterns and the line walk below follow Clear-CodexConfig in
// install/codex-restore.ps1 line by line. PowerShell -match ignores case,
// hence (?i). Tests run both on the same fixtures.
var (
	tableLine     = regexp.MustCompile(`(?i)^[\t ]*\[\[?[^\]]+\]\]?[\t ]*(?:#.*)?$`)
	providerTable = regexp.MustCompile(`(?i)^[\t ]*\[model_providers\.[^\]]+\][\t ]*(?:#.*)?$`)
	modelRootKey  = regexp.MustCompile(`(?i)^[\t ]*(?:model_provider|model|review_model)[\t ]*=`)
	lineBreak     = regexp.MustCompile("\r\n|\n|\r")
)

// CleanConfig removes the root model_provider, model and review_model keys
// and every [model_providers.*] table, keeping all other text unchanged.
func CleanConfig(content string) string {
	// File.ReadAllText drops a UTF-8 byte order mark; the result is written
	// back without one.
	content = strings.TrimPrefix(content, "\ufeff")
	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}
	var kept []string
	inRoot := true
	skipProvider := false
	removedRoot := false
	multiline := ""
	skipValue := false

	for _, line := range lineBreak.Split(content, -1) {
		inString := multiline != ""
		multiline = updateMultiline(line, multiline)
		if inString {
			if skipProvider || skipValue {
				if multiline == "" {
					skipValue = false
				}
				continue
			}
			kept = append(kept, line)
			continue
		}

		if tableLine.MatchString(line) {
			inRoot = false
			if providerTable.MatchString(line) {
				skipProvider = true
				continue
			}
			skipProvider = false
		}
		if skipProvider {
			continue
		}
		if inRoot && modelRootKey.MatchString(line) {
			removedRoot = true
			skipValue = multiline != ""
			continue
		}
		if removedRoot && len(kept) == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, newline)
}

// updateMultiline tracks TOML multi-line strings so their contents are never
// read as tables or keys (Update-MultilineState in the script).
func updateMultiline(line, open string) string {
	rest := line
	for {
		var at int
		if open == "" {
			doubleAt := strings.Index(rest, `"""`)
			singleAt := strings.Index(rest, `'''`)
			if doubleAt < 0 && singleAt < 0 {
				return open
			}
			if doubleAt >= 0 && (singleAt < 0 || doubleAt < singleAt) {
				open, at = `"""`, doubleAt
			} else {
				open, at = `'''`, singleAt
			}
		} else {
			at = strings.Index(rest, open)
			if at < 0 {
				return open
			}
			open = ""
		}
		rest = rest[at+3:]
	}
}
