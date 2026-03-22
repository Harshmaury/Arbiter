# SERVICE-CONTRACT.md — Arbiter

**Role:** tool  
**Version:** v0.1.0  
**ADR:** ADR-047  
**Port:** none (library + CLI; no HTTP server in v0.1.0)  
**Module:** `github.com/Harshmaury/Arbiter`

---

## Public API surface

```go
import arbiter "github.com/Harshmaury/Arbiter/api"

// Packaging gate — called by zp before writing any ZIP.
report, err := arbiter.VerifyPackaging(dir string) (*rules.Report, error)

// Execution gate — called by engx run Enforcing step (Phase 2).
report, err := arbiter.VerifyExecution(nexusAddr, serviceToken, projectDir string) (*rules.Report, error)

// CI gate — runs all static + dynamic rules.
report, err := arbiter.VerifyAll(dir, nexusAddr, serviceToken string) (*rules.Report, error)

// Report rendering — mirrors engx doctor output format.
output := arbiter.FormatReport(report *rules.Report) string
```

## Report type

```go
type Report struct {
    Violations  []*Violation
    Passed      []string      // rule IDs that passed
    EvaluatedAt time.Time
}

func (r *Report) OK() bool        // true when zero violations
func (r *Report) HasErrors() bool // true when any severity="error"

type Violation struct {
    RuleID   string // e.g. "A-C-001"
    Severity string // "error" | "warning"
    Location string // "file.go:line" or "nexus.yaml"
    Message  string
    Hint     string // always non-empty — actionable fix
}
```

## Invariants

- Arbiter is **read-only** — it never modifies any project file
- Arbiter has **no persistent state** — every call is a fresh evaluation
- Arbiter has **no HTTP server** in v0.1.0 — it is a library with a CLI face
- All violations include a non-empty `Hint` field
- Canon is explicitly exempt from A-C-* rules (it is the definition, not a consumer)
- `VerifyExecution` dynamic rules skip gracefully when KnownServiceIDs is empty

## CLI contract

```
arbiter verify [path|./...]   exit 0 (clean) or 1 (violations) or 2 (error)
arbiter rules                 stdout: rule table, exit 0
arbiter help                  stdout: usage, exit 0
```

## Does NOT

- Introduce new architectural rules (rules reflect existing ADRs only)
- Change runtime behavior of any service
- Replace Guardian observation (Arbiter blocks before; Guardian observes after)
- Enforce business logic
