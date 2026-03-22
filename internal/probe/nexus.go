// @arbiter-project: arbiter
// @arbiter-path: internal/probe/nexus.go
// Package probe contains the live Nexus query used by dynamic rules.
// ADR-047 §5.2 — Phase 2 (execution gate).
// Uses stdlib net/http — acceptable in tool packages (Arbiter is not an observer).
package probe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// FetchServiceIDs queries GET /services on Nexus and returns the set of
// registered service IDs. Returns empty map on any error so dynamic rules
// skip gracefully when Nexus is unreachable.
func FetchServiceIDs(nexusAddr, serviceToken string) map[string]bool {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, nexusAddr+"/services", nil)
	if err != nil {
		return map[string]bool{}
	}
	if serviceToken != "" {
		req.Header.Set("X-Service-Token", serviceToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]bool{}
	}
	defer resp.Body.Close()

	var envelope struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return map[string]bool{}
	}
	ids := make(map[string]bool, len(envelope.Data))
	for _, svc := range envelope.Data {
		ids[svc.ID] = true
	}
	return ids
}

// FetchADRFiles queries GET /system/adr on Nexus for the governance ADR list.
// Falls back to empty slice — callers skip A-T-001 gracefully.
func FetchADRFiles(nexusAddr, serviceToken string) []string {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, nexusAddr+"/system/adr", nil)
	if err != nil {
		return nil
	}
	if serviceToken != "" {
		req.Header.Set("X-Service-Token", serviceToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var envelope struct {
		OK   bool     `json:"ok"`
		Data []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil
	}
	return envelope.Data
}

// EmitSkipEnforceAlert fires a SYSTEM_ALERT event to Nexus when --skip-enforce
// is used (ADR-047 §3.2 — audited use). Fires in background; never blocks.
func EmitSkipEnforceAlert(nexusAddr, serviceToken, projectID string) {
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		body := fmt.Sprintf(
			`{"type":"SYSTEM_ALERT","source":"engx","component":"arbiter","outcome":"","payload":{"rule":"skip-enforce","project":%q,"message":"--skip-enforce used: Arbiter execution gate bypassed"}}`,
			projectID,
		)
		req, err := http.NewRequest(http.MethodPost, nexusAddr+"/events",
			strings.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if serviceToken != "" {
			req.Header.Set("X-Service-Token", serviceToken)
		}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
