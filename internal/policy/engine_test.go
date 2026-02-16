package policy

import (
	"testing"

	"github.com/junlov/proxmox-ai/internal/proxmox"
)

func TestEvaluateRiskAndApprovalMappingTableDriven(t *testing.T) {
	engine := NewEngine()
	tests := []struct {
		name             string
		req              proxmox.ActionRequest
		wantRisk         string
		wantApproval     bool
		wantAllowedPlan  bool
		wantAllowedApply bool
	}{
		{
			name: "read vm low risk",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadVM,
				Target:      "vm/101",
			},
			wantRisk:         "low",
			wantApproval:     false,
			wantAllowedPlan:  true,
			wantAllowedApply: true,
		},
		{
			name: "start vm medium risk",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionStartVM,
				Target:      "vm/101",
			},
			wantRisk:         "medium",
			wantApproval:     false,
			wantAllowedPlan:  true,
			wantAllowedApply: true,
		},
		{
			name: "stop vm medium risk requires approval on apply",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionStopVM,
				Target:      "vm/101",
			},
			wantRisk:         "medium",
			wantApproval:     true,
			wantAllowedPlan:  true,
			wantAllowedApply: false,
		},
		{
			name: "delete vm high risk requires approval on apply",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionDeleteVM,
				Target:      "vm/101",
			},
			wantRisk:         "high",
			wantApproval:     true,
			wantAllowedPlan:  true,
			wantAllowedApply: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			planDecision, err := engine.EvaluateForPlan(tc.req)
			if err != nil {
				t.Fatalf("EvaluateForPlan returned error: %v", err)
			}
			if planDecision.RiskLevel != tc.wantRisk {
				t.Fatalf("unexpected plan risk level: got %q want %q", planDecision.RiskLevel, tc.wantRisk)
			}
			if planDecision.RequiresApproval != tc.wantApproval {
				t.Fatalf("unexpected plan approval flag: got %v want %v", planDecision.RequiresApproval, tc.wantApproval)
			}
			if planDecision.Allowed != tc.wantAllowedPlan {
				t.Fatalf("unexpected plan allowed: got %v want %v", planDecision.Allowed, tc.wantAllowedPlan)
			}

			applyDecision, err := engine.EvaluateForApply(tc.req)
			if err != nil {
				t.Fatalf("EvaluateForApply returned error: %v", err)
			}
			if applyDecision.RiskLevel != tc.wantRisk {
				t.Fatalf("unexpected apply risk level: got %q want %q", applyDecision.RiskLevel, tc.wantRisk)
			}
			if applyDecision.RequiresApproval != tc.wantApproval {
				t.Fatalf("unexpected apply approval flag: got %v want %v", applyDecision.RequiresApproval, tc.wantApproval)
			}
			if applyDecision.Allowed != tc.wantAllowedApply {
				t.Fatalf("unexpected apply allowed: got %v want %v", applyDecision.Allowed, tc.wantAllowedApply)
			}
		})
	}
}

func TestEvaluateForPlanAllowsHighRiskWithoutApproval(t *testing.T) {
	engine := NewEngine()
	decision, err := engine.EvaluateForPlan(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "node1/101",
	})
	if err != nil {
		t.Fatalf("EvaluateForPlan returned error: %v", err)
	}
	if !decision.Allowed {
		t.Fatal("plan should be allowed without approval for high-risk action")
	}
	if !decision.RequiresApproval {
		t.Fatal("high-risk action should require approval")
	}
	if decision.RiskLevel != "high" {
		t.Fatalf("unexpected risk level: %q", decision.RiskLevel)
	}
}

func TestEvaluateForApplyDeniesHighRiskWithoutApproval(t *testing.T) {
	engine := NewEngine()
	decision, err := engine.EvaluateForApply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "node1/101",
	})
	if err != nil {
		t.Fatalf("EvaluateForApply returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("apply should be denied without approval for high-risk action")
	}
	if !decision.RequiresApproval {
		t.Fatal("high-risk action should require approval")
	}
	if decision.Reason != "approval required before apply" {
		t.Fatalf("unexpected reason: %q", decision.Reason)
	}
}

func TestEvaluateForApplyAllowsHighRiskWithApproval(t *testing.T) {
	engine := NewEngine()
	decision, err := engine.EvaluateForApply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionMigrateVM,
		Target:      "node1/101",
		ApprovedBy:  "ops-user",
	})
	if err != nil {
		t.Fatalf("EvaluateForApply returned error: %v", err)
	}
	if !decision.Allowed {
		t.Fatal("apply should be allowed with approval metadata")
	}
	if !decision.RequiresApproval {
		t.Fatal("high-risk action should still report approval requirement")
	}
}

func TestEvaluateForApplyDeniesServiceImpactingWithoutApproval(t *testing.T) {
	engine := NewEngine()
	decision, err := engine.EvaluateForApply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionStopVM,
		Target:      "node1/101",
	})
	if err != nil {
		t.Fatalf("EvaluateForApply returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("stop action should be denied on apply without approval")
	}
	if decision.RiskLevel != "medium" {
		t.Fatalf("unexpected risk level: %q", decision.RiskLevel)
	}
}

func TestEvaluateValidationErrors(t *testing.T) {
	engine := NewEngine()
	_, err := engine.EvaluateForPlan(proxmox.ActionRequest{
		Action: proxmox.ActionReadVM,
		Target: "node1/101",
	})
	if err == nil {
		t.Fatal("expected validation error for missing environment")
	}
}
