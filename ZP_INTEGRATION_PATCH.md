# ZP_INTEGRATION_PATCH.md — Arbiter → zp packaging gate

**Applies to:** `~/workspace/projects/tools/zp`  
**Phase:** ADR-047 §3.1 — Packaging gate (Phase 1)  
**Effect:** `zp` calls `arbiter.VerifyPackaging` before writing any ZIP.
            Violations block the package; no ZIP is written.

---

## 1. go.mod — add Arbiter dependency

```
require github.com/Harshmaury/Arbiter v0.1.0
```

After adding: `go mod tidy`

---

## 2. Locate the package-writing entry point in zp

Find the function that constructs and writes the ZIP (typically in `main.go`
or `cmd/package.go`). It will contain something like:

```go
zipWriter := zip.NewWriter(outFile)
// ... adds files ...
zipWriter.Close()
fmt.Printf("✓ %s\n", outputPath)
```

---

## 3. Insert the Arbiter gate immediately before ZIP creation

Add this import:

```go
arbiter "github.com/Harshmaury/Arbiter/api"
```

Add this block **before** `zip.NewWriter(outFile)`:

```go
// ── Arbiter packaging gate (ADR-047) ─────────────────────────────────────────
report, err := arbiter.VerifyPackaging(projectDir)
if err != nil {
    return fmt.Errorf("arbiter: %w", err)
}
if !report.OK() {
    fmt.Fprintf(os.Stderr, "\nArbiter gate — violations found:\n")
    fmt.Fprint(os.Stderr, arbiter.FormatReport(report))
    return fmt.Errorf("package blocked — resolve %d violation(s) and re-run zp", len(report.Violations))
}
// ─────────────────────────────────────────────────────────────────────────────
```

Where `projectDir` is the directory containing the project's `nexus.yaml`
(the root of the project being packaged).

---

## 4. Expected output on violation

```
$ zp atlas

Arbiter gate — violations found:

  ✗ A-C-001  internal/api/handler/auth.go:14 — hardcoded "X-Service-Token"
             → import Canon and use canon.ServiceTokenHeader
  ✗ A-S-001  internal/collector/old.go:7 — cross-service import: atlas imports nexus internal package
             → services communicate via HTTP/JSON only (ADR-003)

  2 violation(s). Package blocked. Resolve errors and re-run.
```

---

## 5. Expected output on clean pass

```
$ zp atlas
  ✓ arbiter   10 rule(s) passed
  ✓ atlas-full-20260322-1316.zip
```

---

## 6. Verification

```bash
# On a clean project
zp atlas
# Should: pass Arbiter gate → write ZIP as normal

# On a project with a Canon violation (test)
# Add a literal "X-Service-Token" string to any .go file, then:
zp atlas
# Should: print violation, exit non-zero, no ZIP written
```

---

## 7. Environment variable passthrough

If `ARBITER_GOVERNANCE_DIR` is set in the shell, Arbiter auto-picks it up.
No special handling needed in zp. The ADR auto-discovery works relative to
the project directory.
