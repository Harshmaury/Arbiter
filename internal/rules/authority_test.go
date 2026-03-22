// @arbiter-project: arbiter
// @arbiter-path: internal/rules/authority_test.go
package rules

import "testing"

func TestRuleRoleFieldPresent(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		wantViolations int
	}{
		{"valid control", "control", 0},
		{"valid observer", "observer", 0},
		{"valid tool", "tool", 0},
		{"valid library", "library", 0},
		{"empty role", "", 1},
		{"invalid role", "daemon", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx("myservice", "")
			ctx.Manifest.Role = tt.role
			got := RuleRoleFieldPresentFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}

func TestRuleObserverWriteCall(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		src            string
		wantViolations int
	}{
		{
			name:           "observer with http.Post — violation",
			serviceID:      "guardian",
			src:            `resp, err := http.Post(url, "application/json", body)`,
			wantViolations: 1,
		},
		{
			name:           "observer read-only — clean",
			serviceID:      "guardian",
			src:            `resp, err := http.Get(url)`,
			wantViolations: 0,
		},
		{
			name:           "non-observer with http.Post — not checked",
			serviceID:      "forge",
			src:            `resp, err := http.Post(url, "application/json", body)`,
			wantViolations: 0,
		},
		{
			name:           "comment not caught",
			serviceID:      "guardian",
			src:            `// resp, err := http.Post(url, ...)`,
			wantViolations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := makeCtx(tt.serviceID, tt.src)
			got := RuleObserverWriteCallFn(ctx)
			if len(got) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(got), tt.wantViolations)
			}
		})
	}
}
