// @arbiter-project: arbiter
// @arbiter-path: internal/rules/spatial_test.go
package rules

import "testing"

func TestRuleCrossServiceImport(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		src            string
		wantViolations int
	}{
		{
			name:           "clean — no cross-service import",
			id:             "atlas",
			src:            `import "github.com/Harshmaury/Canon/identity"`,
			wantViolations: 0,
		},
		{
			name:           "violation — atlas imports nexus internal",
			id:             "atlas",
			src:            `import "github.com/Harshmaury/Nexus/internal/state"`,
			wantViolations: 1,
		},
		{
			name:           "violation — forge imports atlas internal",
			id:             "forge",
			src:            `import "github.com/Harshmaury/Atlas/internal/graph"`,
			wantViolations: 1,
		},
		{
			name:           "self import is fine",
			id:             "nexus",
			src:            `import "github.com/Harshmaury/Nexus/internal/state"`,
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.id, tt.src)
			got := RuleCrossServiceImportFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

func TestRuleModuleNameMatch(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		gomod          string
		wantViolations int
	}{
		{
			name:           "match — title case",
			id:             "atlas",
			gomod:          "github.com/Harshmaury/Atlas",
			wantViolations: 0,
		},
		{
			name:           "mismatch — wrong service",
			id:             "atlas",
			gomod:          "github.com/Harshmaury/Nexus",
			wantViolations: 1,
		},
		{
			name:           "match — exact case",
			id:             "nexus",
			gomod:          "github.com/Harshmaury/Nexus",
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.id, "")
			ctx.GoMod = tt.gomod
			got := RuleModuleNameMatchFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

func TestRuleRawHTTPInCollector(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		path           string
		src            string
		wantViolations int
	}{
		{
			name:           "observer collector with raw http.Get",
			serviceID:      "guardian",
			path:           "internal/collector/nexus.go",
			src:            `resp, err := http.Get(baseURL + "/services")`,
			wantViolations: 1,
		},
		{
			name:           "observer collector using Herald — clean",
			serviceID:      "guardian",
			path:           "internal/collector/nexus.go",
			src:            `svcs, err := c.Services().List(ctx)`,
			wantViolations: 0,
		},
		{
			name:           "non-observer raw http.Get — not checked",
			serviceID:      "forge",
			path:           "internal/collector/nexus.go",
			src:            `resp, err := http.Get(url)`,
			wantViolations: 0,
		},
		{
			name:           "observer non-collector file — not checked",
			serviceID:      "guardian",
			path:           "internal/api/handler/findings.go",
			src:            `resp, err := http.Get(url)`,
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ProjectContext{
				Dir: "/tmp/fake/" + tt.serviceID,
				Manifest: &NexusManifest{
					ID:      tt.serviceID,
					Role:    "observer",
					Version: "v0.1.0",
				},
				SourceFiles: []SourceFile{
					{Path: tt.path, Content: tt.src},
				},
			}
			got := RuleRawHTTPInCollectorFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}
