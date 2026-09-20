package main

import (
	"context"
	"flag"
	"log"

	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/api"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	st, err := store.Open(cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	skills, err := agent.LoadSkills(cfg.Skills.Dir)
	if err != nil {
		log.Fatalf("load skills: %v", err)
	}

	var factory *agent.Factory
	if err := agent.Ready(cfg.LLM); err == nil {
		factory, err = agent.NewFactory(context.Background(), cfg)
		if err != nil {
			log.Printf("llm factory not ready: %v", err)
		}
	} else {
		log.Printf("llm not configured yet: %v", err)
	}

	rt := harness.NewRuntime(cfg, st, factory, skills)
	srv := api.New(cfg, st, rt)
	log.Printf("work harness listening on %s", cfg.Server.Addr)
	if err := srv.Engine().Run(cfg.Server.Addr); err != nil {
		log.Fatal(err)
	}
}
