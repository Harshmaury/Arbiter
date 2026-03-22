// @arbiter-project: arbiter
// @arbiter-path: internal/rules/contract.go
// Contract rules (A-C) — Canon compliance and contract ownership.
// ADR-047: every shared constant must originate from Canon.
// ADR-016, ADR-045: ServiceTokenHeader, TraceIDHeader, IdentityTokenHeader,
//                   EventType constants — never redefined outside Canon.
package rules

import (
	"fmt"
	"os"
	"strings"
)

// ── A-C-001: no hardcoded "X-Service-Token" outside Canon ────────────────────

func RuleHardcodedServiceTokenFn(ctx *ProjectContext) []*Violation {
	return checkHardcodedString(ctx, `"X-Service-Token"`, RuleHardcodedServiceToken,
		`import Canon and use canon.ServiceTokenHeader`,
		`import canonid "github.com/Harshmaury/Canon/identity"`)
}

// ── A-C-002: no hardcoded "X-Trace-ID" outside Canon ─────────────────────────

func RuleHardcodedTraceIDFn(ctx *ProjectContext) []*Violation {
	return checkHardcodedString(ctx, `"X-Trace-ID"`, RuleHardcodedTraceID,
		`import Canon and use canon.TraceIDHeader`,
		`import canonid "github.com/Harshmaury/Canon/identity"`)
}

// ── A-C-003: no hardcoded "X-Identity-Token" outside Canon ───────────────────

func RuleHardcodedIdentityTokenFn(ctx *ProjectContext) []*Violation {
	return checkHardcodedString(ctx, `"X-Identity-Token"`, RuleHardcodedIdentityToken,
		`import Canon and use canon.IdentityTokenHeader`,
		`import canonid "github.com/Harshmaury/Canon/identity"`)
}

// checkHardcodedString scans all non-Canon source files for literal.
// Canon itself is explicitly exempt — it is the definition, not a consumer.
func checkHardcodedString(ctx *ProjectContext, literal, ruleID, message, hint string) []*Violation {
	// Canon is the source-of-truth; it is allowed to define these strings.
	if ctx.Manifest != nil && ctx.Manifest.ID == "canon" {
		return nil
	}

	var violations []*Violation
	for _, sf := range ctx.SourceFiles {
		if !isGoFile(sf.Path) {
			continue
		}
		lines := strings.Split(sf.Content, "\n")
		for i, line := range lines {
			if strings.Contains(line, literal) {
				violations = append(violations, &Violation{
					RuleID:   ruleID,
					Severity: SeverityError,
					Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
					Message:  fmt.Sprintf("hardcoded %s — %s", literal, message),
					Hint:     hint,
				})
			}
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return violations
}

// ── A-C-004: no local EventType constant definitions ─────────────────────────

func RuleLocalEventTypeFn(ctx *ProjectContext) []*Violation {
	// Canon defines EventType. All others must import, not redefine.
	if ctx.Manifest != nil && ctx.Manifest.ID == "canon" {
		return nil
	}

	var violations []*Violation
	for _, sf := range ctx.SourceFiles {
		if !isGoFile(sf.Path) {
			continue
		}
		lines := strings.Split(sf.Content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Detect: `type EventType = string` or `type EventType string`
			if strings.HasPrefix(trimmed, "type EventType") {
				violations = append(violations, &Violation{
					RuleID:   RuleLocalEventType,
					Severity: SeverityError,
					Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
					Message:  "local EventType definition — EventType is owned by Canon",
					Hint:     `import canonevents "github.com/Harshmaury/Canon/events" and use canonevents.EventType`,
				})
			}
			// Detect: EventType const like `EventSomething EventType = "SOMETHING"`
			// only when not inside Canon itself (already guarded above).
			if strings.Contains(trimmed, "EventType =") && strings.Contains(trimmed, "\"") {
				// Allow if the line is an import alias or type assertion context
				if !strings.HasPrefix(trimmed, "//") && !strings.Contains(trimmed, "import") {
					violations = append(violations, &Violation{
						RuleID:   RuleLocalEventType,
						Severity: SeverityWarning,
						Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
						Message:  "local EventType constant — may duplicate Canon definition",
						Hint:     "verify this constant exists in Canon/events; if not, add it there",
					})
				}
			}
		}
	}
	return violations
}

// ── A-C-005: depends_on entries resolve to known service IDs ─────────────────

func RuleDependsOnResolvableFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil || len(ctx.Manifest.DependsOn) == 0 {
		return nil
	}
	if len(ctx.KnownServiceIDs) == 0 {
		// Phase 2 only — skip when no live Nexus data available.
		return nil
	}

	var violations []*Violation
	for _, dep := range ctx.Manifest.DependsOn {
		if !ctx.KnownServiceIDs[dep] {
			violations = append(violations, &Violation{
				RuleID:   RuleDependsOnResolvable,
				Severity: SeverityError,
				Location: "nexus.yaml",
				Message:  fmt.Sprintf("depends_on: %q not registered in Nexus", dep),
				Hint:     fmt.Sprintf("register the service: engx register <path-to-%s>", dep),
			})
		}
	}
	return violations
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// ── A-C-006: go.mod Canon dependency >= v1.0.0 ───────────────────────────────

// RuleCanonVersionFloorFn checks that go.mod requires Canon >= v1.0.0.
// A service using Canon v0.x is missing RelayTokenHeader, SubdomainHeader,
// OwnerHeader, and workspace payload types — all added in v1.0.0 (ADR-045).
func RuleCanonVersionFloorFn(ctx *ProjectContext) []*Violation {
	if ctx.GoMod == "" {
		return nil // non-Go project — skip
	}
	// GoMod holds the module name. We need the full go.mod content.
	// Load it from Dir + "/go.mod".
	gomodPath := ctx.Dir + "/go.mod"
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return nil // file unreadable — skip silently
	}
	src := string(data)

	// Find: github.com/Harshmaury/Canon v<version>
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "Harshmaury/Canon") {
			continue
		}
		// Extract version token — last field on the line
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		version := fields[len(fields)-1]
		if isPreV1Canon(version) {
			return []*Violation{{
				RuleID:   RuleCanonVersionFloor,
				Severity: SeverityError,
				Location: "go.mod",
				Message:  fmt.Sprintf("Canon %s is below v1.0.0 — missing RelayTokenHeader, workspace payload types", version),
				Hint:     "run: go get github.com/Harshmaury/Canon@v1.0.0 && go mod tidy",
			}}
		}
		return nil // Canon >= v1.0.0
	}
	// Canon not found in go.mod at all — services using the config/canon.go shim
	// are not yet on the real module. Warn, not error (migration in progress).
	return []*Violation{{
		RuleID:   RuleCanonVersionFloor,
		Severity: SeverityWarning,
		Location: "go.mod",
		Message:  "Canon not found in go.mod — using local shim or not imported",
		Hint:     "run: go get github.com/Harshmaury/Canon@v1.0.0 && go mod tidy",
	}}
}

// isPreV1Canon returns true when the version string indicates Canon < v1.0.0.
func isPreV1Canon(version string) bool {
	// Versions like v0.1.0, v0.3.0, v0.4.1 are all < v1.0.0
	// Versions like v1.0.0, v1.1.0 are fine
	// Pseudo-versions like v0.0.0-20260101-abc123 are also < v1.0.0
	if strings.HasPrefix(version, "v0.") {
		return true
	}
	if strings.HasPrefix(version, "v0.0.0-") {
		return true
	}
	return false
}
