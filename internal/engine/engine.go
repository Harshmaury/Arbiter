// @arbiter-project: arbiter
// @arbiter-path: internal/engine/engine.go
// Package engine orchestrates Arbiter rule evaluation.
// It wires all rule families (A-T, A-S, A-A, A-C) against a ProjectContext
// and produces a single Report. ADR-047.
package engine

import (
	"time"

	"github.com/Harshmaury/Arbiter/internal/rules"
)

// ruleFunc is the signature of every rule evaluation function.
type ruleFunc func(ctx *rules.ProjectContext) []*rules.Violation

// staticRules is the ordered list of rules evaluated in Phase 1 (packaging gate).
// These require only filesystem access — no live Nexus needed.
var staticRules = []struct {
	id string
	fn ruleFunc
}{
	// Temporal
	{rules.RuleVersionPresent, rules.RuleVersionPresentFn},
	// Spatial
	{rules.RuleModuleNameMatch, rules.RuleModuleNameMatchFn},
	{rules.RuleCrossServiceImport, rules.RuleCrossServiceImportFn},
	{rules.RuleRawHTTPInCollector, rules.RuleRawHTTPInCollectorFn},
	// Authority
	{rules.RuleRoleFieldPresent, rules.RuleRoleFieldPresentFn},
	{rules.RuleObserverWriteCall, rules.RuleObserverWriteCallFn},
	// Contract
	{rules.RuleHardcodedServiceToken, rules.RuleHardcodedServiceTokenFn},
	{rules.RuleHardcodedTraceID, rules.RuleHardcodedTraceIDFn},
	{rules.RuleHardcodedIdentityToken, rules.RuleHardcodedIdentityTokenFn},
	{rules.RuleLocalEventType, rules.RuleLocalEventTypeFn},
	{rules.RuleCanonVersionFloor, rules.RuleCanonVersionFloorFn},
}

// dynamicRules is the ordered list of rules evaluated in Phase 2 (execution gate).
// These require a live Nexus connection.
var dynamicRules = []struct {
	id string
	fn ruleFunc
}{
	{rules.RuleADRCoverage, rules.RuleADRCoverageFn},
	{rules.RuleDependsOnResolvable, rules.RuleDependsOnResolvableFn},
}

// EvaluateStatic runs all static rules and returns a Report.
// Called by the zp packaging gate.
func EvaluateStatic(ctx *rules.ProjectContext) *rules.Report {
	return evaluate(ctx, staticRules)
}

// EvaluateDynamic runs dynamic rules (requires KnownServiceIDs and ADRFiles populated).
// Called by the engx execution gate.
func EvaluateDynamic(ctx *rules.ProjectContext) *rules.Report {
	return evaluate(ctx, dynamicRules)
}

// EvaluateAll runs both static and dynamic rules.
// Called by `arbiter verify ./...` in CI.
func EvaluateAll(ctx *rules.ProjectContext) *rules.Report {
	combined := append(staticRules, dynamicRules...)
	return evaluate(ctx, combined)
}

func evaluate(ctx *rules.ProjectContext, set []struct {
	id string
	fn ruleFunc
}) *rules.Report {
	report := &rules.Report{
		EvaluatedAt: time.Now().UTC(),
	}
	for _, r := range set {
		violations := r.fn(ctx)
		if len(violations) == 0 {
			report.Passed = append(report.Passed, r.id)
		} else {
			report.Violations = append(report.Violations, violations...)
		}
	}
	return report
}
