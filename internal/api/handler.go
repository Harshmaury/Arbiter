// @arbiter-project: arbiter
// @arbiter-path: internal/api/handler.go
// POST /arbiter/verify handler (ADR-048).
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	arbiter "github.com/Harshmaury/Arbiter/api"
	"github.com/Harshmaury/Arbiter/internal/rules"
)

// VerifyRequest is the POST /arbiter/verify request body.
type VerifyRequest struct {
	ProjectDir string `json:"project_dir"`
	Mode       string `json:"mode"` // "packaging" | "execution"
}

// VerifyData is the response data payload.
type VerifyData struct {
	Passed      bool           `json:"passed"`
	Violations  []ViolationDTO `json:"violations"`
	PassedRules []string       `json:"passed_rules"`
	EvaluatedAt string         `json:"evaluated_at"`
}

// ViolationDTO is one rule violation.
type ViolationDTO struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Location string `json:"location"`
	Message  string `json:"message"`
	Hint     string `json:"hint"`
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ProjectDir == "" {
		respondError(w, http.StatusBadRequest, "project_dir is required")
		return
	}
	if req.Mode == "" {
		req.Mode = "packaging"
	}

	var report *rules.Report
	var err error

	switch req.Mode {
	case "packaging":
		report, err = arbiter.VerifyPackaging(req.ProjectDir)
	case "execution":
		report, err = arbiter.VerifyExecution("", "", req.ProjectDir)
	default:
		respondError(w, http.StatusBadRequest,
			fmt.Sprintf("unknown mode %q — use 'packaging' or 'execution'", req.Mode))
		return
	}

	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	data := toVerifyData(report)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": data})
}

func toVerifyData(r *rules.Report) VerifyData {
	vdtos := make([]ViolationDTO, 0, len(r.Violations))
	for _, v := range r.Violations {
		vdtos = append(vdtos, ViolationDTO{
			RuleID:   v.RuleID,
			Severity: v.Severity,
			Location: v.Location,
			Message:  v.Message,
			Hint:     v.Hint,
		})
	}
	return VerifyData{
		Passed:      r.OK(),
		Violations:  vdtos,
		PassedRules: r.Passed,
		EvaluatedAt: r.EvaluatedAt.Format(time.RFC3339),
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}
