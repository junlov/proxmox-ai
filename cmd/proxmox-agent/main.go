package main

import (
	"flag"
	"log"

	"github.com/junlov/proxmox-ai/internal/actions"
	"github.com/junlov/proxmox-ai/internal/config"
	"github.com/junlov/proxmox-ai/internal/policy"
	"github.com/junlov/proxmox-ai/internal/proxmox"
	"github.com/junlov/proxmox-ai/internal/server"
)

func main() {
	configPath := flag.String("config", "./config.example.json", "path to JSON config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client, err := proxmox.NewAPIClient(cfg.Environments)
	if err != nil {
		log.Fatalf("initialize proxmox client: %v", err)
	}
	engine := policy.NewEngine()
	runner := actions.NewRunner(engine, client, cfg.AuditLogPath)

	srv := server.New(cfg, runner)
	log.Printf("starting proxmox-agent on %s", cfg.ListenAddr)
	if err := srv.Start(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
