// @arbiter-project: arbiter
// @arbiter-path: internal/rules/contract.go
// Contract rules (A-C) — Canon compliance and contract ownership.
// ADR-047: every shared constant must originate from Canon.
// ADR-016, ADR-045: ServiceTokenHeader, TraceIDHeader, IdentityTokenHeader,
//                   EventType constants — never redefined outside Canon.
package rules

import (
	"fmt"
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
