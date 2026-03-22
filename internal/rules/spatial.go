// @arbiter-project: arbiter
// @arbiter-path: internal/rules/spatial.go
// Spatial rules (A-S) — import boundaries and service topology.
// ADR-047: no cross-service internal imports; go.mod must match nexus.yaml id;
//          observer collectors must use Herald, not raw net/http.
package rules

import (
	"fmt"
	"strings"
)

// knownServiceModules maps known service IDs to their Go module paths.
// This is the canonical source — update when new services join the platform.
var knownServiceModules = map[string]string{
	"nexus":     "github.com/Harshmaury/Nexus",
	"atlas":     "github.com/Harshmaury/Atlas",
	"forge":     "github.com/Harshmaury/Forge",
	"metrics":   "github.com/Harshmaury/Metrics",
	"navigator": "github.com/Harshmaury/Navigator",
	"guardian":  "github.com/Harshmaury/Guardian",
	"observer":  "github.com/Harshmaury/Observer",
	"sentinel":  "github.com/Harshmaury/Sentinel",
	"gate":      "github.com/Harshmaury/Gate",
}

// ── A-S-001: no cross-service internal/ imports ───────────────────────────────

func RuleCrossServiceImportFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return nil
	}
	myModule := knownServiceModules[ctx.Manifest.ID]

	var violations []*Violation
	for _, sf := range ctx.SourceFiles {
		if !isGoFile(sf.Path) {
			continue
		}
		lines := strings.Split(sf.Content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Only process lines that contain a quoted import path.
			if !strings.Contains(trimmed, `"`) {
				continue
			}
			importPath := extractImportPath(trimmed)
			if importPath == "" {
				continue
			}
		// Check: imports another service's internal/ package
		for svcID, modPath := range knownServiceModules {
			if svcID == ctx.Manifest.ID {
				continue // self-import is fine
			}
			internalPrefix := modPath + "/internal"
			if strings.Contains(importPath, internalPrefix) {
				_ = myModule
				violations = append(violations, &Violation{
					RuleID:   RuleCrossServiceImport,
					Severity: SeverityError,
					Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
					Message:  fmt.Sprintf("cross-service import: %q imports %s internal package", ctx.Manifest.ID, svcID),
					Hint:     "services communicate via HTTP/JSON only (ADR-003) — remove this import and use Herald or the service's public API",
				})
			}
		}
		}
	}
	return violations
}

// extractImportPath returns the Go import path from a line, or "".
// Handles: `import "github.com/..."`, `"github.com/..."`, `alias "github.com/..."`.
func extractImportPath(line string) string {
	line = strings.TrimSpace(line)
	// Strip leading "import " keyword if present
	line = strings.TrimPrefix(line, "import ")
	line = strings.TrimSpace(line)
	start := strings.Index(line, `"`)
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(line, `"`)
	if end <= start {
		return ""
	}
	return line[start+1 : end]
}

// ── A-S-002: go.mod module name matches nexus.yaml id ────────────────────────

func RuleModuleNameMatchFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil || ctx.GoMod == "" {
		return nil
	}
	// go.mod first line: "module github.com/Harshmaury/Nexus"
	// Expected: module path ends with title-cased id
	id := ctx.Manifest.ID
	titleID := strings.ToUpper(id[:1]) + id[1:] // "nexus" → "Nexus"

	if !strings.HasSuffix(ctx.GoMod, "/"+titleID) &&
		!strings.HasSuffix(ctx.GoMod, "/"+strings.ToUpper(id)) &&
		!strings.HasSuffix(ctx.GoMod, "/"+id) {
		return []*Violation{{
			RuleID:   RuleModuleNameMatch,
			Severity: SeverityError,
			Location: "go.mod",
			Message: fmt.Sprintf(
				"go.mod module %q does not match nexus.yaml id %q",
				ctx.GoMod, id,
			),
			Hint: fmt.Sprintf(
				"module path should end with /%s (or title-case variant)",
				id,
			),
		}}
	}
	return nil
}

// ── A-S-003: no raw net/http calls in observer collector files ────────────────

func RuleRawHTTPInCollectorFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return nil
	}
	// Only applies to observer-role services.
	if !ObserverServiceIDs[ctx.Manifest.ID] {
		return nil
	}

	var violations []*Violation
	for _, sf := range ctx.SourceFiles {
		if !isGoFile(sf.Path) {
			continue
		}
		// Target collector files only (ADR-020 — Herald migration, ADR-039).
		if !strings.Contains(sf.Path, "collector") {
			continue
		}
		lines := strings.Split(sf.Content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Flag direct http.Get, http.Post, http.NewRequest calls.
			for _, rawCall := range []string{
				"http.Get(",
				"http.Post(",
				"http.NewRequest(",
				"http.NewRequestWithContext(",
				"http.DefaultClient",
			} {
				if strings.Contains(trimmed, rawCall) {
					violations = append(violations, &Violation{
						RuleID:   RuleRawHTTPInCollector,
						Severity: SeverityError,
						Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
						Message:  fmt.Sprintf("raw net/http call %q in observer collector (ADR-020, ADR-039)", rawCall),
						Hint:     "replace with Herald client — all inter-service calls in observer collectors must use herald.New() or herald.NewForService()",
					})
				}
			}
		}
	}
	return violations
}
