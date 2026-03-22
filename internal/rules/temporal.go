// @arbiter-project: arbiter
// @arbiter-path: internal/rules/temporal.go
// Temporal rules (A-T) — ADR coverage and sequencing.
// ADR-047: every implemented capability must be backed by an ADR.
package rules

import (
	"fmt"
	"strings"
)

// ── A-T-001: nexus.yaml id backed by at least one ADR ────────────────────────
// This rule requires ADRFiles to be populated (Phase 2 — execution gate).

func RuleADRCoverageFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return nil
	}
	if len(ctx.ADRFiles) == 0 {
		// ADR files not available — skip silently (governance dir not found).
		return nil
	}

	id := strings.ToLower(ctx.Manifest.ID)
	for _, adrFile := range ctx.ADRFiles {
		lower := strings.ToLower(adrFile)
		if strings.Contains(lower, id) {
			return nil // at least one ADR references this service
		}
	}

	return []*Violation{{
		RuleID:   RuleADRCoverage,
		Severity: SeverityError,
		Location: "nexus.yaml",
		Message: fmt.Sprintf(
			"service %q has no backing ADR in engx-governance/decisions/",
			ctx.Manifest.ID,
		),
		Hint: "add an ADR that covers this service before packaging or running it (ADR-first rule, §Rule 9)",
	}}
}

// ── A-T-002: nexus.yaml version field is present and non-empty ───────────────

func RuleVersionPresentFn(ctx *ProjectContext) []*Violation {
	if ctx.Manifest == nil {
		return nil
	}
	if strings.TrimSpace(ctx.Manifest.Version) == "" {
		return []*Violation{{
			RuleID:   RuleVersionPresent,
			Severity: SeverityError,
			Location: "nexus.yaml",
			Message:  fmt.Sprintf("nexus.yaml for %q has no version field", ctx.Manifest.ID),
			Hint:     "add: version: v0.1.0 (semver, matches latest git tag)",
		}}
	}
	return nil
}

// ── A-T-003: ADR files sequentially numbered (no gap >5) ─────────────────────

func RuleADRSequenceFn(ctx *ProjectContext) []*Violation {
	if len(ctx.ADRFiles) < 2 {
		return nil
	}

	numbers := extractADRNumbers(ctx.ADRFiles)
	if len(numbers) < 2 {
		return nil
	}

	var violations []*Violation
	prev := numbers[0]
	for _, n := range numbers[1:] {
		gap := n - prev
		if gap > 5 {
			violations = append(violations, &Violation{
				RuleID:   RuleADRSequence,
				Severity: SeverityWarning,
				Location: "engx-governance/decisions/",
				Message:  fmt.Sprintf("ADR sequence gap: ADR-%03d → ADR-%03d (gap of %d)", prev, n, gap),
				Hint:     "large gaps may indicate undocumented decisions — verify intentional",
			})
		}
		prev = n
	}
	return violations
}

// extractADRNumbers parses ADR sequence numbers from filenames like
// "ADR-007-forge-phase3-automation.md" → [7, ...].
func extractADRNumbers(files []string) []int {
	var nums []int
	for _, f := range files {
		base := strings.ToUpper(f)
		if !strings.HasPrefix(base, "ADR-") {
			continue
		}
		rest := f[4:] // strip "ADR-"
		dashIdx := strings.Index(rest, "-")
		var numStr string
		if dashIdx > 0 {
			numStr = rest[:dashIdx]
		} else {
			numStr = rest
		}
		var n int
		fmt.Sscanf(numStr, "%d", &n)
		if n > 0 {
			nums = append(nums, n)
		}
	}
	return nums
}
