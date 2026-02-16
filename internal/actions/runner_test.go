package actions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/junlov/proxmox-ai/internal/policy"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type fakeClient struct {
	calls int
}

func (c *fakeClient) Execute(req proxmox.ActionRequest) (proxmox.ActionResult, error) {
	c.calls++
	return proxmox.ActionResult{Status: "accepted", Message: "ok"}, nil
}

func TestPlanAllowsHighRiskWithoutApproval(t *testing.T) {
	client := &fakeClient{}
	runner := NewRunner(policy.NewEngine(), client, "")

	resp, err := runner.Plan(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "node1/101",
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if !resp.Decision.Allowed {
		t.Fatal("plan should allow high-risk action without approval")
	}
	if !resp.Decision.RequiresApproval {
		t.Fatal("plan should indicate approval is required")
	}
	if client.calls != 0 {
		t.Fatalf("expected no execution calls during plan, got %d", client.calls)
	}
}

func TestApplyDeniesHighRiskWithoutApproval(t *testing.T) {
	client := &fakeClient{}
	runner := NewRunner(policy.NewEngine(), client, "")

	_, err := runner.Apply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "node1/101",
	})
	if err == nil {
		t.Fatal("expected apply to be denied without approval")
	}
	if !strings.Contains(err.Error(), "request denied by policy") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no execution call for denied apply, got %d", client.calls)
	}
}

func TestApplyExecutesWithApproval(t *testing.T) {
	client := &fakeClient{}
	runner := NewRunner(policy.NewEngine(), client, "")

	resp, err := runner.Apply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "node1/101",
		ApprovedBy:  "ops-user",
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected one execution call, got %d", client.calls)
	}
	if resp.Result.Status != "accepted" {
		t.Fatalf("unexpected apply status: %q", resp.Result.Status)
	}
}

func TestRunnerWritesAuditRecordsForPlanDeniedAndApply(t *testing.T) {
	client := &fakeClient{}
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	runner := NewRunner(policy.NewEngine(), client, auditPath)

	_, err := runner.Plan(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "vm/101",
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	_, err = runner.Apply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "vm/101",
	})
	if err == nil {
		t.Fatal("expected denied apply error")
	}

	_, err = runner.Apply(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionDeleteVM,
		Target:      "vm/101",
		ApprovedBy:  "ops-user",
	})
	if err != nil {
		t.Fatalf("approved Apply returned error: %v", err)
	}

	b, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 audit records, got %d", len(lines))
	}

	var kinds []string
	for _, line := range lines {
		var record struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to decode audit record %q: %v", line, err)
		}
		kinds = append(kinds, record.Kind)
	}

	wantKinds := []string{"plan", "apply_denied", "apply"}
	for i, want := range wantKinds {
		if kinds[i] != want {
			t.Fatalf("unexpected audit kind at index %d: got %q want %q", i, kinds[i], want)
		}
	}
	if client.calls != 1 {
		t.Fatalf("expected single execution call for approved apply, got %d", client.calls)
	}
}

func TestRunnerAuditIncludesActorIdentity(t *testing.T) {
	client := &fakeClient{}
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	runner := NewRunner(policy.NewEngine(), client, auditPath)

	_, err := runner.Plan(proxmox.ActionRequest{
		Environment: "home",
		Action:      proxmox.ActionReadVM,
		Target:      "vm/101",
		Actor:       "test-agent",
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	b, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	var record struct {
		Actor string `json:"actor"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(b))), &record); err != nil {
		t.Fatalf("decode audit record: %v", err)
	}
	if record.Actor != "test-agent" {
		t.Fatalf("expected actor %q, got %q", "test-agent", record.Actor)
	}
}
