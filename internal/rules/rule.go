// @arbiter-project: arbiter
// @arbiter-path: internal/rules/rule.go
// Package rules defines Arbiter enforcement rule types and constants.
// Rules are stateless pure functions: given a ProjectContext, they return Violations.
// ADR-047: rule taxonomy A-T-*, A-S-*, A-A-*, A-C-*.
package rules

import "time"

// ── Rule IDs ─────────────────────────────────────────────────────────────────

// Temporal rules — ADR coverage and sequencing.
const (
	RuleADRCoverage    = "A-T-001" // nexus.yaml id backed by at least one ADR
	RuleVersionPresent = "A-T-002" // nexus.yaml version field non-empty
	RuleADRSequence    = "A-T-003" // ADR files sequentially numbered (no gap >5)
)

// Spatial rules — import boundaries and topology.
const (
	RuleCrossServiceImport  = "A-S-001" // no import of another service's internal/
	RuleModuleNameMatch     = "A-S-002" // go.mod module name matches nexus.yaml id
	RuleRawHTTPInCollector  = "A-S-003" // no net/http direct calls in collector files
)

// Authority rules — role compliance.
const (
	RuleObserverWriteCall    = "A-A-001" // observers never call write endpoints
	RuleForgeNexusEndpoints  = "A-A-002" // Forge calls only permitted Nexus endpoints
	RuleRoleFieldPresent     = "A-A-003" // nexus.yaml role is present and valid
)

// Contract rules — Canon compliance and contract ownership.
const (
	RuleHardcodedServiceToken  = "A-C-001" // no literal "X-Service-Token" outside Canon
	RuleHardcodedTraceID       = "A-C-002" // no literal "X-Trace-ID" outside Canon
	RuleHardcodedIdentityToken = "A-C-003" // no literal "X-Identity-Token" outside Canon
	RuleLocalEventType         = "A-C-004" // no local EventType const definitions
	RuleDependsOnResolvable    = "A-C-005" // depends_on entries resolve to known IDs
)

// ── Severity ──────────────────────────────────────────────────────────────────

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// ── Types ─────────────────────────────────────────────────────────────────────

// Violation is a single rule failure.
type Violation struct {
	RuleID   string
	Severity string // SeverityError | SeverityWarning
	Location string // "file.go:line" or "" for project-level checks
	Message  string
	Hint     string // actionable fix suggestion, always non-empty
}

// Report is the result of a full Arbiter evaluation pass.
type Report struct {
	Violations  []*Violation
	Passed      []string // rule IDs that produced no violations
	EvaluatedAt time.Time
}

// OK returns true when the report has no violations.
func (r *Report) OK() bool { return len(r.Violations) == 0 }

// HasErrors returns true when at least one violation has severity "error".
func (r *Report) HasErrors() bool {
	for _, v := range r.Violations {
		if v.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ProjectContext is the full input to Arbiter's rule engine.
// It is populated by the loader before rules are evaluated.
type ProjectContext struct {
	// Dir is the absolute path to the project root.
	Dir string

	// Manifest is the parsed nexus.yaml.
	Manifest *NexusManifest

	// GoMod is the parsed go.mod module line. Empty for non-Go projects.
	GoMod string

	// SourceFiles is the list of .go source files (relative to Dir).
	// Does not include vendor/ or testdata/.
	SourceFiles []SourceFile

	// ADRFiles is the list of ADR filenames found in engx-governance.
	// Populated only when GovernanceDir is resolvable.
	ADRFiles []string

	// KnownServiceIDs is the set of service IDs registered in the live Nexus.
	// Populated only for execution-gate checks (Phase 2).
	KnownServiceIDs map[string]bool
}

// NexusManifest is the parsed content of a nexus.yaml file.
type NexusManifest struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Role        string   `yaml:"role"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	DependsOn   []string `yaml:"depends_on"`
}

// SourceFile represents a single Go source file with its content.
type SourceFile struct {
	Path    string // relative to project Dir
	Content string // full file content
}

// ValidRoles is the set of permitted nexus.yaml role values (ADR-047 §A-A-003).
var ValidRoles = map[string]bool{
	"control":   true,
	"knowledge": true,
	"execution": true,
	"observer":  true,
	"library":   true,
	"tool":      true,
	"contract":  true,
	"client":    true,
}

// ObserverServiceIDs are the five observer services (ports 8083–8087).
// Used by A-A-001 to identify which projects are observer-role.
var ObserverServiceIDs = map[string]bool{
	"metrics":   true,
	"navigator": true,
	"guardian":  true,
	"observer":  true,
	"sentinel":  true,
}
