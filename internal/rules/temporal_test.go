// @arbiter-project: arbiter
// @arbiter-path: internal/rules/temporal_test.go
package rules

import "testing"

func TestRuleVersionPresent(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		wantViolations int
	}{
		{"version set", "v0.1.0", 0},
		{"empty version", "", 1},
		{"whitespace only", "   ", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx("myservice", "")
			ctx.Manifest.Version = tt.version
			got := RuleVersionPresentFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

func TestRuleADRCoverage(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		adrFiles       []string
		wantViolations int
	}{
		{
			name:      "ADR exists for service",
			serviceID: "atlas",
			adrFiles: []string{
				"ADR-006-atlas-context-source-for-forge.md",
				"ADR-009-atlas-phase3-metadata-contract.md",
			},
			wantViolations: 0,
		},
		{
			name:           "no ADR for service",
			serviceID:      "relay",
			adrFiles:       []string{"ADR-001-project-registry.md", "ADR-002-workspace.md"},
			wantViolations: 1,
		},
		{
			name:           "no ADR files available — skip",
			serviceID:      "relay",
			adrFiles:       nil,
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.serviceID, "")
			ctx.ADRFiles = tt.adrFiles
			got := RuleADRCoverageFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

func TestExtractADRNumbers(t *testing.T) {
	files := []string{
		"ADR-001-registry.md",
		"ADR-003-comms.md",
		"ADR-010-forge.md",
	}
	got := extractADRNumbers(files)
	want := []int{1, 3, 10}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("index %d: got %d, want %d", i, got[i], v)
		}
	}
}
