// @arbiter-project: arbiter
// @arbiter-path: cmd/arbiter/main.go
// Arbiter CLI — standalone enforcement tool (ADR-047).
//
// Usage:
//   arbiter verify [path]          — verify project at path (default: .)
//   arbiter verify ./...           — verify all projects in workspace
//   arbiter rules                  — list all rule IDs with descriptions
//   arbiter help                   — print usage
//
// Exit codes:
//   0 — all rules passed
//   1 — one or more violations found
//   2 — internal error (loader failure, bad args)
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	arbiter    "github.com/Harshmaury/Arbiter/api"
	arbiterapi "github.com/Harshmaury/Arbiter/internal/api"
)

func main() {
	// Optional HTTP server — starts only when ARBITER_HTTP_ADDR is set (ADR-048).
	if addr := os.Getenv("ARBITER_HTTP_ADDR"); addr != "" {
		token := os.Getenv("ARBITER_SERVICE_TOKEN")
		srv := arbiterapi.NewServer(addr, token)
		ctx, cancel := context.WithCancel(context.Background())
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() { <-sigCh; cancel() }()
		fmt.Fprintf(os.Stderr, "[arbiter] HTTP server listening on %s\n", addr)
		if err := srv.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[arbiter] server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "verify":
		runVerify()
	case "rules":
		runRules()
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "arbiter: unknown command %q\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Run: arbiter help")
		os.Exit(2)
	}
}

func runVerify() {
	dirs := []string{"."}
	if len(os.Args) > 2 {
		dirs = os.Args[2:]
	}

	// Expand ./... pattern
	if len(dirs) == 1 && dirs[0] == "./..." {
		dirs = findProjectDirs(".")
	}

	totalViolations := 0
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arbiter: %s: %v\n", dir, err)
			os.Exit(2)
		}

		report, err := arbiter.VerifyPackaging(abs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arbiter: %s: %v\n", dir, err)
			os.Exit(2)
		}

		if !report.OK() {
			fmt.Printf("\n  Project: %s\n", dir)
			fmt.Print(arbiter.FormatReport(report))
			totalViolations += len(report.Violations)
		}
	}

	if totalViolations > 0 {
		fmt.Printf("\narbiter: %d total violation(s) across %d project(s)\n",
			totalViolations, len(dirs))
		os.Exit(1)
	}

	fmt.Printf("arbiter: ✓ all %d project(s) passed\n", len(dirs))
}

func runRules() {
	rules := []struct{ id, family, desc string }{
		{"A-T-001", "Temporal", "nexus.yaml id backed by at least one ADR"},
		{"A-T-002", "Temporal", "nexus.yaml version field is present and non-empty"},
		{"A-T-003", "Temporal", "ADR files sequentially numbered (no gap >5)"},
		{"A-S-001", "Spatial", "No cross-service internal/ package imports"},
		{"A-S-002", "Spatial", "go.mod module name matches nexus.yaml id"},
		{"A-S-003", "Spatial", "No raw net/http calls in observer collector files"},
		{"A-A-001", "Authority", "Observer services never call write endpoints"},
		{"A-A-002", "Authority", "Forge calls only permitted Nexus endpoints"},
		{"A-A-003", "Authority", "nexus.yaml role is present and valid"},
		{"A-C-001", "Contract", "No hardcoded \"X-Service-Token\" outside Canon"},
		{"A-C-002", "Contract", "No hardcoded \"X-Trace-ID\" outside Canon"},
		{"A-C-003", "Contract", "No hardcoded \"X-Identity-Token\" outside Canon"},
		{"A-C-004", "Contract", "No local EventType constant definitions"},
		{"A-C-005", "Contract", "depends_on entries resolve to known service IDs"},
		{"A-C-006", "Contract", "Canon dependency in go.mod must be >= v1.0.0"},
	}

	fmt.Println("Arbiter rules (ADR-047)")
	fmt.Println(strings.Repeat("─", 70))
	for _, r := range rules {
		fmt.Printf("  %-10s  %-10s  %s\n", r.id, r.family, r.desc)
	}
}

func printHelp() {
	fmt.Print(`Arbiter — Centralized Architectural Enforcement Engine (ADR-047)

Usage:
  arbiter verify [path|./...]   verify project(s) against all static rules
  arbiter rules                 list all rule IDs and descriptions
  arbiter help                  show this message

Environment:
  ARBITER_GOVERNANCE_DIR   path to engx-governance/decisions/ (auto-detected if absent)

Exit codes:
  0   all rules passed
  1   violations found — packaging/execution blocked
  2   internal error

Rules: A-T (temporal) A-S (spatial) A-A (authority) A-C (contract)
`)
}

// findProjectDirs returns directories containing nexus.yaml under root.
func findProjectDirs(root string) []string {
	var dirs []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "nexus.yaml" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if len(dirs) == 0 {
		return []string{root}
	}
	return dirs
}
