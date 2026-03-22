// @arbiter-project: arbiter
// @arbiter-path: internal/loader/loader.go
// Package loader builds a ProjectContext from a filesystem path.
// ADR-047: loader is read-only — it never modifies the project directory.
// Uses stdlib only — no external YAML library required.
package loader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Harshmaury/Arbiter/internal/probe"
	"github.com/Harshmaury/Arbiter/internal/rules"
)

// LoadProject reads dir and returns a fully populated ProjectContext.
// It does not fail on missing optional files (go.mod, ADR directory);
// it populates what it finds and leaves the rest zero-valued.
func LoadProject(dir string) (*rules.ProjectContext, error) {
	ctx := &rules.ProjectContext{Dir: dir}

	// nexus.yaml (required)
	manifest, err := loadManifest(filepath.Join(dir, "nexus.yaml"))
	if err != nil {
		return nil, fmt.Errorf("arbiter: nexus.yaml: %w", err)
	}
	ctx.Manifest = manifest

	// go.mod module line (optional)
	ctx.GoMod = loadModuleLine(filepath.Join(dir, "go.mod"))

	// Source files
	ctx.SourceFiles, err = loadSourceFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("arbiter: source scan: %w", err)
	}

	// ADR files from sibling engx-governance directory (best-effort)
	ctx.ADRFiles = discoverADRFiles(dir)

	return ctx, nil
}

// LoadProjectWithNexus extends LoadProject by populating KnownServiceIDs
// from a live Nexus query. Used by the execution gate (Phase 2).
func LoadProjectWithNexus(dir, nexusAddr, serviceToken string) (*rules.ProjectContext, error) {
	ctx, err := LoadProject(dir)
	if err != nil {
		return nil, err
	}
	ctx.KnownServiceIDs = fetchKnownServiceIDs(nexusAddr, serviceToken)
	return ctx, nil
}

// ── nexus.yaml — minimal key: value parser ───────────────────────────────────
// nexus.yaml uses simple flat YAML (no nesting beyond depends_on list).
// stdlib bufio scanner is sufficient; avoids external yaml dependency.

func loadManifest(path string) (*rules.NexusManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := &rules.NexusManifest{}
	scanner := bufio.NewScanner(f)
	inDependsOn := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if inDependsOn {
			if strings.HasPrefix(trimmed, "- ") {
				m.DependsOn = append(m.DependsOn, strings.TrimPrefix(trimmed, "- "))
				continue
			}
			inDependsOn = false
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "id":
			m.ID = val
		case "name":
			m.Name = val
		case "role":
			m.Role = val
		case "version":
			m.Version = val
		case "description":
			m.Description = val
		case "depends_on":
			inDependsOn = true
		}
	}
	return m, scanner.Err()
}

// ── go.mod ────────────────────────────────────────────────────────────────────

func loadModuleLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return ""
}

// ── source files ─────────────────────────────────────────────────────────────

var skipDirs = map[string]bool{
	"vendor":   true,
	"testdata": true,
	".git":     true,
	"node_modules": true,
}

func loadSourceFiles(dir string) ([]rules.SourceFile, error) {
	var files []rules.SourceFile
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}
		rel, _ := filepath.Rel(dir, path)
		files = append(files, rules.SourceFile{
			Path:    rel,
			Content: string(content),
		})
		return nil
	})
	return files, err
}

// ── ADR discovery ─────────────────────────────────────────────────────────────

// discoverADRFiles attempts to find engx-governance/decisions/ relative to dir.
// It walks up the directory tree to find the governance repository.
func discoverADRFiles(projectDir string) []string {
	// Try common layout: all services in ~/workspace/projects/engx/services/<id>
	// and governance at ~/workspace/projects/engx/engx-governance/
	candidates := []string{
		filepath.Join(projectDir, "..", "engx-governance", "architecture", "decisions"),
		filepath.Join(projectDir, "..", "..", "engx-governance", "architecture", "decisions"),
		filepath.Join(projectDir, "..", "..", "..", "engx-governance", "architecture", "decisions"),
	}
	// Also check ARBITER_GOVERNANCE_DIR env var.
	if envDir := os.Getenv("ARBITER_GOVERNANCE_DIR"); envDir != "" {
		candidates = append([]string{envDir}, candidates...)
	}

	for _, candidate := range candidates {
		entries, err := os.ReadDir(candidate)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "ADR-") {
				names = append(names, e.Name())
			}
		}
		if len(names) > 0 {
			return names
		}
	}
	return nil
}

// ── Nexus live query ──────────────────────────────────────────────────────────

func fetchKnownServiceIDs(nexusAddr, serviceToken string) map[string]bool {
	return probe.FetchServiceIDs(nexusAddr, serviceToken)
}
