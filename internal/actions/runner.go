package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/junlov/proxmox-ai/internal/policy"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type PlanResponse struct {
	Request  proxmox.ActionRequest `json:"request"`
	Decision policy.Decision       `json:"decision"`
}

type ApplyResponse struct {
	Request  proxmox.ActionRequest `json:"request"`
	Decision policy.Decision       `json:"decision"`
	Result   proxmox.ActionResult  `json:"result"`
}

type Runner struct {
	policy  *policy.Engine
	client  proxmox.Client
	auditTo string
}

func NewRunner(policyEngine *policy.Engine, client proxmox.Client, auditPath string) *Runner {
	return &Runner{policy: policyEngine, client: client, auditTo: auditPath}
}

func (r *Runner) Plan(req proxmox.ActionRequest) (PlanResponse, error) {
	decision, err := r.policy.EvaluateForPlan(req)
	if err != nil {
		return PlanResponse{}, err
	}
	if err := r.audit("plan", req, decision, nil); err != nil {
		return PlanResponse{}, err
	}
	return PlanResponse{Request: req, Decision: decision}, nil
}

func (r *Runner) Apply(req proxmox.ActionRequest) (ApplyResponse, error) {
	decision, err := r.policy.EvaluateForApply(req)
	if err != nil {
		return ApplyResponse{}, err
	}
	if !decision.Allowed {
		if err := r.audit("apply_denied", req, decision, nil); err != nil {
			return ApplyResponse{}, err
		}
		return ApplyResponse{}, fmt.Errorf("request denied by policy: %s", decision.Reason)
	}
	result, err := r.client.Execute(req)
	if err != nil {
		return ApplyResponse{}, err
	}
	if err := r.audit("apply", req, decision, &result); err != nil {
		return ApplyResponse{}, err
	}
	return ApplyResponse{Request: req, Decision: decision, Result: result}, nil
}

func (r *Runner) audit(kind string, req proxmox.ActionRequest, decision policy.Decision, result *proxmox.ActionResult) error {
	if r.auditTo == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.auditTo), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(r.auditTo, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	record := map[string]any{
		"ts":       time.Now().UTC().Format(time.RFC3339),
		"kind":     kind,
		"actor":    req.Actor,
		"request":  req,
		"decision": decision,
	}
	if result != nil {
		record["result"] = result
	}
	enc := json.NewEncoder(f)
	return enc.Encode(record)
}
