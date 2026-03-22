// @arbiter-project: arbiter
// @arbiter-path: internal/probe/nexus.go
// Package probe contains the live Nexus query used by dynamic rules.
// ADR-047 §5.2 — Phase 2 (execution gate).
// Uses stdlib net/http — acceptable in tool packages (Arbiter is not an observer).
//
// CW-4-fix: EmitSkipEnforceAlert now marshals canonevents.SystemAlertPayload
// instead of constructing a raw JSON string. Schema is enforced at compile time.
package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	canonevents "github.com/Harshmaury/Canon/events"
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
//
// CW-4-fix: payload is marshalled from canonevents.SystemAlertPayload —
// no more raw JSON string construction. Schema is enforced at compile time.
func EmitSkipEnforceAlert(nexusAddr, serviceToken, projectID string) {
	go func() {
		payload := canonevents.SystemAlertPayload{
			Rule:      canonevents.AlertRuleSkipEnforce,
			ProjectID: projectID,
			Message:   "--skip-enforce used: Arbiter execution gate bypassed",
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return // marshal failure is not recoverable — skip silently
		}

		event := struct {
			Type      string `json:"type"`
			Source    string `json:"source"`
			Component string `json:"component"`
			Outcome   string `json:"outcome"`
			Payload   string `json:"payload"`
		}{
			Type:      "SYSTEM_ALERT",
			Source:    "engx",
			Component: "arbiter",
			Outcome:   "",
			Payload:   string(payloadJSON),
		}
		body, err := json.Marshal(event)
		if err != nil {
			return
		}

		client := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequest(http.MethodPost, nexusAddr+"/events",
			bytes.NewReader(body))
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

// buildPayloadJSON is the canonical helper for serialising a typed payload
// to the JSON string stored in EventDTO.Payload.
// Exported for use by other Arbiter probes if needed in future.
func buildPayloadJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	return string(b), nil
}
