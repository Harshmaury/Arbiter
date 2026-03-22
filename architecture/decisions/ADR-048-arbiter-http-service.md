# ADR-048 — Arbiter HTTP Service

Date: 2026-03-22
Status: Accepted
Domain: Arbiter (tool) — enforcement gateway
Depends on: ADR-047 (Arbiter enforcement engine), ADR-041 (Relay)
Port: 9094

---

## Context

ADR-047 defines Arbiter as a library + CLI tool. The packaging and execution
gates call Arbiter via Go function calls (`arbiter.VerifyPackaging`,
`arbiter.VerifyExecution`). This works for zp and engx — both are Go binaries
on the same machine.

Relay has a different requirement. Before accepting any inbound tunnel
connection, Relay must verify that the platform's architectural invariants are
satisfied — specifically that the service being exposed has passed Arbiter
validation. Relay is deployed on a remote server (relay.engx.dev). It cannot
call Go functions on the developer's local machine.

The solution: expose Arbiter's verification logic as an HTTP endpoint that
Relay (and future remote services) can call synchronously.

---

## Decision

Arbiter gains an optional HTTP server on port `:9094`.

The server starts only when `ARBITER_HTTP_ADDR` is set. When unset, Arbiter
behaves exactly as before — library + CLI only, no daemon.

### Endpoint

```
POST /arbiter/verify
Content-Type: application/json

{
  "project_dir": "/home/harsh/workspace/projects/engx/services/atlas",
  "mode": "packaging"   // "packaging" | "execution"
}
```

Response:

```json
{
  "ok": true,
  "data": {
    "passed": true,
    "violations": [],
    "passed_rules": ["A-T-002", "A-S-001", "A-S-002", ...],
    "evaluated_at": "2026-03-22T19:09:00Z"
  }
}
```

On violations:

```json
{
  "ok": true,
  "data": {
    "passed": false,
    "violations": [
      {
        "rule_id": "A-C-001",
        "severity": "error",
        "location": "internal/api/handler.go:14",
        "message": "hardcoded \"X-Service-Token\"",
        "hint": "import Canon and use canon.ServiceTokenHeader"
      }
    ],
    "passed_rules": ["A-T-002", "A-S-002"],
    "evaluated_at": "2026-03-22T19:09:01Z"
  }
}
```

### Auth

`X-Service-Token` required (ADR-008). Token set via `ARBITER_SERVICE_TOKEN`.
`GET /health` always exempt.

### Port

`:9094` — registered in nexus.yaml. Follows platform port convention
(8080–8087 observers, 8088 Gate, 9090/91 Relay, 9092/93 Conduit, 9094 Arbiter).

---

## Compliance

A v0.1.0 HTTP implementation satisfies this ADR when:

1. `POST /arbiter/verify` returns a valid report for any project dir.
2. `GET /health` returns `{"status":"ok"}` without auth.
3. Server only starts when `ARBITER_HTTP_ADDR` is set.
4. `X-Service-Token` is validated on all non-health endpoints.
5. `project_dir` missing or unreadable returns HTTP 400 with clear error.

---

## Next ADR

ADR-049 — Relay pre-launch capability check: Relay calls `POST /arbiter/verify`
before forwarding any tunnel connection from a project that has not yet passed
packaging validation.
