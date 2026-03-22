# ADR-047 — Arbiter: Centralized Enforcement Engine

Date: 2026-03-22
Status: Accepted
Domain: Platform-wide — zp, engx, engx-governance
Depends on: ADR-016 (Canon), ADR-019 (zp), ADR-020 (Observer governance),
            ADR-044 (Runtime mode), ADR-045 (Canon v1.0.0)
Unblocks: Relay safe-exposure guarantee, v3 trust model

---

## Context

The platform defines a comprehensive set of architectural invariants across
five correctness dimensions:

| Dimension   | Example invariant                                         | ADR(s)          |
|-------------|-----------------------------------------------------------|-----------------|
| Temporal    | No capability exists without a prior ADR                  | Evo rules       |
| Spatial     | No cross-service internal package imports                 | ADR-003         |
| Authority   | Observers never call write endpoints                      | ADR-020         |
| Contract    | All shared constants imported from Canon only             | ADR-016, ADR-045|
| Operational | Platform mode is declared and visible before execution    | ADR-044         |

These invariants are currently:

- **Defined** in ADRs and governance docs
- **Partially enforced** at runtime (Guardian G-001..G-018 observe violations
  that already exist; ADR-044 makes mode visible)
- **Not enforced** at the packaging or execution gate

This means a developer can produce an invalid artifact (zp), commit a
Canon-violation (hardcoded header string), or start a service that imports
another service's internal package — and receive no signal until runtime
observation detects the consequence.

The 2026-03-22 proposal document states this precisely:

> "There is currently no single enforcement point that guarantees:
> No invalid architectural state can be created, packaged, or executed."

Guardian is the observation layer — it sees what is true at runtime.
Arbiter is the enforcement layer — it prevents invalid states from forming.

---

## Decision

Introduce **Arbiter** as a standalone Go module that:

1. Evaluates all architectural invariants as a library
2. Integrates into `zp` as a pre-packaging gate
3. Integrates into `engx run` as a new plan step (Validate → **Enforce** → Start → Wait → Health)
4. Exposes a standalone CLI (`arbiter verify [path]`) for direct use

### 1. Module identity

```
repo:    github.com/Harshmaury/Arbiter
port:    none (library-first; HTTP service deferred to ADR-048 if needed)
role:    enforcement — evaluate and block, never observe running state
version: v0.1.0
```

Arbiter is a **library with a CLI face**, not a daemon. It has no persistent
state, no background loop, and no HTTP endpoints in v0.1.0. It is called
synchronously at the enforcement points described below.

### 2. Rule taxonomy

All rules carry an `A-` prefix to distinguish them from Guardian rules (G-).

```
A-T-nnn  Temporal   — ADR coverage and sequencing
A-S-nnn  Spatial    — import boundaries and service topology
A-A-nnn  Authority  — role compliance (observer/control/execution/knowledge)
A-C-nnn  Contract   — Canon compliance and contract ownership
```

#### Temporal rules (A-T)

| Rule    | Check                                                                       | Blocked by       |
|---------|-----------------------------------------------------------------------------|------------------|
| A-T-001 | nexus.yaml `id` matches at least one ADR in engx-governance/decisions/      | zp, engx run     |
| A-T-002 | nexus.yaml `version` field is present and non-empty                         | zp               |
| A-T-003 | ADR files are sequentially numbered (no gaps > 5 in sequence)               | CI optional      |

#### Spatial rules (A-S)

| Rule    | Check                                                                       | Blocked by       |
|---------|-----------------------------------------------------------------------------|------------------|
| A-S-001 | No Go source file imports another service's `internal/` package             | zp               |
| A-S-002 | Module name in go.mod matches `nexus.yaml` id (case-normalised)             | zp               |
| A-S-003 | No direct stdlib `net/http` client calls in observer collector files        | zp               |

A-S-003 enforces ADR-020: all inter-service calls in observer services must
go through Herald (no raw `http.Get` in collector files).

#### Authority rules (A-A)

| Rule    | Check                                                                       | Blocked by       |
|---------|-----------------------------------------------------------------------------|------------------|
| A-A-001 | Observer service (port 8083–8087) has no write method (`POST`/`PUT`/`DELETE`/`PATCH`) calls to non-self URLs | zp |
| A-A-002 | Forge HTTP calls to Nexus use only the two permitted endpoints (`/projects/:id/start`, `/projects/:id/stop`) | zp |
| A-A-003 | nexus.yaml `role` field is present and is one of: control, knowledge, execution, observer, library, tool, contract, client | zp |

#### Contract rules (A-C)

| Rule    | Check                                                                       | Blocked by       |
|---------|-----------------------------------------------------------------------------|------------------|
| A-C-001 | No source file contains the literal string `"X-Service-Token"` outside of Canon | zp         |
| A-C-002 | No source file contains the literal string `"X-Trace-ID"` outside of Canon | zp               |
| A-C-003 | No source file contains the literal string `"X-Identity-Token"` outside of Canon | zp         |
| A-C-004 | No service defines its own EventType constants instead of importing Canon   | zp               |
| A-C-005 | `nexus.yaml` `depends_on` entries all resolve to known service IDs          | zp, engx run     |

### 3. Enforcement points

#### 3.1 Packaging gate (zp)

`zp` calls `arbiter.VerifyPackaging(dir)` before writing any ZIP.

If any rule fails, `zp` prints the violations and exits non-zero.
No ZIP is written. The format mirrors `engx doctor`:

```
✗ A-C-001  internal/api/handler/auth.go:14 — hardcoded "X-Service-Token"
           → import from Canon: canon.ServiceTokenHeader
✗ A-S-001  internal/collector/nexus.go:7 — imports Nexus internal/state
           → remove cross-service internal import
2 violation(s). Package blocked. Resolve errors and re-run zp.
```

#### 3.2 Execution gate (engx run)

`buildRunPlan` in `cmd/engx/cmd_run.go` gains a new step between
Validating and Starting:

```go
{
    Label: "Enforcing",
    Kind:  plan.KindEnforce,
    Run:   stepEnforce(httpAddr, serviceToken, id),
},
```

`stepEnforce` calls `arbiter.VerifyExecution(nexusAddr, token, projectID)`,
which runs the subset of rules that require a live Nexus connection
(A-T-001, A-C-005). Static rules (A-S-*, A-A-*, A-C-001..A-C-004) are
intentionally skipped at the execution gate — they must pass the packaging
gate before the artifact can exist.

If execution-gate checks fail:

```
  ✗ Enforcing   A-T-001 — project "relay" has no backing ADR in engx-governance
               → add ADR before implementing this capability
               Use --skip-enforce to bypass (audit logged).
```

The `--skip-enforce` flag is available for migration and development
contexts. Its use is emitted as a `SYSTEM_ALERT` event with level `warn`
so Guardian can detect it (future G-019 candidate).

#### 3.3 CI integration (optional)

`arbiter verify ./...` runs the full static rule set against all projects
in a workspace. This is the CI enforcement path. Exit code 0 = all clear,
1 = violations found.

### 4. Public API surface

```go
// package api — public surface consumed by zp and engx

// VerifyPackaging runs all static rules against the project at dir.
// dir must contain a nexus.yaml. Returns a Report; never panics.
func VerifyPackaging(dir string) (*Report, error)

// VerifyExecution runs dynamic rules that require a live Nexus connection.
func VerifyExecution(nexusAddr, serviceToken, projectID string) (*Report, error)

// Report is the result of a rule evaluation pass.
type Report struct {
    Violations []*Violation
    Passed     []string      // rule IDs that passed
    EvaluatedAt time.Time
}

func (r *Report) OK() bool { return len(r.Violations) == 0 }

// Violation is a single rule failure.
type Violation struct {
    RuleID   string
    Severity string   // "error" | "warning"
    Location string   // "file.go:line" or "" for project-level
    Message  string
    Hint     string   // actionable fix suggestion
}
```

### 5. nexus.yaml for Arbiter

```yaml
id: arbiter
name: Arbiter
role: tool
version: v0.1.0
description: Centralized architectural enforcement engine
runtime:
  binary: arbiter
  port: 0
```

Port 0 signals: this component has no HTTP server in v0.1.0.

### 6. Relationship to existing components

| Component    | Role                           | Arbiter relationship                   |
|--------------|--------------------------------|----------------------------------------|
| ADRs         | Define invariants              | Arbiter reads ADR files as evidence    |
| Canon        | Define shared contracts        | Arbiter enforces Canon adoption        |
| Guardian     | Observe and report (runtime)   | Complementary — Guardian fires after; Arbiter blocks before |
| zp           | Package artifacts              | Arbiter gates zp output                |
| engx run     | Execute services               | Arbiter gates execution                |
| Nexus        | Registry and runtime           | Arbiter queries for A-T-001, A-C-005  |

---

## Compliance

A v0.1.0 Arbiter implementation satisfies this ADR when:

1. `arbiter.VerifyPackaging(dir)` returns a non-empty `Report.Violations` for
   any project that violates rules A-C-001, A-C-002, A-C-003, A-S-001, or A-A-003.

2. `zp` calls `VerifyPackaging` before writing any ZIP and exits non-zero on
   violations.

3. `engx run` plan includes an `Enforcing` step that calls `VerifyExecution`
   and blocks on violations unless `--skip-enforce` is passed.

4. `arbiter verify ./...` exits 0 on a clean platform workspace and 1 when
   any rule fails.

5. All violation output includes a `Hint` field with an actionable fix.

---

## What this ADR does not change

- Guardian rules G-001..G-018 are unchanged.
- Runtime behavior of all services is unchanged.
- No existing endpoint is modified.
- ADRs remain the source of invariant definition — Arbiter enforces them,
  never defines new ones.

---

## Implementation order

**Phase 1 — Static enforcement (v0.1.0)**
- Arbiter module: `internal/rules`, `internal/loader`, `api/`
- Rules: A-C-001..A-C-004, A-S-001..A-S-003, A-A-003, A-T-002
- `arbiter verify` CLI
- zp integration

**Phase 2 — Execution gate (v0.2.0)**
- Rules: A-T-001, A-C-005 (require live Nexus)
- `engx run` Enforcing step
- `--skip-enforce` flag with SYSTEM_ALERT emission

**Phase 3 — Full temporal coverage (v0.3.0)**
- A-T-003: ADR sequence gap detection
- A-A-001, A-A-002: authority boundary checks (require AST analysis)
- Guardian G-019: alert on `--skip-enforce` usage
- CI workflow template

---

## Next ADR

ADR-048 — Arbiter HTTP service: expose `POST /arbiter/verify` for remote
enforcement queries (required before Relay can call Arbiter to validate
its own capability coverage pre-launch).
