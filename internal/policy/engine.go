package policy

import (
	"fmt"

	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type Decision struct {
	Allowed          bool   `json:"allowed"`
	RiskLevel        string `json:"risk_level"`
	RequiresApproval bool   `json:"requires_approval"`
	Reason           string `json:"reason"`
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) EvaluateForPlan(req proxmox.ActionRequest) (Decision, error) {
	return e.evaluate(req, false)
}

func (e *Engine) EvaluateForApply(req proxmox.ActionRequest) (Decision, error) {
	return e.evaluate(req, true)
}

func (e *Engine) evaluate(req proxmox.ActionRequest, enforceApproval bool) (Decision, error) {
	risk := "low"
	requiresApproval := false
	reason := "read/safe operation"

	switch req.Action {
	case proxmox.ActionDeleteVM, proxmox.ActionMigrateVM, proxmox.ActionStorageEdit, proxmox.ActionFirewallEdit:
		risk = "high"
		requiresApproval = true
		reason = "high-impact operation"
	case proxmox.ActionStopVM:
		risk = "medium"
		requiresApproval = true
		reason = "service-impacting operation"
	case proxmox.ActionStartVM, proxmox.ActionSnapshotVM:
		risk = "medium"
		reason = "state-changing operation"
	}

	if requiresApproval && enforceApproval && req.ApprovedBy == "" {
		return Decision{Allowed: false, RiskLevel: risk, RequiresApproval: true, Reason: "approval required before apply"}, nil
	}
	if req.Environment == "" || req.Target == "" {
		return Decision{}, fmt.Errorf("environment and target are required")
	}

	return Decision{Allowed: true, RiskLevel: risk, RequiresApproval: requiresApproval, Reason: reason}, nil
}
