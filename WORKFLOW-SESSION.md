# WORKFLOW-SESSION.md — Arbiter

**Role:** tool — centralized architectural enforcement engine  
**ADR:** ADR-047  
**Version:** v0.1.0  
**Repo:** github.com/Harshmaury/Arbiter  
**Local path:** ~/workspace/projects/engx/services/arbiter

---

## Start of session checklist

```bash
cd ~/workspace/projects/engx/services/arbiter
git pull
go build ./...       # must be clean
go test ./...        # must be clean
go vet ./...         # must be clean
```

---

## What Arbiter does

Arbiter is a **packaging and execution gate**. It enforces architectural
invariants defined in ADRs before code is packaged (zp) or executed (engx run).

It does not run as a daemon. It has no HTTP server. It is a library + CLI.

```
zp <service>        →  arbiter.VerifyPackaging()  →  write ZIP (or block)
engx run <service>  →  arbiter.VerifyExecution()  →  start service (or block)
arbiter verify .    →  full static check, exit 0 or 1
```

---

## Rules at a glance

| ID       | Family    | What it checks                                    | Phase |
|----------|-----------|---------------------------------------------------|-------|
| A-T-001  | Temporal  | Service backed by an ADR                          | 2     |
| A-T-002  | Temporal  | nexus.yaml version present                        | 1 ✓   |
| A-T-003  | Temporal  | ADR sequence has no gap >5                        | 3     |
| A-S-001  | Spatial   | No cross-service internal/ imports                | 1 ✓   |
| A-S-002  | Spatial   | go.mod module name matches nexus.yaml id          | 1 ✓   |
| A-S-003  | Spatial   | No raw net/http in observer collector files       | 1 ✓   |
| A-A-001  | Authority | Observer services never call write endpoints      | 1 ✓   |
| A-A-002  | Authority | Forge only calls permitted Nexus endpoints        | 3     |
| A-A-003  | Authority | nexus.yaml role is present and valid              | 1 ✓   |
| A-C-001  | Contract  | No hardcoded "X-Service-Token" outside Canon      | 1 ✓   |
| A-C-002  | Contract  | No hardcoded "X-Trace-ID" outside Canon           | 1 ✓   |
| A-C-003  | Contract  | No hardcoded "X-Identity-Token" outside Canon     | 1 ✓   |
| A-C-004  | Contract  | No local EventType definitions                    | 1 ✓   |
| A-C-005  | Contract  | depends_on entries resolve to known service IDs   | 2     |

Phase 1 ✓ = implemented and tested in v0.1.0.

---

## Using Arbiter locally

```bash
# Build the binary
cd ~/workspace/projects/engx/services/arbiter
go build -o ~/bin/arbiter ./cmd/arbiter/

# Verify a specific service
arbiter verify ~/workspace/projects/engx/services/guardian

# Verify all services at once
cd ~/workspace/projects/engx/services
arbiter verify ./...

# List all rules
arbiter rules
```

Set governance dir if auto-detection fails:
```bash
export ARBITER_GOVERNANCE_DIR=~/workspace/projects/engx/engx-governance/architecture/decisions
arbiter verify .
```

---

## Integration status

| Integration point       | Status          | Notes                                  |
|-------------------------|-----------------|----------------------------------------|
| zp packaging gate       | Patch written   | ZP_INTEGRATION_PATCH.md                |
| engx run Enforcing step | Patch written   | NEXUS_INTEGRATION_PATCH.md             |
| arbiter verify CLI      | ✅ Implemented  | Phase 1 static rules                   |
| Guardian G-019          | Planned         | Detect --skip-enforce usage (Phase 3)  |
| CI template             | Planned         | Phase 3                                |

Apply the Nexus and ZP patches before tagging Arbiter v0.1.0 as production-active.

---

## Adding a new rule

1. Add the rule ID constant to `internal/rules/rule.go`
2. Implement the rule function in the appropriate family file
3. Register the function in `internal/engine/engine.go` (staticRules or dynamicRules)
4. Add test cases in the corresponding `_test.go` file
5. Update the rules table in this file and README.md
6. Update ADR-047 compliance section if the rule changes the compliance bar

Rule function signature: `func(ctx *rules.ProjectContext) []*rules.Violation`

---

## Never

- Add daemon behavior (HTTP server, background loop) without a new ADR
- Import Herald or any platform service — Arbiter must have zero platform dependencies
- Enforce business logic — rules must reflect existing ADRs only
- Modify any file in the project being verified
