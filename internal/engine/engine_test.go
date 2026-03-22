// @arbiter-project: arbiter
// @arbiter-path: internal/engine/engine_test.go
package engine

import (
	"testing"

	"github.com/Harshmaury/Arbiter/internal/rules"
)

func TestEvaluateStatic_CleanProject(t *testing.T) {
	ctx := &rules.ProjectContext{
		Dir: "/tmp/fake/forge",
		Manifest: &rules.NexusManifest{
			ID:      "forge",
			Role:    "execution",
			Version: "v0.5.0",
		},
		GoMod: "github.com/Harshmaury/Forge",
		SourceFiles: []rules.SourceFile{
			{
				Path:    "internal/executor/executor.go",
				Content: `import canonid "github.com/Harshmaury/Canon/identity"`,
			},
		},
	}
	report := EvaluateStatic(ctx)
	if !report.OK() {
		for _, v := range report.Violations {
			t.Logf("violation: %s %s %s", v.RuleID, v.Location, v.Message)
		}
		t.Errorf("expected clean report, got %d violation(s)", len(report.Violations))
	}
	if len(report.Passed) == 0 {
		t.Error("expected at least one passed rule")
	}
}

func TestEvaluateStatic_MultipleViolations(t *testing.T) {
	ctx := &rules.ProjectContext{
		Dir: "/tmp/fake/atlas",
		Manifest: &rules.NexusManifest{
			ID:      "atlas",
			Role:    "knowledge",
			Version: "v0.5.0",
		},
		GoMod: "github.com/Harshmaury/Atlas",
		SourceFiles: []rules.SourceFile{
			{
				Path: "internal/api/handler.go",
				// Two Canon violations in one file
				Content: `
req.Header.Set("X-Service-Token", token)
req.Header.Set("X-Trace-ID", traceID)
`,
			},
		},
	}
	report := EvaluateStatic(ctx)
	if report.OK() {
		t.Fatal("expected violations, got clean report")
	}

	// Count error-severity violations
	errCount := 0
	for _, v := range report.Violations {
		if v.Severity == rules.SeverityError {
			errCount++
		}
	}
	if errCount < 2 {
		t.Errorf("expected at least 2 error violations, got %d", errCount)
	}
}

func TestEvaluateStatic_MissingRole(t *testing.T) {
	ctx := &rules.ProjectContext{
		Dir: "/tmp/fake/unknown",
		Manifest: &rules.NexusManifest{
			ID:      "unknown",
			Role:    "", // missing
			Version: "v0.1.0",
		},
		GoMod: "github.com/Harshmaury/Unknown",
	}
	report := EvaluateStatic(ctx)
	found := false
	for _, v := range report.Violations {
		if v.RuleID == rules.RuleRoleFieldPresent {
			found = true
		}
	}
	if !found {
		t.Error("expected A-A-003 violation for missing role")
	}
}

func TestReport_HasErrors(t *testing.T) {
	r := &rules.Report{
		Violations: []*rules.Violation{
			{RuleID: "A-C-001", Severity: rules.SeverityError},
		},
	}
	if !r.HasErrors() {
		t.Error("expected HasErrors() == true")
	}

	r2 := &rules.Report{
		Violations: []*rules.Violation{
			{RuleID: "A-T-003", Severity: rules.SeverityWarning},
		},
	}
	if r2.HasErrors() {
		t.Error("expected HasErrors() == false for warning-only report")
	}
}
