package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type idempotencyRecord struct {
	payloadHash string
	statusCode  int
	contentType string
	body        []byte
}

type idempotencyStore struct {
	mu      sync.Mutex
	records map[string]idempotencyRecord
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{
		records: make(map[string]idempotencyRecord),
	}
}

func (s *idempotencyStore) Get(scope, key string) (idempotencyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[scope+"|"+key]
	if !ok {
		return idempotencyRecord{}, false
	}
	rec.body = append([]byte(nil), rec.body...)
	return rec, true
}

func (s *idempotencyStore) Put(scope, key string, rec idempotencyRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec.body = append([]byte(nil), rec.body...)
	s.records[scope+"|"+key] = rec
}

func hashActionRequest(req proxmox.ActionRequest) (string, error) {
	b, err := json.Marshal(struct {
		Environment    string             `json:"environment"`
		Action         proxmox.ActionType `json:"action"`
		Target         string             `json:"target"`
		Params         map[string]any     `json:"params,omitempty"`
		DryRun         bool               `json:"dry_run"`
		ApprovedBy     string             `json:"approved_by,omitempty"`
		ApprovalTicket string             `json:"approval_ticket,omitempty"`
		Reason         string             `json:"reason,omitempty"`
		ExpiresAt      string             `json:"expires_at,omitempty"`
	}{
		Environment:    req.Environment,
		Action:         req.Action,
		Target:         req.Target,
		Params:         req.Params,
		DryRun:         req.DryRun,
		ApprovedBy:     req.ApprovedBy,
		ApprovalTicket: req.ApprovalTicket,
		Reason:         req.Reason,
		ExpiresAt:      req.ExpiresAt,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
