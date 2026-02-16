package server

import (
	"testing"

	"github.com/junlov/proxmox-ai/internal/config"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

func TestValidateActionRequest(t *testing.T) {
	v := newRequestValidator(config.Config{
		Environments: []config.Environment{
			{Name: "home"},
			{Name: "cloud"},
		},
	})

	tests := []struct {
		name    string
		req     proxmox.ActionRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadVM,
				Target:      "vm/100",
			},
		},
		{
			name: "valid inventory target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadInventory,
				Target:      "inventory/running",
			},
		},
		{
			name: "valid task status target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadTaskStatus,
				Target:      "task/status",
			},
		},
		{
			name: "valid task list target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadTasks,
				Target:      "task/list",
			},
		},
		{
			name: "valid storage target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionStorageEdit,
				Target:      "storage/local-lvm",
			},
		},
		{
			name: "valid clone target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionCloneVM,
				Target:      "vm/103",
			},
		},
		{
			name: "valid firewall node target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionFirewallEdit,
				Target:      "firewall/node/pve1",
			},
		},
		{
			name: "valid approval metadata",
			req: proxmox.ActionRequest{
				Environment:    "home",
				Action:         proxmox.ActionDeleteVM,
				Target:         "vm/100",
				ApprovedBy:     "ops-user",
				ApprovalTicket: "CHG-2026-001",
				Reason:         "approved by on-call lead",
				ExpiresAt:      "2026-02-16T12:00:00Z",
			},
		},
		{
			name: "missing environment",
			req: proxmox.ActionRequest{
				Action: proxmox.ActionReadVM,
				Target: "vm/100",
			},
			wantErr: true,
		},
		{
			name: "unknown environment",
			req: proxmox.ActionRequest{
				Environment: "dev",
				Action:      proxmox.ActionReadVM,
				Target:      "vm/100",
			},
			wantErr: true,
		},
		{
			name: "missing action",
			req: proxmox.ActionRequest{
				Environment: "home",
				Target:      "vm/100",
			},
			wantErr: true,
		},
		{
			name: "missing target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadVM,
			},
			wantErr: true,
		},
		{
			name: "invalid vm target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadVM,
				Target:      "100",
			},
			wantErr: true,
		},
		{
			name: "invalid inventory target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadInventory,
				Target:      "inventory/active",
			},
			wantErr: true,
		},
		{
			name: "invalid task status target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadTaskStatus,
				Target:      "task/pve",
			},
			wantErr: true,
		},
		{
			name: "invalid task list target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionReadTasks,
				Target:      "task/all",
			},
			wantErr: true,
		},
		{
			name: "invalid storage target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionStorageEdit,
				Target:      "vm/100",
			},
			wantErr: true,
		},
		{
			name: "invalid firewall target",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionFirewallEdit,
				Target:      "firewall/datacenter",
			},
			wantErr: true,
		},
		{
			name: "unsupported action",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionType("do_everything"),
				Target:      "vm/100",
			},
			wantErr: true,
		},
		{
			name: "invalid approved_by format",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionDeleteVM,
				Target:      "vm/100",
				ApprovedBy:  "bad approver",
			},
			wantErr: true,
		},
		{
			name: "invalid approval_ticket format",
			req: proxmox.ActionRequest{
				Environment:    "home",
				Action:         proxmox.ActionDeleteVM,
				Target:         "vm/100",
				ApprovedBy:     "ops-user",
				ApprovalTicket: "bad ticket !",
			},
			wantErr: true,
		},
		{
			name: "reason too short",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionDeleteVM,
				Target:      "vm/100",
				ApprovedBy:  "ops-user",
				Reason:      "short",
			},
			wantErr: true,
		},
		{
			name: "invalid expires_at",
			req: proxmox.ActionRequest{
				Environment: "home",
				Action:      proxmox.ActionDeleteVM,
				Target:      "vm/100",
				ApprovedBy:  "ops-user",
				ExpiresAt:   "tomorrow",
			},
			wantErr: true,
		},
		{
			name: "approval metadata requires approved_by",
			req: proxmox.ActionRequest{
				Environment:    "home",
				Action:         proxmox.ActionDeleteVM,
				Target:         "vm/100",
				ApprovalTicket: "CHG-2026-001",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateActionRequest(tt.req)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
