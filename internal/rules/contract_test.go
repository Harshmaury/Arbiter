// @arbiter-project: arbiter
// @arbiter-path: internal/rules/contract_test.go
package rules

import (
	"testing"
)

func TestRuleHardcodedServiceToken(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		src        string
		wantViolations int
	}{
		{
			name: "clean file",
			id:   "atlas",
			src:  `req.Header.Set(canon.ServiceTokenHeader, token)`,
			wantViolations: 0,
		},
		{
			name: "hardcoded literal",
			id:   "atlas",
			src:  `req.Header.Set("X-Service-Token", token)`,
			wantViolations: 1,
		},
		{
			name: "canon is exempt",
			id:   "canon",
			src:  `const ServiceTokenHeader = "X-Service-Token"`,
			wantViolations: 0,
		},
		{
			name: "comment is still caught",
			id:   "forge",
			src:  `// header: "X-Service-Token" — do not use directly`,
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.id, tt.src)
			got := RuleHardcodedServiceTokenFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d (violations: %v)", len(got), tt.wantViolations, got)
			}
		})
	}
}

func TestRuleHardcodedTraceID(t *testing.T) {
	ctx := makeCtx("nexus", `req.Header.Set("X-Trace-ID", id)`)
	got := RuleHardcodedTraceIDFn(ctx)
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(got))
	}
	if got[0].RuleID != RuleHardcodedTraceID {
		t.Errorf("wrong rule ID: %s", got[0].RuleID)
	}
	if got[0].Hint == "" {
		t.Error("hint must be non-empty")
	}
}

func TestRuleLocalEventType(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		src            string
		wantViolations int
	}{
		{
			name: "redefines EventType",
			id:   "forge",
			src:  "type EventType = string",
			wantViolations: 1,
		},
		{
			name: "canon exempt",
			id:   "canon",
			src:  "type EventType = string",
			wantViolations: 0,
		},
		{
			name: "clean import usage",
			id:   "forge",
			src:  `canonevents.EventServiceStarted`,
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.id, tt.src)
			got := RuleLocalEventTypeFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

// makeCtx builds a minimal ProjectContext with one source file.
func makeCtx(serviceID, src string) *ProjectContext {
	return &ProjectContext{
		Dir: "/tmp/fake/" + serviceID,
		Manifest: &NexusManifest{
			ID:      serviceID,
			Role:    "execution",
			Version: "v0.1.0",
		},
		SourceFiles: []SourceFile{
			{Path: "internal/handler/handler.go", Content: src},
		},
	}
}
