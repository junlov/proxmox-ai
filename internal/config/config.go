package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Environment struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	TokenID        string `json:"token_id"`
	TokenSecretEnv string `json:"token_secret_env"`
}

type Config struct {
	ListenAddr   string        `json:"listen_addr"`
	AuditLogPath string        `json:"audit_log_path"`
	Environments []Environment `json:"environments"`
}

func Load(path string) (Config, error) {
	var cfg Config

	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.ListenAddr == "" {
		return cfg, fmt.Errorf("listen_addr is required")
	}
	if len(cfg.Environments) == 0 {
		return cfg, fmt.Errorf("at least one environment is required")
	}
	for _, env := range cfg.Environments {
		if env.Name == "" || env.BaseURL == "" || env.TokenID == "" || env.TokenSecretEnv == "" {
			return cfg, fmt.Errorf("invalid environment config for %q", env.Name)
		}
	}
	if cfg.AuditLogPath == "" {
		cfg.AuditLogPath = "./data/audit.log"
	}
	return cfg, nil
}
