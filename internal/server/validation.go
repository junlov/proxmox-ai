package server

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/junlov/proxmox-ai/internal/config"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

var (
	vmTargetPattern         = regexp.MustCompile(`^vm/[0-9]+$`)
	inventoryTargetPattern  = regexp.MustCompile(`^inventory/(all|running)$`)
	taskStatusTargetPattern = regexp.MustCompile(`^task/status$`)
	taskListTargetPattern   = regexp.MustCompile(`^task/list$`)
	storageTargetPattern    = regexp.MustCompile(`^storage/[A-Za-z0-9._:-]+$`)
	firewallTargetPattern   = regexp.MustCompile(`^firewall/(cluster|node/[A-Za-z0-9._-]+|vm/[0-9]+)$`)
	approvedByPattern       = regexp.MustCompile(`^[A-Za-z0-9._:@/\-]{3,128}$`)
	approvalTicketPattern   = regexp.MustCompile(`^[A-Za-z0-9._:\-]{3,128}$`)
)

type requestValidator struct {
	environments map[string]struct{}
	actions      map[proxmox.ActionType]struct{}
}

func newRequestValidator(cfg config.Config) *requestValidator {
	envs := make(map[string]struct{}, len(cfg.Environments))
	for _, env := range cfg.Environments {
		envs[env.Name] = struct{}{}
	}
	return &requestValidator{
		environments: envs,
		actions: map[proxmox.ActionType]struct{}{
			proxmox.ActionReadVM:         {},
			proxmox.ActionReadInventory:  {},
			proxmox.ActionReadTaskStatus: {},
			proxmox.ActionReadTasks:      {},
			proxmox.ActionStartVM:        {},
			proxmox.ActionStopVM:         {},
			proxmox.ActionSnapshotVM:     {},
			proxmox.ActionCloneVM:        {},
			proxmox.ActionMigrateVM:      {},
			proxmox.ActionDeleteVM:       {},
			proxmox.ActionStorageEdit:    {},
			proxmox.ActionFirewallEdit:   {},
		},
	}
}

func (v *requestValidator) ValidateActionRequest(req proxmox.ActionRequest) error {
	if strings.TrimSpace(req.Environment) == "" {
		return fmt.Errorf("environment is required")
	}
	if _, ok := v.environments[req.Environment]; !ok {
		return fmt.Errorf("unknown environment %q", req.Environment)
	}
	if strings.TrimSpace(string(req.Action)) == "" {
		return fmt.Errorf("action is required")
	}
	if _, ok := v.actions[req.Action]; !ok {
		return fmt.Errorf("unsupported action %q", req.Action)
	}
	if strings.TrimSpace(req.Target) == "" {
		return fmt.Errorf("target is required")
	}
	if err := validateTargetByAction(req.Action, req.Target); err != nil {
		return err
	}
	if err := validateApprovalMetadata(req); err != nil {
		return err
	}
	return nil
}

func validateTargetByAction(action proxmox.ActionType, target string) error {
	switch action {
	case proxmox.ActionReadTasks:
		if !taskListTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected task/list", action)
		}
	case proxmox.ActionReadTaskStatus:
		if !taskStatusTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected task/status", action)
		}
	case proxmox.ActionReadInventory:
		if !inventoryTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected inventory/all or inventory/running", action)
		}
	case proxmox.ActionReadVM,
		proxmox.ActionStartVM,
		proxmox.ActionStopVM,
		proxmox.ActionSnapshotVM,
		proxmox.ActionCloneVM,
		proxmox.ActionMigrateVM,
		proxmox.ActionDeleteVM:
		if !vmTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected vm/<id>", action)
		}
	case proxmox.ActionStorageEdit:
		if !storageTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected storage/<name>", action)
		}
	case proxmox.ActionFirewallEdit:
		if !firewallTargetPattern.MatchString(target) {
			return fmt.Errorf("invalid target for %q: expected firewall/cluster, firewall/node/<name>, or firewall/vm/<id>", action)
		}
	}
	return nil
}

func validateApprovalMetadata(req proxmox.ActionRequest) error {
	approvedBy := strings.TrimSpace(req.ApprovedBy)
	approvalTicket := strings.TrimSpace(req.ApprovalTicket)
	reason := strings.TrimSpace(req.Reason)
	expiresAt := strings.TrimSpace(req.ExpiresAt)

	if approvedBy != "" && !approvedByPattern.MatchString(approvedBy) {
		return fmt.Errorf("invalid approved_by format")
	}
	if approvalTicket != "" && !approvalTicketPattern.MatchString(approvalTicket) {
		return fmt.Errorf("invalid approval_ticket format")
	}
	if reason != "" && len(reason) < 8 {
		return fmt.Errorf("reason must be at least 8 characters when provided")
	}
	if expiresAt != "" {
		if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
			return fmt.Errorf("expires_at must be RFC3339 format")
		}
	}
	if approvedBy == "" && (approvalTicket != "" || reason != "" || expiresAt != "") {
		return fmt.Errorf("approved_by is required when approval metadata is provided")
	}
	return nil
}
