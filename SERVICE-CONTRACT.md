// @arbiter-project: arbiter
// @arbiter-path: SERVICE-CONTRACT.md
# SERVICE-CONTRACT.md — Arbiter
# @version: 0.1.0
# @updated: 2026-03-25

**Type:** Library + CLI · **Module:** `github.com/Harshmaury/Arbiter` · **Domain:** Tool

---

## Code

```
api/arbiter.go              VerifyPackaging / VerifyExecution / VerifyAll / FormatReport
internal/engine/engine.go   rule evaluation loop
internal/rules/temporal.go  A-T-* -- ADR coverage, version field present
internal/rules/spatial.go   A-S-* -- import boundaries, Herald usage in collectors
internal/rules/authority.go A-A-* -- observer write prohibition, role validity
internal/rules/contract.go  A-C-* -- Canon usage, no local EventType definitions
internal/loader/loader.go   loads project context from nexus.yaml + go files
internal/probe/nexus.go     EmitSkipEnforceAlert -- typed SystemAlertPayload
cmd/arbiter/main.go         CLI: verify / rules / help
```

---

## Contract

**Go API:**
```go
report, err := arbiter.VerifyPackaging(dir)
report, err := arbiter.VerifyExecution(addr, token, dir)
report, err := arbiter.VerifyAll(dir, addr, token)
output   := arbiter.FormatReport(report)
```

**Report types:**
```go
Report{Violations []*Violation, Passed []string, EvaluatedAt time.Time}
Violation{RuleID, Severity, Location, Message, Hint string}  // Hint always non-empty
```

**CLI:**
```
arbiter verify [path|./...]   exit 0 clean / 1 violations / 2 error
arbiter rules                 stdout: rule table
```

Canon is exempt from A-C-* rules.

---

## Control

| Gate | Trigger | Runs |
|------|---------|------|
| Packaging | `zp` pre-ZIP | `VerifyPackaging` -- static only |
| Execution | `engx run` Enforce step | `VerifyExecution` -- static + dynamic |

No persistent state. Every call is a fresh evaluation. Read-only.

`VerifyExecution` dynamic rules skip gracefully when `KnownServiceIDs` is empty.

---

## Context

Enforcement layer -- blocks before. Guardian observes after. Does not replace Guardian. All rules reflect existing ADRs only.
