package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/junlov/proxmox-ai/internal/actions"
	"github.com/junlov/proxmox-ai/internal/config"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type Server struct {
	cfg       config.Config
	runner    *actions.Runner
	validator *requestValidator
	idem      *idempotencyStore
	authToken string
}

func New(cfg config.Config, runner *actions.Runner) *Server {
	return &Server{
		cfg:       cfg,
		runner:    runner,
		validator: newRequestValidator(cfg),
		idem:      newIdempotencyStore(),
		authToken: strings.TrimSpace(os.Getenv("PROXMOX_AGENT_API_TOKEN")),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v1/environments", s.environments)
	mux.HandleFunc("/v1/inventory", s.inventory)
	mux.HandleFunc("/v1/actions/plan", s.plan)
	mux.HandleFunc("/v1/actions/apply", s.apply)

	return http.ListenAndServe(s.cfg.ListenAddr, s.logRequests(mux))
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) environments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.requireAuth(w, r); !ok {
		return
	}
	envs := make([]map[string]string, 0, len(s.cfg.Environments))
	for _, env := range s.cfg.Environments {
		envs = append(envs, map[string]string{
			"name":     env.Name,
			"base_url": env.BaseURL,
			"token_id": env.TokenID,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"environments": envs})
}

func (s *Server) inventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := s.requireAuth(w, r)
	if !ok {
		return
	}

	environment := strings.TrimSpace(r.URL.Query().Get("environment"))
	if environment == "" {
		http.Error(w, "environment query parameter is required", http.StatusBadRequest)
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		state = "all"
	}

	target := "inventory/" + state
	req := proxmox.ActionRequest{
		Environment: environment,
		Action:      proxmox.ActionReadInventory,
		Target:      target,
		Actor:       actor,
	}
	if err := s.validator.ValidateActionRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, handled := s.tryReplayIdempotent(w, r, req); handled {
		return
	}

	planResp, err := s.runner.Plan(req)
	if err != nil {
		s.writeAndStoreError(w, r, req, http.StatusBadRequest, err.Error())
		return
	}
	applyResp, err := s.runner.Apply(req)
	if err != nil {
		s.writeAndStoreError(w, r, req, http.StatusForbidden, err.Error())
		return
	}

	s.writeAndStoreJSON(w, r, req, http.StatusOK, map[string]any{
		"request": req,
		"plan":    planResp.Decision,
		"result":  applyResp.Result,
	})
}

func (s *Server) plan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req proxmox.ActionRequest
	if err := decodeStrictJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := s.validator.ValidateActionRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Actor = actor
	if _, handled := s.tryReplayIdempotent(w, r, req); handled {
		return
	}

	resp, err := s.runner.Plan(req)
	if err != nil {
		s.writeAndStoreError(w, r, req, http.StatusBadRequest, err.Error())
		return
	}
	s.writeAndStoreJSON(w, r, req, http.StatusOK, resp)
}

func (s *Server) apply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req proxmox.ActionRequest
	if err := decodeStrictJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := s.validator.ValidateActionRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Actor = actor
	if _, handled := s.tryReplayIdempotent(w, r, req); handled {
		return
	}

	resp, err := s.runner.Apply(req)
	if err != nil {
		s.writeAndStoreError(w, r, req, http.StatusForbidden, err.Error())
		return
	}
	s.writeAndStoreJSON(w, r, req, http.StatusOK, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) tryReplayIdempotent(w http.ResponseWriter, r *http.Request, req proxmox.ActionRequest) (replayed bool, handled bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return false, false
	}
	hash, err := hashActionRequest(req)
	if err != nil {
		http.Error(w, "failed to hash request", http.StatusInternalServerError)
		return false, true
	}
	if rec, ok := s.idem.Get(r.URL.Path, key); ok {
		if rec.payloadHash != hash {
			http.Error(w, "idempotency key reused with different payload", http.StatusConflict)
			return false, true
		}
		s.writeRaw(w, rec.statusCode, rec.contentType, rec.body)
		return true, true
	}
	return false, false
}

func (s *Server) writeAndStoreJSON(w http.ResponseWriter, r *http.Request, req proxmox.ActionRequest, status int, body any) {
	respBody, contentType := marshalJSONBody(body)
	s.writeRaw(w, status, contentType, respBody)
	s.storeIdempotencyResponse(r, req, status, contentType, respBody)
}

func (s *Server) writeAndStoreError(w http.ResponseWriter, r *http.Request, req proxmox.ActionRequest, status int, message string) {
	contentType := "text/plain; charset=utf-8"
	body := []byte(message + "\n")
	s.writeRaw(w, status, contentType, body)
	s.storeIdempotencyResponse(r, req, status, contentType, body)
}

func (s *Server) storeIdempotencyResponse(r *http.Request, req proxmox.ActionRequest, status int, contentType string, body []byte) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return
	}
	hash, err := hashActionRequest(req)
	if err != nil {
		return
	}
	s.idem.Put(r.URL.Path, key, idempotencyRecord{
		payloadHash: hash,
		statusCode:  status,
		contentType: contentType,
		body:        body,
	})
}

func marshalJSONBody(v any) ([]byte, string) {
	b, err := json.Marshal(v)
	if err != nil {
		fallback := []byte(fmt.Sprintf(`{"error":"failed to marshal response: %s"}`, err.Error()))
		return fallback, "application/json"
	}
	b = append(b, '\n')
	return b, "application/json"
}

func (s *Server) writeRaw(w http.ResponseWriter, status int, contentType string, body []byte) {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(status)
	_, _ = io.Copy(w, bytes.NewReader(body))
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	if s.authToken == "" {
		http.Error(w, "server auth token is not configured", http.StatusServiceUnavailable)
		return "", false
	}
	rawAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(rawAuth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(rawAuth, "Bearer "))
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) != 1 {
		http.Error(w, "invalid bearer token", http.StatusUnauthorized)
		return "", false
	}

	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if actor == "" {
		actor = "authenticated"
	}
	return actor, true
}

func decodeStrictJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}
