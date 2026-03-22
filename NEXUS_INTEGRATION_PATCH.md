# NEXUS_INTEGRATION_PATCH.md — Arbiter → engx run

**Applies to:** `github.com/Harshmaury/Nexus`  
**Phase:** ADR-047 §3.2 — Execution gate (Phase 2)  
**Files:** `internal/plan/plan.go`, `cmd/engx/cmd_run.go`, `go.mod`

---

## 1. go.mod — add Arbiter dependency

```
require github.com/Harshmaury/Arbiter v0.1.0
```

After adding: `go mod tidy`

---

## 2. internal/plan/plan.go — add KindEnforce

### Locate this block:

```go
const (
	KindValidate StepKind = iota // pre-flight check — fail-hard on deny
	KindExecute                  // service call with expected side effect
	KindWait                     // poll loop until condition met or timeout
	KindObserve                  // read-only check — fail-open (warn, continue)
)
```

### Replace with:

```go
const (
	KindValidate StepKind = iota // pre-flight check — fail-hard on deny
	KindEnforce                  // architectural gate — fail-hard on violation (ADR-047)
	KindExecute                  // service call with expected side effect
	KindWait                     // poll loop until condition met or timeout
	KindObserve                  // read-only check — fail-open (warn, continue)
)
```

### Locate String() switch:

```go
func (k StepKind) String() string {
	switch k {
	case KindValidate:
		return "validate"
	case KindExecute:
		return "execute"
```

### Add KindEnforce case:

```go
	case KindEnforce:
		return "enforce"
```

---

## 3. cmd/engx/cmd_run.go — add Enforcing step

### Add import (top of file, alongside existing imports):

```go
arbiter "github.com/Harshmaury/Arbiter/api"
```

### Locate buildRunPlan:

```go
func buildRunPlan(socketPath, httpAddr, id string, timeoutSecs int) *plan.Plan {
	return plan.Build("run:"+id, []*plan.Step{
		{
			Label: "Validating",
			Kind:  plan.KindValidate,
			Run:   stepValidate(httpAddr, id),
		},
		{
			Label: "Starting",
			Kind:  plan.KindExecute,
			Run:   stepStart(socketPath, id),
		},
```

### Replace with (insert Enforcing step between Validating and Starting):

```go
func buildRunPlan(socketPath, httpAddr, id string, timeoutSecs int, skipEnforce bool) *plan.Plan {
	return plan.Build("run:"+id, []*plan.Step{
		{
			Label: "Validating",
			Kind:  plan.KindValidate,
			Run:   stepValidate(httpAddr, id),
		},
		{
			Label: "Enforcing",
			Kind:  plan.KindEnforce,
			Run:   stepEnforce(httpAddr, id, skipEnforce),
		},
		{
			Label: "Starting",
			Kind:  plan.KindExecute,
			Run:   stepStart(socketPath, id),
		},
```

### Update runProject signature and call site:

```go
func runProject(socketPath, httpAddr, id string, timeoutSecs int, dryRun bool, skipEnforce bool) error {
	cfg := plan.RunConfig{NexusAddr: httpAddr}
	p := buildRunPlan(socketPath, httpAddr, id, timeoutSecs, skipEnforce)
	if dryRun {
		plan.Print(p, os.Stdout)
		return nil
	}
	return plan.Run(context.Background(), p, os.Stdout, cfg)
}
```

### Update runCmd to add --skip-enforce flag:

```go
func runCmd(socketPath, httpAddr *string) *cobra.Command {
	var timeout int
	var dryRun bool
	var skipEnforce bool
	cmd := &cobra.Command{
		Use:   "run <project>",
		Short: "Start a project and confirm it is running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			return runProject(*socketPath, *httpAddr, id, timeout, dryRun, skipEnforce)
		},
	}
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 60, "seconds to wait for running state")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print execution plan without running")
	cmd.Flags().BoolVar(&skipEnforce, "skip-enforce", false, "bypass Arbiter gate (audit logged)")
	return cmd
}
```

### Add stepEnforce function (new function, anywhere in cmd_run.go):

```go
// stepEnforce runs the Arbiter architectural gate (ADR-047 §3.2).
// If skipEnforce is true, the gate is bypassed and a SYSTEM_ALERT is emitted.
func stepEnforce(httpAddr, projectID string, skipEnforce bool) plan.StepFunc {
	return func(ctx context.Context) plan.StepResult {
		if skipEnforce {
			arbiter.SkipEnforceAlert(httpAddr, "", projectID)
			return plan.StepResult{
				OK:      true,
				Skip:    true,
				Message: "bypassed (audit logged)",
			}
		}

		// Resolve project directory from nexus.yaml in registered path.
		// If directory cannot be determined, skip the gate fail-open.
		projectDir := resolveProjectDir(projectID)
		if projectDir == "" {
			return plan.StepResult{
				OK:      true,
				Skip:    true,
				Message: "skipped (project dir unresolvable)",
			}
		}

		report, err := arbiter.VerifyExecution(httpAddr, "", projectDir)
		if err != nil {
			// Fail-open: if Arbiter itself errors, log and continue.
			return plan.StepResult{
				OK:      true,
				Skip:    true,
				Message: "skipped (arbiter error)",
				Detail:  err.Error(),
			}
		}

		if report.OK() {
			return plan.StepResult{
				OK:      true,
				Message: fmt.Sprintf("✓ (%d rules)", len(report.Passed)),
			}
		}

		return plan.StepResult{
			OK:     false,
			Detail: arbiter.FormatReport(report),
			Err: &plan.UserError{
				What:     "Arbiter enforcement gate failed",
				Where:    "Enforcing",
				Why:      fmt.Sprintf("%d architectural violation(s) detected", len(report.Violations)),
				NextStep: "resolve violations and re-run, or use --skip-enforce to bypass (audited)",
			},
		}
	}
}

// resolveProjectDir returns the local filesystem path for a registered project.
// Returns "" if the path cannot be determined — callers skip the gate fail-open.
func resolveProjectDir(projectID string) string {
	// Convention: ~/.nexus/projects/<id>/path stored by engx register.
	// Simplified lookup: check common workspace path.
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(home, "workspace", "projects", "engx", "services", projectID)
	if _, err := os.Stat(filepath.Join(candidate, "nexus.yaml")); err == nil {
		return candidate
	}
	return ""
}
```

### Add to imports (if not already present):

```go
"path/filepath"
```

---

## 4. Verification

After applying the patch:

```bash
go build ./cmd/engx/
engx run atlas --dry-run
# Should show: 1 Validating  2 Enforcing  3 Starting  4 Waiting  5 Health

engx run atlas
# Enforcing step passes → proceeds normally

engx run atlas --skip-enforce
# Enforcing step shows "bypassed (audit logged)"
# SYSTEM_ALERT event emitted to Nexus
```

---

## 5. What this does NOT change

- `stepValidate` — unchanged (V-001..V-003 system/validate checks)
- Reconciler, daemon, any service — unchanged
- Existing plan steps — unchanged (indices shift by 1 due to KindEnforce insertion)
- `engx platform start` — not affected (does not use buildRunPlan)
