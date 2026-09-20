package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/api"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
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
	reg := connector.New(st, cfg.Workspace.Root)
	rt.SetOnSuccess(func(ctx context.Context, task *store.Task, arts []store.Artifact) {
		reg.Notify(ctx, connector.Event{Task: task, Artifacts: arts})
	})
	sched := scheduler.New(st, rt, 20*time.Second)
	sched.Start(context.Background())
	srv := api.New(cfg, st, rt, sched, reg)
	log.Printf("work harness listening on %s", cfg.Server.Addr)
	if err := srv.Engine().Run(cfg.Server.Addr); err != nil {
		log.Fatal(err)
	}
}
