// @arbiter-project: arbiter
// @arbiter-path: api/arbiter.go
// Package api is the public surface of Arbiter consumed by zp and engx.
// ADR-047 §4: VerifyPackaging and VerifyExecution are the two integration
// points. Both return a *Report; callers decide how to render violations.
package api

import (
	"fmt"
	"strings"

	"github.com/Harshmaury/Arbiter/internal/engine"
	"github.com/Harshmaury/Arbiter/internal/loader"
	"github.com/Harshmaury/Arbiter/internal/probe"
	"github.com/Harshmaury/Arbiter/internal/rules"
)

// VerifyPackaging runs all static Arbiter rules against the project at dir.
// dir must contain a nexus.yaml. Returns a Report; never panics.
//
// Called by zp before writing any ZIP (ADR-047 §3.1).
func VerifyPackaging(dir string) (*rules.Report, error) {
	ctx, err := loader.LoadProject(dir)
	if err != nil {
		return nil, fmt.Errorf("arbiter: load: %w", err)
	}
	return engine.EvaluateStatic(ctx), nil
}

// VerifyExecution runs dynamic Arbiter rules that require a live Nexus
// connection. projectID is the nexus.yaml id of the project being started.
//
// Called by `engx run` Enforcing step (ADR-047 §3.2).
func VerifyExecution(nexusAddr, serviceToken, projectDir string) (*rules.Report, error) {
	ctx, err := loader.LoadProjectWithNexus(projectDir, nexusAddr, serviceToken)
	if err != nil {
		return nil, fmt.Errorf("arbiter: load: %w", err)
	}
	return engine.EvaluateDynamic(ctx), nil
}

// VerifyAll runs the full static + dynamic rule set.
// Used by `arbiter verify ./...` and CI pipelines (ADR-047 §3.3).
func VerifyAll(dir, nexusAddr, serviceToken string) (*rules.Report, error) {
	ctx, err := loader.LoadProjectWithNexus(dir, nexusAddr, serviceToken)
	if err != nil {
		return nil, fmt.Errorf("arbiter: load: %w", err)
	}
	return engine.EvaluateAll(ctx), nil
}

// SkipEnforceAlert fires a SYSTEM_ALERT event to Nexus when --skip-enforce is
// used, enabling future Guardian G-019 detection of enforcement bypass.
// Fires asynchronously — never blocks the caller.
func SkipEnforceAlert(nexusAddr, serviceToken, projectID string) {
	probe.EmitSkipEnforceAlert(nexusAddr, serviceToken, projectID)
}

// FormatReport renders a Report as human-readable CLI output.
// Format mirrors `engx doctor` output for consistency.
func FormatReport(r *rules.Report) string {
	if r.OK() {
		return fmt.Sprintf("  ✓ arbiter   %d rule(s) passed\n", len(r.Passed))
	}

	var sb strings.Builder
	for _, v := range r.Violations {
		icon := "✗"
		if v.Severity == rules.SeverityWarning {
			icon = "○"
		}
		loc := ""
		if v.Location != "" {
			loc = " " + v.Location + " —"
		}
		sb.WriteString(fmt.Sprintf("  %s %-10s%s %s\n", icon, v.RuleID, loc, v.Message))
		if v.Hint != "" {
			sb.WriteString(fmt.Sprintf("             → %s\n", v.Hint))
		}
	}
	errCount := 0
	for _, v := range r.Violations {
		if v.Severity == rules.SeverityError {
			errCount++
		}
	}
	sb.WriteString(fmt.Sprintf("\n  %d violation(s). Package blocked. Resolve errors and re-run.\n", errCount))
	return sb.String()
}
