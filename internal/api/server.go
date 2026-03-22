// @arbiter-project: arbiter
// @arbiter-path: internal/api/server.go
// HTTP server for Arbiter remote verification (ADR-048).
// Starts only when ARBITER_HTTP_ADDR is set — zero change to existing behavior.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server is the Arbiter HTTP server.
type Server struct {
	srv          *http.Server
	serviceToken string
}

// NewServer creates an Arbiter HTTP server.
// serviceToken is required on all non-health endpoints (ADR-008).
func NewServer(addr, serviceToken string) *Server {
	s := &Server{serviceToken: serviceToken}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/arbiter/verify", s.authMiddleware(http.HandlerFunc(s.handleVerify)))
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(shutCtx)
	}()
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("arbiter http: %w", err)
	}
	return nil
}

// handleHealth handles GET /health — always unauthenticated.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","service":"arbiter"}`)
}

// authMiddleware enforces X-Service-Token on protected endpoints (ADR-008).
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.serviceToken != "" && r.Header.Get("X-Service-Token") != s.serviceToken {
			http.Error(w, `{"ok":false,"error":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
