// @arbiter-project: arbiter
// @arbiter-path: internal/rules/authority.go
// Authority rules (A-A) — role compliance.
// ADR-047: observer services never call write endpoints;
//          nexus.yaml role is present and valid.
package rules

import (
	"fmt"
	"strings"
)

// writeMethods is the set of HTTP method patterns that constitute a write operation.
// Patterns are matched as substrings — keep them specific enough to avoid
// false-positives on read-only methods (e.g. ".Get(" must not appear here).
var writeMethods = []string{
	`http.Post(`,
	`http.Put(`,
	`http.Patch(`,
	`http.NewRequest("POST"`,
	`http.NewRequest("PUT"`,
	`http.NewRequest("PATCH"`,
	`http.NewRequest("DELETE"`,
	`http.NewRequestWithContext(ctx, "POST"`,
	`http.NewRequestWithContext(ctx, "PUT"`,
	`http.NewRequestWithContext(ctx, "PATCH"`,
	`http.NewRequestWithContext(ctx, "DELETE"`,
}

// ── A-A-001: observer services never call write endpoints ─────────────────────

func RuleObserverWriteCallFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return nil
	}
	if !ObserverServiceIDs[ctx.Manifest.ID] {
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
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, method := range writeMethods {
				if strings.Contains(trimmed, method) {
					violations = append(violations, &Violation{
						RuleID:   RuleObserverWriteCall,
						Severity: SeverityError,
						Location: fmt.Sprintf("%s:%d", sf.Path, i+1),
						Message:  fmt.Sprintf("observer service %q contains write call %q (ADR-020)", ctx.Manifest.ID, method),
						Hint:     "observer services are strictly read-only — remove this write call. If this is intentional, escalate via a new ADR before proceeding",
					})
				}
			}
		}
	}
	return violations
}

// ── A-A-003: nexus.yaml role is present and valid ─────────────────────────────

func RuleRoleFieldPresentFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return []*Violation{{
			RuleID:   RuleRoleFieldPresent,
			Severity: SeverityError,
			Location: "nexus.yaml",
			Message:  "nexus.yaml not found or could not be parsed",
			Hint:     "every project must have a valid nexus.yaml with an id, role, and version field",
		}}
	}
	if ctx.Manifest.Role == "" {
		return []*Violation{{
			RuleID:   RuleRoleFieldPresent,
			Severity: SeverityError,
			Location: "nexus.yaml",
			Message:  fmt.Sprintf("nexus.yaml for %q has no role field", ctx.Manifest.ID),
			Hint: fmt.Sprintf(
				"add: role: <one of %s>",
				validRoleList(),
			),
		}}
	}
	if !ValidRoles[ctx.Manifest.Role] {
		return []*Violation{{
			RuleID:   RuleRoleFieldPresent,
			Severity: SeverityError,
			Location: "nexus.yaml",
			Message:  fmt.Sprintf("invalid role %q in nexus.yaml for %q", ctx.Manifest.Role, ctx.Manifest.ID),
			Hint:     fmt.Sprintf("valid roles: %s", validRoleList()),
		}}
	}
	return nil
}

func validRoleList() string {
	roles := make([]string, 0, len(ValidRoles))
	for r := range ValidRoles {
		roles = append(roles, r)
	}
	return strings.Join(roles, " | ")
}
