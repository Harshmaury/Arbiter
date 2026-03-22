# Arbiter

**Centralized Architectural Enforcement Engine**  
`role: tool` | `version: v0.1.0` | ADR-047

---

## What it does

Arbiter ensures that architectural invariants defined in ADRs cannot be bypassed.  
It runs **before** code is packaged (via `zp`) or executed (via `engx run`), making correctness deterministic rather than observed after the fact.

| Layer | Component | When it fires |
|---|---|---|
| Definition | ADRs (engx-governance) | Authoring time |
| Enforcement | **Arbiter** | Pre-package, pre-execution |
| Observation | Guardian (G-001..G-018) | Runtime |

---

## Rule families

| Family | Prefix | What it checks |
|---|---|---|
| Temporal | A-T | Every service is backed by an ADR; version field present |
| Spatial | A-S | No cross-service `internal/` imports; Herald used in collectors |
| Authority | A-A | Observers never call write endpoints; `role` field valid |
| Contract | A-C | No hardcoded Canon constants; no local `EventType` definitions |

Run `arbiter rules` for the full list.

---

## Usage

```bash
# Verify the current project
arbiter verify .

# Verify all projects in a workspace
arbiter verify ./...

# List all rule IDs
arbiter rules
```

**Exit codes:** `0` = all clear · `1` = violations (package blocked) · `2` = internal error

---

## Integration

### zp (packaging gate)

```go
import arbiter "github.com/Harshmaury/Arbiter/api"

report, err := arbiter.VerifyPackaging(projectDir)
if err != nil || !report.OK() {
    fmt.Print(arbiter.FormatReport(report))
    return fmt.Errorf("arbiter gate failed — package blocked")
}
```

### engx run (execution gate — Phase 2)

A new `Enforcing` step is inserted between `Validating` and `Starting` in `buildRunPlan`.  
See ADR-047 §3.2.

---

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `ARBITER_GOVERNANCE_DIR` | auto-detected | Path to `engx-governance/decisions/` |

Auto-detection walks up the directory tree looking for `engx-governance/architecture/decisions/` relative to the project being verified.

---

## Build

```bash
go build ./cmd/arbiter/
./arbiter rules
```

No external dependencies. Stdlib only.

---

## Phases

| Phase | Version | What ships |
|---|---|---|
| 1 | v0.1.0 | Static rules (A-C, A-S, A-A-003, A-T-002). `arbiter verify`. `zp` integration. |
| 2 | v0.2.0 | Dynamic rules (A-T-001, A-C-005). `engx run` Enforcing step. `--skip-enforce`. |
| 3 | v0.3.0 | A-T-003, A-A-001, A-A-002. Guardian G-019. CI template. |

---

## ADR

[ADR-047 — Arbiter: Centralized Enforcement Engine](architecture/decisions/ADR-047-arbiter-enforcement-engine.md)
